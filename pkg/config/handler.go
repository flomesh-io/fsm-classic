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

package config

import (
	"github.com/flomesh-io/fsm/pkg/commons"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type FlomeshConfigurationHandler struct {
	listeners []MeshConfigChangeListener
}

var _ ConfigEventHandler = &FlomeshConfigurationHandler{}

func NewFlomeshConfigurationHandler(listeners []MeshConfigChangeListener) *FlomeshConfigurationHandler {
	return &FlomeshConfigurationHandler{
		listeners: listeners,
	}
}

func (f FlomeshConfigurationHandler) OnConfigMapAdd(cm *corev1.ConfigMap) {
	klog.V(5).Infof("OnConfigMapAdd(), ConfigMap namespace = %q, name = %q", cm.Namespace, cm.Name)

	switch cm.Name {
	case commons.MeshConfigName:
		// create the config, and set default values according to the cm
		cfg, err := ParseMeshConfig(cm)
		if err != nil {
			return
		}

		for _, listener := range f.listeners {
			go listener.OnConfigCreate(cfg)
		}
	default:
		//ignore
	}
}

func (f FlomeshConfigurationHandler) OnConfigMapUpdate(oldCm, cm *corev1.ConfigMap) {
	klog.V(5).Infof("OnConfigMapUpdate(), ConfigMap namespace = %q, name = %q", cm.Namespace, cm.Name)

	switch cm.Name {
	case commons.MeshConfigName:
		// update the config
		oldCfg, err := ParseMeshConfig(oldCm)
		if err != nil {
			return
		}

		cfg, err := ParseMeshConfig(cm)
		if err != nil {
			return
		}

		for _, listener := range f.listeners {
			go listener.OnConfigUpdate(oldCfg, cfg)
		}
	default:
		//ignore
	}
}

func (f FlomeshConfigurationHandler) OnConfigMapDelete(cm *corev1.ConfigMap) {
	klog.V(5).Infof("OnConfigMapDelete(), ConfigMap namespace = %q, name = %q", cm.Namespace, cm.Name)

	switch cm.Name {
	case commons.MeshConfigName:
		// Reset the config to default values
		// Actually for now, as ingress-controller mounts the fsm-mesh-config, if it's deleted will cause an error
		//f.updateMeshConfig(nil)
		cfg, err := ParseMeshConfig(cm)
		if err != nil {
			return
		}

		for _, listener := range f.listeners {
			go listener.OnConfigDelete(cfg)
		}

	default:
		//ignore
	}
}
