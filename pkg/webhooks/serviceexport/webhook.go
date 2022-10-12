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

package serviceexport

import (
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
)

const (
	kind      = "ServiceExport"
	groups    = "gateway.networking.k8s.io"
	resources = "serviceexports"
	versions  = "v1alpha2"

	mwPath = commons.ServiceExportMutatingWebhookPath
	mwName = "mserviceexport.kb.flomesh.io"
	vwPath = commons.ServiceExportValidatingWebhookPath
	vwName = "vserviceexport.kb.flomesh.io"
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

type ServiceExportDefaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func NewDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *ServiceExportDefaulter {
	return &ServiceExportDefaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *ServiceExportDefaulter) Kind() string {
	return kind
}

func (w *ServiceExportDefaulter) SetDefaults(obj interface{}) {
	//serviceExport, ok := obj.(*svcexpv1alpha1.ServiceExport)
	//if !ok {
	//	return
	//}
	//
	//klog.V(5).Infof("Default Webhook, name=%s", serviceExport.Name)
	//klog.V(4).Infof("Before setting default values, spec=%#v", serviceExport.Spec)
	//
	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}
	//
	//klog.V(4).Infof("After setting default values, spec=%#v", serviceExport.Spec)
}

type ServiceExportValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ServiceExportValidator) Kind() string {
	return kind
}

func (w *ServiceExportValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *ServiceExportValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return doValidation(obj)
}

func (w *ServiceExportValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI) *ServiceExportValidator {
	return &ServiceExportValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	//serviceExport, ok := obj.(*svcexpv1alpha1.ServiceExport)
	//if !ok {
	//    return nil
	//}

	return nil
}