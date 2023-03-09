/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package cache

import (
	"context"
	svcimpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceimport/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/discovery"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayCache struct {
	client client.Client
	cache  cache.Cache

	processors map[ProcessorType]Processor

	gatewayclass   *client.ObjectKey
	gateways       map[string]client.ObjectKey // ns -> gateway
	services       map[client.ObjectKey]bool
	serviceimports map[client.ObjectKey]bool
	endpoints      map[client.ObjectKey]bool
	endpointslices map[client.ObjectKey]map[client.ObjectKey]bool // svc -> endpointslices
	namespaces     map[string]bool
	httproutes     map[client.ObjectKey]bool
}

type GatewayCacheConfig struct {
	Client client.Client
	Cache  cache.Cache
}

func NewGatewayCache(config GatewayCacheConfig) *GatewayCache {
	return &GatewayCache{
		client: config.Client,
		cache:  config.Cache,

		processors: map[ProcessorType]Processor{
			ServicesProcessorType:       &ServicesProcessor{},
			ServiceImportsProcessorType: &ServiceImportsProcessor{},
			EndpointsProcessorType:      &EndpointsProcessor{},
			EndpointSlicesProcessorType: &EndpointSlicesProcessor{},
			NamespacesProcessorType:     &NamespacesProcessor{},
			GatewayClassesProcessorType: &GatewayClassesProcessor{},
			GatewaysProcessorType:       &GatewaysProcessor{},
			HTTPRoutesProcessorType:     &HTTPRoutesProcessor{},
		},

		gateways:       make(map[string]client.ObjectKey),
		services:       make(map[client.ObjectKey]bool),
		serviceimports: make(map[client.ObjectKey]bool),
		endpoints:      make(map[client.ObjectKey]bool),
		endpointslices: make(map[client.ObjectKey]map[client.ObjectKey]bool),
		namespaces:     make(map[string]bool),
		httproutes:     make(map[client.ObjectKey]bool),
	}
}

func (c *GatewayCache) Insert(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Insert(obj, c)
	}

	return false
}

func (c *GatewayCache) Delete(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Delete(obj, c)
	}

	return false
}

func (c *GatewayCache) getProcessor(obj interface{}) Processor {
	switch obj.(type) {
	case *corev1.Service:
		return c.processors[ServicesProcessorType]
	case *svcimpv1alpha1.ServiceImport:
		return c.processors[ServiceImportsProcessorType]
	case *corev1.Endpoints:
		return c.processors[EndpointsProcessorType]
	case *discovery.EndpointSlice:
		return c.processors[EndpointSlicesProcessorType]
	case *corev1.Namespace:
		return c.processors[NamespacesProcessorType]
	case *gwv1beta1.GatewayClass:
		return c.processors[GatewayClassesProcessorType]
	case *gwv1beta1.Gateway:
		return c.processors[GatewaysProcessorType]
	case *gwv1beta1.HTTPRoute:
		return c.processors[HTTPRoutesProcessorType]
	}

	return nil
}

func (c *GatewayCache) isRoutableService(service client.ObjectKey) bool {
	for key := range c.httproutes {
		// Get HTTPRoute from client-go cache
		route := &gwv1beta1.HTTPRoute{}
		if err := c.cache.Get(context.TODO(), key, route); err != nil {
			klog.Error("Failed to get HTTPRoute %q from cache: %s", key.String(), err)
			continue
		}

		for _, rule := range route.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, route.Namespace) {
					return true
				}
			}
		}
	}

	return false
}

func isRefToService(ref gwv1beta1.BackendObjectReference, service client.ObjectKey, ns string) bool {
	if ref.Group != nil {
		switch string(*ref.Group) {
		case "", "flomesh.io":
			klog.V(5).Infof("Ref group is %q", string(*ref.Group))
		default:
			return false
		}
	}

	if ref.Kind != nil {
		switch string(*ref.Kind) {
		case "Service", "ServiceImport":
			klog.V(5).Infof("Ref kind is %q", string(*ref.Kind))
		default:
			return false
		}
	}

	if ref.Namespace == nil {
		if ns != service.Namespace {
			return false
		}
	} else {
		if string(*ref.Namespace) != service.Namespace {
			return false
		}
	}

	return string(ref.Name) == service.Name
}

func (c *GatewayCache) isEffectiveRoute(parentRefs []gwv1beta1.ParentReference) bool {
	if len(c.gateways) == 0 {
		return false
	}

	for _, parentRef := range parentRefs {
		for _, gw := range c.gateways {
			if isRefToGateway(parentRef, gw) {
				return true
			}
		}
	}

	return false
}

func isRefToGateway(parentRef gwv1beta1.ParentReference, gateway client.ObjectKey) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gwv1beta1.GroupName {
		return false
	}

	if parentRef.Kind != nil && string(*parentRef.Kind) != "Gateway" {
		return false
	}

	if parentRef.Namespace != nil && string(*parentRef.Namespace) != gateway.Namespace {
		return false
	}

	return string(parentRef.Name) == gateway.Name
}

func objectKey(obj client.Object) client.ObjectKey {
	ns := obj.GetNamespace()
	if ns == "" {
		ns = metav1.NamespaceDefault
	}

	return client.ObjectKey{Namespace: ns, Name: obj.GetName()}
}
