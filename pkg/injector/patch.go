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

package injector

import (
	pfv1alpha1 "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

func (pi *ProxyInjector) mutatingPod(pod *corev1.Pod, pf *pfv1alpha1.ProxyProfile) error {
	// load config template and apply ralated values
	sidecarTemplate, err := pi.toSidecarTemplate(pod, pf)
	if err != nil {
		// treat not found as normal, no injection will be done
		if errors.IsNotFound(err) {
			return nil
		}

		return err
	}
	klog.V(3).Infof("Loaded sidecarTemplate: %v", sidecarTemplate)
	//pi.Recorder.Eventf(pf, corev1.EventTypeNormal, "Found",
	//	"Found Sidecar Template in Namespace %s.", pod.Namespace)

	setInitContainers(pod, sidecarTemplate)
	setContainers(pod, sidecarTemplate)
	setVolumes(pod, sidecarTemplate)
	setAnnotations(pod, pf)
	setLabels(pod, pf)

	//pi.Recorder.Eventf(pf, corev1.EventTypeNormal, "Mutated",
	//	"Successfully Mutated in Namespace %s.", pod.Namespace)

	return nil
}

func setInitContainers(pod *corev1.Pod, template *SidecarTemplate) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, template.InitContainers...)
}

func setContainers(pod *corev1.Pod, template *SidecarTemplate) {
	// sidecars are before service containers
	containers := make([]corev1.Container, 0)
	// append sidecars first
	containers = append(containers, template.Containers...)
	// append common env & service env to each service container
	for index := range pod.Spec.Containers {
		pod.Spec.Containers[index].Env = append(pod.Spec.Containers[index].Env, defaultEnv...)
		pod.Spec.Containers[index].Env = append(pod.Spec.Containers[index].Env, template.ServiceEnv...)
	}
	// append service containers
	pod.Spec.Containers = append(containers, pod.Spec.Containers...)
}

func setVolumes(pod *corev1.Pod, template *SidecarTemplate) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, template.Volumes...)
}

func setAnnotations(pod *corev1.Pod, pf *pfv1alpha1.ProxyProfile) {
	if len(pod.Annotations) == 0 {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations[commons.ProxyInjectStatusAnnotation] = commons.ProxyInjectdStatus
}

func setLabels(pod *corev1.Pod, pf *pfv1alpha1.ProxyProfile) {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	lastUpdate := pf.Annotations[commons.ProxyProfileLastUpdated]
	if lastUpdate != "" {
		pod.Labels[commons.ProxyProfileLastUpdated] = lastUpdate
	}

	pod.Labels[commons.MatchedProxyProfile] = pf.Name
}
