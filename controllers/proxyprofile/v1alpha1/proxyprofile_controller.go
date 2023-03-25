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
	"fmt"
	pfv1alpha1 "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1"
	pfhelper "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1/helper"
	"github.com/flomesh-io/fsm/controllers"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

var (
	hashStore = map[string]string{}
)

// ProxyProfileReconciler reconciles a ProxyProfile object
type reconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.FsmContext
}

func NewReconciler(ctx *fctx.FsmContext) controllers.Reconciler {
	return &reconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ProxyProfile"),
		fctx:     ctx,
	}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = context.Background()
	klog.V(3).Infof("|=======> ProxyProfileReconciler received request for: %s <=======|", req.Name)

	// Fetch the ProxyProfile instance
	pf := &pfv1alpha1.ProxyProfile{}
	if err := r.fctx.Client.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		pf,
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

	klog.V(3).Infof("Processing ProxyProfile %s with ResourceVersion: %s", pf.Name, pf.ResourceVersion)
	klog.V(3).Infof("ProxyProfile %q, ConfigMode=%s, RestartPolicy=%s, RestartScope=%s",
		pf.Name, pf.GetConfigMode(), pf.Spec.RestartPolicy, pf.Spec.RestartScope)

	mc := r.fctx.ConfigStore.MeshConfig.GetConfig()

	switch pf.GetConfigMode() {
	case pfv1alpha1.ProxyConfigModeLocal:
		return r.reconcileLocalMode(ctx, pf)
	case pfv1alpha1.ProxyConfigModeRemote:
		return r.reconcileRemoteMode(ctx, pf, mc)
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) reconcileLocalMode(ctx context.Context, pf *pfv1alpha1.ProxyProfile) (ctrl.Result, error) {
	// FIXME: for Local mode, this's not top priority, but need to implement the logic based on
	//      RestartPolicy and RestartScope.
	// apply resources, create/update
	result, err := r.applyResources(ctx, pf)
	if err != nil {
		r.recorder.Eventf(pf, corev1.EventTypeWarning, "Failed",
			"Failed to create resources, %#v ", err)
		return result, err
	}
	if result.RequeueAfter > 0 || result.Requeue {
		klog.V(3).Infof("Requeue ProxyProfile %s with ResourceVersion: %s due to resources change", pf.Name, pf.ResourceVersion)
		return result, nil
	}

	// update status
	statusResult, statusErr := r.updateProxyProfileStatus(ctx, pf)
	if err != nil {
		r.recorder.Eventf(pf, corev1.EventTypeWarning, "Failed",
			"Failed to update status, %#v ", statusErr)
		return statusResult, statusErr
	}
	if statusResult.RequeueAfter > 0 || statusResult.Requeue {
		klog.V(3).Infof("Requeue ProxyProfile %s with ResourceVersion: %s due to status change", pf.Name, pf.ResourceVersion)
		return statusResult, nil
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) reconcileRemoteMode(ctx context.Context, pf *pfv1alpha1.ProxyProfile, mc *config.MeshConfig) (ctrl.Result, error) {
	// check if the spec is changed, only changed ProxyProfile triggers the restart
	oldHash := hashStore[pf.Name]
	hash := pf.Annotations[commons.SpecHashAnnotation]
	if hash == "" {
		// It should not be empty, if it's empty, recalculate and update
		hash = pf.SpecHash()
		pf.Annotations[commons.SpecHashAnnotation] = hash
		if err := r.fctx.Client.Update(ctx, pf); err != nil {
			return ctrl.Result{}, err
		}
	}
	klog.V(5).Infof("Old Hash=%q, New Hash=%q.", oldHash, hash)
	if oldHash == hash {
		klog.V(5).Infof("Hash of ProxyProfile %q doesn't change, skipping...", pf.Name)
		return ctrl.Result{}, nil
	}

	result, err := r.deriveCodebases(pf, mc)
	if err != nil {
		klog.Errorf("Deriving codebase error: %#v", err)
		return result, err
	}

	switch pf.Spec.RestartPolicy {
	case pfv1alpha1.ProxyRestartPolicyAlways:
		// find all existing PODs those injected with this ProxyProfile, update them and restart
		pods, err := r.findInjectedPods(ctx, pf)
		if err != nil {
			klog.Errorf("Finding controllee of ProxyProfile %q, %#v", pf.Name, err)
			return ctrl.Result{}, err
		}

		// TODO: need to consider partial restarted PF pods.
		//  PF has annotation flomesh.io/last-updated,
		//  POD has annotation kubectl.kubernetes.io/restartedAt,
		//  Their values should equal for those pods restarted successfully
		switch pf.Spec.RestartScope {
		//case pfv1alpha1.ProxyRestartScopePod:
		//    result, err := r.proxyRestartScopePod(ctx, pods)
		//    if err != nil {
		//        return result, err
		//    }
		case pfv1alpha1.ProxyRestartScopeOwner:
			result, err := r.proxyRestartScopeOwner(ctx, pf, pods)
			if err != nil {
				return result, err
			}
		//case pfv1alpha1.ProxyRestartScopeSidecar:
		// FIXME: implement it， restart ONLY sidecars
		// Not implemented yet, as restart sidecar may have potential issue as the init containers doesn't restarted as well
		//  Should consider if and how, probably we need to REMOVE this.
		default:
			// do nothing
		}
	case pfv1alpha1.ProxyRestartPolicyNever:
		// do nothing, ONLY inject new created POD with new config values
		klog.V(5).Infof("RestartPolicy of ProxyProfile %q is Never, only new created POD will be injected with latest version.", pf.Name)
	}

	// update the local hash store in case of success
	klog.V(5).Infof("Updating hash ...")
	hashStore[pf.Name] = hash
	klog.V(5).Infof("Hash of %q has been updated to %q", pf.Name, hashStore[pf.Name])

	return ctrl.Result{}, nil
}

func (r *reconciler) proxyRestartScopePod(ctx context.Context, pods []corev1.Pod) (ctrl.Result, error) {
	for _, po := range pods {
		klog.V(5).Infof("|=================> Found pod %s/%s\n", po.Namespace, po.Name)
		if po.Status.Phase != corev1.PodRunning {
			// ignore not running pods
			continue
		}

		if metav1.GetControllerOf(&po) != nil {
			// Delete the POD triggers a restart controlled by owner deployment/replicaset etc.
			if err := r.fctx.Client.Delete(ctx, &po); err != nil {
				klog.Errorf("Restart POD %s/%s error, %s", po.Namespace, po.Name, err.Error())
				return ctrl.Result{}, err
			}
		} else {
			// It's a POD has no controller, create a copy, delete the old pod then create a new one with the copy
			result, err := r.restartSinglePod(ctx, po)
			if err != nil {
				return result, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) proxyRestartScopeOwner(ctx context.Context, pf *pfv1alpha1.ProxyProfile, pods []corev1.Pod) (ctrl.Result, error) {
	replicaSets := sets.String{}
	deployments := sets.String{}
	daemonSets := sets.String{}
	replicationControllers := sets.String{}
	statefulSets := sets.String{}

	errs := map[string]error{}

	for _, po := range pods {
		klog.V(5).Infof("|=================> Found pod %s/%s\n", po.Namespace, po.Name)

		owner := metav1.GetControllerOf(&po)
		if owner != nil {
			resource := fmt.Sprintf("%s/%s", po.Namespace, owner.Name)
			kind := strings.ToLower(owner.Kind)

			// ignore jobs & cronjobs, pods may have same owner, need to aggregate
			switch kind {
			case "replicaset":
				replicaSets.Insert(resource)
			case "deployment":
				deployments.Insert(resource)
			case "daemonset":
				daemonSets.Insert(resource)
			case "replicationcontroller":
				replicationControllers.Insert(resource)
			case "statefulset":
				statefulSets.Insert(resource)
			}
		} else {
			// It's a POD has no controller, create a copy, delete the old pod then create a new one with the copy
			result, err := r.restartSinglePod(ctx, po)
			if err != nil {
				return result, err
			}
		}
	}

	patch := fmt.Sprintf(
		`{"spec": {"template":{"metadata": {"labels": {"%s": "%s"}}}}}`,
		commons.ProxyProfileLastUpdated,
		pf.Annotations[commons.ProxyProfileLastUpdated],
	)
	klog.V(5).Infof("patch = %s", patch)

	for _, rs := range replicaSets.List() {
		klog.V(5).Infof("Rollout restart ReplicaSet %q ...", rs)
		strs := strings.Split(rs, "/")
		_, err := r.fctx.K8sAPI.Client.AppsV1().
			ReplicaSets(strs[0]).
			Patch(context.TODO(), strs[1], types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})

		if err != nil {
			klog.Errorf("Rollout restart ReplicaSet %s/%s error, %s", strs[0], strs[1], err.Error())
			errs[fmt.Sprintf("rs/%s", rs)] = err
		}
	}

	for _, dp := range deployments.List() {
		klog.V(5).Infof("Rollout restart Deployment %q ...", dp)
		strs := strings.Split(dp, "/")
		_, err := r.fctx.K8sAPI.Client.AppsV1().
			Deployments(strs[0]).
			Patch(context.TODO(), strs[1], types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("Rollout restart Deployment %s/%s error, %s", strs[0], strs[1], err.Error())
			errs[fmt.Sprintf("dp/%s", dp)] = err
		}
	}

	for _, ds := range daemonSets.List() {
		klog.V(5).Infof("Rollout restart DaemonSet %q ...", ds)
		strs := strings.Split(ds, "/")
		_, err := r.fctx.K8sAPI.Client.AppsV1().
			DaemonSets(strs[0]).
			Patch(context.TODO(), strs[1], types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("Rollout restart DaemonSet %s/%s error, %s", strs[0], strs[1], err.Error())
			errs[fmt.Sprintf("ds/%s", ds)] = err
		}
	}

	for _, ss := range statefulSets.List() {
		klog.V(5).Infof("Rollout restart StatefulSet %q ...", ss)
		strs := strings.Split(ss, "/")
		_, err := r.fctx.K8sAPI.Client.AppsV1().
			StatefulSets(strs[0]).
			Patch(context.TODO(), strs[1], types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("Rollout restart StatefulSet %s/%s error, %s", strs[0], strs[1], err.Error())
			errs[fmt.Sprintf("ss/%s", ss)] = err
		}
	}

	for _, rc := range replicationControllers.List() {
		klog.V(5).Infof("Rollout restart ReplicationController %q ...", rc)
		strs := strings.Split(rc, "/")
		_, err := r.fctx.K8sAPI.Client.CoreV1().
			ReplicationControllers(strs[0]).
			Patch(context.TODO(), strs[1], types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("Rollout restart ReplicationController %s/%s error, %s", strs[0], strs[1], err.Error())
			errs[fmt.Sprintf("rc/%s", rc)] = err
		}
	}

	if len(errs) != 0 {
		return ctrl.Result{RequeueAfter: 3 * time.Second}, fmt.Errorf("%#v", errs)
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) restartSinglePod(ctx context.Context, po corev1.Pod) (ctrl.Result, error) {
	podCopy := po.DeepCopy()

	if err := r.fctx.Client.Delete(ctx, &po); err != nil {
		klog.Errorf("Delete POD %s/%s error, %s", po.Namespace, po.Name, err.Error())
		return ctrl.Result{}, err
	}

	if podCopy.GenerateName != "" {
		// do nothing, just go ahead
	} else {
		//FIXME: it has a static name, need to wait till the old pod is deleted and is terminated???
	}

	if err := r.fctx.Client.Create(context.TODO(), podCopy); err != nil {
		klog.Errorf("Create POD %s/%s error, %s", podCopy.Namespace, podCopy.Name, err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) deriveCodebases(pf *pfv1alpha1.ProxyProfile, mc *config.MeshConfig) (ctrl.Result, error) {
	repoClient := repo.NewRepoClient(mc.RepoRootURL())

	// ProxyProfile codebase derives service codebase
	pfPath := pfhelper.GetProxyProfilePath(pf.Name, mc)
	pfParentPath := pfhelper.GetProxyProfileParentPath(mc)
	klog.V(5).Infof("Deriving service codebase of ProxyProfile %q", pf.Name)
	if err := repoClient.DeriveCodebase(pfPath, pfParentPath); err != nil {
		klog.Errorf("Deriving service codebase of ProxyProfile %q error: %#v", pf.Name, err)
		return ctrl.Result{RequeueAfter: 3 * time.Second}, err
	}

	// sidecar codebase derives ProxyProfile codebase
	for _, sidecar := range pf.Spec.Sidecars {
		sidecarPath := pfhelper.GetSidecarPath(pf.Name, sidecar.Name, mc)
		klog.V(5).Infof("Deriving codebase of sidecar %q of ProxyProfile %q", sidecar.Name, pf.Name)
		if err := repoClient.DeriveCodebase(sidecarPath, pfPath); err != nil {
			klog.Errorf("Deriving codebase of sidecar %q of ProxyProfile %q error: %#v", sidecar.Name, pf.Name, err)
			return ctrl.Result{RequeueAfter: 3 * time.Second}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) findInjectedPods(ctx context.Context, pf *pfv1alpha1.ProxyProfile) ([]corev1.Pod, error) {
	ns := pf.Spec.Namespace
	if ns == "" {
		ns = corev1.NamespaceAll
	}

	klog.V(5).Infof("ProxyProfile %q, %s=%s", pf.Name, commons.MatchedProxyProfile, pf.Name)
	klog.V(5).Infof("ProxyProfile %q, %s=%s", pf.Name, commons.ProxyProfileLastUpdated, pf.Annotations[commons.ProxyProfileLastUpdated])

	// flomesh.io/proxy-profile=pf.Name && flomesh.io/last-updated=pf.lastUpdated
	labelSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      commons.MatchedProxyProfile,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{pf.Name},
			},
			{
				Key:      commons.ProxyProfileLastUpdated,
				Operator: metav1.LabelSelectorOpNotIn,
				Values:   []string{pf.Annotations[commons.ProxyProfileLastUpdated]},
			},
		},
	}

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		klog.Errorf("Converting LabelSelector to Selector error, %#v", err)
		return nil, err
	}
	klog.V(5).Infof("Selector is %#v", selector)

	pods := &corev1.PodList{}
	if err := r.fctx.Client.List(
		ctx,
		pods,
		client.InNamespace(ns),
		client.MatchingLabelsSelector{
			Selector: selector,
		},
	); err != nil {
		klog.Errorf("Not able to list pods in namespace %q injected with ProxyProfile %q", ns, pf.Name)
		return nil, err
	}

	klog.V(5).Infof("Found %d PODs.", len(pods.Items))

	return pods.Items, nil
}

func (r *reconciler) applyResources(ctx context.Context, proxyProfile *pfv1alpha1.ProxyProfile) (ctrl.Result, error) {
	requeue := false
	// If the ProxyProfile watches all applicable namespaces
	if proxyProfile.Spec.Namespace == "" {
		// 1. list all injectable namespaces
		namespaces := &corev1.NamespaceList{}
		if err := r.fctx.Client.List(
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
		if !injector.IsNamespaceProxyInjectLabelEnabled(r.fctx.Client, ns) {
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

func (r *reconciler) createConfigMap(ctx context.Context, namespace string, proxyProfile *pfv1alpha1.ProxyProfile) (bool, error) {
	// check if ns exists
	ns := &corev1.Namespace{}
	if err := r.fctx.Client.Get(
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
	if err := r.fctx.Client.List(
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
		if err := r.fctx.Client.Create(ctx, cm); err != nil {
			klog.Errorf("Failed to create new ConfigMap %s/%s for ProxyProfile %s, error=%#v", namespace, cmName, proxyProfile.Name, err)
			return false, err
		}
		// ConfigMap created successfully - return and requeue
		r.recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Created",
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
		if err := r.fctx.Client.Update(ctx, found); err != nil {
			klog.Errorf("Not able to update ConfigMap, %#v", err)
			return false, err
		}
		r.recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Updated",
			"ConfigMap %s/%s is updated successfully.", found.Namespace, found.Name)
		return true, nil
	}

	return false, nil
}

func (r *reconciler) configMapForProxyProfile(namespace string, cmName string, proxyProfile *pfv1alpha1.ProxyProfile) *corev1.ConfigMap {
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

	ctrl.SetControllerReference(proxyProfile, cm, r.fctx.Scheme)

	return cm
}

func (r *reconciler) updateProxyProfileStatus(ctx context.Context, proxyProfile *pfv1alpha1.ProxyProfile) (ctrl.Result, error) {
	// update status
	configmaps := &corev1.ConfigMapList{}
	if err := r.fctx.Client.List(
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
		if injector.IsNamespaceProxyInjectLabelEnabled(r.fctx.Client, cm.Namespace) {
			cfgs[cm.Namespace] = cm.Name
		} else {
			// GracePeriodSeconds: The value zero indicates delete immediately.
			// PropagationPolicy: DeletePropagationBackground
			//   Deletes the object from the key-value store, the garbage aggregator will
			//	 delete the dependents in the background.
			if err := r.fctx.Client.Delete(
				ctx,
				&cm,
				client.GracePeriodSeconds(0),
				client.PropagationPolicy(metav1.DeletePropagationBackground),
			); err != nil {
				return ctrl.Result{}, err
			}

			r.recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Deleted",
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
		if err := r.fctx.Client.Status().Update(ctx, proxyProfile); err != nil {
			if errors.IsConflict(err) {
				// doesn't matter
				klog.Warning("Ignore duplicate/conflict updating, the object is stale.")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{RequeueAfter: 3 * time.Second, Requeue: true}, err
		}

		klog.V(3).Infof("Successfully updated status.")
		r.recorder.Eventf(proxyProfile, corev1.EventTypeNormal, "Updated", "Successfully updated status.")

		return ctrl.Result{}, nil
	}

	klog.V(3).Infof("No status change, go ahead.")
	return ctrl.Result{}, nil
}

func (r *reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pfv1alpha1.ProxyProfile{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&source.Kind{Type: &corev1.Namespace{}},
			&NamespaceEventHandler{Client: mgr.GetClient()},
			builder.WithPredicates(namespacePredicates),
		).
		Complete(r)
}
