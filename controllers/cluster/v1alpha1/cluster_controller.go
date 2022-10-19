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

package v1alpha1

import (
	"context"
	_ "embed"
	clusterv1alpha1 "github.com/flomesh-io/fsm/apis/cluster/v1alpha1"
	svcexpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
	conn "github.com/flomesh-io/fsm/pkg/cluster"
	cctx "github.com/flomesh-io/fsm/pkg/cluster/context"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/event"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/util"
	"io/ioutil"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"sync"
	"time"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	k8sAPI      *kube.K8sAPI
	recorder    record.EventRecorder
	configStore *config.Store
	broker      *event.Broker
	backgrounds map[string]*connectorBackground
	mu          sync.Mutex
}

type connectorBackground struct {
	isInCluster bool
	context     cctx.ConnectorContext
	connector   conn.Connector
}

func New(
	client client.Client,
	api *kube.K8sAPI,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	store *config.Store,
	broker *event.Broker,
	stop <-chan struct{},
) *ClusterReconciler {
	r := &ClusterReconciler{
		Client:      client,
		Scheme:      scheme,
		k8sAPI:      api,
		recorder:    recorder,
		configStore: store,
		broker:      broker,
		backgrounds: make(map[string]*connectorBackground),
	}

	go r.processEvent(broker, stop)

	return r
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Cluster instance
	cluster := &clusterv1alpha1.Cluster{}
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		cluster,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("Cluster resource not found. Stopping the connector and remove the reference.")
			r.destroyConnector(cluster)

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Cluster, %#v", err)
		return ctrl.Result{}, err
	}

	mc := r.configStore.MeshConfig.GetConfig()

	result, err := r.deriveCodebases(mc)
	if err != nil {
		return result, err
	}

	key := cluster.Key()
	klog.V(5).Infof("Cluster key is %s", key)
	bg, exists := r.backgrounds[key]
	if exists && bg.context.SpecHash != util.SimpleHash(cluster.Spec) {
		klog.V(5).Infof("Background context of cluster [%s] exists, ")
		// exists and the spec changed, then stop it and start a new one
		if result, err = r.recreateConnector(bg, cluster); err != nil {
			return result, err
		}
	} else if !exists {
		// doesn't exist, just create a new one
		if result, err = r.createConnector(cluster); err != nil {
			return result, err
		}
	} else {
		klog.V(2).Infof("The connector %s already exists and the spec doesn't change", key)
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) deriveCodebases(mc *config.MeshConfig) (ctrl.Result, error) {
	repoClient := repo.NewRepoClientWithApiBaseUrl(mc.RepoApiBaseURL())

	defaultServicesPath := mc.GetDefaultServicesPath()
	if err := repoClient.DeriveCodebase(defaultServicesPath, commons.DefaultServiceBasePath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	defaultIngressPath := mc.GetDefaultIngressPath()
	if err := repoClient.DeriveCodebase(defaultIngressPath, commons.DefaultIngressBasePath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) createConnector(cluster *clusterv1alpha1.Cluster) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.newConnector(cluster)
}

func (r *ClusterReconciler) recreateConnector(bg *connectorBackground, cluster *clusterv1alpha1.Cluster) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(bg.context.StopCh)
	delete(r.backgrounds, cluster.Key())

	return r.newConnector(cluster)
}

func (r *ClusterReconciler) destroyConnector(cluster *clusterv1alpha1.Cluster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := cluster.Key()
	if bg, exists := r.backgrounds[key]; exists {
		close(bg.context.StopCh)
		delete(r.backgrounds, key)
	}
}

