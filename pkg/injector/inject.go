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

package injector

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm/api/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// Check whether the target resoured need to be mutated
func (pi *ProxyInjector) isInjectionRequired(pod *corev1.Pod) bool {
	// 0. if pod use host network, ignore it
	if pod.Spec.HostNetwork {
		return false
	}

	// 1. check owner reference, if the owner is Job/CronJob, just ignore it
	if !isAllowedOwner(pod) {
		return false
	}

	// 2. the sidecar is already injected, ignore it
	if wasInjeted(pod) {
		return false
	}

	// 3. if the pod has flomesh.io/inject=true annotation, then it's enabled for injection no matter the namespace is enabled or not
	if IsProxyInjectAnnotationEnabled(pod) {
		return true
	}

	// 4. if the namespace of pod has flomesh.io/inject = true label, then the namespace is enabled for injection
	// 5. if the pod has flomesh.io/inject = false annotation, then it's excluded from injection
	if IsNamespaceProxyInjectLabelEnabled(pi.Client, pod.Namespace) && !isProxyInjectDisabled(pod) {
		return true
	}

	return false
}

func (pi *ProxyInjector) hasService(pod *corev1.Pod) (bool, *corev1.Service) {
	// find pod services, a POD must have relevent service to provide enough information for the injector to work properly
	// first by hints annotation
	svc, err := pi.getPodServiceByHints(pod)
	if err == nil && svc != nil {
		return true, svc
	}
	// if not hits, we have to iterate over all services in the namespace of POD to find matched services,
	//      this's time-consuming.
	services, err := pi.getPodServices(pod)
	if err == nil && services != nil && len(services) == 1 {
		// found EXACT one service
		return true, services[0]
	}

	return false, nil
}

func isAllowedOwner(pod *corev1.Pod) bool {
	klog.V(5).Info("Going to check owner reference...")
	//2. check owner reference, if the owner is Job/CronJob, just ignore it
	owner := metav1.GetControllerOf(pod)
	kind := strings.ToLower(owner.Kind)
	klog.V(4).Infof("Onwer Kind = %s", kind)

	switch kind {
	case "deployment", "daemonset", "replicationcontroller", "replicaset", "statefulset", "pod": // do nothing, go ahead
		return true
	case "cronjob", "job":
		return false
	}

	return false
}

func IsProxyInjectAnnotationEnabled(pod *corev1.Pod) bool {
	klog.V(5).Infof("Going to check annotations...")

	return isProxyInjectEnabled(pod.GetAnnotations())
}

func IsNamespaceProxyInjectLabelEnabled(c client.Client, nsName string) bool {
	namespace := &corev1.Namespace{}

	if err := c.Get(
		context.TODO(),
		client.ObjectKey{Name: nsName},
		namespace,
	); err != nil {
		klog.Errorf("Not able to get Namespace %s, error=%#v", nsName, err)
		return false
	}

	return isProxyInjectEnabled(namespace.GetLabels())
}

func IsProxyInjectLabelEnabled(labels map[string]string) bool {
	return isProxyInjectEnabled(labels)
}

func isProxyInjectEnabled(data map[string]string) bool {
	if len(data) == 0 {
		return false
	}

	klog.V(3).Infof("%s=%s", commons.ProxyInjectIndicator, data[commons.ProxyInjectIndicator])
	switch strings.ToLower(data[commons.ProxyInjectIndicator]) {
	case commons.ProxyInjectEnabled:
		return true
	}

	return false
}

func isProxyInjectDisabled(pod *corev1.Pod) bool {
	annotations := pod.GetAnnotations()

	switch strings.ToLower(annotations[commons.ProxyInjectIndicator]) {
	case commons.ProxyInjectDisabled:
		return true
	}

	return false
}

func wasInjeted(pod *corev1.Pod) bool {
	annotations := pod.GetAnnotations()
	if len(annotations) == 0 {
		return false
	}

	status := annotations[commons.ProxyInjectStatusAnnotation]
	klog.V(3).Infof("%s=%s", commons.ProxyInjectStatusAnnotation, status)

	// check annotation flomesh.io/inject-status == 'injected'?
	if strings.ToLower(status) == commons.ProxyInjectdStatus {
		klog.V(3).Infof("The sidecar is already injected, ignore it.")
		return true
	}

	return false
}

