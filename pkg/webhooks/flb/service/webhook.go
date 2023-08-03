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

package secret

import (
	"context"
	flomeshadmission "github.com/flomesh-io/fsm-classic/pkg/admission"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/flb"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/webhooks"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
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
		[]string{""},
		[]string{"v1"},
		[]string{"services"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mflbservice.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.FLBServiceMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vflbservice.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.FLBServiceValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.FLBServiceMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI, r.ConfigStore)),
		commons.FLBServiceValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI, r.ConfigStore)),
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
	return &corev1.Service{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	//service, ok := obj.(*corev1.Service)
	//if !ok {
	//	return
	//}
	//
	//mc := w.configStore.MeshConfig.GetConfig()
	//if mc.FLB.StrictMode {
	//
	//}
}

type validator struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func (w *validator) RuntimeObject() runtime.Object {
	return &corev1.Service{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	return w.doValidation(obj)
}

func (w *validator) ValidateDelete(obj interface{}) error {
	return nil
}

func newValidator(k8sAPI *kube.K8sAPI, configStore *config.Store) *validator {
	return &validator{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}

	if !flb.IsFlbEnabled(service, w.k8sAPI) {
		return nil
	}

	mc := w.configStore.MeshConfig.GetConfig()
	if mc.FLB.StrictMode {
		if _, err := w.k8sAPI.Client.CoreV1().
			Secrets(service.Namespace).
			Get(context.TODO(), mc.FLB.SecretName, metav1.GetOptions{}); err != nil {
			return err
		}
	}

	return nil
}