func (r *ClusterReconciler) newConnector(cluster *clusterv1alpha1.Cluster) (ctrl.Result, error) {
	key := cluster.Key()

	kubeconfig, result, err := getKubeConfig(cluster)
	if err != nil {
		return result, err
	}

	background := cctx.ConnectorContext{
		ClusterKey:      key,
		KubeConfig:      kubeconfig,
		ConnectorConfig: connectorConfig(cluster),
		SpecHash:        util.SimpleHash(cluster.Spec),
	}
	_, cancel := context.WithCancel(&background)
	stop := util.RegisterExitHandlers(cancel)
	background.Cancel = cancel
	background.StopCh = stop

	connector, err := conn.NewConnector(&background, r.broker, 15*time.Minute)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.backgrounds[key] = &connectorBackground{
		isInCluster: cluster.Spec.IsInCluster,
		context:     background,
		connector:   connector,
	}

	go func() {
		if err := connector.Run(stop); err != nil {
			close(stop)
			delete(r.backgrounds, key)
		}
	}()

	return ctrl.Result{}, nil
}

func getKubeConfig(cluster *clusterv1alpha1.Cluster) (*rest.Config, ctrl.Result, error) {
	if cluster.Spec.IsInCluster {
		kubeconfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		return kubeconfig, ctrl.Result{}, nil
	} else {
		return remoteKubeConfig(cluster)
	}
}

func remoteKubeConfig(cluster *clusterv1alpha1.Cluster) (*rest.Config, ctrl.Result, error) {
	if _, err := os.Stat(clientcmd.RecommendedConfigDir); os.IsNotExist(err) {
		if err := os.MkdirAll(clientcmd.RecommendedConfigDir, 0644); err != nil {
			return nil, ctrl.Result{}, err
		}
	}

	kubeconfigPath := filepath.Join(clientcmd.RecommendedConfigDir, strings.ReplaceAll(cluster.Key(), "/", "-"))
	if err := ioutil.WriteFile(kubeconfigPath, []byte(cluster.Spec.Kubeconfig), 0644); err != nil {
		return nil, ctrl.Result{}, err
	}

	// use the current context in kubeconfig
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	return kubeconfig, ctrl.Result{}, nil
}

func connectorConfig(cluster *clusterv1alpha1.Cluster) *config.ConnectorConfig {
	return config.NewConnectorConfig(
		cluster.Spec.Region,
		cluster.Spec.Zone,
		cluster.Spec.Group,
		cluster.Name,
		cluster.Spec.Gateway,
		cluster.Spec.IsInCluster,
	)
}

func (r *ClusterReconciler) processEvent(broker *event.Broker, stop <-chan struct{}) {
	msgBus := broker.GetMessageBus()
	svcExportCreatedCh := msgBus.Sub(string(event.ServiceExportCreated))
	defer broker.Unsub(msgBus, svcExportCreatedCh)

	for {
		// FIXME: refine it later
		mc := r.configStore.MeshConfig.GetConfig()
		// ONLY Control Plane takes care of the federation of service export/import
		if !mc.IsControlPlane && mc.IsManaged {
			klog.V(5).Infof("Ignore processing ServiceExportCreated event due to cluster is managed ...")
			continue
		}

		select {
		case msg, ok := <-svcExportCreatedCh:
			if !ok {
				klog.Warningf("Channel closed for ServiceExport")
				continue
			}
			klog.V(5).Infof("Received event ServiceExportCreated %v", msg)

			e, ok := msg.(event.Message)
			if !ok {
				klog.Errorf("Received unexpected message %T on channel, expected Message", e)
				continue
			}

			svcExportEvt, ok := e.NewObj.(*event.ServiceExportEvent)
			if !ok {
				klog.Errorf("Received unexpected object %T, expected ServiceExportEvent", svcExportEvt)
				continue
			}

			// check ServiceExport Status, Invalid and Conflict ServiceExport is ignored
			export := svcExportEvt.ServiceExport
			if metautil.IsStatusConditionFalse(export.Status.Conditions, string(svcexpv1alpha1.ServiceExportValid)) {
				klog.Warningf("ServiceExport %#v is ignored due to Valid status is false", export)
				continue
			}
			if metautil.IsStatusConditionTrue(export.Status.Conditions, string(svcexpv1alpha1.ServiceExportConflict)) {
				klog.Warningf("ServiceExport %#v is ignored due to Conflict status is true", export)
				continue
			}

			r.processServiceExportCreatedEvent(svcExportEvt)
		case <-stop:
			klog.Infof("Received stop signal.")
			return
		}
	}
}

