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
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	"github.com/flomesh-io/traffic-guru/pkg/webhooks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"strings"
)

// +kubebuilder:webhook:path=/mutate-core-v1-configmap,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=configmaps,verbs=create;update,versions=v1,name=mconfigmap.kb.flomesh.io,admissionReviewVersions=v1

const Kind = "ConfigMap"

type ConfigMapDefaulter struct {
	k8sAPI *kube.K8sAPI
}

var _ webhooks.Defaulter = &ConfigMapDefaulter{}

// TODO: check if it works or not, perhaps the name is empty at this phase
func isNotWatchedConfigmap(cm *corev1.ConfigMap) bool {
	klog.V(5).Infof("Configmap namespace = %q, name = %q.", cm.Namespace, cm.Name)
	return cm.Namespace != commons.DefaultFlomeshNamespace || !commons.DefaultWatchedConfigMaps.Has(cm.Name)
}

func NewConfigMapDefaulter(k8sAPI *kube.K8sAPI) *ConfigMapDefaulter {
	return &ConfigMapDefaulter{
		k8sAPI: k8sAPI,
	}
}

func (w *ConfigMapDefaulter) Kind() string {
	return Kind
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
		// TODO: set default values
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

		cm.Data[commons.MeshConfigJsonName] = cfg.ToJson()
	default:
		// ignore
	}
}

// +kubebuilder:webhook:path=/validate-core-v1-configmap,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=configmaps,verbs=create;update,versions=v1,name=vconfigmap.kb.flomesh.io,admissionReviewVersions=v1

type ConfigMapValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ConfigMapValidator) Kind() string {
	return Kind
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

var _ webhooks.Validator = &ConfigMapValidator{}

// TODO: register configmaps to watch, for now it's only mesh-config

func NewConfigMapValidator(k8sAPI *kube.K8sAPI) *ConfigMapValidator {
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
		// TODO: validate the config
	default:
		// ignore
	}

	return nil
}
