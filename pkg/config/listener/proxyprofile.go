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
	pfv1alpha1 "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	lcfg "github.com/flomesh-io/fsm/pkg/config/listener/config"
	"k8s.io/klog/v2"
	"time"
)

type proxyProfileConfigChangeListener struct {
	listenerCfg *lcfg.ListenerConfig
}

func NewProxyProfileConfigListener(cfg *lcfg.ListenerConfig) config.MeshConfigChangeListener {
	return &proxyProfileConfigChangeListener{
		listenerCfg: cfg,
	}
}

func (l proxyProfileConfigChangeListener) OnConfigCreate(cfg *config.MeshConfig) {
	// TODO: implement it if needed
}

func (l proxyProfileConfigChangeListener) OnConfigUpdate(oldCfg, cfg *config.MeshConfig) {
	klog.V(5).Infof("Updating ProxyProfile...")
	profiles := &pfv1alpha1.ProxyProfileList{}
	if err := l.listenerCfg.Client.List(context.TODO(), profiles); err != nil {
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
		if err := l.listenerCfg.Client.Update(context.TODO(), &pf); err != nil {
			klog.Errorf("update ProxyProfile %s error, %s", pf.Name, err.Error())
			continue
		}
	}
}

func (l proxyProfileConfigChangeListener) OnConfigDelete(cfg *config.MeshConfig) {
	// TODO: implement it if needed
}
