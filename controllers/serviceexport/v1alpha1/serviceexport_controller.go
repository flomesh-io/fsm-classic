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

package v1alpha1

import (
	"context"
	_ "embed"
	"fmt"
	svcexpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// ServiceExportReconciler reconciles a ServiceExport object
type ServiceExportReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the ServiceExport closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceExport object against the actual ServiceExport state, and then
// perform operations to make the ServiceExport state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ServiceExportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()

	serviceExport := &svcexpv1alpha1.ServiceExport{}
	if err := r.Get(
		ctx,
		req.NamespacedName,
		serviceExport,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("[ServiceExport] ServiceExport resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get ServiceExport, %#v", err)
		return ctrl.Result{}, err
	}

	svc := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		// the service doesn't exist
		if errors.IsNotFound(err) {
			serviceExport.Status.Conditions = []metav1.Condition{
				{
					Type:               string(svcexpv1alpha1.ServiceExportValid),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: serviceExport.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             "Failed",
					Message:            fmt.Sprintf("Service %s not found", req.NamespacedName),
				},
			}
			if err := r.Status().Update(ctx, serviceExport); err != nil {
				return ctrl.Result{}, err
			}
		}

		// unknown errors
		serviceExport.Status.Conditions = []metav1.Condition{
			{
				Type:               string(svcexpv1alpha1.ServiceExportValid),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: serviceExport.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "Failed",
				Message:            fmt.Sprintf("Get Service %s error: %s", req.NamespacedName, err),
			},
		}
		if err := r.Status().Update(ctx, serviceExport); err != nil {
			return ctrl.Result{}, err
		}

		// stop processing
		return ctrl.Result{}, nil
	}

	// Found service

	// service is being deleted
	if svc.DeletionTimestamp != nil {
		serviceExport.Status.Conditions = []metav1.Condition{
			{
				Type:               string(svcexpv1alpha1.ServiceExportValid),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: serviceExport.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "Failed",
				Message:            fmt.Sprintf("Service %s is being deleted.", req.NamespacedName),
			},
		}
		if err := r.Status().Update(ctx, serviceExport); err != nil {
			return ctrl.Result{}, err
		}

		// stop processing
		return ctrl.Result{}, nil
	}

	// ExternalName service cannot be exported
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		serviceExport.Status.Conditions = []metav1.Condition{
			{
				Type:               string(svcexpv1alpha1.ServiceExportValid),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: serviceExport.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "Failed",
				Message:            fmt.Sprintf("Type of Service %s is %s, cannot be exported.", req.NamespacedName, corev1.ServiceTypeExternalName),
			},
		}
		if err := r.Status().Update(ctx, serviceExport); err != nil {
			return ctrl.Result{}, err
		}

		// stop processing
		return ctrl.Result{}, nil
	}

	// service is exported successfully
	serviceExport.Status.Conditions = []metav1.Condition{
		{
			Type:               string(svcexpv1alpha1.ServiceExportValid),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: serviceExport.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Success",
			Message:            fmt.Sprintf("Service %s is exported successfully.", req.NamespacedName),
		},
	}

	if err := r.Status().Update(ctx, serviceExport); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceExportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&svcexpv1alpha1.ServiceExport{}).
		Complete(r)
}
