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
	"fmt"
	svcimpv1alpha1 "github.com/flomesh-io/fsm-classic/apis/serviceimport/v1alpha1"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	fctx "github.com/flomesh-io/fsm-classic/pkg/context"
	gwpkg "github.com/flomesh-io/fsm-classic/pkg/gateway"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/route"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	gwutils "github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	"github.com/flomesh-io/fsm-classic/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayCache struct {
	client      client.Client
	cache       cache.Cache
	repoClient  *repo.PipyRepoClient
	configStore *config.Store

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
		client:      fctx.Client,
		cache:       fctx.Manager.GetCache(),
		repoClient:  fctx.RepoClient,
		configStore: fctx.ConfigStore,

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
		r := &gwv1beta1.HTTPRoute{}
		if err := c.cache.Get(context.TODO(), key, r); err != nil {
			klog.Error("Failed to get HTTPRoute %q from cache: %s", key.String(), err)
			continue
		}

		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}
			}
		}
	}

	for key := range c.grpcroutes {
		// Get GRPCRoute from client-go cache
		r := &gwv1alpha2.GRPCRoute{}
		if err := c.cache.Get(context.TODO(), key, r); err != nil {
			klog.Error("Failed to get GRPCRoute %q from cache: %s", key.String(), err)
			continue
		}

		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}
			}
		}
	}

	for key := range c.tlsroutes {
		// Get TLSRoute from client-go cache
		r := &gwv1alpha2.TLSRoute{}
		if err := c.cache.Get(context.TODO(), key, r); err != nil {
			klog.Error("Failed to get TLSRoute %q from cache: %s", key.String(), err)
			continue
		}

		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}
			}
		}
	}

	for key := range c.tcproutes {
		// Get TCPRoute from client-go cache
		r := &gwv1alpha2.TCPRoute{}
		if err := c.cache.Get(context.TODO(), key, r); err != nil {
			klog.Error("Failed to get TCPRoute %q from cache: %s", key.String(), err)
			continue
		}

		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
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

func (c *GatewayCache) BuildConfigs() {
	configs := make(map[string]*route.ConfigSpec)
	ctx := context.TODO()

	for ns, key := range c.gateways {
		gw := &gwv1beta1.Gateway{}
		if err := c.cache.Get(ctx, key, gw); err != nil {
			klog.Errorf("Failed to get Gateway %s: %s", key, err)
			continue
		}

		validListeners := gwutils.GetValidListenersFromGateway(gw)
		listenerCfg := c.listeners(gw, validListeners)
		rules, referredServices := c.routeRules(gw, validListeners)
		svcConfigs := c.serviceConfigs(referredServices)

		configSpec := &route.ConfigSpec{
			Listeners:  listenerCfg,
			RouteRules: rules,
			Services:   svcConfigs,
			Chains:     chains(),
		}
		configs[ns] = configSpec
	}

	mc := c.configStore.MeshConfig.GetConfig()
	parentPath := mc.GetDefaultGatewaysPath()
	for ns, cfg := range configs {
		gatewayPath := mc.GatewayCodebasePath(ns)

		go func(cfg *route.ConfigSpec) {
			if err := c.repoClient.DeriveCodebase(gatewayPath, parentPath); err != nil {
				klog.Errorf("Gateway codebase %q failed to derive codebase %q: %s", gatewayPath, parentPath, err)
				return
			}

			batches := []repo.Batch{
				{
					Basepath: gatewayPath,
					Items: []repo.BatchItem{
						{
							Path:     "/",
							Filename: "config.json",
							Content:  cfg,
						},
					},
				},
			}

			if err := c.repoClient.Batch(batches); err != nil {
				klog.Errorf("Sync gateway config to repo failed: %s", err)
				return
			}
		}(cfg)
	}
}

func (c *GatewayCache) listeners(gw *gwv1beta1.Gateway, validListeners []gwpkg.Listener) []route.Listener {
	listeners := make([]route.Listener, 0)
	for _, l := range validListeners {
		listener := route.Listener{
			Protocol: l.Protocol,
			Port:     l.Port,
		}

		switch l.Protocol {
		case gwv1beta1.HTTPSProtocolType:
			// Terminate
			if l.TLS != nil {
				switch *l.TLS.Mode {
				case gwv1beta1.TLSModeTerminate:
					listener.TLS = &route.TLS{
						TLSModeType:  gwv1beta1.TLSModeTerminate,
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
						TLSModeType:  gwv1beta1.TLSModeTerminate,
						MTLS:         false, // FIXME: source of mTLS
						Certificates: c.certificates(gw, l),
					}
				case gwv1beta1.TLSModePassthrough:
					listener.TLS = &route.TLS{
						TLSModeType: gwv1beta1.TLSModePassthrough,
						MTLS:        false, // FIXME: source of mTLS
					}
				}
			}
		}

		listeners = append(listeners, listener)
	}

	return listeners
}

func (c *GatewayCache) certificates(gw *gwv1beta1.Gateway, l gwpkg.Listener) []route.Certificate {
	certs := make([]route.Certificate, 0)
	for _, ref := range l.TLS.CertificateRefs {
		if string(*ref.Kind) == "Secret" && string(*ref.Group) == "" {
			secret := &corev1.Secret{}
			key := secretKey(gw, ref)
			if err := c.client.Get(context.TODO(), key, secret); err != nil {
				klog.Errorf("Failed to get Secret %s: %s", key, err)
				continue
			}

			cert := route.Certificate{
				CertChain:  string(secret.Data[corev1.TLSCertKey]),
				PrivateKey: string(secret.Data[corev1.TLSPrivateKeyKey]),
			}

			ca := string(secret.Data[corev1.ServiceAccountRootCAKey])
			if ca != "" {
				cert.IssuingCA = ca
			}

			certs = append(certs, cert)
		}
	}
	return certs
}

func (c *GatewayCache) routeRules(gw *gwv1beta1.Gateway, validListeners []gwpkg.Listener) (map[int32]route.RouteRule, map[string]serviceInfo) {
	rules := make(map[int32]route.RouteRule)
	services := make(map[string]serviceInfo)

	for key := range c.httproutes {
		httpRoute := &gwv1beta1.HTTPRoute{}
		if err := c.cache.Get(context.TODO(), key, httpRoute); err != nil {
			klog.Errorf("Failed to get HTTPRoute %s: %s", key, err)
			continue
		}

		processHttpRoute(gw, validListeners, httpRoute, rules)
		processHttpRouteBackendFilters(httpRoute, services)
	}

	for key := range c.grpcroutes {
		grpcRoute := &gwv1alpha2.GRPCRoute{}
		if err := c.cache.Get(context.TODO(), key, grpcRoute); err != nil {
			klog.Errorf("Failed to get GRPCRoute %s: %s", key, err)
			continue
		}

		processGrpcRoute(gw, validListeners, grpcRoute, rules)
		processGrpcRouteBackendFilters(grpcRoute, services)
	}

	for key := range c.tlsroutes {
		tlsRoute := &gwv1alpha2.TLSRoute{}
		if err := c.cache.Get(context.TODO(), key, tlsRoute); err != nil {
			klog.Errorf("Failed to get TLSRoute %s: %s", key, err)
			continue
		}

		processTlsRoute(gw, validListeners, tlsRoute, rules)
		processTlsBackends(tlsRoute, services)
	}

	for key := range c.tcproutes {
		tcpRoute := &gwv1alpha2.TCPRoute{}
		if err := c.cache.Get(context.TODO(), key, tcpRoute); err != nil {
			klog.Errorf("Failed to get TCPRoute %s: %s", key, err)
			continue
		}

		processTcpRoute(gw, validListeners, tcpRoute, rules)
		processTcpBackends(tcpRoute, services)
	}

	return rules, services
}

func processHttpRoute(gw *gwv1beta1.Gateway, validListeners []gwpkg.Listener, httpRoute *gwv1beta1.HTTPRoute, rules map[int32]route.RouteRule) {
	for _, ref := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, httpRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, httpRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			httpRule := route.L7RouteRule{}
			for _, hostname := range hostnames {
				httpRule[hostname] = generateHttpRouteConfig(httpRoute)
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(route.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, httpRule)
				}
			} else {
				rules[port] = httpRule
			}
		}
	}
}

