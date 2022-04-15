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

package proxyprofile

import (
	pfv1alpha1 "github.com/flomesh-io/traffic-guru/apis/proxyprofile/v1alpha1"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	"github.com/flomesh-io/traffic-guru/pkg/util"
	"github.com/flomesh-io/traffic-guru/pkg/webhooks"
	"k8s.io/klog/v2"
	"time"
)

//+kubebuilder:webhook:path=/mutate-flomesh-io-v1alpha1-proxyprofile,mutating=true,failurePolicy=fail,sideEffects=None,groups=flomesh.io,resources=proxyprofiles,verbs=create;update,versions=v1alpha1,name=mproxyprofile.kb.flomesh.io,admissionReviewVersions=v1

const Kind = "ProxyProfile"

type ProxyProfileDefaulter struct {
	k8sAPI *kube.K8sAPI
}

var _ webhooks.Defaulter = &ProxyProfileDefaulter{}

func NewProxyProfileDefaulter(k8sAPI *kube.K8sAPI) *ProxyProfileDefaulter {
	return &ProxyProfileDefaulter{
		k8sAPI: k8sAPI,
	}
}

//
//func DefaultingWebhookFor(defaulter *ProxyProfileDefaulter) *admission.Webhook {
//	return &admission.Webhook{
//		Handler: &mutatingHandler{defaulter: defaulter},
//	}
//}

func (w *ProxyProfileDefaulter) Kind() string {
	return Kind
}

func (w *ProxyProfileDefaulter) SetDefaults(obj interface{}) {
	pf, ok := obj.(*pfv1alpha1.ProxyProfile)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", pf.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", pf.Spec)

	meshConfig := config.GetMeshConfig(w.k8sAPI)

	if meshConfig == nil {
		return
	}

	if pf.Spec.RestartPolicy == "" {
		pf.Spec.RestartPolicy = pfv1alpha1.ProxyRestartPolicyNever
	}

	if pf.Spec.RestartScope == "" {
		pf.Spec.RestartScope = pfv1alpha1.ProxyRestartScopePod
	}

	// set default values if it's not set
	for index, sidecar := range pf.Spec.Sidecars {
		if sidecar.Image == "" {
			pf.Spec.Sidecars[index].Image = commons.DefaultProxyImage
		}

		if sidecar.ImagePullPolicy == "" {
			pf.Spec.Sidecars[index].ImagePullPolicy = util.ImagePullPolicyByTag(pf.Spec.Sidecars[index].Image)
		}

		switch pf.Spec.ConfigMode {
		case pfv1alpha1.ProxyConfigModeLocal:
			if sidecar.StartupScriptName == "" {
				pf.Spec.Sidecars[index].StartupScriptName = sidecar.Name + ".js"
			}
		case pfv1alpha1.ProxyConfigModeRemote:
			// do nothing
		}
	}

	// calculate the hash, this must be the last step as the spec may change due to set default values
	if pf.Annotations == nil {
		pf.Annotations = make(map[string]string)
	}
	pf.Annotations[commons.ProxyProfileLastUpdatedAnnotation] = time.Now().String()

	switch pf.Spec.ConfigMode {
	case pfv1alpha1.ProxyConfigModeLocal:
		pf.Annotations[commons.ConfigHashAnnotation] = pf.ConfigHash()
	case pfv1alpha1.ProxyConfigModeRemote:
		pf.Annotations[commons.SpecHashAnnotation] = pf.SpecHash()
	}

	klog.V(4).Infof("After setting default values, spec=%#v", pf.Spec)
}

// change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-flomesh-io-v1alpha1-proxyprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=flomesh.io,resources=proxyprofiles,verbs=create;update,versions=v1alpha1,name=vproxyprofile.kb.flomesh.io,admissionReviewVersions=v1

type ProxyProfileValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ProxyProfileValidator) Kind() string {
	return Kind
}

func (w *ProxyProfileValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *ProxyProfileValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return doValidation(obj)
}

func (w *ProxyProfileValidator) ValidateDelete(obj interface{}) error {
	return nil
}

var _ webhooks.Validator = &ProxyProfileValidator{}

func NewProxyProfileValidator(k8sAPI *kube.K8sAPI) *ProxyProfileValidator {
	return &ProxyProfileValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	//pf, ok := obj.(*pfv1alpha1.ProxyProfile)
	//if !ok {
	//    return nil
	//}

	return nil
}