func (r *ClusterReconciler) processServiceExportCreatedEvent(svcExportEvt *event.ServiceExportEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	export := svcExportEvt.ServiceExport
	if r.isFirstTimeExport(svcExportEvt) {
		klog.V(5).Infof("[%s] ServiceExport %s/%s is exported first in the cluster set, will be accepted", svcExportEvt.Geo.Key(), export.Namespace, export.Namespace)
		r.acceptServiceExport(svcExportEvt)
	} else {
		valid, err := r.isValidServiceExport(svcExportEvt)
		if valid {
			klog.V(5).Infof("[%s] ServiceExport %s/%s is valid, will be accepted", svcExportEvt.Geo.Key(), export.Namespace, export.Namespace)
			r.acceptServiceExport(svcExportEvt)
		} else {
			klog.V(5).Infof("[%s] ServiceExport %s/%s is invalid, will be rejected", svcExportEvt.Geo.Key(), export.Namespace, export.Namespace)
			r.rejectServiceExport(svcExportEvt, err)
		}
	}
}

func (r *ClusterReconciler) isFirstTimeExport(event *event.ServiceExportEvent) bool {
	export := event.ServiceExport
	for _, bg := range r.backgrounds {
		if bg.isInCluster {
			continue
		}
		remoteConnector := bg.connector.(*conn.RemoteConnector)
		if remoteConnector.ServiceImportExists(export) {
			klog.Warningf("[%s] ServiceExport %s/%s exists in Cluster %s", event.Geo.Key(), export.Namespace, export.Namespace, bg.context.ClusterKey)
			return false
		}
	}

	return true
}

func (r *ClusterReconciler) isValidServiceExport(svcExportEvt *event.ServiceExportEvent) (bool, error) {
	export := svcExportEvt.ServiceExport
	for _, bg := range r.backgrounds {
		if bg.isInCluster {
			continue
		}

		connectorContext := bg.context
		if connectorContext.ClusterKey == svcExportEvt.ClusterKey() {
			// no need to test against itself
			continue
		}

		remoteConnector := bg.connector.(*conn.RemoteConnector)
		if err := remoteConnector.ValidateServiceExport(svcExportEvt.ServiceExport, svcExportEvt.Service); err != nil {
			klog.Warningf("[%s] ServiceExport %s/%s has conflict in Cluster %s", svcExportEvt.Geo.Key(), export.Namespace, export.Namespace, connectorContext.ClusterKey)
			return false, err
		}
	}

	return true, nil
}

func (r *ClusterReconciler) acceptServiceExport(svcExportEvt *event.ServiceExportEvent) {
	r.broker.GetQueue().Add(
		event.NewServiceExportMessage(
			event.ServiceExportAccepted,
			svcExportEvt.Geo,
			svcExportEvt.ServiceExport,
			svcExportEvt.Service,
			svcExportEvt.Data,
		),
	)
}

func (r *ClusterReconciler) rejectServiceExport(svcExportEvt *event.ServiceExportEvent, err error) {
	if svcExportEvt.Data == nil {
		svcExportEvt.Data = make(map[string]interface{})
	}
	svcExportEvt.Data["reason"] = err

	r.broker.GetQueue().Add(
		event.NewServiceExportMessage(
			event.ServiceExportRejected,
			svcExportEvt.Geo,
			svcExportEvt.ServiceExport,
			svcExportEvt.Service,
			svcExportEvt.Data,
		),
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.Cluster{}).
		Owns(&corev1.Secret{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
