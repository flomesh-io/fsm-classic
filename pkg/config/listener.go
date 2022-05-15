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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	klog.V(5).Infof("Old RepoBaseURL = %q", oldCfg.RepoBaseURL())
	klog.V(5).Infof("New RepoBaseURL = %q", cfg.RepoBaseURL())
	klog.V(5).Infof("Old IngressCodebasePath = %q", oldCfg.IngressCodebasePath())
	klog.V(5).Infof("New IngressCodebasePath = %q", cfg.IngressCodebasePath())

	// if repo base URL or ingress codebase path is changed, we need to edit ingress-controller deployment
	if oldCfg.Repo.RootURL != cfg.Repo.RootURL ||
		oldCfg.Repo.Path != cfg.Repo.Path ||
		oldCfg.Repo.ApiPath != cfg.Repo.ApiPath ||
		oldCfg.IngressCodebasePath() != cfg.IngressCodebasePath() {
		l.updateIngressController(cfg)
	}
}

func (l meshCfgChangeListenerForIngress) updateIngressController(mc *MeshConfig) {
	// patch the deployment spec template triggers the action of rollout restart like with kubectl
	patch := fmt.Sprintf(
		`{"spec": {"template":{"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`,
		time.Now().String(),
	)
	klog.V(5).Infof("patch = %s", patch)

	_, err := l.k8sApi.Client.AppsV1().
		Deployments(GetFsmNamespace()).
		Patch(context.TODO(), mc.Ingress.DeployName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		klog.Errorf("Patch deployment %s/%s error, %s", GetFsmNamespace(), mc.Ingress.DeployName, err.Error())
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
		pf.Annotations[commons.ProxyProfileLastUpdatedAnnotation] = time.Now().String()

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
