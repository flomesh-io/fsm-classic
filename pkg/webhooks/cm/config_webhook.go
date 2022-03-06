/*
 * The NEU License
 *
 * Copyright (c) 2021-2022.  flomesh.io
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is furnished to do
 * so, subject to the following conditions:
 *
 * (1)The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * (2)If the software or part of the code will be directly used or used as a
 * component for commercial purposes, including but not limited to: public cloud
 *  services, hosting services, and/or commercial software, the logo as following
 *  shall be displayed in the eye-catching position of the introduction materials
 * of the relevant commercial services or products (such as website, product
 * publicity print), and the logo shall be linked or text marked with the
 * following URL.
 *
 * LOGO : http://flomesh.cn/assets/flomesh-logo.png
 * URL : https://github.com/flomesh-io
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
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// +kubebuilder:webhook:path=/mutate-core-v1-configmap,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=configmaps,verbs=create;update,versions=v1,name=mconfigmap.kb.flomesh.io,admissionReviewVersions=v1

var watchedConfigmaps = sets.String{}

type ConfigMapDefaulter struct {
	client client.Client
	k8sAPI *kube.K8sAPI
}

func init() {
	watchedConfigmaps.Insert(commons.OperatorConfigName)
	//watchedConfigmaps.Insert(commons.ClusterConfigName)
}

// TODO: check if it works or not, perhaps the name is empty at this phase
func isNotWatchedConfigmap(cm *corev1.ConfigMap) bool {
	klog.V(5).Infof("Configmap namespace = %q, name = %q.", cm.Namespace, cm.Name)
	return cm.Namespace != commons.DefaultFlomeshNamespace || !commons.DefaultWatchedConfigMaps.Has(cm.Name)
}

func NewConfigMapDefaulter(client client.Client, k8sAPI *kube.K8sAPI) *ConfigMapDefaulter {
	return &ConfigMapDefaulter{
		client: client,
		k8sAPI: k8sAPI,
	}
}

func DefaultingWebhookFor(defaulter *ConfigMapDefaulter) *admission.Webhook {
	return &admission.Webhook{
		Handler: &mutatingHandler{defaulter: defaulter},
	}
}

func (w *ConfigMapDefaulter) SetDefaults(cm *corev1.ConfigMap) {
	if isNotWatchedConfigmap(cm) {
		return
	}

	switch cm.Name {
	case commons.OperatorConfigName:
		// TODO: set default values
		cfg := config.ParseOperatorConfig(cm)
		if cfg == nil {
			return
		}

		if cfg.DefaultPipyImage == "" {
			cfg.DefaultPipyImage = commons.DefaultPipyImage
		}

		//if !strings.HasPrefix(cfg.IngressCodebasePath, "/") {
		//	cfg.IngressCodebasePath = "/" + cfg.IngressCodebasePath
		//}
		//
		//if !strings.HasSuffix(cfg.IngressCodebasePath, "/") {
		//	cfg.IngressCodebasePath = cfg.IngressCodebasePath + "/"
		//}

		if strings.HasSuffix(cfg.RepoRootURL, "/") {
			cfg.RepoRootURL = strings.TrimSuffix(cfg.RepoRootURL, "/")
		}

		cm.Data[commons.OperatorConfigJsonName] = cfg.ToJson()
	//case commons.ClusterConfigName:
	//	// TODO: implement it
	default:
		// ignore
	}
}

// +kubebuilder:webhook:path=/validate-core-v1-configmap,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=configmaps,verbs=create;update,versions=v1,name=vconfigmap.kb.flomesh.io,admissionReviewVersions=v1

type ConfigMapValidator struct {
	client client.Client
	k8sAPI *kube.K8sAPI
}

// TODO: register configmaps to watch, for now it's only operator-config

func NewConfigMapValidator(client client.Client, k8sAPI *kube.K8sAPI) *ConfigMapValidator {
	return &ConfigMapValidator{
		client: client,
		k8sAPI: k8sAPI,
	}
}

func ValidatingWebhookFor(validator *ConfigMapValidator) *admission.Webhook {
	return &admission.Webhook{
		Handler: &validatingHandler{validator: validator},
	}
}

func (w *ConfigMapValidator) ValidateCreate(cm *corev1.ConfigMap) error {
	return doValidation(cm)
}

func (w *ConfigMapValidator) ValidateUpdate(cm *corev1.ConfigMap) error {

	return doValidation(cm)
}

func doValidation(cm *corev1.ConfigMap) error {
	if isNotWatchedConfigmap(cm) {
		return nil
	}

	switch cm.Name {
	case commons.OperatorConfigName:
		// TODO: validate the config
	//case commons.ClusterConfigName:
	//	// TODO: implement it
	default:
		// ignore
	}

	return nil
}
