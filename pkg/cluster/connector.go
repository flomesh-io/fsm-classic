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

package cluster

import (
	"context"
	"fmt"
	flomeshiov1alpha1 "github.com/flomesh-io/fsm/api/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/cache"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	clustercfg "github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

type Connector struct {
	K8sAPI *kube.K8sAPI
	Cache  *cache.Cache
	// ConnectorConfig, the config passed from control-plane operator manager to the connector
	ConnectorConfig clustercfg.ConnectorConfig
}

func NewConnector(kubeconfig *rest.Config, connectorConfig clustercfg.ConnectorConfig, resyncPeriod time.Duration) (*Connector, error) {
	k8sAPI, err := kube.NewAPIForConfig(kubeconfig, 30*time.Second)
	if err != nil {
		return nil, err
	}

	if !version.IsSupportedK8sVersion(k8sAPI) {
		err := fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String())
		klog.Error(err)

		return nil, err
	}

	// checks if flomesh-service-mesh is installed in the cluster, this's a MUST otherwise it doesn't work
	_, err = k8sAPI.Client.AppsV1().
		Deployments(commons.DefaultFlomeshNamespace).
		Get(context.TODO(), commons.OperatorManagerComponentName, metav1.GetOptions{})
	if err != nil {
		klog.Error("Flomesh operator-manager is not installed or not in a proper state, please check it.")
		return nil, err
	}

	clusterCache := cache.NewCache(connectorConfig, k8sAPI, resyncPeriod)

	return &Connector{
		K8sAPI:          k8sAPI,
		Cache:           clusterCache,
		ConnectorConfig: connectorConfig,
	}, nil
}

func (c *Connector) Run() error {
	err := c.updateConfigsOfLinkedCluster()
	if err != nil {
		return err
	}

	if c.Cache.GetBroadcaster() != nil && c.K8sAPI.EventClient != nil {
		klog.V(3).Infof("Starting broadcaster ......")
		stopCh := make(chan struct{})
		c.Cache.GetBroadcaster().StartRecordingToSink(stopCh)
	}

	errCh := make(chan error)

	// register event handlers
	klog.V(3).Infof("Registering event handlers ......")
	controllers := c.Cache.GetControllers()

	go controllers.Service.Run(wait.NeverStop)
	go controllers.Endpoints.Run(wait.NeverStop)
	go controllers.IngressClassv1.Run(wait.NeverStop)
	go controllers.Ingressv1.Run(wait.NeverStop)
	//go controllers.ConfigMap.Run(wait.NeverStop)

	// start the informers manually
	klog.V(3).Infof("Starting informers(svc, ep & ingress class) ......")
	go controllers.Service.Informer.Run(wait.NeverStop)
	go controllers.Endpoints.Informer.Run(wait.NeverStop)
	//go controllers.ConfigMap.Informer.Run(wait.NeverStop)
	go controllers.IngressClassv1.Informer.Run(wait.NeverStop)

	klog.V(3).Infof("Waiting for caches to be synced ......")
	// Ingress depends on service & enpoints, they must be synced first
	if !k8scache.WaitForCacheSync(wait.NeverStop,
		controllers.Endpoints.HasSynced,
		controllers.Service.HasSynced,
		//controllers.ConfigMap.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for services & endpoints caches to sync"))
	}

	// Ingress also depends on IngressClass, but it'c not needed to have relation with svc & ep
	if !k8scache.WaitForCacheSync(wait.NeverStop, controllers.IngressClassv1.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for ingress class cache to sync"))
	}

	// Sleep for a while, so that there'c enough time for processing
	klog.V(5).Infof("Sleep for a while ......")
	time.Sleep(1 * time.Second)

	// start the Ingress Informer
	klog.V(3).Infof("Starting ingress informer ......")
	go controllers.Ingressv1.Informer.Run(wait.NeverStop)
	if !k8scache.WaitForCacheSync(wait.NeverStop, controllers.Ingressv1.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for ingress caches to sync"))
	}

	// start the cache runner
	go c.Cache.SyncLoop()

	return <-errCh
}

func (c *Connector) updateConfigsOfLinkedCluster() error {
	connectorCfg := c.ConnectorConfig

	klog.V(5).Infof("ClusterConnectorMode = %q", connectorCfg.ClusterConnectorMode)
	if connectorCfg.ClusterConnectorMode == string(flomeshiov1alpha1.OutCluster) {
		if connectorCfg.ClusterControlPlaneRepoRootUrl == "" {
			return fmt.Errorf("controlPlaneRepoBaseUrl cannot be empty in OutCluster mode")
		}

		operatorConfig := config.GetOperatorConfig(c.K8sAPI)
		operatorConfig.RepoRootURL = connectorCfg.ClusterControlPlaneRepoRootUrl
		operatorConfig.RepoPath = connectorCfg.ClusterControlPlaneRepoPath
		operatorConfig.RepoApiPath = connectorCfg.ClusterControlPlaneRepoApiPath

		//operatorConfig.IngressCodebasePath = fmt.Sprintf(
		//	"/%s/%s/%s/%s/ingress/",
		//	connectorCfg.ClusterRegion,
		//	connectorCfg.ClusterZone,
		//	connectorCfg.ClusterGroup,
		//	connectorCfg.ClusterName,
		//)
		operatorConfig.Cluster.Region = connectorCfg.ClusterRegion
		operatorConfig.Cluster.Zone = connectorCfg.ClusterZone
		operatorConfig.Cluster.Group = connectorCfg.ClusterGroup
		operatorConfig.Cluster.Name = connectorCfg.ClusterName

		config.UpdateOperatorConfig(c.K8sAPI, operatorConfig)
	}

	return nil
}
