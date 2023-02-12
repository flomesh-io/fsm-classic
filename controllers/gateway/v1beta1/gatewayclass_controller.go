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
    "github.com/flomesh-io/fsm/pkg/commons"
    "github.com/flomesh-io/fsm/pkg/kube"
    "k8s.io/apimachinery/pkg/api/errors"
    metautil "k8s.io/apimachinery/pkg/api/meta"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/client-go/tools/record"
    "k8s.io/klog/v2"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/builder"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/predicate"
    gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
    "time"
)

type GatewayClassReconciler struct {
	client.Client
	K8sAPI   *kube.K8sAPI
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    gatewayClass := &gwv1beta1.GatewayClass{}
    if err := r.Get(
        ctx,
        client.ObjectKey{Name: req.Name},
        gatewayClass,
    ); err != nil {
        if errors.IsNotFound(err) {
            // Request object not found, could have been deleted after reconcile request.
            // Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
            // Return and don't requeue
            klog.V(3).Info("GatewayClass resource not found. Ignoring since object must be deleted")
            return ctrl.Result{}, nil
        }
        // Error reading the object - requeue the request.
        klog.Errorf("Failed to get GatewayClass, %#v", err)
        return ctrl.Result{}, err
    }

    metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
        Type:               string(gwv1beta1.GatewayClassConditionStatusAccepted),
        Status:             metav1.ConditionTrue,
        ObservedGeneration: gatewayClass.Generation,
        LastTransitionTime: metav1.Time{Time: time.Now()},
        Reason:             string(gwv1beta1.GatewayClassReasonAccepted),
        Message:            fmt.Sprintf("GatewayClass %q is accepted.", req.Name),
    })

    if err := r.Status().Update(ctx, gatewayClass); err != nil {
        return ctrl.Result{}, err
    }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
    gwclsPrct := predicate.NewPredicateFuncs(func(object client.Object) bool {
        gatewayClass, ok := object.(*gwv1beta1.GatewayClass)
        if !ok {
            klog.Infof("unexpected object type: %T", object)
            return false
        }

        return gatewayClass.Spec.ControllerName == commons.GatewayController
    })

	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.GatewayClass{}, builder.WithPredicates(gwclsPrct)).
		Complete(r)
}


