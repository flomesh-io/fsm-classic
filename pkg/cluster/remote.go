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

package cluster

import (
	"context"
	"fmt"
	svcexpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
	svcimpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceimport/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/cache/controller"
	conn "github.com/flomesh-io/fsm/pkg/cluster/context"
	"github.com/flomesh-io/fsm/pkg/event"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func (c *RemoteConnector) Run(stopCh <-chan struct{}) error {
	errCh := make(chan error)

	err := c.updateConfigsOfManagedCluster()
	if err != nil {
		return err
	}

	if c.cache.GetBroadcaster() != nil && c.k8sAPI.EventClient != nil {
		klog.V(3).Infof("Starting broadcaster ......")
		c.cache.GetBroadcaster().StartRecordingToSink(stopCh)
	}

	// register event handlers
	klog.V(3).Infof("Registering event handlers ......")
	controllers := c.cache.GetControllers().(*controller.RemoteControllers)
	go controllers.ServiceExport.Run(stopCh)

	// start the ServiceExport Informer
	klog.V(3).Infof("Starting ServiceExport informer ......")
	go controllers.ServiceExport.Informer.Run(stopCh)
	if !k8scache.WaitForCacheSync(stopCh, controllers.ServiceExport.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for ServiceExport to sync"))
	}

	// Sleep for a while, so that there's enough time for processing
	klog.V(5).Infof("Sleep for a while ......")
	time.Sleep(1 * time.Second)

	// register event handler
	go c.processEvent(c.broker, stopCh)

	// start the cache runner
	go c.cache.SyncLoop(stopCh)

	return <-errCh
}

func (c *RemoteConnector) updateConfigsOfManagedCluster() error {
	ctx := c.context.(*conn.ConnectorContext)
	connectorCfg := ctx.ConnectorConfig

	klog.V(5).Infof("IsInCluster = %q", connectorCfg.IsInCluster)
	if !connectorCfg.IsInCluster {
		//if connectorCfg.ClusterControlPlaneRepoRootUrl == "" {
		//	return fmt.Errorf("controlPlaneRepoBaseUrl cannot be empty in OutCluster mode")
		//}

		mcClient := c.clusterCfg.MeshConfig
		mc := mcClient.GetConfig()
		if mc.IsManaged {
			return fmt.Errorf("cluster %s is already managed, cannot join the MultiCluster", connectorCfg.Key())
		} else {
			mc.IsControlPlane = false
			mc.IsManaged = true
			//mc.Repo.RootURL = connectorCfg.ClusterControlPlaneRepoRootUrl
			//mc.Repo.Path = connectorCfg.ClusterControlPlaneRepoPath
			//mc.Repo.ApiPath = connectorCfg.ClusterControlPlaneRepoApiPath
			mc.Cluster.Region = connectorCfg.Region
			mc.Cluster.Zone = connectorCfg.Zone
			mc.Cluster.Group = connectorCfg.Group
			mc.Cluster.Name = connectorCfg.Name

			mcClient.UpdateConfig(mc)
		}
	}

	return nil
}

func (c *RemoteConnector) processEvent(broker *event.Broker, stopCh <-chan struct{}) {
	mc := c.clusterCfg.MeshConfig.GetConfig()
	msgBus := broker.GetMessageBus()

	svcExportDeletedCh := msgBus.Sub(string(event.ServiceExportDeleted))
	defer broker.Unsub(msgBus, svcExportDeletedCh)
	svcExportAcceptedCh := msgBus.Sub(string(event.ServiceExportAccepted))
	defer broker.Unsub(msgBus, svcExportAcceptedCh)
	svcExportRejectedCh := msgBus.Sub(string(event.ServiceExportRejected))
	defer broker.Unsub(msgBus, svcExportRejectedCh)

	for {
		// FIXME: refine it later
		// ONLY Control Plane takes care of the managed cluster
		if !mc.IsManaged {
			continue
		}

		select {
		case msg, ok := <-svcExportDeletedCh:
			if !ok {
				klog.Warningf("Channel closed for ServiceExport")
				continue
			}

			e, ok := msg.(event.Message)
			if !ok {
				klog.Errorf("Received unexpected message %T on channel, expected Message", e)
				continue
			}

			svcExportEvt, ok := e.OldObj.(*event.ServiceExportEvent)
			if !ok {
				klog.Errorf("Received unexpected object %T, expected ServiceExportEvent", svcExportEvt)
				continue
			}

			if err := c.deleteServiceImport(svcExportEvt); err != nil {
				klog.Errorf("Failed to delete ServiceImport %s/%s", svcExportEvt.ServiceExport.Namespace, svcExportEvt.ServiceExport.Name)
			}
		case msg, ok := <-svcExportAcceptedCh:
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
				klog.Errorf("Received unexpected object %T, ServiceExportEvent", svcExportEvt)
				continue
			}

			if err := c.upsertServiceImport(svcExportEvt); err != nil {
				klog.Errorf("Failed to upsert ServiceImport %s/%s", svcExportEvt.ServiceExport.Namespace, svcExportEvt.ServiceExport.Name)
			}
		case msg, ok := <-svcExportRejectedCh:
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

			export := svcExportEvt.ServiceExport
			reason := svcExportEvt.Data["reason"]
			connectorCtx := c.context.(*conn.ConnectorContext)
			if connectorCtx.ClusterKey == svcExportEvt.ClusterKey() {
				exp, err := c.k8sAPI.FlomeshClient.ServiceexportV1alpha1().
					ServiceExports(export.Namespace).
					Get(context.TODO(), export.Name, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("Failed to update status of ServiceExport %s/%s: %s", export.Namespace, export.Name, err)
					continue
				}

				c.cache.GetRecorder().Eventf(exp, nil, corev1.EventTypeWarning, "Rejected", "ServiceExport %s/%s is invalid, %s", exp.Namespace, exp.Name, reason)

				metautil.SetStatusCondition(&exp.Status.Conditions, metav1.Condition{
					Type:               string(svcexpv1alpha1.ServiceExportConflict),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: exp.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             "Conflict",
					Message:            fmt.Sprintf("ServiceExport %s/%s conflicts, %s", exp.Namespace, exp.Name, reason),
				})

				if _, err := c.k8sAPI.FlomeshClient.ServiceexportV1alpha1().
					ServiceExports(export.Namespace).
					UpdateStatus(context.TODO(), exp, metav1.UpdateOptions{}); err != nil {
					klog.Errorf("Failed to update status of ServiceExport %s/%s: %s", exp.Namespace, exp.Name, err)
					continue
				}
			}

		case <-stopCh:
			return
		}
	}
}

