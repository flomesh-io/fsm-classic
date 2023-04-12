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
	_ "embed"
	"fmt"
	"github.com/flomesh-io/fsm/controllers"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/helm"
	ghodssyaml "github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"sort"
	"time"
)

var (
	//go:embed chart.tgz
	chartSource []byte

	// namespace <-> active gateway
	activeGateways map[string]*gwv1beta1.Gateway
)

type gatewayValues struct {
	Gateway *gwv1beta1.Gateway `json:"gwy,omitempty"`
}

type gatewayReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.FsmContext
}

func init() {
	activeGateways = make(map[string]*gwv1beta1.Gateway)
}

func NewGatewayReconciler(ctx *fctx.FsmContext) controllers.Reconciler {
	return &gatewayReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Gateway"),
		fctx:     ctx,
	}
}

func (r *gatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := &gwv1beta1.Gateway{}
	if err := r.fctx.Client.Get(
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

	var gatewayClasses gwv1beta1.GatewayClassList
	if err := r.fctx.Client.List(ctx, &gatewayClasses); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list gateway classes: %s", err)
	}

	var effectiveGatewayClass *gwv1beta1.GatewayClass
	for idx, cls := range gatewayClasses.Items {
		if isEffectiveGatewayClass(&cls) {
			effectiveGatewayClass = &gatewayClasses.Items[idx]
			break
		}
	}

	if effectiveGatewayClass == nil {
		klog.Warningf("No effective GatewayClass, ignore processing Gateway resource %s.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// 1. List all Gateways in the namespace whose GatewayClass is current effective class
	gatewayList := &gwv1beta1.GatewayList{}
	if err := r.fctx.Client.List(ctx, gatewayList, client.InNamespace(gateway.Namespace)); err != nil {
		klog.Errorf("Failed to list all gateways in namespace %s: %s", gateway.Namespace, err)
		return ctrl.Result{}, err
	}

	// 2. Find the oldest Gateway in the namespace, if CreateTimestamp is equal, then sort by alphabet order asc.
	// If spec.GatewayClassName equals effectiveGatewayClass then it's a valid gateway
	// Otherwise, it's invalid
	validGateways := make([]*gwv1beta1.Gateway, 0)
	invalidGateways := make([]*gwv1beta1.Gateway, 0)

	for _, gw := range gatewayList.Items {
		if string(gw.Spec.GatewayClassName) == effectiveGatewayClass.Name {
			validGateways = append(validGateways, &gw)
		} else {
			invalidGateways = append(invalidGateways, &gw)
		}
	}

	sort.Slice(validGateways, func(i, j int) bool {
		if validGateways[i].CreationTimestamp.Time.Equal(validGateways[j].CreationTimestamp.Time) {
			return validGateways[i].Name < validGateways[j].Name
		} else {
			return validGateways[i].CreationTimestamp.Time.Before(validGateways[j].CreationTimestamp.Time)
		}
	})

	// 3. Set the oldest as Accepted and the rest are unaccepted
	statusChangedGateways := make([]*gwv1beta1.Gateway, 0)
	for i := range validGateways {
		if i == 0 {
			if !isAcceptedGateway(validGateways[i]) {
				r.setAccepted(validGateways[i])
				statusChangedGateways = append(statusChangedGateways, validGateways[i])
			}
		} else {
			if isAcceptedGateway(validGateways[i]) {
				r.setUnaccepted(validGateways[i])
				statusChangedGateways = append(statusChangedGateways, validGateways[i])
			}
		}
	}

	// in case of effective GatewayClass changed or spec.GatewayClassName was changed
	for i := range invalidGateways {
		if isAcceptedGateway(invalidGateways[i]) {
			r.setUnaccepted(invalidGateways[i])
			statusChangedGateways = append(statusChangedGateways, invalidGateways[i])
		}
	}

	// 4. update status
	for _, gw := range statusChangedGateways {
		if err := r.fctx.Client.Status().Update(ctx, gw); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 5. after all status of gateways in the namespace have been updated successfully
	//   list all gateways in the namespace and deploy/redeploy the effective one
	activeGateway, result, err := r.findActiveGateway(ctx, gateway)
	if err != nil {
		return result, err
	}

	if activeGateway != nil && !isSameGateway(activeGateways[gateway.Namespace], activeGateway) {
		result, err = r.applyGateway(activeGateway)
		if err != nil {
			return result, err
		}

		activeGateways[gateway.Namespace] = activeGateway
	}

	// TODO: implement it
	// 6. update addresses of Gateway status if any IP is allocated

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) findActiveGateway(ctx context.Context, gateway *gwv1beta1.Gateway) (*gwv1beta1.Gateway, ctrl.Result, error) {
	gatewayList := &gwv1beta1.GatewayList{}
	if err := r.fctx.Client.List(ctx, gatewayList, client.InNamespace(gateway.Namespace)); err != nil {
		klog.Errorf("Failed to list all gateways in namespace %s: %s", gateway.Namespace, err)
		return nil, ctrl.Result{}, err
	}

	for _, gw := range gatewayList.Items {
		if isAcceptedGateway(&gw) {
			return &gw, ctrl.Result{}, nil
		}
	}

	return nil, ctrl.Result{}, nil
}

func isSameGateway(oldGateway, newGateway *gwv1beta1.Gateway) bool {
	return equality.Semantic.DeepEqual(oldGateway, newGateway)
}

func (r *gatewayReconciler) applyGateway(gateway *gwv1beta1.Gateway) (ctrl.Result, error) {
	mc := r.fctx.ConfigStore.MeshConfig.GetConfig()

	result, err := r.deriveCodebases(gateway, mc)
	if err != nil {
		return result, err
	}

	result, err = r.updateConfig(gateway, mc)
	if err != nil {
		return result, err
	}

	return r.deployGateway(gateway, mc)
}

func (r *gatewayReconciler) deriveCodebases(gw *gwv1beta1.Gateway, mc *config.MeshConfig) (ctrl.Result, error) {
	gwPath := mc.GatewayCodebasePath(gw.Namespace)
	parentPath := mc.GetDefaultGatewaysPath()
	if err := r.fctx.RepoClient.DeriveCodebase(gwPath, parentPath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateConfig(gw *gwv1beta1.Gateway, mc *config.MeshConfig) (ctrl.Result, error) {
	// TODO: update pipy repo
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) deployGateway(gw *gwv1beta1.Gateway, mc *config.MeshConfig) (ctrl.Result, error) {
	releaseName := fmt.Sprintf("fsm-gateway-%s", gw.Namespace)
	if ctrlResult, err := helm.RenderChart(releaseName, gw, chartSource, mc, r.fctx.Client, r.fctx.Scheme, resolveValues); err != nil {
		return ctrlResult, err
	}

	return ctrl.Result{}, nil
}

func resolveValues(object metav1.Object, mc *config.MeshConfig) (map[string]interface{}, error) {
	gateway, ok := object.(*gwv1beta1.Gateway)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of *gwv1beta1.Gateway", object)
	}

	klog.V(5).Infof("[GW] Resolving Values ...")

	gwBytes, err := ghodssyaml.Marshal(&gatewayValues{Gateway: gateway})
	if err != nil {
		return nil, fmt.Errorf("convert Gateway to yaml, err = %#v", err)
	}
	klog.V(5).Infof("\n\nGATEWAY VALUES YAML:\n\n\n%s\n\n", string(gwBytes))
	nsigValues, err := chartutil.ReadValues(gwBytes)
	if err != nil {
		return nil, err
	}

	finalValues := nsigValues.AsMap()

	overrides := []string{
		"fsm.gatewayApi.enabled=true",
		"fsm.ingress.enabled=false",
		fmt.Sprintf("fsm.image.repository=%s", mc.Images.Repository),
		fmt.Sprintf("fsm.namespace=%s", config.GetFsmNamespace()),
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func (r *gatewayReconciler) setAccepted(gateway *gwv1beta1.Gateway) {
	metautil.SetStatusCondition(&gateway.Status.Conditions, metav1.Condition{
		Type:               string(gwv1beta1.GatewayConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1beta1.GatewayReasonAccepted),
		Message:            fmt.Sprintf("Gateway %s/%s is accepted.", gateway.Namespace, gateway.Name),
	})
}

func (r *gatewayReconciler) setUnaccepted(gateway *gwv1beta1.Gateway) {
	metautil.SetStatusCondition(&gateway.Status.Conditions, metav1.Condition{
		Type:               string(gwv1beta1.GatewayConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Unaccepted",
		Message:            fmt.Sprintf("Gateway %s/%s is not accepted as it's not the oldest one in namespace %q.", gateway.Namespace, gateway.Name, gateway.Namespace),
	})
}

func isAcceptedGateway(gateway *gwv1beta1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1beta1.GatewayConditionAccepted))
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.Gateway{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			gateway, ok := obj.(*gwv1beta1.Gateway)
			if !ok {
				klog.Errorf("unexpected object type %T", obj)
				return false
			}

			gatewayClass, err := r.fctx.K8sAPI.GatewayAPIClient.
				GatewayV1beta1().
				GatewayClasses().
				Get(context.TODO(), string(gateway.Spec.GatewayClassName), metav1.GetOptions{})
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

func (r *gatewayReconciler) gatewayClassToGateways(obj client.Object) []reconcile.Request {
	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		klog.Errorf("unexpected object type: %T", obj)
		return nil
	}

	if isEffectiveGatewayClass(gatewayClass) {
		var gateways gwv1beta1.GatewayList
		if err := r.fctx.Client.List(context.TODO(), &gateways); err != nil {
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
