/*
 * The NEU License
 *
 * Copyright (c) 2022.  flomesh.io
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

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type OperatorConfig struct {
	IsControlPlane bool   `yaml:"isControlPlane,omitempty" json:"isControlPlane,omitempty"`
	IngressEnabled bool   `yaml:"ingressEnabled,omitempty" json:"ingressEnabled,omitempty"`
	RepoRootURL    string `yaml:"repoRootURL,omitempty" json:"repoRootURL,omitempty"`
	RepoPath       string `yaml:"repoPath,omitempty" json:"repoPath,omitempty"`
	RepoApiPath    string `yaml:"repoApiPath,omitempty" json:"repoApiPath,omitempty"`
	//IngressCodebasePath string  `yaml:"ingressCodebasePath,omitempty" json:"ingressCodebasePath,omitempty"`
	DefaultPipyImage string  `yaml:"defaultPipyImage,omitempty" json:"defaultPipyImage,omitempty"`
	Cluster          Cluster `yaml:"cluster,omitempty" json:"cluster,omitempty"`

	// TODO: add a config option for indicating if no matched ProxyProfile is found
	//      should we inject a default sidecar or just ignore it?
}

type Cluster struct {
	Region string `yaml:"region,omitempty" json:"region,omitempty"`
	Zone   string `yaml:"zone,omitempty" json:"zone,omitempty"`
	Group  string `yaml:"group,omitempty" json:"group,omitempty"`
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
}

func DefaultOperatorConfig(k8sApi *kube.K8sAPI) *OperatorConfig {
	return GetOperatorConfig(k8sApi)
}

func (o *OperatorConfig) RepoBaseURL() string {
	return fmt.Sprintf("%s%s", o.RepoRootURL, o.RepoPath)
}

func (o *OperatorConfig) RepoApiBaseURL() string {
	return fmt.Sprintf("%s%s", o.RepoRootURL, o.RepoApiPath)
}

func (o *OperatorConfig) IngressCodebasePath() string {
	return fmt.Sprintf(
		"/%s/%s/%s/%s/ingress/",
		o.Cluster.Region,
		o.Cluster.Zone,
		o.Cluster.Group,
		o.Cluster.Name,
	)
}

func (o *OperatorConfig) ToJson() string {
	cfgBytes, err := json.Marshal(o)
	if err != nil {
		klog.Errorf("Not able to marshal OperatorConfig %#v to json, %s", o, err.Error())
		return ""
	}

	return string(cfgBytes)
}

func GetOperatorConfig(k8sApi *kube.K8sAPI) *OperatorConfig {
	cm := GetOperatorConfigMap(k8sApi)

	if cm != nil {
		return ParseOperatorConfig(cm)
	}

	return nil
}

func UpdateOperatorConfig(k8sApi *kube.K8sAPI, config *OperatorConfig) {
	cm := GetOperatorConfigMap(k8sApi)

	if cm == nil {
		return
	}

	cfgBytes, err := json.Marshal(config)
	if err != nil {
		klog.Errorf("Not able to marshal OperatorConfig %#v to json, %s", config, err.Error())
		return
	}
	cm.Data[commons.OperatorConfigJsonName] = string(cfgBytes)

	cm, err = k8sApi.Client.CoreV1().
		ConfigMaps(commons.DefaultFlomeshNamespace).
		Update(context.TODO(), cm, metav1.UpdateOptions{})

	if err != nil {
		klog.Errorf("Update ConfigMap flomesh/operator-config error, %s", err.Error())
		return
	}

	klog.V(5).Infof("After updating, ConfigMap flomesh/operator-config = %#v", cm)
}

func GetOperatorConfigMap(k8sApi *kube.K8sAPI) *corev1.ConfigMap {
	cm, err := k8sApi.Client.CoreV1().
		ConfigMaps(commons.DefaultFlomeshNamespace).
		Get(context.TODO(), commons.OperatorConfigName, metav1.GetOptions{})

	if err != nil {
		klog.Errorf("Get ConfigMap flomesh/operator-config error, %s", err.Error())
		return nil
	}

	return cm
}

func ParseOperatorConfig(cm *corev1.ConfigMap) *OperatorConfig {
	cfgJson, ok := cm.Data[commons.OperatorConfigJsonName]
	if !ok {
		klog.Error("Config file operator_config.json not found, please check ConfigMap flomesh/operator-config.")
		return nil
	}
	klog.V(5).Infof("Found operator_config.json, content: %s", cfgJson)

	cfg := OperatorConfig{}
	err := json.Unmarshal([]byte(cfgJson), &cfg)
	if err != nil {
		klog.Errorf("Unable to unmarshal operator_config.json to operator.Config, %s", err.Error())
		return nil
	}

	return &cfg
}
