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
	pfv1alpha1 "github.com/flomesh-io/traffic-guru/apis/proxyprofile/v1alpha1"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		oldCfg = l.configStore.MeshConfig
	}

	if cfg == nil { // cfg is deleted
		l.configStore.MeshConfig = &MeshConfig{}
	} else {
		l.configStore.MeshConfig = cfg
	}

	klog.V(5).Infof("Operator Config is updated, new values: %#v", l.configStore.MeshConfig)
	klog.V(5).Infof("Old RepoBaseURL = %q", oldCfg.RepoBaseURL())
	klog.V(5).Infof("New RepoBaseURL = %q", l.configStore.MeshConfig.RepoBaseURL())
	klog.V(5).Infof("Old IngressCodebasePath = %q", oldCfg.IngressCodebasePath())
	klog.V(5).Infof("New IngressCodebasePath = %q", l.configStore.MeshConfig.IngressCodebasePath())

	// if repo base URL or ingress codebase path is changed, we need to edit ingress-controller deployment
	if oldCfg.RepoRootURL != l.configStore.MeshConfig.RepoRootURL ||
		oldCfg.RepoPath != l.configStore.MeshConfig.RepoPath ||
		oldCfg.RepoApiPath != l.configStore.MeshConfig.RepoApiPath ||
		oldCfg.IngressCodebasePath() != l.configStore.MeshConfig.IngressCodebasePath() {
		l.updateIngressController()
	}
}

func (l meshCfgChangeListenerForIngress) updateIngressController() {
	deploy, err := l.k8sApi.Client.AppsV1().
		Deployments(commons.DefaultFlomeshNamespace).
		Get(context.TODO(), "ingress-pipy-controller", metav1.GetOptions{})

	if err != nil {
		klog.Errorf("Get deployment flomesh/ingress-pipy-controller error, %s", err.Error())
		return
	}

	// FIXME: for now, just assume there's only ONE container and it's ingress
	ingressCtn := deploy.Spec.Template.Spec.Containers[0]
	l.updateIngressEnv(ingressCtn, "REPO_BASE_URL", l.configStore.MeshConfig.RepoBaseURL())
	l.updateIngressEnv(ingressCtn, "INGRESS_CODEBASE_PATH", l.configStore.MeshConfig.IngressCodebasePath())

	klog.V(5).Infof("Env of ingress container = %#v", ingressCtn.Env)
	deploy.Spec.Template.Spec.Containers[0] = ingressCtn

	deploy, err = l.k8sApi.Client.AppsV1().
		Deployments(commons.DefaultFlomeshNamespace).
		Update(context.TODO(), deploy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Update deployment flomesh/ingress-pipy-controller error, %s", err.Error())
	}

	klog.V(5).Infof("New env of deployment flomesh/ingress-pipy-controller = %#v", deploy.Spec.Template.Spec.Containers[0].Env)
}

func (l meshCfgChangeListenerForIngress) updateIngressEnv(ingressCtn corev1.Container, envName string, envValue string) {
	found := false
	for index, env := range ingressCtn.Env {
		if env.Name == envName {
			klog.V(5).Infof("Old repo base URL = %q", env.Value)
			ingressCtn.Env[index].Value = envValue
			found = true
			break
		}
	}
	if !found {
		ingressCtn.Env = append(ingressCtn.Env, corev1.EnvVar{
			Name:  envName,
			Value: envValue,
		})
	}
}

type meshCfgChangeListenerForProxyProfile struct {
	client      client.Client
	k8sApi      *kube.K8sAPI
	configStore *Store
}

func (l meshCfgChangeListenerForProxyProfile) OnConfigCreate(cfg *MeshConfig) {
	// TODO: implement it
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
		pf.Annotations[commons.ProxyProfileLastUpdatedAnnotation] = time.Now().String()

		if err := l.client.Update(context.TODO(), &pf); err != nil {
			klog.Errorf("update ProxyProfile %s error, %s", pf.Name, err.Error())
			continue
		}
	}
}

func (l meshCfgChangeListenerForProxyProfile) OnConfigDelete(cfg *MeshConfig) {
	// TODO: implement it
}

var _ MeshConfigChangeListener = &meshCfgChangeListenerForProxyProfile{}
