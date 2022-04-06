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

type operatorCfgChangeListenerForIngress struct {
	k8sApi      *kube.K8sAPI
	configStore *Store
}

var _ OperatorConfigChangeListener = &operatorCfgChangeListenerForIngress{}

func (l operatorCfgChangeListenerForIngress) OnConfigCreate(cfg *OperatorConfig) {
	l.onUpdate(nil, cfg)
}

func (l operatorCfgChangeListenerForIngress) OnConfigUpdate(oldCfg, cfg *OperatorConfig) {
	l.onUpdate(oldCfg, cfg)
}

func (l operatorCfgChangeListenerForIngress) OnConfigDelete(cfg *OperatorConfig) {
	l.onUpdate(cfg, nil)
}

func (l operatorCfgChangeListenerForIngress) onUpdate(oldCfg, cfg *OperatorConfig) {
	if oldCfg == nil {
		oldCfg = l.configStore.OperatorConfig
	}

	if cfg == nil { // cfg is deleted
		l.configStore.OperatorConfig = &OperatorConfig{}
	} else {
		l.configStore.OperatorConfig = cfg
	}

	klog.V(5).Infof("Operator Config is updated, new values: %#v", l.configStore.OperatorConfig)
	klog.V(5).Infof("Old RepoBaseURL = %q", oldCfg.RepoBaseURL())
	klog.V(5).Infof("New RepoBaseURL = %q", l.configStore.OperatorConfig.RepoBaseURL())
	klog.V(5).Infof("Old IngressCodebasePath = %q", oldCfg.IngressCodebasePath())
	klog.V(5).Infof("New IngressCodebasePath = %q", l.configStore.OperatorConfig.IngressCodebasePath())

	// if repo base URL or ingress codebase path is changed, we need to edit ingress-controller deployment
	if oldCfg.RepoRootURL != l.configStore.OperatorConfig.RepoRootURL ||
		oldCfg.RepoPath != l.configStore.OperatorConfig.RepoPath ||
		oldCfg.RepoApiPath != l.configStore.OperatorConfig.RepoApiPath ||
		oldCfg.IngressCodebasePath() != l.configStore.OperatorConfig.IngressCodebasePath() {
		l.updateIngressController()
	}
}

func (l operatorCfgChangeListenerForIngress) updateIngressController() {
	deploy, err := l.k8sApi.Client.AppsV1().
		Deployments(commons.DefaultFlomeshNamespace).
		Get(context.TODO(), "ingress-pipy-controller", metav1.GetOptions{})

	if err != nil {
		klog.Errorf("Get deployment flomesh/ingress-pipy-controller error, %s", err.Error())
		return
	}

	// FIXME: for now, just assume there's only ONE container and it's ingress
	ingressCtn := deploy.Spec.Template.Spec.Containers[0]
	l.updateIngressEnv(ingressCtn, "REPO_BASE_URL", l.configStore.OperatorConfig.RepoBaseURL())
	l.updateIngressEnv(ingressCtn, "INGRESS_CODEBASE_PATH", l.configStore.OperatorConfig.IngressCodebasePath())

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

func (l operatorCfgChangeListenerForIngress) updateIngressEnv(ingressCtn corev1.Container, envName string, envValue string) {
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

type operatorCfgChangeListenerForProxyProfile struct {
	client      client.Client
	k8sApi      *kube.K8sAPI
	configStore *Store
}

func (l operatorCfgChangeListenerForProxyProfile) OnConfigCreate(cfg *OperatorConfig) {
	// TODO: implement it
	klog.Errorf("Implement me!")
	klog.V(5).Infof("Updating ProxyProfile...")
}

func (l operatorCfgChangeListenerForProxyProfile) OnConfigUpdate(oldCfg, cfg *OperatorConfig) {
	klog.V(5).Infof("Updating ProxyProfile...")
	profiles := &pfv1alpha1.ProxyProfileList{}
	if err := l.client.List(context.TODO(), profiles); err != nil {
		// skip creating cm
		return
	}

	for _, pf := range profiles.Items {
		//pf.Spec.RepoBaseUrl = cfg.RepoBaseURL()

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

func (l operatorCfgChangeListenerForProxyProfile) OnConfigDelete(cfg *OperatorConfig) {
	// TODO: implement it
	klog.Errorf("Implement me!")
	klog.V(5).Infof("Updating ProxyProfile...")
}

var _ OperatorConfigChangeListener = &operatorCfgChangeListenerForProxyProfile{}
