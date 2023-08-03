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

package tcproute

import (
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
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1alpha2validation "sigs.k8s.io/gateway-api/apis/v1alpha2/validation"
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
		[]string{"gateway.networking.k8s.io"},
		[]string{"v1alpha2"},
		[]string{"tcproutes"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mtcproute.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.TCPRouteMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vtcproute.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.TCPRouteValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.TCPRouteMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI, r.ConfigStore)),
		commons.TCPRouteValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI)),
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
	return &gwv1alpha2.TCPRoute{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", route.Name)
	klog.V(4).Infof("Before setting default values, spec=%v", route.Spec)

	meshConfig := w.configStore.MeshConfig.GetConfig()

	if meshConfig == nil {
		return
	}

	klog.V(4).Infof("After setting default values, spec=%v", route.Spec)
}

type validator struct {
	k8sAPI *kube.K8sAPI
}

func (w *validator) RuntimeObject() runtime.Object {
	return &gwv1alpha2.TCPRoute{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
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
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		return nil
	}

	errorList := gwv1alpha2validation.ValidateTCPRoute(route)
	errorList = append(errorList, webhooks.ValidateParentRefs(route.Spec.ParentRefs)...)
	if len(errorList) > 0 {
		return util.ErrorListToError(errorList)
	}

	return nil
}
