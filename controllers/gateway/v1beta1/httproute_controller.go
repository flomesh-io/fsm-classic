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

package v1beta1

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm-classic/controllers"
	gwctlutils "github.com/flomesh-io/fsm-classic/controllers/gateway/utils"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	fctx "github.com/flomesh-io/fsm-classic/pkg/context"
	gwutils "github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"time"
)

type httpRouteReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.FsmContext
}

type gatewayInfo struct {
	key   client.ObjectKey
	ports map[gwv1beta1.SectionName]gwv1beta1.PortNumber
}

func NewHTTPRouteReconciler(ctx *fctx.FsmContext) controllers.Reconciler {
	return &httpRouteReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("HTTPRoute"),
		fctx:     ctx,
	}
}

func (r *httpRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	httpRoute := &gwv1beta1.HTTPRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, httpRoute)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwv1beta1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if httpRoute.DeletionTimestamp != nil {
		r.fctx.EventHandler.OnDelete(httpRoute)
		return ctrl.Result{}, nil
	}

	gatewayList := &gwv1beta1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList); err != nil {
		return ctrl.Result{}, err
	}

	activeGateways := make([]*gwv1beta1.Gateway, 0)
	for _, gw := range gatewayList.Items {
		if gwutils.IsActiveGateway(&gw) {
			activeGateways = append(activeGateways, &gw)
		}
	}

	if len(activeGateways) > 0 {
		httpRoute.Status.Parents = nil
		for _, gw := range activeGateways {
			validListeners := gwctlutils.GetValidListenersFromGateway(gw)

			for _, parentRef := range httpRoute.Spec.ParentRefs {
				if !gwutils.IsRefToGateway(parentRef, gwutils.ObjectKey(gw)) {
					continue
				}

				routeParentStatus := gwv1beta1.RouteParentStatus{
					ParentRef:      parentRef,
					ControllerName: commons.GatewayController,
					Conditions:     make([]metav1.Condition, 0),
				}

				allowedListeners := gwctlutils.GetAllowedListeners(httpRoute, parentRef, validListeners, routeParentStatus)
				if len(allowedListeners) == 0 {

				}

				count := 0
				for _, listener := range allowedListeners {
					hostnames := gwctlutils.GetValidHostnames(listener.Hostname, httpRoute.Spec.Hostnames)

					if len(hostnames) == 0 {
						continue
					}

					count += len(hostnames)
				}

				if count == 0 && metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionAccepted)) == nil {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.RouteConditionAccepted),
						Status:             metav1.ConditionFalse,
						ObservedGeneration: httpRoute.GetGeneration(),
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.RouteReasonNoMatchingListenerHostname),
						Message:            fmt.Sprintf("%s", err),
					})
				}

				if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionResolvedRefs)) == nil {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.RouteConditionResolvedRefs),
						Status:             metav1.ConditionTrue,
						ObservedGeneration: httpRoute.GetGeneration(),
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.RouteReasonResolvedRefs),
						Message:            fmt.Sprintf("%s", err),
					})
				}

				if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionAccepted)) == nil {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.RouteConditionAccepted),
						Status:             metav1.ConditionTrue,
						ObservedGeneration: httpRoute.GetGeneration(),
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.RouteReasonAccepted),
						Message:            fmt.Sprintf("%s", err),
					})
				}

				httpRoute.Status.Parents = append(httpRoute.Status.Parents, routeParentStatus)
			}
		}

		if len(httpRoute.Status.Parents) > 0 {
			if err := r.fctx.Status().Update(ctx, httpRoute); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	r.fctx.EventHandler.OnAdd(httpRoute)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *httpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.HTTPRoute{}).
		Complete(r)
}
