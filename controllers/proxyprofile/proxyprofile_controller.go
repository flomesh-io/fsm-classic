/*
 * MIT License
 *
 * Copyright (c) 2021-2022.  flomesh.io
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

package proxyprofile

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm/api/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProxyProfileReconciler reconciles a ProxyProfile object
type ProxyProfileReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	K8sApi   *kube.K8sAPI
}

// +kubebuilder:rbac:groups=flomesh.io,resources=proxyprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=flomesh.io,resources=proxyprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=flomesh.io,resources=proxyprofiles/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=core,resources=volumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func (r *ProxyProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = context.Background()
	klog.V(3).Infof("|=======> ProxyProfileReconciler received request for: %s <=======|", req.Name)

	// Fetch the ProxyProfile instance
	proxyProfile := &v1alpha1.ProxyProfile{}
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		proxyProfile,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("ProxyProfile resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get ProxyProfile, %#v", err)
		return ctrl.Result{}, err
	}

	klog.V(3).Infof("Processing ProxyProfile %s with ResourceVersion: %s", proxyProfile.Name, proxyProfile.ResourceVersion)
	klog.V(3).Infof("ProxyProfile %q, ConfigMode=%s, RestartPolicy=%s, RestartScope=%s",
		proxyProfile.Name, proxyProfile.GetConfigMode(), proxyProfile.Spec.RestartPolicy, proxyProfile.Spec.RestartScope)

	switch proxyProfile.GetConfigMode() {
	case v1alpha1.ProxyConfigModeLocal:
		// FIXME: for Local mode, this's not top priority, but need to implement the logic based on
		//      RestartPolicy and RestartScope.
		// apply resources, create/update
		result, err := r.applyResources(ctx, proxyProfile)
		if err != nil {
			r.Recorder.Eventf(proxyProfile, corev1.EventTypeWarning, "Failed",
				"Failed to create resoueces, %#v ", err)
			return result, err
		}
		if result.RequeueAfter > 0 || result.Requeue {
			klog.V(3).Infof("Requeuing ProxyProfile %s with ResourceVersion: %s due to resources change", proxyProfile.Name, proxyProfile.ResourceVersion)
			return result, nil
		}

		// update status
		statusResult, statusErr := r.updateProxyProfileStatus(ctx, proxyProfile)
		if err != nil {
			r.Recorder.Eventf(proxyProfile, corev1.EventTypeWarning, "Failed",
				"Failed to update status, %#v ", statusErr)
			return statusResult, statusErr
		}
		if statusResult.RequeueAfter > 0 || statusResult.Requeue {
			klog.V(3).Infof("Requeuing ProxyProfile %s with ResourceVersion: %s due to status change", proxyProfile.Name, proxyProfile.ResourceVersion)
			return statusResult, nil
		}
	case v1alpha1.ProxyConfigModeRemote:
		// do nothing
		// FIXME: for new ProxyProfile spec, if RepoBaseUrl, ParentCodebasePath, CodebasePath changes,
		//      it need to trigger reloading the pipy config/script from new localtion. And futher more, depending
		//      on different RestartPolicy and RestartScope, need to adjust the logic accordingly.
		switch proxyProfile.Spec.RestartPolicy {
		case v1alpha1.ProxyRestartPolicyAlways:
			// FIXME: check if the spec is changed, only changed ProxyProfile triggers the restart

			// FIXME: find all existing PODs those injected with this ProxyProfile, update them and restart

			ns := proxyProfile.Spec.Namespace
			if ns == "" {
				ns = corev1.NamespaceAll
			}

			pods := &corev1.PodList{}
			if err := r.List(
				ctx,
				pods,
				client.InNamespace(ns),
				client.MatchingLabels{
					commons.MatchedProxyProfileAnnotation: proxyProfile.Name,
				},
			); err != nil {
				klog.Errorf("Not able to list pods in namespace %q injected with ProxyProfile %q", ns, proxyProfile.Name)
			}

			switch proxyProfile.Spec.RestartScope {
			case v1alpha1.ProxyRestartScopePod:
				for _, po := range pods.Items {
					klog.V(5).Infof("|=================> Found pod %s/%s\n", po.Namespace, po.Name)

					// Delete the POD triggers a restart controlled by owner deployment/replicaset etc.
					if err := r.Delete(ctx, &po); err != nil {
						klog.Errorf("Restart POD %s/%s error, %s", po.Namespace, po.Name, err.Error())
						continue
					}
				}
			case v1alpha1.ProxyRestartScopeOwner:
				// FIXME: implement it， find owner of POD and rollout the POD by owner controller

			case v1alpha1.ProxyRestartScopeSidecar:
				// FIXME: implement it， restart ONLY sidecars
				// Not implemented yet, as restart sidecar may have potential issue as the init containers doesn't restarted as well
				//  Should consider if and how, probably we need to REMOVE this.
			default:
				// do nothing
			}
		case v1alpha1.ProxyRestartPolicyNever:
			// do nothing, ONLY inject new created POD with new config values
			klog.V(5).Infof("RestartPolicy of ProxyProfile %q is Never, only new created POD will be injected with latest version.", proxyProfile.Name)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ProxyProfileReconciler) applyResources(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile) (ctrl.Result, error) {
	requeue := false
	// If the ProxyProfile watches all applicable namespaces
	if proxyProfile.Spec.Namespace == "" {
		// 1. list all injectable namespaces
		namespaces := &corev1.NamespaceList{}
		if err := r.List(
			ctx,
			namespaces,
			client.MatchingLabels{
				commons.ProxyInjectNamespaceLabel: commons.ProxyInjectEnabled,
			},
		); err != nil {
			return ctrl.Result{}, err
		}

		// 2. create a configmap for each namespace
		for _, ns := range namespaces.Items {
			changed, err := r.createConfigMap(ctx, ns.Name, proxyProfile)
			if err != nil {
				klog.Errorf("Failed to create ConfigMap for ProxyProfile[%s] in Namespace[%s]", proxyProfile.Name, ns.Name)
				return ctrl.Result{}, err
			}
			if changed {
				requeue = true
			}
		}
	} else {
		// ONLY create ConfigMap in designated namespace
		ns := proxyProfile.Spec.Namespace
		if !injector.IsNamespaceProxyInjectLabelEnabled(r.Client, ns) {
			// Probably it's a wrong configuration, should be awared of un-injectable namespaces
			klog.V(3).Infof("The namespace[%s] in ProxyProfile[%s] is an ignored namespace as it doesn't have Label flomesh.io/inject=true.", ns, proxyProfile.Name)
			return ctrl.Result{}, nil
		}

		changed, err := r.createConfigMap(ctx, ns, proxyProfile)
		if err != nil {
			klog.Errorf("Failed to create ConfigMap for ProxyProfile[%s] in Namespace[%s]", proxyProfile.Name, ns)
			return ctrl.Result{}, err
		}
		if changed {
			requeue = true
		}
	}

	if requeue {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ProxyProfileReconciler) createConfigMap(ctx context.Context, namespace string, proxyProfile *v1alpha1.ProxyProfile) (bool, error) {
	// check if ns exists
	ns := &corev1.Namespace{}
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: namespace},
		ns,
	); err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("Namespace %s doesn't exist.", namespace)
			return false, nil
		}
		return false, err
	}

	// ns is being terminated
	if ns.Status.Phase == corev1.NamespaceTerminating {
		klog.V(3).Infof("Namespace %s is being terminated, ignore creating cm in it.", namespace)
		return false, nil
	}

	configmaps := &corev1.ConfigMapList{}
	if err := r.List(
		ctx,
		configmaps,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{
			Selector: proxyProfile.ConstructLabelSelector(),
		},
	); err != nil {
		klog.Errorf("Not able to list ConfigMaps in namespace %s by selector %#v, error=%#v", namespace, proxyProfile.ConstructLabelSelector(), err)
		return false, err
	}

	var found *corev1.ConfigMap
	switch len(configmaps.Items) {
	// Not found, create one in the namespace
	case 0:
		cmName := proxyProfile.GenerateConfigMapName(namespace)
		klog.V(3).Infof("Creating a new ConfigMap %s/%s for ProxyProfile %s", namespace, cmName, proxyProfile.Name)
		cm := r.configMapForProxyProfile(namespace, cmName, proxyProfile)
		if err := r.Create(ctx, cm); err != nil {
			klog.Errorf("Failed to create new ConfigMap %s/%s for ProxyProfile %s, error=%#v", namespace, cmName, proxyProfile.Name, err)
			return false, err
		}
		// ConfigMap created successfully - return and requeue
		r.Recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Created",
			"ConfigMap %s/%s is created successfully.", cm.Namespace, cm.Name)
		return true, nil
	// Found exactly ONE
	case 1:
		found = &configmaps.Items[0]
	// Other case, more than ONE?
	default:
		klog.Errorf("Found totally %d ConfigMap(s) for ProxyProfile %s in Namespace %s", len(configmaps.Items), proxyProfile.Name, namespace)
		return false, fmt.Errorf("more than ONE ConfigMaps are found in Namespace %s, it should be an 1:1 relationship(ProxyProfile:ConfigMap) in certain namespace", namespace)
	}

	// no errors, update ConfigMap
	// add an annotation of content hash to reduce the chance of update cm
	foundHash := found.Annotations[commons.ConfigHashAnnotation]
	proxyProfileHash := proxyProfile.Annotations[commons.ConfigHashAnnotation]
	// config changed
	if foundHash != proxyProfileHash {
		klog.V(3).Infof("ConfigMap %s/%s content changed, Old hash: %s, New hash: %s",
			namespace, found.Name, foundHash, proxyProfileHash)
		// update the annotation value to latest hash and the content of cm
		found.Annotations[commons.ConfigHashAnnotation] = proxyProfileHash
		found.Data = proxyProfile.Spec.Config
		if err := r.Update(ctx, found); err != nil {
			klog.Errorf("Not able to update ConfigMap, %#v", err)
			return false, err
		}
		r.Recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Updated",
			"ConfigMap %s/%s is updated successfully.", found.Namespace, found.Name)
		return true, nil
	}

	return false, nil
}

func (r *ProxyProfileReconciler) configMapForProxyProfile(namespace string, cmName string, proxyProfile *v1alpha1.ProxyProfile) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
			Labels:    proxyProfile.ConstructLabels(),
			Annotations: map[string]string{
				commons.ConfigHashAnnotation: proxyProfile.Annotations[commons.ConfigHashAnnotation],
			},
		},

		Data: proxyProfile.Spec.Config,
	}

	ctrl.SetControllerReference(proxyProfile, cm, r.Scheme)

	return cm
}

func (r *ProxyProfileReconciler) updateProxyProfileStatus(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile) (ctrl.Result, error) {
	// update status
	configmaps := &corev1.ConfigMapList{}
	if err := r.List(
		ctx,
		configmaps,
		client.MatchingLabelsSelector{
			Selector: proxyProfile.ConstructLabelSelector(),
		},
	); err != nil {
		klog.Errorf("Not able to list ConfigMaps error=%#v", err)
		return ctrl.Result{}, err
	}

	klog.V(3).Infof("Before cleaning, Found %d ConfigMaps for ProxyProfile %s", len(configmaps.Items), proxyProfile.Name)

	cfgs := make(map[string]string, 0)
	for _, cm := range configmaps.Items {
		if injector.IsNamespaceProxyInjectLabelEnabled(r.Client, cm.Namespace) {
			cfgs[cm.Namespace] = cm.Name
		} else {
			// GracePeriodSeconds: The value zero indicates delete immediately.
			// PropagationPolicy: DeletePropagationBackground
			//   Deletes the object from the key-value store, the garbage collector will
			//	 delete the dependents in the background.
			if err := r.Delete(
				ctx,
				&cm,
				client.GracePeriodSeconds(0),
				client.PropagationPolicy(metav1.DeletePropagationBackground),
			); err != nil {
				return ctrl.Result{}, err
			}

			r.Recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Deleted",
				"ConfigMap %s/%s is deleted successfully.", cm.Namespace, cm.Name)
		}
	}

	klog.V(3).Infof("After cleaning, there're %d ConfigMaps for ProxyProfile %s", len(cfgs), proxyProfile.Name)

	// some configmaps are cleaned up
	if len(cfgs) != len(configmaps.Items) {
		klog.V(3).Infof("Some related ConfigMaps are cleaned up due to namespace label is deleted/updated.")
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	if !reflect.DeepEqual(cfgs, proxyProfile.Status.ConfigMaps) {
		klog.V(3).Infof("Going to update ProxyProfile status...")
		klog.V(3).Infof("Current status configmaps: %#v", proxyProfile.Status.ConfigMaps)
		klog.V(3).Infof("New status configmaps: %#v", cfgs)

		proxyProfile.Status.ConfigMaps = cfgs
		if err := r.Status().Update(ctx, proxyProfile); err != nil {
			if errors.IsConflict(err) {
				// doesn't matter
				klog.Warning("Ignore duplicate/conflict updating, the object is stale.")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{RequeueAfter: 3 * time.Second, Requeue: true}, err
		}

		klog.V(3).Infof("Successfully updated status.")
		r.Recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Updated", "Successfully updated status.")

		return ctrl.Result{}, nil
	}

	klog.V(3).Infof("No status change, go ahead.")
	return ctrl.Result{}, nil
}

func (r *ProxyProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ProxyProfile{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&source.Kind{Type: &corev1.Namespace{}},
			&NamespaceEventHandler{Client: mgr.GetClient()},
			builder.WithPredicates(namespacePredicates),
		).
		//Watches(&source.Kind{Type: &corev1.Pod{}}, &PodRequestHandler{Client: mgr.GetClient()}).
		Complete(r)
}
