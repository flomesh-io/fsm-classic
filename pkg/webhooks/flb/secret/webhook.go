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
	flomeshadmission "github.com/flomesh-io/fsm-classic/pkg/admission"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	kind      = "Secret"
	groups    = ""
	resources = "secrets"
	versions  = "v1"

	mwPath = commons.FLBSecretMutatingWebhookPath
	mwName = "mflbsecret.kb.flomesh.io"
	vwPath = commons.FLBSecretValidatingWebhookPath
	vwName = "vflbsecret.kb.flomesh.io"
)

func RegisterWebhooks(webhookSvcNs, webhookSvcName string, caBundle []byte) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{groups},
		[]string{versions},
		[]string{resources},
	)

	mutatingWebhook := flomeshadmission.NewMutatingWebhook(
		mwName,
		webhookSvcNs,
		webhookSvcName,
		mwPath,
		caBundle,
		nil,
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				commons.FlbSecretLabel: "true",
			},
		},
		admissionregv1.Ignore,
		[]admissionregv1.RuleWithOperations{rule},
	)

	validatingWebhook := flomeshadmission.NewValidatingWebhook(
		vwName,
		webhookSvcNs,
		webhookSvcName,
		vwPath,
		caBundle,
		nil,
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				commons.FlbSecretLabel: "true",
			},
		},
		admissionregv1.Ignore,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type SecretDefaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func NewDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *SecretDefaulter {
	return &SecretDefaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *SecretDefaulter) RuntimeObject() runtime.Object {
	return &corev1.Secret{}
}

func (w *SecretDefaulter) SetDefaults(obj interface{}) {
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

type SecretValidator struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func (w *SecretValidator) RuntimeObject() runtime.Object {
	return &corev1.Secret{}
}

func (w *SecretValidator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

func (w *SecretValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return w.doValidation(obj)
}

func (w *SecretValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI, configStore *config.Store) *SecretValidator {
	return &SecretValidator{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *SecretValidator) doValidation(obj interface{}) error {
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
