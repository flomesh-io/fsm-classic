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

package cm

import (
	"fmt"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/webhooks"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"net/http"
	"strings"
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
		[]string{"configmaps"},
	)

	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			commons.FlomeshControlPlaneLabel: "true",
		},
	}

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mconfigmap.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.ConfigMapMutatingWebhookPath,
			r.CaBundle,
			nsSelector,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vconfigmap.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			commons.ConfigMapValidatingWebhookPath,
			r.CaBundle,
			nsSelector,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		commons.ConfigMapMutatingWebhookPath:   webhooks.DefaultingWebhookFor(newDefaulter(r.K8sAPI)),
		commons.ConfigMapValidatingWebhookPath: webhooks.ValidatingWebhookFor(newValidator(r.K8sAPI)),
	}
}

type defaulter struct {
	k8sAPI *kube.K8sAPI
}

func isNotWatchedConfigmap(cm *corev1.ConfigMap, fsmNamespace string) bool {
	klog.V(5).Infof("Configmap namespace = %q, name = %q.", cm.Namespace, cm.Name)
	return cm.Namespace != fsmNamespace || !config.DefaultWatchedConfigMaps.Has(cm.Name)
}

func newDefaulter(k8sAPI *kube.K8sAPI) *defaulter {
	return &defaulter{
		k8sAPI: k8sAPI,
	}
}

func (w *defaulter) RuntimeObject() runtime.Object {
	return &corev1.ConfigMap{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return
	}

	if isNotWatchedConfigmap(cm, config.GetFsmNamespace()) {
		return
	}

	switch cm.Name {
	case commons.MeshConfigName:
		cfg, err := config.ParseMeshConfig(cm)
		if err != nil {
			return
		}

		if cfg.Images.Repository == "" {
			cfg.Images.Repository = "flomesh"
		}

		if cfg.Images.PipyImage == "" {
			cfg.Images.PipyImage = "pipy:latest"
		}

		if cfg.Images.ProxyInitImage == "" {
			cfg.Images.ProxyInitImage = "fsm-proxy-init:latest"
		}

		if cfg.Images.KlipperLbImage == "" {
			cfg.Images.KlipperLbImage = "mirrored-klipper-lb:v0.3.5"
		}

		if strings.HasSuffix(cfg.Repo.RootURL, "/") {
			cfg.Repo.RootURL = strings.TrimSuffix(cfg.Repo.RootURL, "/")
		}

		if cfg.Certificate.Manager == "" {
			cfg.Certificate.Manager = string(certificate.Archon)
		}

		if cfg.Webhook.ServiceName == "" {
			cfg.Webhook.ServiceName = commons.DefaultWebhookServiceName
		}

		cm.Data[commons.MeshConfigJsonName] = cfg.ToJson()
	default:
		// ignore
	}
}

type validator struct {
	k8sAPI *kube.K8sAPI
}

func (w *validator) RuntimeObject() runtime.Object {
	return &corev1.ConfigMap{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	return w.doValidation(obj)
}

func (w *validator) ValidateDelete(obj interface{}) error {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	if isNotWatchedConfigmap(cm, config.GetFsmNamespace()) {
		return nil
	}

	switch cm.Name {
	case commons.MeshConfigName:
		// protect the MeshConfig from deletion
		return fmt.Errorf("ConfigMap %s/%s cannot be deleted", cm.Namespace, cm.Name)
	default:
		// ignore
	}

	return nil
}

func newValidator(k8sAPI *kube.K8sAPI) *validator {
	return &validator{
		k8sAPI: k8sAPI,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	if isNotWatchedConfigmap(cm, config.GetFsmNamespace()) {
		return nil
	}

	switch cm.Name {
	case commons.MeshConfigName:
		// validate the config
	default:
		// ignore
	}

	return nil
}
