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
	"fmt"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/webhooks"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
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
		[]string{"secrets"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mflbsecret.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.FLBSecretMutatingWebhookPath,
			r.CaBundle,
			nil,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vflbsecret.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.FLBSecretValidatingWebhookPath,
			r.CaBundle,
			nil,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.FLBSecretMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI, r.ConfigStore)),
		commons.FLBSecretValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI, r.ConfigStore)),
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
	return &corev1.Secret{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}

	mc := w.configStore.MeshConfig.GetConfig()
	if secret.Name != mc.FLB.SecretName {
		return
	}

	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	if len(secret.Data[commons.FLBSecretKeyDefaultAlgo]) == 0 {
		secret.Data[commons.FLBSecretKeyDefaultAlgo] = []byte("rr")
	}
}

type validator struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func (w *validator) RuntimeObject() runtime.Object {
	return &corev1.Secret{}
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
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	mc := w.configStore.MeshConfig.GetConfig()
	if secret.Name != mc.FLB.SecretName {
		return nil
	}

	if mc.FLB.StrictMode {
		for _, key := range []string{
			commons.FLBSecretKeyBaseUrl,
			commons.FLBSecretKeyUsername,
			commons.FLBSecretKeyPassword,
			commons.FLBSecretKeyDefaultCluster,
			commons.FLBSecretKeyDefaultAddressPool,
			commons.FLBSecretKeyDefaultAlgo,
		} {
			value, ok := secret.Data[key]
			if !ok {
				return fmt.Errorf("%q is required", key)
			}

			if len(value) == 0 {
				return fmt.Errorf("%q has an empty value", key)
			}
		}
	}

	return nil
}