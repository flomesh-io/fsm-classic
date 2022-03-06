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
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type configChangeListener struct {
	operatorConfig []OperatorConfigChangeListener
	//clusterConfig  []ClusterConfigChangeListener
}

type FlomeshConfigurationHandler struct {
	configStore *Store
	listeners   *configChangeListener
}

var _ ConfigEventHandler = &FlomeshConfigurationHandler{}

func NewFlomeshConfigurationHandler(client client.Client, k8sApi *kube.K8sAPI, store *Store) *FlomeshConfigurationHandler {
	return &FlomeshConfigurationHandler{
		configStore: store,
		listeners: &configChangeListener{
			operatorConfig: []OperatorConfigChangeListener{
				&operatorCfgChangeListenerForIngress{k8sApi: k8sApi, configStore: store},
				&operatorCfgChangeListenerForProxyProfile{client: client, k8sApi: k8sApi, configStore: store},
			},
			//clusterConfig: []ClusterConfigChangeListener{},
		},
	}
}

func (f FlomeshConfigurationHandler) OnConfigMapAdd(cm *corev1.ConfigMap) {
	klog.V(5).Infof("OnConfigMapAdd(), ConfigMap namespace = %q, name = %q", cm.Namespace, cm.Name)

	switch cm.Name {
	case commons.OperatorConfigName:
		// create the config, and set default values according to the cm
		cfg := ParseOperatorConfig(cm)
		if cfg == nil {
			return
		}

		for _, listener := range f.listeners.operatorConfig {
			listener.OnConfigCreate(cfg)
		}
	//case commons.ClusterConfigName:
	//	cfg := ParseClusterConfig(cm)
	//	if cfg == nil {
	//		return
	//	}
	//
	//	for _, listener := range f.listeners.clusterConfig {
	//		listener.OnConfigCreate(cfg)
	//	}
	default:
		//ignore
	}
}

func (f FlomeshConfigurationHandler) OnConfigMapUpdate(oldCm, cm *corev1.ConfigMap) {
	klog.V(5).Infof("OnConfigMapUpdate(), ConfigMap namespace = %q, name = %q", cm.Namespace, cm.Name)

	switch cm.Name {
	case commons.OperatorConfigName:
		// update the config
		oldCfg := ParseOperatorConfig(oldCm)
		if oldCfg == nil {
			return
		}

		cfg := ParseOperatorConfig(cm)
		if cfg == nil {
			return
		}

		for _, listener := range f.listeners.operatorConfig {
			listener.OnConfigUpdate(oldCfg, cfg)
		}
	//case commons.ClusterConfigName:
	//	oldCfg := ParseClusterConfig(oldCm)
	//	if oldCfg == nil {
	//		return
	//	}
	//
	//	cfg := ParseClusterConfig(cm)
	//	if cfg == nil {
	//		return
	//	}
	//
	//	for _, listener := range f.listeners.clusterConfig {
	//		listener.OnConfigUpdate(oldCfg, cfg)
	//	}
	default:
		//ignore
	}
}

func (f FlomeshConfigurationHandler) OnConfigMapDelete(cm *corev1.ConfigMap) {
	klog.V(5).Infof("OnConfigMapDelete(), ConfigMap namespace = %q, name = %q", cm.Namespace, cm.Name)

	switch cm.Name {
	case commons.OperatorConfigName:
		// Reset the config to default values
		// Actually for now, as ingress-controller mounts the operator-config, if it's deleted will cause an error
		//f.updateOperatorConfig(nil)
		cfg := ParseOperatorConfig(cm)
		if cfg == nil {
			return
		}

		for _, listener := range f.listeners.operatorConfig {
			listener.OnConfigDelete(cfg)
		}

		klog.V(5).Infof("Operator Config is reverted to default, new values: %#v", f.configStore.OperatorConfig)
	//case commons.ClusterConfigName:
	//	cfg := ParseClusterConfig(cm)
	//	if cfg == nil {
	//		return
	//	}
	//
	//	for _, listener := range f.listeners.clusterConfig {
	//		listener.OnConfigDelete(cfg)
	//	}
	default:
		//ignore
	}
}
