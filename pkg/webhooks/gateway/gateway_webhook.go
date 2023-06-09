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

package gateway

import (
	"context"
	"fmt"
	flomeshadmission "github.com/flomesh-io/fsm-classic/pkg/admission"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	"github.com/flomesh-io/fsm-classic/pkg/webhooks"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"net/http"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gwv1beta1validation "sigs.k8s.io/gateway-api/apis/v1beta1/validation"
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
		[]string{"v1beta1"},
		[]string{"gateways"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mgateway.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.GatewayMutatingWebhookPath,
			r.CaBundle,
			nil,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vgateway.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.GatewayValidatingWebhookPath,
			r.CaBundle,
			nil,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.GatewayMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI, r.ConfigStore)),
		commons.GatewayValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI)),
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
	return &gwv1beta1.Gateway{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	gateway, ok := obj.(*gwv1beta1.Gateway)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", gateway.Name)
	klog.V(4).Infof("Before setting default values, spec=%v", gateway.Spec)

	meshConfig := w.configStore.MeshConfig.GetConfig()

	if meshConfig == nil {
		return
	}

	klog.V(4).Infof("After setting default values, spec=%v", gateway.Spec)
}

type validator struct {
	k8sAPI *kube.K8sAPI
}

func (w *validator) RuntimeObject() runtime.Object {
	return &gwv1beta1.Gateway{}
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

func newValidator(k8sAPI *kube.K8sAPI) *validator {
	return &validator{
		k8sAPI: k8sAPI,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	gateway, ok := obj.(*gwv1beta1.Gateway)
	if !ok {
		return nil
	}

	errorList := gwv1beta1validation.ValidateGateway(gateway)
	errorList = append(errorList, w.validateListenerHostname(gateway)...)
	errorList = append(errorList, w.validateCertificateSecret(gateway)...)
	if len(errorList) > 0 {
		return util.ErrorListToError(errorList)
	}

	return nil
}

func (w *validator) validateCertificateSecret(gateway *gwv1beta1.Gateway) field.ErrorList {
	var errs field.ErrorList

	for i, c := range gateway.Spec.Listeners {
		switch c.Protocol {
		case gwv1beta1.HTTPSProtocolType:
			if c.TLS != nil && c.TLS.Mode != nil {
				switch *c.TLS.Mode {
				case gwv1beta1.TLSModeTerminate:
					errs = append(errs, w.validateSecretsExistence(gateway, c, i)...)
				case gwv1beta1.TLSModePassthrough:
					path := field.NewPath("spec").
						Child("listeners").Index(i).
						Child("tls").
						Child("mode")
					errs = append(errs, field.Forbidden(path, fmt.Sprintf("TLSModeType %s is not supported when Protocol is %s, please use Protocol %s", gwv1beta1.TLSModePassthrough, gwv1beta1.HTTPSProtocolType, gwv1beta1.TLSProtocolType)))
				}
			}
		case gwv1beta1.TLSProtocolType:
			if c.TLS != nil && c.TLS.Mode != nil {
				switch *c.TLS.Mode {
				case gwv1beta1.TLSModeTerminate:
					errs = append(errs, w.validateSecretsExistence(gateway, c, i)...)
				case gwv1beta1.TLSModePassthrough:
					if len(c.TLS.CertificateRefs) > 0 {
						path := field.NewPath("spec").
							Child("listeners").Index(i).
							Child("tls").
							Child("certificateRefs")
						errs = append(errs, field.Forbidden(path, fmt.Sprintf("No need to provide certificates when Protocol is %s and TLSModeType is %s", gwv1beta1.TLSProtocolType, gwv1beta1.TLSModePassthrough)))
					}
				}
			}
		}
	}

	return errs
}

func (w *validator) validateSecretsExistence(gateway *gwv1beta1.Gateway, c gwv1beta1.Listener, i int) field.ErrorList {
	var errs field.ErrorList

	for j, ref := range c.TLS.CertificateRefs {
		if string(*ref.Kind) == "Secret" && string(*ref.Group) == "" {
			ns := ""
			if ref.Namespace == nil {
				ns = gateway.Namespace
			} else {
				ns = string(*ref.Namespace)
			}
			name := string(ref.Name)

			path := field.NewPath("spec").
				Child("listeners").Index(i).
				Child("tls").
				Child("certificateRefs").Index(j)
			secret, err := w.k8sAPI.Client.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				errs = append(errs, field.NotFound(path, fmt.Sprintf("Failed to get Secret %s/%s: %s", ns, name, err)))
				continue
			}

			v, ok := secret.Data[corev1.TLSCertKey]
			if ok {
				if string(v) == "" {
					errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s is empty", ns, name, corev1.TLSCertKey)))
				}
			} else {
				errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s doesn't have required data by key %s", ns, name, corev1.TLSCertKey)))
			}

			v, ok = secret.Data[corev1.TLSPrivateKeyKey]
			if ok {
				if string(v) == "" {
					errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s is empty", ns, name, corev1.TLSPrivateKeyKey)))
				}
			} else {
				errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s doesn't have required data by key %s", ns, name, corev1.TLSPrivateKeyKey)))
			}
		}
	}

	return errs
}

func (w *validator) validateListenerHostname(gateway *gwv1beta1.Gateway) field.ErrorList {
	var errs field.ErrorList

	for i, listener := range gateway.Spec.Listeners {
		if listener.Hostname != nil {
			hostname := string(*listener.Hostname)
			if err := webhooks.IsValidHostname(hostname); err != nil {
				path := field.NewPath("spec").
					Child("listeners").Index(i).
					Child("hostname")

				errs = append(errs, field.Invalid(path, hostname, fmt.Sprintf("%s", err)))
			}
		}
	}

	return errs
}
