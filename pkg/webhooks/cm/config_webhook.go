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
	flomeshadmission "github.com/flomesh-io/traffic-guru/pkg/admission"
	"github.com/flomesh-io/traffic-guru/pkg/certificate"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"strings"
)

const (
	kind      = "ConfigMap"
	groups    = ""
	resources = "configmaps"
	versions  = "v1"

	mwPath = commons.ConfigMapMutatingWebhookPath
	mwName = "mconfigmap.kb.flomesh.io"
	vwPath = commons.ConfigMapValidatingWebhookPath
	vwName = "vconfigmap.kb.flomesh.io"
)

func RegisterWebhooks(webhookSvcNs, webhookSvcName string, caBundle []byte) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{groups},
		[]string{versions},
		[]string{resources},
	)

	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			commons.FlomeshControlPlaneLabel: "true",
		},
	}

	mutatingWebhook := flomeshadmission.NewMutatingWebhook(
		mwName,
		webhookSvcNs,
		webhookSvcName,
		mwPath,
		caBundle,
		nsSelector,
		[]admissionregv1.RuleWithOperations{rule},
	)

	validatingWebhook := flomeshadmission.NewValidatingWebhook(
		vwName,
		webhookSvcNs,
		webhookSvcName,
		vwPath,
		caBundle,
		nsSelector,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type ConfigMapDefaulter struct {
	k8sAPI *kube.K8sAPI
}

func isNotWatchedConfigmap(cm *corev1.ConfigMap) bool {
	klog.V(5).Infof("Configmap namespace = %q, name = %q.", cm.Namespace, cm.Name)
	return cm.Namespace != commons.DefaultFlomeshNamespace || !commons.DefaultWatchedConfigMaps.Has(cm.Name)
}

func NewDefaulter(k8sAPI *kube.K8sAPI) *ConfigMapDefaulter {
	return &ConfigMapDefaulter{
		k8sAPI: k8sAPI,
	}
}

func (w *ConfigMapDefaulter) Kind() string {
	return kind
}

func (w *ConfigMapDefaulter) SetDefaults(obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return
	}

	if isNotWatchedConfigmap(cm) {
		return
	}

	switch cm.Name {
	case commons.MeshConfigName:
		cfg := config.ParseMeshConfig(cm)
		if cfg == nil {
			return
		}

		if cfg.DefaultPipyImage == "" {
			cfg.DefaultPipyImage = commons.DefaultPipyImage
		}

		if strings.HasSuffix(cfg.RepoRootURL, "/") {
			cfg.RepoRootURL = strings.TrimSuffix(cfg.RepoRootURL, "/")
		}

		if cfg.Certificate.Manager == "" {
			cfg.Certificate.Manager = string(certificate.Archon)
		}

		cm.Data[commons.MeshConfigJsonName] = cfg.ToJson()
	default:
		// ignore
	}
}

type ConfigMapValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ConfigMapValidator) Kind() string {
	return kind
}

func (w *ConfigMapValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *ConfigMapValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return doValidation(obj)
}

func (w *ConfigMapValidator) ValidateDelete(obj interface{}) error {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	if isNotWatchedConfigmap(cm) {
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

func NewValidator(k8sAPI *kube.K8sAPI) *ConfigMapValidator {
	return &ConfigMapValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	if isNotWatchedConfigmap(cm) {
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
