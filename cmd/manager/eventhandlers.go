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

package main

import (
	"context"
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/config/listener"
	lcfg "github.com/flomesh-io/fsm-classic/pkg/config/listener/config"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

func registerEventHandler(mgr manager.Manager, api *kube.K8sAPI, configStore *config.Store, certMgr certificate.Manager, repoClient *repo.PipyRepoClient) {

	// FIXME: make it configurable
	resyncPeriod := 15 * time.Minute

	configmapInformer, err := mgr.GetCache().GetInformer(context.TODO(), &corev1.ConfigMap{})

	if err != nil {
		klog.Error(err, "unable to get informer for ConfigMap")
		os.Exit(1)
	}

	listenerConfig := &lcfg.ListenerConfig{
		Client:             mgr.GetClient(),
		K8sApi:             api,
		ConfigStore:        configStore,
		CertificateManager: certMgr,
		RepoClient:         repoClient,
	}

	listeners := []config.MeshConfigChangeListener{
		listener.NewBasicConfigListener(listenerConfig),
		listener.NewIngressConfigListener(listenerConfig),
		listener.NewProxyProfileConfigListener(listenerConfig),
		listener.NewLoggingConfigListener(listenerConfig),
	}

	config.RegisterConfigurationHanlder(
		config.NewFlomeshConfigurationHandler(configStore, listeners),
		configmapInformer,
		resyncPeriod,
	)
}
