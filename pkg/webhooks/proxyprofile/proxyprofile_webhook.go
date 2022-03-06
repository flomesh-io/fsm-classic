/*
 * The NEU License
 *
 * Copyright (c) 2021-2022.  flomesh.io
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is furnished to do
 * so, subject to the following conditions:
 *
 * (1)The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * (2)If the software or part of the code will be directly used or used as a
 * component for commercial purposes, including but not limited to: public cloud
 *  services, hosting services, and/or commercial software, the logo as following
 *  shall be displayed in the eye-catching position of the introduction materials
 * of the relevant commercial services or products (such as website, product
 * publicity print), and the logo shall be linked or text marked with the
 * following URL.
 *
 * LOGO : http://flomesh.cn/assets/flomesh-logo.png
 * URL : https://github.com/flomesh-io
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
	"fmt"
	"github.com/flomesh-io/fsm/api/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/util"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-flomesh-io-v1alpha1-proxyprofile,mutating=true,failurePolicy=fail,sideEffects=None,groups=flomesh.io,resources=proxyprofiles,verbs=create;update,versions=v1alpha1,name=mproxyprofile.kb.flomesh.io,admissionReviewVersions=v1

type ProxProfileDefaulter struct {
	client client.Client
	k8sAPI *kube.K8sAPI
}

func NewProxyProfileDefaulter(client client.Client, k8sAPI *kube.K8sAPI) *ProxProfileDefaulter {
	return &ProxProfileDefaulter{
		client: client,
		k8sAPI: k8sAPI,
	}
}

func DefaultingWebhookFor(defaulter *ProxProfileDefaulter) *admission.Webhook {
	return &admission.Webhook{
		Handler: &mutatingHandler{defaulter: defaulter},
	}
}

func (w *ProxProfileDefaulter) SetDefaults(pf *v1alpha1.ProxyProfile) {
	klog.V(5).Infof("Default Webhook, name=%s", pf.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", pf.Spec)

	//configHash, err := pf.ConfigHash()
	//if err != nil {
	//	klog.Errorf("Not able convert ProxyProfile Config to bytes, ProxyProfile: %s, error: %#v", pf.Name, err)
	//	panic(err)
	//}

	operatorConfig := config.GetOperatorConfig(w.k8sAPI)

	if operatorConfig == nil {
		return
	}

	if pf.Spec.ConfigMode == v1alpha1.ProxyConfigModeRemote {
		if pf.Spec.RepoBaseUrl == "" {
			// get from opertor-config
			pf.Spec.RepoBaseUrl = operatorConfig.RepoBaseURL()
		}

		if pf.Spec.ParentCodebasePath == "" {
			pf.Spec.ParentCodebasePath = commons.DefaultProxyParentCodebasePathTpl
		}
	}

	if pf.Spec.RestartPolicy == "" {
		pf.Spec.RestartPolicy = v1alpha1.ProxyRestartPolicyNever
	}

	if pf.Spec.RestartScope == "" {
		pf.Spec.RestartScope = v1alpha1.ProxyRestartScopePod
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
		case v1alpha1.ProxyConfigModeLocal:
			if sidecar.StartupScriptName == "" {
				pf.Spec.Sidecars[index].StartupScriptName = sidecar.Name + ".js"
			}
		case v1alpha1.ProxyConfigModeRemote:
			if sidecar.CodebasePath == "" {
				// /[region]/[zone]/[group]/[cluster]/sidecars/[namespace]/[service-name]
				// /default/default/default/cluster1/sidecars/test/service1
				pf.Spec.Sidecars[index].CodebasePath = commons.DefaultProxyCodebasePathTpl
			}
		}
	}

	// calculate the hash, this must be the last step as the spec may change due to set default values
	if pf.Annotations == nil {
		pf.Annotations = make(map[string]string)
	}
	switch pf.Spec.ConfigMode {
	case v1alpha1.ProxyConfigModeLocal:
		pf.Annotations[commons.ConfigHashAnnotation] = pf.ConfigHash()
	case v1alpha1.ProxyConfigModeRemote:
		pf.Annotations[commons.SpecHashAnnotation] = pf.SpecHash()
	}

	klog.V(4).Infof("After setting default values, spec=%#v", pf.Spec)
}

// change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-flomesh-io-v1alpha1-proxyprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=flomesh.io,resources=proxyprofiles,verbs=create;update,versions=v1alpha1,name=vproxyprofile.kb.flomesh.io,admissionReviewVersions=v1

type ProxProfileValidator struct {
	client client.Client
	k8sAPI *kube.K8sAPI
}

func NewProxyProfileValidator(client client.Client, k8sAPI *kube.K8sAPI) *ProxProfileValidator {
	return &ProxProfileValidator{
		client: client,
		k8sAPI: k8sAPI,
	}
}

func ValidatingWebhookFor(validator *ProxProfileValidator) *admission.Webhook {
	return &admission.Webhook{
		Handler: &validatingHandler{validator: validator},
	}
}

func (w *ProxProfileValidator) ValidateCreate(pf *v1alpha1.ProxyProfile) error {
	return doValidation(pf)
}

func (w *ProxProfileValidator) ValidateUpdate(pf *v1alpha1.ProxyProfile) error {

	return doValidation(pf)
}

func doValidation(pf *v1alpha1.ProxyProfile) error {
	if len(pf.Spec.Config) == 0 && pf.Spec.RepoBaseUrl == "" {
		return fmt.Errorf("either Config or RepoBaseUrl must be provided")
	}

	if len(pf.Spec.Config) > 0 && pf.Spec.RepoBaseUrl != "" {
		return fmt.Errorf("mutually exclusive options found, only either Config or RepoBaseUrl is permitted")
	}
	return nil
}
