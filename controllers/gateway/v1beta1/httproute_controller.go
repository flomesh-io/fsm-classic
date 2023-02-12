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
	"github.com/flomesh-io/fsm/pkg/kube"
    corev1 "k8s.io/api/core/v1"
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

type HTTPRouteReconciler struct {
	client.Client
	K8sAPI   *kube.K8sAPI
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.HTTPRoute{}).
        Watches(
            &source.Kind{Type: &gwv1beta1.Gateway{}},
            handler.EnqueueRequestsFromMapFunc(r.gatewayToHTTPRoutes),
            builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
                gateway, ok := obj.(*gwv1beta1.Gateway)
                if !ok {
                    klog.Infof("unexpected object type: %T", obj)
                    return false
                }

                gateway.
            })),
        ).
        Watches(
            &source.Kind{Type: &corev1.Service{}},
            handler.EnqueueRequestsFromMapFunc(r.serviceToHTTPRoutes),
        ).
		Complete(r)
}

func (r *HTTPRouteReconciler) gatewayToHTTPRoutes(gateway client.Object) []reconcile.Request {
    var httpRoutes gwv1beta1.HTTPRouteList
    if err := r.Client.List(context.Background(), &httpRoutes); err != nil {
        klog.Error("error listing gateways")
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

func (r *HTTPRouteReconciler) serviceToHTTPRoutes(obj client.Object) []reconcile.Request {
    svc, ok := obj.(*corev1.Service)
    if !ok {
        klog.Infof("unexpected object type: %T", obj)
        return nil
    }

    ingresses, err := r.K8sAPI.GatewayAPIClient.GatewayV1beta1().HTTPRoutes()
        Ingresses(svc.Namespace).
        List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        klog.Error("error listing ingresses: %s", err)
        return nil
    }

    var reconciles []reconcile.Request
    for _, ing := range ingresses.Items {
        for _, rule := range ing.Spec.Rules {
            if rule.HTTP == nil {
                continue
            }

            for _, path := range rule.HTTP.Paths {
                if path.Backend.Service.Name == svc.Name {
                    reconciles = append(reconciles, reconcile.Request{
                        NamespacedName: types.NamespacedName{
                            Namespace: ing.Namespace,
                            Name:      ing.Name,
                        },
                    })
                }
            }
        }
    }

    return reconciles
}