func processGrpcRoute(gw *gwv1beta1.Gateway, validListeners []gwpkg.Listener, grpcRoute *gwv1alpha2.GRPCRoute, rules map[int32]route.RouteRule) {
	for _, ref := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, grpcRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, grpcRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			grpcRule := route.L7RouteRule{}
			for _, hostname := range hostnames {
				grpcRule[hostname] = generateGrpcRouteCfg(grpcRoute)
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(route.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, grpcRule)
				}
			} else {
				rules[port] = grpcRule
			}
		}
	}
}

func processTlsRoute(gw *gwv1beta1.Gateway, validListeners []gwpkg.Listener, tlsRoute *gwv1alpha2.TLSRoute, rules map[int32]route.RouteRule) {
	for _, ref := range tlsRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, tlsRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			if listener.Protocol != gwv1beta1.TLSProtocolType {
				continue
			}

			if listener.TLS == nil {
				continue
			}

			if listener.TLS.Mode == nil {
				continue
			}

			if *listener.TLS.Mode != gwv1beta1.TLSModePassthrough {
				continue
			}

			hostnames := gwutils.GetValidHostnames(listener.Hostname, tlsRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			tlsRule := route.TLSPassthroughRouteRule{}
			for _, hostname := range hostnames {
				if target := generateTLSPassthroughRouteCfg(tlsRoute); target != nil {
					tlsRule[hostname] = *target
				}
			}

			rules[int32(listener.Port)] = tlsRule
		}
	}
}

