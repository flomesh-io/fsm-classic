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
	svcimpv1alpha1 "github.com/flomesh-io/fsm-classic/apis/serviceimport/v1alpha1"
	fctx "github.com/flomesh-io/fsm-classic/pkg/context"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/route"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	"github.com/flomesh-io/fsm-classic/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayCache struct {
	client     client.Client
	cache      cache.Cache
	repoClient *repo.PipyRepoClient

	processors map[ProcessorType]Processor

	gatewayclass   *gwv1beta1.GatewayClass
	gateways       map[string]client.ObjectKey // ns -> gateway
	services       map[client.ObjectKey]bool
	serviceimports map[client.ObjectKey]bool
	endpoints      map[client.ObjectKey]bool
	endpointslices map[client.ObjectKey]map[client.ObjectKey]bool // svc -> endpointslices
	namespaces     map[string]bool
	httproutes     map[client.ObjectKey]bool
	grpcroutes     map[client.ObjectKey]bool
	tcproutes      map[client.ObjectKey]bool
	tlsroutes      map[client.ObjectKey]bool
}

func NewGatewayCache(fctx *fctx.FsmContext) *GatewayCache {
	return &GatewayCache{
		client:     fctx.Client,
		cache:      fctx.Manager.GetCache(),
		repoClient: fctx.RepoClient,

		processors: map[ProcessorType]Processor{
			ServicesProcessorType:       &ServicesProcessor{},
			ServiceImportsProcessorType: &ServiceImportsProcessor{},
			EndpointSlicesProcessorType: &EndpointSlicesProcessor{},
			EndpointsProcessorType:      &EndpointsProcessor{},
			NamespacesProcessorType:     &NamespacesProcessor{},
			GatewayClassesProcessorType: &GatewayClassesProcessor{},
			GatewaysProcessorType:       &GatewaysProcessor{},
			HTTPRoutesProcessorType:     &HTTPRoutesProcessor{},
			GRPCRoutesProcessorType:     &GRPCRoutesProcessor{},
			TCPRoutesProcessorType:      &TCPRoutesProcessor{},
			TLSRoutesProcessorType:      &TLSRoutesProcessor{},
		},

		gateways:       make(map[string]client.ObjectKey),
		services:       make(map[client.ObjectKey]bool),
		serviceimports: make(map[client.ObjectKey]bool),
		endpointslices: make(map[client.ObjectKey]map[client.ObjectKey]bool),
		endpoints:      make(map[client.ObjectKey]bool),
		namespaces:     make(map[string]bool),
		httproutes:     make(map[client.ObjectKey]bool),
		grpcroutes:     make(map[client.ObjectKey]bool),
		tcproutes:      make(map[client.ObjectKey]bool),
		tlsroutes:      make(map[client.ObjectKey]bool),
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

func (c *GatewayCache) WaitForCacheSync(ctx context.Context) bool {
	return c.cache.WaitForCacheSync(ctx)
}

func (c *GatewayCache) getProcessor(obj interface{}) Processor {
	switch obj.(type) {
	case *corev1.Service:
		return c.processors[ServicesProcessorType]
	case *svcimpv1alpha1.ServiceImport:
		return c.processors[ServiceImportsProcessorType]
	case *corev1.Endpoints:
		return c.processors[EndpointsProcessorType]
	case *discoveryv1.EndpointSlice:
		return c.processors[EndpointSlicesProcessorType]
	case *corev1.Namespace:
		return c.processors[NamespacesProcessorType]
	case *gwv1beta1.GatewayClass:
		return c.processors[GatewayClassesProcessorType]
	case *gwv1beta1.Gateway:
		return c.processors[GatewaysProcessorType]
	case *gwv1beta1.HTTPRoute:
		return c.processors[HTTPRoutesProcessorType]
	case *gwv1alpha2.GRPCRoute:
		return c.processors[GRPCRoutesProcessorType]
	case *gwv1alpha2.TCPRoute:
		return c.processors[TCPRoutesProcessorType]
	case *gwv1alpha2.TLSRoute:
		return c.processors[TLSRoutesProcessorType]
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

	for key := range c.grpcroutes {
		// Get GRPCRoute from client-go cache
		route := &gwv1alpha2.GRPCRoute{}
		if err := c.cache.Get(context.TODO(), key, route); err != nil {
			klog.Error("Failed to get GRPCRoute %q from cache: %s", key.String(), err)
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

	for key := range c.tlsroutes {
		// Get TLSRoute from client-go cache
		route := &gwv1alpha2.TLSRoute{}
		if err := c.cache.Get(context.TODO(), key, route); err != nil {
			klog.Error("Failed to get TLSRoute %q from cache: %s", key.String(), err)
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

	for key := range c.tcproutes {
		// Get TCPRoute from client-go cache
		route := &gwv1alpha2.TCPRoute{}
		if err := c.cache.Get(context.TODO(), key, route); err != nil {
			klog.Error("Failed to get TCPRoute %q from cache: %s", key.String(), err)
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
			if utils.IsRefToGateway(parentRef, gw) {
				return true
			}
		}
	}

	return false
}

func objectKey(obj client.Object) client.ObjectKey {
	ns := obj.GetNamespace()
	if ns == "" {
		ns = metav1.NamespaceDefault
	}

	return client.ObjectKey{Namespace: ns, Name: obj.GetName()}
}

func (c *GatewayCache) BuildConfigs() {
	configs := make(map[string]*route.ConfigSpec)
	ctx := context.TODO()

	for ns, key := range c.gateways {
		gw := &gwv1beta1.Gateway{}
		if err := c.cache.Get(ctx, key, gw); err != nil {
			klog.Errorf("Failed to get Gateway %s: %s", key, err)
			continue
		}

		config := &route.ConfigSpec{
			Listeners:  c.listeners(gw),
			RouteRules: c.routeRules(gw),
			Chains:     chains(),
		}
		configs[ns] = config
	}
}

func (c *GatewayCache) listeners(gw *gwv1beta1.Gateway) []route.Listener {
	listeners := make([]route.Listener, 0)

	for _, l := range gw.Spec.Listeners {
		listener := route.Listener{
			Protocol: string(l.Protocol),
			Port:     int32(l.Port),
		}

		switch l.Protocol {
		case gwv1beta1.HTTPSProtocolType:
			// Terminate
			if l.TLS != nil {
				switch *l.TLS.Mode {
				case gwv1beta1.TLSModeTerminate:
					listener.TLS = &route.TLS{
						TLSModeType:  string(gwv1beta1.TLSModeTerminate),
						MTLS:         false, // FIXME: source of mTLS
						Certificates: c.certificates(gw, l),
					}
				default:
					klog.Warningf("Invalid TLSModeType %q for Protocol %s", l.TLS.Mode, l.Protocol)
				}
			}
		case gwv1beta1.TLSProtocolType:
			// Terminate & Passthrough
			if l.TLS != nil {
				switch *l.TLS.Mode {
				case gwv1beta1.TLSModeTerminate:
					listener.TLS = &route.TLS{
						TLSModeType:  string(gwv1beta1.TLSModeTerminate),
						MTLS:         false, // FIXME: source of mTLS
						Certificates: c.certificates(gw, l),
					}
				case gwv1beta1.TLSModePassthrough:
					listener.TLS = &route.TLS{
						TLSModeType: string(gwv1beta1.TLSModePassthrough),
						MTLS:        false, // FIXME: source of mTLS
					}
				}
			}
		}

		listeners = append(listeners, listener)
	}

	return listeners
}

func (c *GatewayCache) certificates(gw *gwv1beta1.Gateway, l gwv1beta1.Listener) []route.Certificate {
	certs := make([]route.Certificate, 0)
	for _, ref := range l.TLS.CertificateRefs {
		if string(*ref.Kind) == "Secret" && string(*ref.Group) == "" {
			secret := &corev1.Secret{}
			key := secretKey(gw, ref)
			if err := c.client.Get(context.TODO(), key, secret); err != nil {
				klog.Errorf("Failed to get Secret %s: %s", key, err)
				continue
			}
			certs = append(certs, route.Certificate{
				CertChain:  string(secret.Data["tls.crt"]),
				PrivateKey: string(secret.Data["tls.key"]),
				IssuingCA:  string(secret.Data["ca.crt"]),
			})
		}
	}
	return certs
}

func secretKey(gw *gwv1beta1.Gateway, secretRef gwv1beta1.SecretObjectReference) client.ObjectKey {
	ns := ""
	if secretRef.Namespace == nil {
		ns = gw.Namespace
	} else {
		ns = string(*secretRef.Namespace)
	}

	name := string(secretRef.Name)

	return client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
}

func (c *GatewayCache) routeRules(gw *gwv1beta1.Gateway) map[int32]route.RouteRule {
	rules := make(map[int32]route.RouteRule)

	for key := range c.httproutes {
		httpRoute := &gwv1beta1.HTTPRoute{}
		if err := c.client.Get(context.TODO(), key, httpRoute); err != nil {
			klog.Errorf("Failed to get HTTPRoute %s: %s", key, err)
			continue
		}

		for _, ref := range httpRoute.Spec.ParentRefs {
			if string(*ref.Kind) == gw.Kind && string(*ref.Group) == gw.GroupVersionKind().Group {

				//if *ref.Port == gw.Spec.Listeners {}
			}
		}
	}

	return rules
}

func chains() route.Chains {
	return route.Chains{
		InboundHTTP: []string{
			"modules/inbound-tls-termination.js",
			"modules/inbound-http-routing.js",
			"plugins/inbound-http-default-routing.js",
			"modules/inbound-metrics-http.js",
			"modules/inbound-tracing-http.js",
			"modules/inbound-logging-http.js",
			"modules/inbound-throttle-service.js",
			"modules/inbound-throttle-route.js",
			"modules/inbound-http-load-balancing.js",
			"modules/inbound-http-default.js",
		},
		InboundTCP: []string{
			"modules/inbound-tls-termination.js",
			"modules/inbound-tcp-routing.js",
			"modules/inbound-tcp-load-balancing.js",
			"modules/inbound-tcp-default.js",
		},
		OutboundHTTP: []string{
			"modules/outbound-http-routing.js",
			"plugins/outbound-http-default-routing.js",
			"modules/outbound-metrics-http.js",
			"modules/outbound-tracing-http.js",
			"modules/outbound-logging-http.js",
			"modules/outbound-circuit-breaker.js",
			"modules/outbound-http-load-balancing.js",
			"modules/outbound-http-default.js",
		},
		OutboundTCP: []string{
			"modules/outbound-tcp-routing.js",
			"modules/outbound-tcp-load-balancing.js",
			"modules/outbound-tcp-default.js",
		},
	}
}
