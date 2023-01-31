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
	"context"
	"fmt"
	pfv1alpha1 "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type meshCfgChangeListenerForIngress struct {
	k8sApi      *kube.K8sAPI
	configStore *Store
}

var _ MeshConfigChangeListener = &meshCfgChangeListenerForIngress{}

func (l meshCfgChangeListenerForIngress) OnConfigCreate(cfg *MeshConfig) {
	l.onUpdate(nil, cfg)
}

func (l meshCfgChangeListenerForIngress) OnConfigUpdate(oldCfg, cfg *MeshConfig) {
	l.onUpdate(oldCfg, cfg)
}

func (l meshCfgChangeListenerForIngress) OnConfigDelete(cfg *MeshConfig) {
	l.onUpdate(cfg, nil)
}

func (l meshCfgChangeListenerForIngress) onUpdate(oldCfg, cfg *MeshConfig) {
	if oldCfg == nil {
		oldCfg = l.configStore.MeshConfig.GetConfig()
	}

	if cfg == nil { // cfg is deleted
		cfg = &MeshConfig{}
	}

	klog.V(5).Infof("Operator Config is updated, new values: %#v", l.configStore.MeshConfig)
	//klog.V(5).Infof("Old RepoBaseURL = %q", oldCfg.RepoBaseURL())
	//klog.V(5).Infof("New RepoBaseURL = %q", cfg.RepoBaseURL())
	klog.V(5).Infof("Old IngressCodebasePath = %q", oldCfg.IngressCodebasePath())
	klog.V(5).Infof("New IngressCodebasePath = %q", cfg.IngressCodebasePath())

	// if ingress codebase path is changed, we need to edit ingress-controller deployment
	if oldCfg.IngressCodebasePath() != cfg.IngressCodebasePath() {
		l.updateIngressController(cfg)
	}
}

func (l meshCfgChangeListenerForIngress) updateIngressController(mc *MeshConfig) {
	// patch the deployment spec template triggers the action of rollout restart like with kubectl
	patch := fmt.Sprintf(
		`{"spec": {"template":{"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`,
		time.Now().Format(commons.ProxyProfileLastUpdatedTimeFormat),
	)
	klog.V(5).Infof("patch = %s", patch)

	selector := labels.SelectorFromSet(
		map[string]string{
			"app.kubernetes.io/component": "controller",
			"app.kubernetes.io/instance":  "fsm-ingress-pipy",
		},
	)
	ingressList, err := l.k8sApi.Client.AppsV1().
		Deployments(corev1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		klog.Errorf("Error listing all ingress-pipy instances: %s", err)
		return
	}

	for _, ing := range ingressList.Items {
		_, err := l.k8sApi.Client.AppsV1().
			Deployments(ing.Namespace).
			Patch(context.TODO(), ing.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("Patch deployment %s/%s error, %s", ing.Namespace, ing.Name, err)
		}
	}
}

type meshCfgChangeListenerForProxyProfile struct {
	client      client.Client
	k8sApi      *kube.K8sAPI
	configStore *Store
}

func (l meshCfgChangeListenerForProxyProfile) OnConfigCreate(cfg *MeshConfig) {
	// TODO: implement it if needed
}

func (l meshCfgChangeListenerForProxyProfile) OnConfigUpdate(oldCfg, cfg *MeshConfig) {
	klog.V(5).Infof("Updating ProxyProfile...")
	profiles := &pfv1alpha1.ProxyProfileList{}
	if err := l.client.List(context.TODO(), profiles); err != nil {
		// skip updating
		return
	}

	for _, pf := range profiles.Items {
		if pf.Annotations == nil {
			pf.Annotations = make(map[string]string)
		}
		pf.Annotations[commons.ProxyProfileLastUpdated] = time.Now().Format(commons.ProxyProfileLastUpdatedTimeFormat)

		for index, sidecar := range pf.Spec.Sidecars {
			if oldCfg.PipyImage() != cfg.PipyImage() && sidecar.Image == oldCfg.PipyImage() {
				pf.Spec.Sidecars[index].Image = cfg.PipyImage()
			}
		}
		if err := l.client.Update(context.TODO(), &pf); err != nil {
			klog.Errorf("update ProxyProfile %s error, %s", pf.Name, err.Error())
			continue
		}
	}
}

func (l meshCfgChangeListenerForProxyProfile) OnConfigDelete(cfg *MeshConfig) {
	// TODO: implement it if needed
}

var _ MeshConfigChangeListener = &meshCfgChangeListenerForProxyProfile{}

type meshCfgChangeListenerForBasicConfig struct {
	client      client.Client
	k8sApi      *kube.K8sAPI
	configStore *Store
}

func (l meshCfgChangeListenerForBasicConfig) OnConfigCreate(cfg *MeshConfig) {
	// TODO: implement it if needed
}

func (l meshCfgChangeListenerForBasicConfig) OnConfigUpdate(oldCfg, cfg *MeshConfig) {
	klog.V(5).Infof("Updating basic config ...")

	if cfg.Ingress.Enabled &&
		(oldCfg.Ingress.HTTP.Enabled != cfg.Ingress.HTTP.Enabled ||
			oldCfg.Ingress.HTTP.Listen != cfg.Ingress.HTTP.Listen) {
		if err := UpdateIngressHTTPConfig(commons.DefaultIngressBasePath, repo.NewRepoClient(cfg.RepoRootURL()), cfg); err != nil {
			klog.Errorf("Failed to update HTTP config: %s", err)
		}
	}

	if oldCfg.Ingress.TLS.Enabled != cfg.Ingress.TLS.Enabled ||
		oldCfg.Ingress.TLS.Listen != cfg.Ingress.TLS.Listen ||
		oldCfg.Ingress.TLS.MTLS != cfg.Ingress.TLS.MTLS {
		if err := UpdateIngressTLSConfig(commons.DefaultIngressBasePath, repo.NewRepoClient(cfg.RepoRootURL()), cfg); err != nil {
			klog.Errorf("Failed to update TLS config: %s", err)
		}
	}

	selector := labels.SelectorFromSet(
		map[string]string{
			"app.kubernetes.io/component":   "controller",
			"app.kubernetes.io/instance":    "fsm-ingress-pipy",
			"ingress.flomesh.io/namespaced": "false",
		},
	)
	svcList, err := l.k8sApi.Client.CoreV1().
		Services(GetFsmNamespace()).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})

	if err != nil {
        klog.Errorf("Failed to list all ingress-pipy services: %s", err)
        return
	}

	// as container port of pod is informational, only change svc spec is enough
	for _, svc := range svcList.Items {
		service := svc.DeepCopy()
		ports := make([]corev1.ServicePort, 0)

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
			ports = append(ports, httpPort)
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
			ports = append(ports, tlsPort)
		}

		if len(ports) > 0 {
			service.Spec.Ports = ports
			if _, err := l.k8sApi.Client.CoreV1().
				Services(GetFsmNamespace()).
				Update(context.TODO(), service, metav1.UpdateOptions{}); err != nil {
				klog.Errorf("Failed update spec of ingress-pipy service: %s", err)
			}
		} else {
			klog.Warningf("Both HTTP and TLS are disabled, ignore updating ingress-pipy service")
		}
	}
}

func (l meshCfgChangeListenerForBasicConfig) OnConfigDelete(cfg *MeshConfig) {
	// TODO: implement it if needed
}

var _ MeshConfigChangeListener = &meshCfgChangeListenerForBasicConfig{}
