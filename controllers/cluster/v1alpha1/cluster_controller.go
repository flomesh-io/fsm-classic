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
	"flag"
	"fmt"
	clusterv1alpha1 "github.com/flomesh-io/fsm/apis/cluster/v1alpha1"
	"github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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
	meshConfig  *config.MeshConfig
	backgrounds map[string]*connectorBackground
}

type connectorBackground struct {
	isInCluster bool
	context     cctx.ConnectorContext
	connector   conn.Connector
}

func New(client client.Client, api *kube.K8sAPI, scheme *runtime.Scheme, recorder record.EventRecorder, store *config.Store, broker *event.Broker, stop <-chan struct{}) *ClusterReconciler {
	reconciler := &ClusterReconciler{
		Client:      client,
		Scheme:      scheme,
		k8sAPI:      api,
		recorder:    recorder,
		configStore: store,
		broker:      broker,
		meshConfig:  store.MeshConfig.GetConfig(),
		backgrounds: make(map[string]*connectorBackground),
	}

	go processEvent(broker, reconciler, stop)

	return reconciler
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
			key := cluster.Key()
			if bg, exists := r.backgrounds[key]; exists {
				close(bg.context.StopCh)
				delete(r.backgrounds, key)
			}

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Cluster, %#v", err)
		return ctrl.Result{}, err
	}

	mc := r.meshConfig

	result, err := r.deriveCodebases(mc)
	if err != nil {
		return result, err
	}

	key := cluster.Key()
	bg, exists := r.backgrounds[key]
	if exists && bg.context.SpecHash != util.SimpleHash(cluster.Spec) {
		// exists and the spec changed, then stop it and start a new one
		close(bg.context.StopCh)
		delete(r.backgrounds, key)

		if result, err = r.newConnector(cluster); err != nil {
			return result, err
		}
	} else if !exists {
		// doesn't exist, just create a new one
		if result, err = r.newConnector(cluster); err != nil {
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
	connectorCtx, cancel := context.WithCancel(&background)
	stop := util.RegisterExitHandlers(cancel)
	background.Cancel = cancel
	background.StopCh = stop

	connector, err := conn.NewConnector(connectorCtx, r.broker, 15*time.Minute)
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
	filename := fmt.Sprintf("%s", strings.ReplaceAll(cluster.Key(), "/", "-"))
	if err := ioutil.WriteFile(filename, []byte(cluster.Spec.Kubeconfig), 0644); err != nil {
		return nil, ctrl.Result{}, err
	}

	kubeconfigFile := flag.String("kubeconfig", filename, "absolute path to the kubeconfig file")
	flag.Parse()

	// use the current context in kubeconfig
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfigFile)
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	return kubeconfig, ctrl.Result{}, nil
}

func connectorConfig(cluster *clusterv1alpha1.Cluster) config.ConnectorConfig {
	return config.ConnectorConfig{
		Name:        cluster.Name,
		Region:      cluster.Spec.Region,
		Zone:        cluster.Spec.Zone,
		Group:       cluster.Spec.Group,
		Gateway:     cluster.Spec.Gateway,
		IsInCluster: cluster.Spec.IsInCluster,
	}
}

func processEvent(broker *event.Broker, reconciler *ClusterReconciler, stop <-chan struct{}) {
	mc := reconciler.meshConfig
	msgBus := broker.GetMessageBus()
	svcExportCreatedCh := msgBus.Sub(string(event.ServiceExportCreated))
	defer broker.Unsub(msgBus, svcExportCreatedCh)

	for {
		// FIXME: refine it later
		// ONLY Control Plane takes care of the federation of service export/import
		if !mc.IsControlPlane && mc.IsManaged {
			continue
		}

		select {
		case msg, ok := <-svcExportCreatedCh:
			if !ok {
				klog.Warningf("Channel closed for ServiceExport")
				continue
			}

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

			if isFirstTimeExport(reconciler, svcExportEvt.ServiceExport) {
				acceptServiceExport(reconciler, svcExportEvt)
			} else {
				valid, err := isValidServiceExport(reconciler, svcExportEvt)
				if valid {
					acceptServiceExport(reconciler, svcExportEvt)
				} else {
					rejectServiceExport(reconciler, svcExportEvt, err)
				}
			}
		case <-stop:
			return
		}
	}
}

func isFirstTimeExport(reconciler *ClusterReconciler, export *v1alpha1.ServiceExport) bool {
	for _, bg := range reconciler.backgrounds {
		if bg.isInCluster {
			continue
		}
		remoteConnector := bg.connector.(*conn.RemoteConnector)
		if remoteConnector.ServiceImportExists(export) {
			return false
		}
	}

	return true
}

func isValidServiceExport(reconciler *ClusterReconciler, svcExportEvt *event.ServiceExportEvent) (bool, error) {
	for _, bg := range reconciler.backgrounds {
		if bg.isInCluster {
			continue
		}

		connectorContext := bg.context
		if connectorContext.ClusterKey == svcExportEvt.ClusterKey() {
			// no need to test against itself
			continue
		}

		remoteConnector := bg.connector.(*conn.RemoteConnector)
		export := svcExportEvt.ServiceExport
		if err := remoteConnector.ValidateServiceExport(export, svcExportEvt.Service); err != nil {
			return false, err
		}
	}

	return true, nil
}

func acceptServiceExport(reconciler *ClusterReconciler, svcExportEvt *event.ServiceExportEvent) {
	reconciler.broker.GetQueue().Add(
		event.NewServiceExportMessage(
			event.ServiceExportAccepted,
			svcExportEvt.Geo,
			svcExportEvt.ServiceExport,
			svcExportEvt.Service,
			svcExportEvt.Data,
		),
	)
}

func rejectServiceExport(reconciler *ClusterReconciler, svcExportEvt *event.ServiceExportEvent, err error) {
	if svcExportEvt.Data == nil {
		svcExportEvt.Data = make(map[string]interface{})
	}
	svcExportEvt.Data["reason"] = err

	reconciler.broker.GetQueue().Add(
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
