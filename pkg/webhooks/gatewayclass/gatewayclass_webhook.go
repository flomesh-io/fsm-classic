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
	flomeshadmission "github.com/flomesh-io/fsm-classic/pkg/admission"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gwv1beta1validation "sigs.k8s.io/gateway-api/apis/v1beta1/validation"
)

const (
	kind      = "GatewayClass"
	groups    = "gateway.networking.k8s.io"
	resources = "gatewayclasses"
	versions  = "v1beta1"

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

type GatewayClassDefaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func NewDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *GatewayClassDefaulter {
	return &GatewayClassDefaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *GatewayClassDefaulter) RuntimeObject() runtime.Object {
	return &gwv1beta1.GatewayClass{}
}

func (w *GatewayClassDefaulter) SetDefaults(obj interface{}) {
	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", gatewayClass.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", gatewayClass.Spec)

	meshConfig := w.configStore.MeshConfig.GetConfig()

	if meshConfig == nil {
		return
	}

	klog.V(4).Infof("After setting default values, spec=%#v", gatewayClass.Spec)
}

type GatewayClassValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *GatewayClassValidator) RuntimeObject() runtime.Object {
	return &gwv1beta1.GatewayClass{}
}

func (w *GatewayClassValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *GatewayClassValidator) ValidateUpdate(oldObj, obj interface{}) error {
	oldGatewayClass, ok := oldObj.(*gwv1beta1.GatewayClass)
	if !ok {
		return nil
	}

	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		return nil
	}

	errorList := gwv1beta1validation.ValidateGatewayClassUpdate(oldGatewayClass, gatewayClass)
	if len(errorList) > 0 {
		return util.ErrorListToError(errorList)
	}

	return nil
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
	//gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	//if !ok {
	//    return nil
	//}

	return nil
}