func processTcpRoute(gw *gwv1beta1.Gateway, validListeners []gwpkg.Listener, tcpRoute *gwv1alpha2.TCPRoute, rules map[int32]route.RouteRule) {
	for _, ref := range tcpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, tcpRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1beta1.TLSProtocolType:
				if listener.TLS == nil {
					continue
				}

				if listener.TLS.Mode == nil {
					continue
				}

				if *listener.TLS.Mode != gwv1beta1.TLSModeTerminate {
					continue
				}

				hostnames := gwutils.GetValidHostnames(listener.Hostname, nil)

				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}

				tlsRule := route.TLSTerminateRouteRule{}
				for _, hostname := range hostnames {
					tlsRule[hostname] = generateTLSTerminateRouteCfg(tcpRoute)
				}

				rules[int32(listener.Port)] = tlsRule
			case gwv1beta1.TCPProtocolType:
				rules[int32(listener.Port)] = generateTcpRouteCfg(tcpRoute)
			}
		}
	}
}

func processHttpRouteBackendFilters(httpRoute *gwv1beta1.HTTPRoute, services map[string]serviceInfo) {
	ns := httpRoute.Namespace

	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
	for _, rule := range httpRoute.Spec.Rules {
		ruleLevelFilters := make(map[gwv1beta1.HTTPRouteFilterType]route.Filter)

		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters[ruleFilter.Type] = ruleFilter
		}

		for _, backend := range rule.BackendRefs {
			if *backend.Group == "" && *backend.Kind == "Service" {
				if backend.Namespace != nil {
					ns = string(*backend.Namespace)
				}

				svcPort := route.ServicePortName{
					NamespacedName: types.NamespacedName{
						Namespace: ns,
						Name:      string(backend.Name),
					},

					Port: pointer.Int32(int32(*backend.Port)),
				}

				svcFilters := copyMap(ruleLevelFilters)
				for _, svcFilter := range backend.Filters {
					svcFilters[svcFilter.Type] = svcFilter
				}

				svcInfo := serviceInfo{
					svcPortName: svcPort,
					filters:     make([]route.Filter, 0),
				}
				for _, f := range svcFilters {
					svcInfo.filters = append(svcInfo.filters, f)
				}
				services[svcPort.String()] = svcInfo
			}
		}
	}
}

func processGrpcRouteBackendFilters(grpcRoute *gwv1alpha2.GRPCRoute, services map[string]serviceInfo) {
	ns := grpcRoute.Namespace

	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
	for _, rule := range grpcRoute.Spec.Rules {
		ruleLevelFilters := make(map[gwv1alpha2.GRPCRouteFilterType]route.Filter)

		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters[ruleFilter.Type] = ruleFilter
		}

		for _, backend := range rule.BackendRefs {
			if *backend.Group == "" && *backend.Kind == "Service" {
				if backend.Namespace != nil {
					ns = string(*backend.Namespace)
				}

				svcPort := route.ServicePortName{
					NamespacedName: types.NamespacedName{
						Namespace: ns,
						Name:      string(backend.Name),
					},
					Port: pointer.Int32(int32(*backend.Port)),
				}

				svcFilters := copyMap(ruleLevelFilters)
				for _, svcFilter := range backend.Filters {
					svcFilters[svcFilter.Type] = svcFilter
				}

				svcInfo := serviceInfo{
					svcPortName: svcPort,
					filters:     make([]route.Filter, 0),
				}
				for _, f := range svcFilters {
					svcInfo.filters = append(svcInfo.filters, f)
				}
				services[svcPort.String()] = svcInfo
			}
		}
	}
}

func processTlsBackends(tlsRoute *gwv1alpha2.TLSRoute, services map[string]serviceInfo) {
	// DO nothing for now
}

