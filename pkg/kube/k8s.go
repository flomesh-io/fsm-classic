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

package kube

import (
	"fmt"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	flomesh "github.com/flomesh-io/traffic-guru/pkg/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	cfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

type K8sAPI struct {
	*rest.Config
	Client          kubernetes.Interface
	EventClient     v1core.EventsGetter
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
	FlomeshClient   flomesh.Interface
	Listers         *listers
}

type listers struct {
	ConfigMap v1.ConfigMapLister
}

/**
 * Config precedence
 *      --kubeconfig flag pointing at a file
 *      KUBECONFIG environment variable pointing at a file
 *      In-cluster config if running in cluster
 *      $HOME/.kube/config if exists.
 */

func NewAPI(timeout time.Duration) (*K8sAPI, error) {
	config, err := cfg.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("error get config for K8s API client: %v", err)
	}
	//clientcmd.w
	return NewAPIForConfig(config, timeout)
}

func NewAPIForContext(kubeContext string, timeout time.Duration) (*K8sAPI, error) {
	config, err := cfg.GetConfigWithContext(kubeContext)
	if err != nil {
		return nil, fmt.Errorf("error get config for K8s API client: %v", err)
	}
	return NewAPIForConfig(config, timeout)
}

func NewAPIForConfig(config *rest.Config, timeout time.Duration) (*K8sAPI, error) {
	return NewAPIForConfigOrDie(config, timeout)
}

func NewAPIForConfigOrDie(config *rest.Config, timeout time.Duration) (*K8sAPI, error) {
	config.Timeout = timeout

	clientset := kubernetes.NewForConfigOrDie(config)
	eventClient := kubernetes.NewForConfigOrDie(config)
	dynamicClient := dynamic.NewForConfigOrDie(config)
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config)
	flomeshClient := flomesh.NewForConfigOrDie(config)

	informerFactory := informers.NewSharedInformerFactoryWithOptions(clientset, 60*time.Second, informers.WithNamespace(commons.DefaultFlomeshNamespace))
	configmapLister := informerFactory.Core().V1().ConfigMaps().Lister()
	configmapInformer := informerFactory.Core().V1().ConfigMaps().Informer()
	go configmapInformer.Run(wait.NeverStop)

	if !k8scache.WaitForCacheSync(wait.NeverStop, configmapInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for configmap to sync"))
	}

	return &K8sAPI{
		Config:          config,
		Client:          clientset,
		EventClient:     eventClient.CoreV1(),
		DynamicClient:   dynamicClient,
		DiscoveryClient: discoveryClient,
		FlomeshClient:   flomeshClient,
		Listers: &listers{
			ConfigMap: configmapLister,
		},
	}, nil
}

func MetaNamespaceKey(obj interface{}) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Warning(err)
	}

	return key
}