func (c *RemoteConnector) ServiceImportExists(svcExp *svcexpv1alpha1.ServiceExport) bool {
	if _, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Get(context.TODO(), svcExp.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return false
		}
	}

	return true
}

func (c *RemoteConnector) ValidateServiceExport(svcExp *svcexpv1alpha1.ServiceExport, service *corev1.Service) error {
	localSvc, err := c.k8sAPI.Client.CoreV1().
		Services(svcExp.Namespace).
		Get(context.TODO(), svcExp.Namespace, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			// If not found this svc in the cluster, then there' no conflict possibility
			return nil
		}
		return err
	}

	if service.Spec.Type != localSvc.Spec.Type {
		return fmt.Errorf("service type doesn't match: %s vs %s", service.Spec.Type, localSvc.Spec.Type)
	}

	if !reflect.DeepEqual(service.Spec.Ports, localSvc.Spec.Ports) {
		return fmt.Errorf("spec.ports conflict, please check service spec")
	}

	return nil
}

func (c *RemoteConnector) upsertServiceImport(export *event.ServiceExportEvent) error {
	ctx := c.context.(*conn.ConnectorContext)
	clusterKey := export.ClusterKey()
	if clusterKey == ctx.ClusterKey {
		return nil
	}

	svcExp := export.ServiceExport

	imp, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Get(context.TODO(), svcExp.Name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			imp = newServiceImport(export)

			if imp, err = c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
				ServiceImports(svcExp.Namespace).
				Create(context.TODO(), imp, metav1.CreateOptions{}); err != nil {
				klog.Errorf("Failed to create ServiceImport %s/%s: %s", svcExp.Namespace, svcExp.Name, err)
				return err
			}

			return nil
		}

		klog.Errorf("Failed to get ServiceImport %s/%s: %s", svcExp.Namespace, svcExp.Name, err)
		return err
	}

	ports := make([]svcimpv1alpha1.ServicePort, 0)
	for _, p := range imp.Spec.Ports {
		endpoints := make([]svcimpv1alpha1.Endpoint, 0)
		if len(p.Endpoints) == 0 {
			for _, r := range svcExp.Spec.Rules {
				if r.PortNumber == p.Port {
					endpoints = append(endpoints, newEndpoint(export, r))
				}
			}
		} else {
			for _, r := range svcExp.Spec.Rules {
				if r.PortNumber == p.Port {
					for _, ep := range p.Endpoints {
						if ep.ClusterKey == clusterKey {
							endpoints = append(endpoints, newEndpoint(export, r))
						} else {
							endpoints = append(endpoints, *ep.DeepCopy())
						}
					}
				}
			}
		}

		p.Endpoints = endpoints
		ports = append(ports, *p.DeepCopy())
	}
	imp.Spec.Ports = ports

	if _, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Update(context.TODO(), imp, metav1.UpdateOptions{}); err != nil {
		klog.Errorf("Failed to update ServiceImport %s/%s: %s", svcExp.Namespace, svcExp.Name, err)
		return err
	}

	return nil
}

