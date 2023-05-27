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

package listener

import (
	"context"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	lcfg "github.com/flomesh-io/fsm-classic/pkg/config/listener/config"
	"github.com/flomesh-io/fsm-classic/pkg/config/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

type basicConfigChangeListener struct {
	listenerCfg *lcfg.ListenerConfig
}

func NewBasicConfigListener(cfg *lcfg.ListenerConfig) config.MeshConfigChangeListener {
	return &basicConfigChangeListener{
		listenerCfg: cfg,
	}
}

func (l basicConfigChangeListener) OnConfigCreate(cfg *config.MeshConfig) {
	// TODO: implement it if needed
}

func (l basicConfigChangeListener) OnConfigUpdate(oldCfg, cfg *config.MeshConfig) {
	klog.V(5).Infof("Updating basic config ...")

	if isHTTPConfigChanged(oldCfg, cfg) {
		if err := utils.UpdateIngressHTTPConfig(commons.DefaultIngressBasePath, l.listenerCfg.RepoClient, cfg); err != nil {
			klog.Errorf("Failed to update HTTP config: %s", err)
		}
	}

	if isTLSConfigChanged(oldCfg, cfg) {
		if cfg.Ingress.TLS.Enabled {
			if err := utils.IssueCertForIngress(commons.DefaultIngressBasePath, l.listenerCfg.RepoClient, l.listenerCfg.CertificateManager, cfg); err != nil {
				klog.Errorf("Failed to update TLS config and issue default cert: %s", err)
			}
		} else {
			if err := utils.UpdateIngressTLSConfig(commons.DefaultIngressBasePath, l.listenerCfg.RepoClient, cfg); err != nil {
				klog.Errorf("Failed to update TLS config: %s", err)
			}
		}
	}

	if shouldUpdateIngressControllerServiceSpec(oldCfg, cfg) {
		l.updateIngressControllerSpec(oldCfg, cfg)
	}
}

func (l basicConfigChangeListener) updateIngressControllerSpec(oldCfg *config.MeshConfig, cfg *config.MeshConfig) {
	selector := labels.SelectorFromSet(
		map[string]string{
			"app.kubernetes.io/component":   "controller",
			"app.kubernetes.io/instance":    "fsm-ingress-pipy",
			"ingress.flomesh.io/namespaced": "false",
		},
	)
	svcList, err := l.listenerCfg.K8sApi.Client.CoreV1().
		Services(config.GetFsmNamespace()).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})

	if err != nil {
		klog.Errorf("Failed to list all ingress-pipy services: %s", err)
		return
	}

	// as container port of pod is informational, only change svc spec is enough
	for _, svc := range svcList.Items {
		service := svc.DeepCopy()
		service.Spec.Ports = nil

		if cfg.Ingress.HTTP.Enabled {
			httpPort := corev1.ServicePort{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       cfg.Ingress.HTTP.Bind,
				TargetPort: intstr.FromInt(int(cfg.Ingress.HTTP.Listen)),
			}
			if cfg.Ingress.HTTP.NodePort > 0 {
				httpPort.NodePort = cfg.Ingress.HTTP.NodePort
			}
			service.Spec.Ports = append(service.Spec.Ports, httpPort)
		}

		if cfg.Ingress.TLS.Enabled {
			tlsPort := corev1.ServicePort{
				Name:       "https",
				Protocol:   corev1.ProtocolTCP,
				Port:       cfg.Ingress.TLS.Bind,
				TargetPort: intstr.FromInt(int(cfg.Ingress.TLS.Listen)),
			}
			if cfg.Ingress.TLS.NodePort > 0 {
				tlsPort.NodePort = cfg.Ingress.TLS.NodePort
			}
			service.Spec.Ports = append(service.Spec.Ports, tlsPort)
		}

		if len(service.Spec.Ports) > 0 {
			if _, err := l.listenerCfg.K8sApi.Client.CoreV1().
				Services(config.GetFsmNamespace()).
				Update(context.TODO(), service, metav1.UpdateOptions{}); err != nil {
				klog.Errorf("Failed update spec of ingress-pipy service: %s", err)
			}
		} else {
			klog.Warningf("Both HTTP and TLS are disabled, ignore updating ingress-pipy service")
		}
	}
}

func isHTTPConfigChanged(oldCfg *config.MeshConfig, cfg *config.MeshConfig) bool {
	return cfg.Ingress.Enabled &&
		(oldCfg.Ingress.HTTP.Enabled != cfg.Ingress.HTTP.Enabled ||
			oldCfg.Ingress.HTTP.Listen != cfg.Ingress.HTTP.Listen)
}

func isTLSConfigChanged(oldCfg *config.MeshConfig, cfg *config.MeshConfig) bool {
	return cfg.Ingress.Enabled &&
		(oldCfg.Ingress.TLS.Enabled != cfg.Ingress.TLS.Enabled ||
			oldCfg.Ingress.TLS.Listen != cfg.Ingress.TLS.Listen ||
			oldCfg.Ingress.TLS.MTLS != cfg.Ingress.TLS.MTLS)
}

func shouldUpdateIngressControllerServiceSpec(oldCfg, cfg *config.MeshConfig) bool {
	return cfg.Ingress.Enabled &&
		(oldCfg.Ingress.TLS.Enabled != cfg.Ingress.TLS.Enabled ||
			oldCfg.Ingress.TLS.Listen != cfg.Ingress.TLS.Listen ||
			oldCfg.Ingress.TLS.Bind != cfg.Ingress.TLS.Bind ||
			oldCfg.Ingress.TLS.NodePort != cfg.Ingress.TLS.NodePort ||
			oldCfg.Ingress.HTTP.Enabled != cfg.Ingress.HTTP.Enabled ||
			oldCfg.Ingress.HTTP.Listen != cfg.Ingress.HTTP.Listen ||
			oldCfg.Ingress.HTTP.NodePort != cfg.Ingress.HTTP.NodePort ||
			oldCfg.Ingress.HTTP.Bind != cfg.Ingress.HTTP.Bind)
}

func (l basicConfigChangeListener) OnConfigDelete(cfg *config.MeshConfig) {
	// TODO: implement it if needed
}