func processTcpBackends(tcpRoute *gwv1alpha2.TCPRoute, services map[string]serviceInfo) {
	ns := tcpRoute.Namespace

	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if *backend.Group == "" && *backend.Kind == "Service" {
				if backend.Namespace != nil {
					ns = string(*backend.Namespace)
				}

				svcPort := route.ServicePortName{
					NamespacedName: types.NamespacedName{
						Namespace: ns,
						Name:      string(backend.Name),
					},
					Port: pointer.Int32(int32(*backend.Port)),
				}

				services[svcPort.String()] = serviceInfo{
					svcPortName: svcPort,
				}
			}
		}
	}
}

func (c *GatewayCache) serviceConfigs(services map[string]serviceInfo) map[string]route.ServiceConfig {
	configs := make(map[string]route.ServiceConfig)

	for svcPortName, svcInfo := range services {
		svcKey := svcInfo.svcPortName.NamespacedName
		svc := &corev1.Service{}
		if err := c.cache.Get(context.TODO(), svcKey, svc); err != nil {
			klog.Errorf("Failed to get Service %s: %s", svcKey, err)
			continue
		}

		endpointSliceList := &discoveryv1.EndpointSliceList{}
		if err := c.cache.List(
			context.TODO(),
			endpointSliceList,
			client.InNamespace(svc.Namespace),
			client.MatchingLabels{
				"kubernetes.io/service-name": svc.Name,
			},
		); err != nil {
			klog.Errorf("Failed to list EndpointSlice of Service %s: %s", svcKey, err)
			continue
		}

		if len(endpointSliceList.Items) == 0 {
			continue
		}

		svcPort, err := getServicePort(svc, svcInfo.svcPortName.Port)
		if err != nil {
			klog.Errorf("Failed to get ServicePort: %s", err)
			continue
		}

		filteredSlices := filterEndpointSliceList(endpointSliceList, svcPort)
		if len(filteredSlices) == 0 {
			klog.Errorf("no valid endpoints found for Service %s and port %+v", svcKey, svcPort)
			continue
		}

		endpointSet := make(map[endpointInfo]struct{})

		for _, eps := range filteredSlices {
			for _, endpoint := range eps.Endpoints {

				if !isEndpointReady(endpoint) {
					continue
				}
				endpointPort := findPort(eps.Ports, svcPort)

				for _, address := range endpoint.Addresses {
					ep := endpointInfo{address: address, port: endpointPort}
					endpointSet[ep] = struct{}{}
				}
			}
		}

		svcCfg := route.ServiceConfig{
			Filters:   svcInfo.filters,
			Endpoints: make(map[string]route.Endpoint),
		}

		for ep := range endpointSet {
			hostport := fmt.Sprintf("%s:%d", ep.address, ep.port)
			svcCfg.Endpoints[hostport] = route.Endpoint{
				Weight: 1,
			}
		}

		configs[svcPortName] = svcCfg
	}

	return configs
}

func getServicePort(svc *corev1.Service, port *int32) (corev1.ServicePort, error) {
	if port == nil && len(svc.Spec.Ports) == 1 {
		return svc.Spec.Ports[0], nil
	}

	if port != nil {
		for _, p := range svc.Spec.Ports {
			if p.Port == *port {
				return p, nil
			}
		}
	}

	return corev1.ServicePort{}, fmt.Errorf("no matching port for Service %s and port %d", svc.Name, port)
}

func filterEndpointSliceList(
	endpointSliceList *discoveryv1.EndpointSliceList,
	port corev1.ServicePort,
) []discoveryv1.EndpointSlice {
	filtered := make([]discoveryv1.EndpointSlice, 0, len(endpointSliceList.Items))

	for _, endpointSlice := range endpointSliceList.Items {
		if !ignoreEndpointSlice(endpointSlice, port) {
			filtered = append(filtered, endpointSlice)
		}
	}

	return filtered
}

func ignoreEndpointSlice(endpointSlice discoveryv1.EndpointSlice, port corev1.ServicePort) bool {
	if endpointSlice.AddressType != discoveryv1.AddressTypeIPv4 {
		return true
	}

	// ignore endpoint slices that don't have a matching port.
	return findPort(endpointSlice.Ports, port) == 0
}

func findPort(ports []discoveryv1.EndpointPort, svcPort corev1.ServicePort) int32 {
	portName := svcPort.Name

	for _, p := range ports {

		if p.Port == nil {
			return getDefaultPort(svcPort)
		}

		if p.Name != nil && *p.Name == portName {
			return *p.Port
		}
	}

	return 0
}

func getDefaultPort(svcPort corev1.ServicePort) int32 {
	switch svcPort.TargetPort.Type {
	case intstr.Int:
		if svcPort.TargetPort.IntVal != 0 {
			return svcPort.TargetPort.IntVal
		}
	}

	return svcPort.Port
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
