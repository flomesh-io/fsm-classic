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

package gatewayclass

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
	kind      = "GatewayClass"
	groups    = "gateway.networking.k8s.io"
	resources = "gatewayclasses"
	versions  = "v1alpha2"

	mwPath = commons.GatewayClassMutatingWebhookPath
	mwName = "mgatewayclass.kb.flomesh.io"
	vwPath = commons.GatewayClassValidatingWebhookPath
	vwName = "vgatewayclass.kb.flomesh.io"
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

type GatewayClassDefaulter struct {
	k8sAPI *kube.K8sAPI
}

func NewDefaulter(k8sAPI *kube.K8sAPI) *GatewayClassDefaulter {
	return &GatewayClassDefaulter{
		k8sAPI: k8sAPI,
	}
}

func (w *GatewayClassDefaulter) Kind() string {
	return kind
}

func (w *GatewayClassDefaulter) SetDefaults(obj interface{}) {
	gatewayClass, ok := obj.(*gwv1alpha2.GatewayClass)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", gatewayClass.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", gatewayClass.Spec)

	meshConfig := config.GetMeshConfig(w.k8sAPI)

	if meshConfig == nil {
		return
	}

	klog.V(4).Infof("After setting default values, spec=%#v", gatewayClass.Spec)
}

type GatewayClassValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *GatewayClassValidator) Kind() string {
	return kind
}

func (w *GatewayClassValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *GatewayClassValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return doValidation(obj)
}

func (w *GatewayClassValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI) *GatewayClassValidator {
	return &GatewayClassValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	//gatewayClass, ok := obj.(*gwv1alpha2.GatewayClass)
	//if !ok {
	//    return nil
	//}

	return nil
}
