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
	"fmt"
	gwpkg "github.com/flomesh-io/fsm-classic/pkg/gateway"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/route"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

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

func generateHttpRouteConfig(httpRoute *gwv1beta1.HTTPRoute) route.HTTPRouteRuleSpec {
	httpSpec := route.HTTPRouteRuleSpec{
		RouteType: "HTTP",
		Matches:   make([]route.HTTPTrafficMatch, 0),
	}

	for _, rule := range httpRoute.Spec.Rules {
		backends := map[string]int32{}

		for _, bk := range rule.BackendRefs {
			svcPort := backendRefToServicePortName(bk.BackendRef, httpRoute.Namespace)
			if svcPort != nil {
				backends[*svcPort] = backendWeight(bk.BackendRef)
			}
		}

		for _, m := range rule.Matches {
			match := route.HTTPTrafficMatch{
				BackendService: backends,
			}

			if m.Path != nil {
				match.Path = &route.Path{
					MatchType: string(*m.Path.Type),
					Path:      *m.Path.Value,
				}
			}

			if m.Method != nil {
				match.Methods = []string{string(*m.Method)}
			}

			if len(m.Headers) > 0 {
				match.Headers = append(match.Headers, httpMatchHeaders(m)...)
			}

			if len(m.QueryParams) > 0 {
				match.RequestParams = append(match.RequestParams, httpMatchQueryParams(m)...)
			}

			httpSpec.Matches = append(httpSpec.Matches, match)
		}
	}
	return httpSpec
}

func httpMatchHeaders(m gwv1beta1.HTTPRouteMatch) []route.Headers {
	headers := make([]route.Headers, 0)

	exact := route.Headers{
		MatchType: string(gwv1beta1.HeaderMatchExact),
		Headers:   make(map[string]string),
	}
	regex := route.Headers{
		MatchType: string(gwv1beta1.HeaderMatchExact),
		Headers:   make(map[string]string),
	}
	for _, header := range m.Headers {
		switch *header.Type {
		case gwv1beta1.HeaderMatchExact:
			exact.Headers[string(header.Name)] = header.Value
		case gwv1beta1.HeaderMatchRegularExpression:
			regex.Headers[string(header.Name)] = header.Value
		}
	}

	if len(exact.Headers) > 0 {
		headers = append(headers, exact)
	}

	if len(regex.Headers) > 0 {
		headers = append(headers, regex)
	}

	return headers
}

func httpMatchQueryParams(m gwv1beta1.HTTPRouteMatch) []route.RequestParams {
	params := make([]route.RequestParams, 0)

	exact := route.RequestParams{
		MatchType:     string(gwv1beta1.QueryParamMatchExact),
		RequestParams: make(map[string]string),
	}
	regex := route.RequestParams{
		MatchType:     string(gwv1beta1.QueryParamMatchRegularExpression),
		RequestParams: make(map[string]string),
	}
	for _, param := range m.QueryParams {
		switch *param.Type {
		case gwv1beta1.QueryParamMatchExact:
			exact.RequestParams[string(param.Name)] = param.Value
		case gwv1beta1.QueryParamMatchRegularExpression:
			regex.RequestParams[string(param.Name)] = param.Value
		}
	}

	if len(exact.RequestParams) > 0 {
		params = append(params, exact)
	}

	if len(regex.RequestParams) > 0 {
		params = append(params, regex)
	}

	return params
}

func generateGrpcRouteCfg(grpcRoute *gwv1alpha2.GRPCRoute) route.GRPCRouteRuleSpec {
	grpcSpec := route.GRPCRouteRuleSpec{
		RouteType: "GRPC",
		Matches:   make([]route.GRPCTrafficMatch, 0),
	}

	for _, rule := range grpcRoute.Spec.Rules {
		backends := map[string]int32{}

		for _, bk := range rule.BackendRefs {
			svcPort := backendRefToServicePortName(bk.BackendRef, grpcRoute.Namespace)
			if svcPort != nil {
				backends[*svcPort] = backendWeight(bk.BackendRef)
			}
		}

		for _, m := range rule.Matches {
			match := route.GRPCTrafficMatch{
				BackendService: backends,
			}

			if m.Method != nil {
				match.Method = &route.GRPCMethod{
					MatchType: string(*m.Method.Type),
				}
			}

			if len(m.Headers) > 0 {
				match.Headers = append(match.Headers, grpcMatchHeaders(m)...)
			}

			grpcSpec.Matches = append(grpcSpec.Matches, match)
		}
	}

	return grpcSpec
}

