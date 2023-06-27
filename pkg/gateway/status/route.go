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

package status

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	fctx "github.com/flomesh-io/fsm-classic/pkg/context"
	"github.com/flomesh-io/fsm-classic/pkg/gateway"
	gwutils "github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"time"
)

type RouteStatusProcessor struct {
	Fctx *fctx.FsmContext
}

func (p *RouteStatusProcessor) ProcessRouteStatus(ctx context.Context, route client.Object) ([]gwv1beta1.RouteParentStatus, error) {
	gatewayList := &gwv1beta1.GatewayList{}
	if err := p.Fctx.List(ctx, gatewayList); err != nil {
		return nil, err
	}

	activeGateways := make([]*gwv1beta1.Gateway, 0)
	for _, gw := range gatewayList.Items {
		if gwutils.IsActiveGateway(&gw) {
			activeGateways = append(activeGateways, &gw)
		}
	}

	if len(activeGateways) > 0 {
		var params *gateway.ComputeParams = nil
		switch route := route.(type) {
		case *gwv1beta1.HTTPRoute:
			params = &gateway.ComputeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  route.Spec.Hostnames,
			}
		case *gwv1alpha2.GRPCRoute:
			params = &gateway.ComputeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  route.Spec.Hostnames,
			}
		case *gwv1alpha2.TLSRoute:
			params = &gateway.ComputeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  route.Spec.Hostnames,
			}
		case *gwv1alpha2.TCPRoute:
			params = &gateway.ComputeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  nil,
			}
		default:
			klog.Warningf("Unsupported route type: %T", route)
			return nil, fmt.Errorf("unsupported route type: %T", route)
		}

		if params != nil {
			return p.computeRouteParentStatus(activeGateways, params), nil
		}
	}

	return nil, nil
}

func (p *RouteStatusProcessor) computeRouteParentStatus(
	activeGateways []*gwv1beta1.Gateway,
	params *gateway.ComputeParams,
) []gwv1beta1.RouteParentStatus {
	status := make([]gwv1beta1.RouteParentStatus, 0)

	for _, gw := range activeGateways {
		validListeners := gwutils.GetValidListenersFromGateway(gw)

		for _, parentRef := range params.ParentRefs {
			if !gwutils.IsRefToGateway(parentRef, gwutils.ObjectKey(gw)) {
				continue
			}

			routeParentStatus := gwv1beta1.RouteParentStatus{
				ParentRef:      parentRef,
				ControllerName: commons.GatewayController,
				Conditions:     make([]metav1.Condition, 0),
			}

			allowedListeners := gwutils.GetAllowedListeners(parentRef, params.RouteGvk, params.RouteGeneration, validListeners, routeParentStatus)
			if len(allowedListeners) == 0 {

			}

			count := 0
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, params.RouteHostnames)

				//if len(hostnames) == 0 {
				//	continue
				//}

				count += len(hostnames)
			}

			switch params.RouteGvk.Kind {
			case "HTTPRoute", "TLSRoute", "GRPCRoute":
				if count == 0 && metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionAccepted)) == nil {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.RouteConditionAccepted),
						Status:             metav1.ConditionFalse,
						ObservedGeneration: params.RouteGeneration,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.RouteReasonNoMatchingListenerHostname),
						Message:            "No matching hostnames were found between the listener and the route.",
					})
				}
			}

			if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionResolvedRefs)) == nil {
				metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
					Type:               string(gwv1beta1.RouteConditionResolvedRefs),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: params.RouteGeneration,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1beta1.RouteReasonResolvedRefs),
					Message:            fmt.Sprintf("References of %s is resolved", params.RouteGvk.Kind),
				})
			}

			if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionAccepted)) == nil {
				metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
					Type:               string(gwv1beta1.RouteConditionAccepted),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: params.RouteGeneration,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1beta1.RouteReasonAccepted),
					Message:            fmt.Sprintf("%s is Accepted", params.RouteGvk.Kind),
				})
			}

			status = append(status, routeParentStatus)
		}
	}

	return status
}