func (pi *ProxyInjector) getMatchedProxyProfile(ctx context.Context, pod *corev1.Pod) (*v1alpha1.ProxyProfile, error) {
	proxyProfiles := &v1alpha1.ProxyProfileList{}
	if err := pi.List(ctx, proxyProfiles); err != nil {
		klog.Errorf("Error happened while listing all ProxyProfiles, error=%v", err)
		return nil, err
	}
	// No ProxyProfile exists
	if len(proxyProfiles.Items) == 0 {
		klog.V(5).Infof("No ProxyProfile exists in cluster yet.")
		return nil, nil
	}
	klog.V(3).Infof("There's totally %d ProxyProfile(s) in the cluster.", len(proxyProfiles.Items))

	matchedProxyProfiles := make([]*v1alpha1.ProxyProfile, 0)
	for _, pf := range proxyProfiles.Items {
		klog.V(5).Info("======================== Finding matched ProxyProfile ========================")
		matched, err := isMatched(pod, &pf)
		if err != nil {
			klog.Errorf("Error happened while tesing matched ProxyProfile, error=%v", err)
			return nil, err
		}
		if matched {
			matchedProxyProfiles = append(matchedProxyProfiles, pf.DeepCopy())
		}
		klog.V(5).Info("================================= Finding End =================================")
	}
	numOfMatched := len(matchedProxyProfiles)
	klog.V(3).Infof("Found %d matched ProxyProfiles.", numOfMatched)

	// if ONLY ONE pf matches, go ahead, otherwise return false
	switch numOfMatched {
	case 1:
		return matchedProxyProfiles[0], nil
	default:
		klog.Warningf("Totally %d matched ProxyProfiles were found, injecting with default ProxyProfile.", numOfMatched)

		defaultProxyProfiles := &v1alpha1.ProxyProfileList{}
		if err := pi.Client.List(
			ctx,
			defaultProxyProfiles,
			client.MatchingLabels{
				commons.ProxyDefaultProxyProfileLabel: "true",
			},
		); err != nil {
			return nil, err
		}

		if len(defaultProxyProfiles.Items) != 1 {
			return nil, fmt.Errorf("totally %d default ProxyProfiles were found, there should be ONLY one", len(defaultProxyProfiles.Items))
		}

		return &defaultProxyProfiles.Items[0], nil
	}
}

func isMatched(pod *corev1.Pod, pf *v1alpha1.ProxyProfile) (bool, error) {
	klog.V(4).Infof("Evaluating ProxyProfile %s, Disabled=%t", pf.Name, pf.Spec.Disabled)
	klog.V(4).Infof("pf.Spec.Namespace=%s, pod.Namespace=%s", pf.Spec.Namespace, pod.Namespace)

	// If disabled, it's not matched
	if pf.Spec.Disabled {
		return false, nil
	}

	// If ProxyProfile is enabled, then go through the logic, otherwise just return false
	//If matched Namespace is not empty, ProxyProfile will only match the pods in the namespace
	if pf.Spec.Namespace != "" && pf.Spec.Namespace != pod.Namespace {
		return false, nil
	}

	// if selector is nil, it matches nothing
	if pf.Spec.Selector == nil {
		return false, nil
	}

	// if selector not matched, then continue
	selector, err := metav1.LabelSelectorAsSelector(pf.Spec.Selector)
	if err != nil {
		return false, err
	}
	klog.V(4).Infof("selector = %#v", selector)
	klog.V(4).Infof("Pod Labels = %#v", pod.Labels)

	if !selector.Empty() && selector.Matches(labels.Set(pod.Labels)) {
		klog.V(3).Infof("ProxyProfile %s matches Pod %s/%s? --> %t", pf.Name, pod.Namespace, pod.Name, true)
		return true, nil
	}

	klog.V(3).Infof("ProxyProfile %s matches Pod %s/%s? --> %t", pf.Name, pod.Namespace, pod.Name, false)
	return false, nil
}

// FIXME: we need to derive the parent repo and create the repo for a new registered service!!!!!!

// getPodServices, get all services of this pod, we must know the service as when inject the sidecar
//      it uses the information from service to compose the final repo URL, it leads to implication which
//      pod must be exposed by service so that the injection works.
func (pi *ProxyInjector) getPodServices(pod *corev1.Pod) ([]*corev1.Service, error) {
	allServices, err := pi.K8sAPI.Client.CoreV1().
		Services(pod.Namespace).
		List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	var services []*corev1.Service
	for _, service := range allServices.Items {
		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}
		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(pod.Labels)) {
			klog.V(5).Infof("Found matched service %s/%s", service.Namespace, service.Name)
			services = append(services, service.DeepCopy())
		}
	}

	return services, nil
}

func (pi *ProxyInjector) getPodServiceByHints(pod *corev1.Pod) (*corev1.Service, error) {
	annotations := pod.Annotations

	if annotations == nil {
		return nil, nil
	}

	serviceName := annotations[commons.ProxyServiceNameAnnotation]

	if serviceName == "" {
		return nil, nil
	}

	svc, err := pi.K8sAPI.Client.CoreV1().
		Services(pod.Namespace).
		Get(context.TODO(), serviceName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return svc, nil
}
