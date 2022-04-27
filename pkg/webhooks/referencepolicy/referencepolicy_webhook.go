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

package referencepolicy

import (
	flomeshadmission "github.com/flomesh-io/traffic-guru/pkg/admission"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/klog/v2"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

const (
	kind      = "ReferencePolicy"
	groups    = "gateway.networking.k8s.io"
	resources = "referencepolices"
	versions  = "v1alpha2"

	mwPath = commons.ReferencePolicyMutatingWebhookPath
	mwName = "mreferencepolicy.kb.flomesh.io"
	vwPath = commons.ReferencePolicyValidatingWebhookPath
	vwName = "vreferencepolicy.kb.flomesh.io"
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
		[]admissionregv1.RuleWithOperations{rule},
	)

	validatingWebhook := flomeshadmission.NewValidatingWebhook(
		vwName,
		webhookSvcNs,
		webhookSvcName,
		vwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type ReferencePolicyDefaulter struct {
	k8sAPI *kube.K8sAPI
}

func NewDefaulter(k8sAPI *kube.K8sAPI) *ReferencePolicyDefaulter {
	return &ReferencePolicyDefaulter{
		k8sAPI: k8sAPI,
	}
}

func (w *ReferencePolicyDefaulter) Kind() string {
	return kind
}

func (w *ReferencePolicyDefaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwv1alpha2.ReferencePolicy)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", policy.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", policy.Spec)

	meshConfig := config.GetMeshConfig(w.k8sAPI)

	if meshConfig == nil {
		return
	}

	klog.V(4).Infof("After setting default values, spec=%#v", policy.Spec)
}

type ReferencePolicyValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ReferencePolicyValidator) Kind() string {
	return kind
}

func (w *ReferencePolicyValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *ReferencePolicyValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return doValidation(obj)
}

func (w *ReferencePolicyValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI) *ReferencePolicyValidator {
	return &ReferencePolicyValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	//policy, ok := obj.(*gwv1alpha2.ReferencePolicy)
	//if !ok {
	//    return nil
	//}

	return nil
}
