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
	"errors"
	pfv1alpha1 "github.com/flomesh-io/fsm-classic/apis/proxyprofile/v1alpha1"
	flomeshadmission "github.com/flomesh-io/fsm-classic/pkg/admission"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	"github.com/flomesh-io/fsm-classic/pkg/webhooks"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

type register struct {
	*webhooks.RegisterConfig
}

func NewRegister(cfg *webhooks.RegisterConfig) webhooks.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"proxyprofiles"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mproxyprofile.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.ProxyProfileMutatingWebhookPath,
			r.CaBundle,
			nil,
            nil,
            admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vproxyprofile.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.ProxyProfileValidatingWebhookPath,
			r.CaBundle,
			nil,
            nil,
            admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.ProxyProfileMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI, r.ConfigStore)),
		commons.ProxyProfileValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI)),
	}
}

type defaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func newDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *defaulter {
	return &defaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *defaulter) RuntimeObject() runtime.Object {
	return &pfv1alpha1.ProxyProfile{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	pf, ok := obj.(*pfv1alpha1.ProxyProfile)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", pf.Name)
	klog.V(4).Infof("Before setting default values, spec=%v", pf.Spec)

	mc := w.configStore.MeshConfig.GetConfig()

	if mc == nil {
		return
	}

	if pf.Spec.RestartPolicy == "" {
		pf.Spec.RestartPolicy = pfv1alpha1.ProxyRestartPolicyNever
	}

	if pf.Spec.RestartScope == "" {
		pf.Spec.RestartScope = pfv1alpha1.ProxyRestartScopeOwner
	}

	if pf.Annotations == nil {
		pf.Annotations = make(map[string]string)
	}

	// set default values if it's not set
	for index, sidecar := range pf.Spec.Sidecars {
		if sidecar.Image == "" {
			pf.Spec.Sidecars[index].Image = mc.PipyImage()
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
	pf.Annotations[commons.ProxyProfileLastUpdated] = time.Now().Format(commons.ProxyProfileLastUpdatedTimeFormat)

	switch pf.Spec.ConfigMode {
	case pfv1alpha1.ProxyConfigModeLocal:
		pf.Annotations[commons.ConfigHashAnnotation] = pf.ConfigHash()
	case pfv1alpha1.ProxyConfigModeRemote:
		pf.Annotations[commons.SpecHashAnnotation] = pf.SpecHash()
	}

	klog.V(4).Infof("After setting default values, spec=%v", pf.Spec)
}

type validator struct {
	k8sAPI *kube.K8sAPI
}

func (w *validator) RuntimeObject() runtime.Object {
	return &pfv1alpha1.ProxyProfile{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	oldPf, ok := oldObj.(*pfv1alpha1.ProxyProfile)
	if !ok {
		return nil
	}

	pf, ok := obj.(*pfv1alpha1.ProxyProfile)
	if !ok {
		return nil
	}

	if oldPf.Spec.ConfigMode != pf.Spec.ConfigMode {
		return errors.New("cannot update an immutable field: spec.ConfigMode")
	}

	return doValidation(obj)
}

func (w *validator) ValidateDelete(obj interface{}) error {
	return nil
}

func newValidator(k8sAPI *kube.K8sAPI) *validator {
	return &validator{
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