func grpcMatchHeaders(m gwv1alpha2.GRPCRouteMatch) []route.Headers {
	headers := make([]route.Headers, 0)

	exact := route.Headers{
		MatchType: string(gwv1beta1.HeaderMatchExact),
		Headers:   make(map[string]string),
	}
	regex := route.Headers{
		MatchType: string(gwv1beta1.HeaderMatchRegularExpression),
		Headers:   make(map[string]string),
	}
	for _, header := range m.Headers {
		switch *header.Type {
		case gwv1beta1.HeaderMatchExact:
			exact.Headers[string(header.Name)] = header.Value
		case gwv1beta1.HeaderMatchRegularExpression:
			regex.Headers[string(header.Name)] = header.Value
		}
	}

	if len(exact.Headers) > 0 {
		headers = append(headers, exact)
	}

	if len(regex.Headers) > 0 {
		headers = append(headers, regex)
	}

	return headers
}

func generateTLSTerminateRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) route.TLSBackendService {
	backends := route.TLSBackendService{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			svcPort := backendRefToServicePortName(bk, tcpRoute.Namespace)
			if svcPort != nil {
				backends[*svcPort] = backendWeight(bk)
			}
		}
	}

	return backends
}

func generateTLSPassthroughRouteCfg(tlsRoute *gwv1alpha2.TLSRoute) *string {
	for _, rule := range tlsRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			// return the first ONE
			return passthroughTarget(bk)
		}
	}

	return nil
}

func generateTcpRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) route.RouteRule {
	backends := route.TCPRouteRule{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			svcPort := backendRefToServicePortName(bk, tcpRoute.Namespace)
			if svcPort != nil {
				backends[*svcPort] = backendWeight(bk)
			}
		}
	}

	return backends
}

func allowedListeners(
	parentRef gwv1beta1.ParentReference,
	routeGvk schema.GroupVersionKind,
	validListeners []gwpkg.Listener,
) []gwpkg.Listener {
	var selectedListeners []gwpkg.Listener
	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}

	if len(selectedListeners) == 0 {
		return nil
	}

	var allowedListeners []gwpkg.Listener
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(routeGvk) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		return nil
	}

	return allowedListeners
}

func backendRefToServicePortName(ref gwv1beta1.BackendRef, defaultNs string) *string {
	// ONLY supports service backend now
	if *ref.Kind == "Service" && *ref.Group == "" {
		ns := defaultNs
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}

		svcPort := route.ServicePortName{
			NamespacedName: types.NamespacedName{
				Namespace: ns,
				Name:      string(ref.Name),
			},
			Port: pointer.Int32(int32(*ref.Port)),
		}

		result := svcPort.String()
		return &result
	}

	return nil
}

func passthroughTarget(ref gwv1beta1.BackendRef) *string {
	// ONLY supports service backend now
	if *ref.Kind == "Service" && *ref.Group == "" {
		port := int32(443)
		if ref.Port != nil {
			port = int32(*ref.Port)
		}

		target := fmt.Sprintf("%s:%d", ref.Name, port)

		return &target
	}

	return nil
}

func backendWeight(bk gwv1beta1.BackendRef) int32 {
	if bk.Weight != nil {
		return *bk.Weight
	}

	return 1
}

func mergeL7RouteRule(rule1 route.L7RouteRule, rule2 route.L7RouteRule) route.L7RouteRule {
	mergedRule := route.L7RouteRule{}

	for hostname, rule := range rule1 {
		mergedRule[hostname] = rule
	}

	for hostname, rule := range rule2 {
		if r1, exists := mergedRule[hostname]; exists {
			// can only merge same type of route into one hostname
			switch r1 := r1.(type) {
			case route.GRPCRouteRuleSpec:
				switch r2 := rule.(type) {
				case route.GRPCRouteRuleSpec:
					r1.Matches = append(r1.Matches, r2.Matches...)
					mergedRule[hostname] = r1
				default:
					klog.Errorf("%s has been already mapped to RouteRule[%s] %v, current RouteRule %v will be dropped.", hostname, r1.RouteType, r1, r2)
				}
			case route.HTTPRouteRuleSpec:
				switch r2 := rule.(type) {
				case route.HTTPRouteRuleSpec:
					r1.Matches = append(r1.Matches, r2.Matches...)
					mergedRule[hostname] = r1
				default:
					klog.Errorf("%s has been already mapped to RouteRule[%s] %v, current RouteRule %v will be dropped.", hostname, r1.RouteType, r1, r2)
				}
			}
		} else {
			mergedRule[hostname] = rule
		}
	}

	return mergedRule
}

func copyMap[K, V comparable](m map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func isEndpointReady(ep discoveryv1.Endpoint) bool {
	if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
		return true
	}

	return false
}