func newServiceImport(export *event.ServiceExportEvent) *svcimpv1alpha1.ServiceImport {
	svcExp := export.ServiceExport
	service := export.Service

	ports := make([]svcimpv1alpha1.ServicePort, 0)
	for _, r := range svcExp.Spec.Rules {
		for _, p := range service.Spec.Ports {
			if r.PortNumber == p.Port {
				ports = append(ports, svcimpv1alpha1.ServicePort{
					Name:        p.Name,
					Port:        p.Port,
					Protocol:    p.Protocol,
					AppProtocol: p.AppProtocol,
					Endpoints: []svcimpv1alpha1.Endpoint{
						newEndpoint(export, r),
					},
				})
			}
		}
	}

	return &svcimpv1alpha1.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcExp.Name,
			Namespace: svcExp.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "flomesh.io/v1alpha1",
			Kind:       "ServiceImport",
		},
		Spec: svcimpv1alpha1.ServiceImportSpec{
			Ports: ports,
		},
	}
}

func newEndpoint(export *event.ServiceExportEvent, r svcexpv1alpha1.ServiceExportRule) svcimpv1alpha1.Endpoint {
	return svcimpv1alpha1.Endpoint{
		ClusterKey: export.ClusterKey(),
		Targets: []string{
			fmt.Sprintf("%s%s", export.Geo.Gateway, r.Path),
		},
	}
}

func (c *RemoteConnector) deleteServiceImport(export *event.ServiceExportEvent) error {
	ctx := c.context.(*conn.ConnectorContext)
	clusterKey := export.ClusterKey()
	if clusterKey == ctx.ClusterKey {
		return nil
	}

	svcExp := export.ServiceExport

	imp, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Get(context.TODO(), svcExp.Name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			klog.Warningf("ServiceImport %s had been deleted.", client.ObjectKeyFromObject(svcExp))
			return nil
		}

		return err
	}

	// update service import, remove the export entry
	ports := make([]svcimpv1alpha1.ServicePort, 0)
	for _, r := range svcExp.Spec.Rules {
		for _, p := range imp.Spec.Ports {
			if r.PortNumber == p.Port {
				endpoints := make([]svcimpv1alpha1.Endpoint, 0)
				for _, ep := range p.Endpoints {
					if ep.ClusterKey == clusterKey {
						continue
					} else {
						endpoints = append(endpoints, *ep.DeepCopy())
					}
				}

				if len(endpoints) > 0 {
					p.Endpoints = endpoints
					ports = append(ports, *p.DeepCopy())
				}
			}
		}
	}

	if len(ports) > 0 {
		imp.Spec.Ports = ports
		if _, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
			ServiceImports(svcExp.Namespace).
			Update(context.TODO(), imp, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Failed to update ServiceImport %s/%s: %s", svcExp.Namespace, svcExp.Name, err)
			return err
		}
	} else {
		if err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
			ServiceImports(svcExp.Namespace).
			Delete(context.TODO(), svcExp.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("Failed to delete ServiceImport %s/%s: %s", svcExp.Namespace, svcExp.Name, err)
			return err
		}
	}

	return nil
}
