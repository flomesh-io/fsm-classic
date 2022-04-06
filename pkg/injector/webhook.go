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
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/proxy-injector-flomesh-io-v1alpha1,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=injector.kb.flomesh.io,admissionReviewVersions=v1

type ProxyInjector struct {
	client.Client
	decoder        *admission.Decoder
	Recorder       record.EventRecorder
	ProxyImage     string
	ProxyInitImage string
	ConfigStore    *config.Store
	K8sAPI         *kube.K8sAPI
}

func (pi *ProxyInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	if err := pi.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// ensure the progragation
	// if pod.namespace is empty, using req.namespace
	if pod.Namespace == "" {
		pod.Namespace = req.Namespace
	}

	klog.V(3).Infof("AdmissionRequest for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// determine whether to perform mutation
	if pi.isInjectionRequired(pod) {
		klog.V(3).Infof("Mutation policy for %s/%s: required:%t", pod.Namespace, pod.Name, true)

		// list ProxyProfiles, to see if there's any pf matches current pod
		proxyProfile, matchErr := pi.getMatchedProxyProfile(ctx, pod)
		if matchErr != nil {
			return admission.Errored(http.StatusInternalServerError, matchErr)
		}
		// No matched ProxyProfile
		if proxyProfile == nil {
			return admission.Allowed("No matched ProxyProfile.")
		}
		klog.V(3).Infof("Found matched ProxyProfile: %s", proxyProfile.Name)

		if err := pi.mutatingPod(pod, proxyProfile); err != nil {
			//pi.Recorder.Eventf(proxyProfile, corev1.EventTypeWarning, "Failed",
			//	"Failed to mutate Pod, %#v ", err)
			return admission.Errored(http.StatusInternalServerError, err)
		}

		marshalled, err := json.Marshal(pod)
		if err != nil {
			//pi.Recorder.Eventf(proxyProfile, corev1.EventTypeWarning, "Failed",
			//	"Failed to marshal Pod, %#v ", err)
			return admission.Errored(http.StatusInternalServerError, err)
		}
		return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshalled)
	}

	info := fmt.Sprintf("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
	klog.V(3).Info(info)

	return admission.Allowed(info)
}

func (pi *ProxyInjector) InjectDecoder(d *admission.Decoder) error {
	pi.decoder = d
	return nil
}
