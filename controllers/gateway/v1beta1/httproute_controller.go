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
	"github.com/flomesh-io/fsm-classic/controllers"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	fctx "github.com/flomesh-io/fsm-classic/pkg/context"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type httpRouteReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.FsmContext
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

	r.fctx.EventHandler.OnAdd(httpRoute)

	gatewayList := &gwv1beta1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList); err != nil {
		return ctrl.Result{}, err
	}

	activeGateways := make([]*gwv1beta1.Gateway, 0)
	for _, gw := range gatewayList.Items {
		if utils.IsAcceptedGateway(&gw) {
			activeGateways = append(activeGateways, &gw)
		}
	}

	if len(activeGateways) > 0 {
		//activeGateway.Spec.GatewayClassName
		for _, gw := range activeGateways {
			httpRoute.Status.Parents = nil

			for _, parentRef := range httpRoute.Spec.ParentRefs {
				if utils.IsRefToGateway(parentRef, utils.ObjectKey(gw)) {

					for _, listener := range gw.Spec.Listeners {

					}
				}
			}
			//gw.Spec.Listeners[0].Hostname
			//httpRoute.Spec.Hostnames[]

			for _, ref := range httpRoute.Spec.ParentRefs {
				//if ref.Group ==

				status := gwv1beta1.RouteParentStatus{
					ParentRef:      ref,
					ControllerName: commons.GatewayController,
				}

				httpRoute.Status.Parents = append(httpRoute.Status.Parents, status)
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *httpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.HTTPRoute{}).
		//Watches(
		//&source.Kind{Type: &gwv1beta1.Gateway{}},
		//handler.EnqueueRequestsFromMapFunc(r.gatewayToRoutes),
		//).
		Complete(r)
}

//func (r *httpRouteReconciler) gatewayToRoutes(obj client.Object) []reconcile.Request {
//    gateway, ok := obj.(*gwv1beta1.Gateway)
//    if !ok {
//        klog.Errorf("unexpected object type: %T", obj)
//        return nil
//    }
//
//
//}
