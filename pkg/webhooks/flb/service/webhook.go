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
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	kind      = "Service"
	groups    = ""
	resources = "services"
	versions  = "v1"

	mwPath = commons.FLBServiceMutatingWebhookPath
	mwName = "mflbservice.kb.flomesh.io"
	vwPath = commons.FLBServiceValidatingWebhookPath
	vwName = "vflbservice.kb.flomesh.io"
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
		nil,
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
		nil,
		admissionregv1.Ignore,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type ServiceDefaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func NewDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *ServiceDefaulter {
	return &ServiceDefaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *ServiceDefaulter) RuntimeObject() runtime.Object {
	return &corev1.Service{}
}

func (w *ServiceDefaulter) SetDefaults(obj interface{}) {
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

type ServiceValidator struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func (w *ServiceValidator) RuntimeObject() runtime.Object {
	return &corev1.Service{}
}

func (w *ServiceValidator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

func (w *ServiceValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return w.doValidation(obj)
}

func (w *ServiceValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI, configStore *config.Store) *ServiceValidator {
	return &ServiceValidator{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *ServiceValidator) doValidation(obj interface{}) error {
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
