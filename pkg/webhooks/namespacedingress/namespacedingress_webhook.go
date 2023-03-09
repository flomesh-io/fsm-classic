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

package namespacedingress

import (
	"context"
	nsigv1alpha1 "github.com/flomesh-io/fsm/apis/namespacedingress/v1alpha1"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/webhooks"
	"github.com/pkg/errors"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
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
		[]string{"flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"namespacedingresses"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mnamespacedingress.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.NamespacedIngressMutatingWebhookPath,
			r.CaBundle,
			nil,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vnamespacedingress.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.NamespacedIngressValidatingWebhookPath,
			r.CaBundle,
			nil,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.NamespacedIngressMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI, r.ConfigStore)),
		commons.NamespacedIngressValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI)),
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
	return &nsigv1alpha1.NamespacedIngress{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", c.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", c.Spec)

	meshConfig := w.configStore.MeshConfig.GetConfig()

	if meshConfig == nil {
		return
	}

	if c.Spec.ServiceAccountName == "" {
		c.Spec.ServiceAccountName = "fsm-namespaced-ingress"
	}

	if c.Spec.LogLevel == nil {
		c.Spec.LogLevel = pointer.Int(2)
	}

	if c.Spec.Replicas == nil {
		c.Spec.Replicas = pointer.Int32(1)
	}

	if c.Spec.TLS.SSLPassthrough.UpstreamPort == nil {
		c.Spec.TLS.SSLPassthrough.UpstreamPort = pointer.Int32(443)
	}

	if c.Spec.PodSecurityContext == nil {
		c.Spec.PodSecurityContext = &corev1.PodSecurityContext{
			RunAsNonRoot: pointer.Bool(true),
			RunAsUser:    pointer.Int64(65532),
			RunAsGroup:   pointer.Int64(65532),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	}

	if c.Spec.SecurityContext == nil {
		c.Spec.SecurityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: pointer.Bool(false),
		}
	}

	klog.V(4).Infof("After setting default values, spec=%#v", c.Spec)
}

type validator struct {
	k8sAPI *kube.K8sAPI
}

func (w *validator) RuntimeObject() runtime.Object {
	return &nsigv1alpha1.NamespacedIngress{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	namespacedingress, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return nil
	}

	list, err := w.k8sAPI.FlomeshClient.
		NamespacedingressV1alpha1().
		NamespacedIngresses(namespacedingress.Namespace).
		List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return err
	}

	// There's already an NamespacedIngress in this namespace, return error
	if len(list.Items) > 0 {
		return errors.Errorf(
			"There's already %d IngressDeploymnent(s) in namespace %q. Each namespace can have ONLY ONE NamespacedIngress.",
			len(list.Items),
			namespacedingress.Namespace,
		)
	}

	return doValidation(namespacedingress)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	//oldNamespacedIngress, ok := oldObj.(*nsigv1alpha1.NamespacedIngress)
	//if !ok {
	//	return nil
	//}
	//
	//namespacedingress, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	//if !ok {
	//	return nil
	//}
	//
	//if oldNamespacedIngress.Namespace != namespacedingress.Namespace {
	//    return errors.Errorf("")
	//}

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
	return nil
}
