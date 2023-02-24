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
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/kube"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayReconciler struct {
	client.Client
	K8sAPI   *kube.K8sAPI
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    gateway := &gwv1beta1.Gateway{}
    if err := r.Get(
        ctx,
        req.NamespacedName,
        gateway,
    ); err != nil {
        if errors.IsNotFound(err) {
            // Request object not found, could have been deleted after reconcile request.
            // Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
            // Return and don't requeue
            klog.V(3).Info("Gateway resource not found. Ignoring since object must be deleted")
            return ctrl.Result{}, nil
        }
        // Error reading the object - requeue the request.
        klog.Errorf("Failed to get Gateway, %#v", err)
        return ctrl.Result{}, err
    }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.Gateway{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
            gateway, ok := obj.(*gwv1beta1.Gateway)
            if !ok {
                klog.Errorf("unexpected object type %T", obj)
                return false
            }

            gatewayClass, err := r.K8sAPI.GatewayAPIClient.
                GatewayV1beta1().
                GatewayClasses().
                Get(context.TODO(),  string(gateway.Spec.GatewayClassName), metav1.GetOptions{})
            if err != nil {
                klog.Errorf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
                return false
            }

            if gatewayClass.Spec.ControllerName != commons.GatewayController {
                klog.Warningf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, commons.GatewayController)
                return false
            }

            return true
        }))).
		Watches(
			&source.Kind{Type: &gwv1beta1.GatewayClass{}},
			handler.EnqueueRequestsFromMapFunc(r.gatewayClassToGateways),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
				if !ok {
					klog.Errorf("unexpected object type: %T", obj)
					return false
				}

				return gatewayClass.Spec.ControllerName == commons.GatewayController
			})),
		).
		Complete(r)
}

func (r *GatewayReconciler) gatewayClassToGateways(obj client.Object) []reconcile.Request {
	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		klog.Errorf("unexpected object type: %T", obj)
		return nil
	}

	if isEffectiveGatewayClass(gatewayClass) {
		var gateways gwv1beta1.GatewayList
		if err := r.Client.List(context.Background(), &gateways); err != nil {
			klog.Error("error listing gateways: %s", err)
			return nil
		}

		var reconciles []reconcile.Request
		for _, gw := range gateways.Items {
			if string(gw.Spec.GatewayClassName) == gatewayClass.GetName() {
				reconciles = append(reconciles, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: gw.Namespace,
						Name:      gw.Name,
					},
				})
			}
		}

		return reconciles
	}

	return nil
}
