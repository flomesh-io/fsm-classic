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

package cache

import (
	"context"
	"fmt"
	svcexpv1alpha1 "github.com/flomesh-io/fsm-classic/apis/serviceexport/v1alpha1"
	"github.com/flomesh-io/fsm-classic/pkg/event"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *RemoteCache) OnServiceExportAdd(export *svcexpv1alpha1.ServiceExport) {
	klog.V(5).Infof("[%s] OnServiceExportAdd: %#v", c.connectorConfig.Key(), export)

	c.OnUpdate(nil, export)
}

func (c *RemoteCache) OnServiceExportUpdate(oldExport, export *svcexpv1alpha1.ServiceExport) {
	klog.V(5).Infof("[%s] OnServiceExportUpdate: %#v", c.connectorConfig.Key(), export)

	if oldExport.ResourceVersion == export.ResourceVersion {
		klog.Warningf("[%s] OnServiceExportUpdate %s is ignored as ResourceVersion doesn't change", client.ObjectKeyFromObject(export), c.connectorConfig.Key())
		return
	}

	c.OnUpdate(oldExport, export)
}

func (c *RemoteCache) OnUpdate(oldExport, export *svcexpv1alpha1.ServiceExport) {
	mc := c.clusterCfg.MeshConfig.GetConfig()
	if !mc.IsManaged {
		klog.Warningf("[%s] Cluster is not managed, ignore processing ServiceExport %s", c.connectorConfig.Key(), client.ObjectKeyFromObject(export))
		return
	}

	svc, err := c.getService(export)
	if err != nil {
		klog.Errorf("[%s] Ignore processing ServiceExport %s", c.connectorConfig.Key(), client.ObjectKeyFromObject(export))
		return
	}

	c.broker.Enqueue(
		event.Message{
			Kind:   event.ServiceExportCreated,
			OldObj: nil,
			NewObj: &event.ServiceExportEvent{
				Geo:           c.connectorConfig,
				ServiceExport: export,
				Service:       svc,
			},
		},
	)
}

func (c *RemoteCache) OnServiceExportDelete(export *svcexpv1alpha1.ServiceExport) {
	klog.V(5).Infof("[%s] OnServiceExportDelete: %#v", c.connectorConfig.Key(), export)

	mc := c.clusterCfg.MeshConfig.GetConfig()
	if !mc.IsManaged {
		klog.Warningf("[%s] Cluster is not managed, ignore processing ServiceExport %s", c.connectorConfig.Key(), client.ObjectKeyFromObject(export))
		return
	}

	svc, err := c.getService(export)
	if err != nil {
		klog.Errorf("[%s] Ignore processing ServiceExport %s due to get service failed", c.connectorConfig.Key(), client.ObjectKeyFromObject(export))
		return
	}

	c.broker.Enqueue(
		event.Message{
			Kind:   event.ServiceExportDeleted,
			NewObj: nil,
			OldObj: &event.ServiceExportEvent{
				Geo:           c.connectorConfig,
				ServiceExport: export,
				Service:       svc,
			},
		},
	)
}

func (c *RemoteCache) OnServiceExportSynced() {
	c.mu.Lock()
	c.serviceExportSynced = true
	c.setInitialized(c.serviceExportSynced)
	c.mu.Unlock()

	c.syncManagedCluster()
}

func (c *RemoteCache) getService(export *svcexpv1alpha1.ServiceExport) (*corev1.Service, error) {
	klog.V(5).Infof("[%s] Getting service %s/%s", c.connectorConfig.Key(), export.Namespace, export.Name)

	svc, err := c.k8sAPI.Client.CoreV1().
		Services(export.Namespace).
		Get(context.TODO(), export.Name, metav1.GetOptions{})

	if err != nil {
		klog.Errorf("[%s] Failed to get svc %s/%s, %s", c.connectorConfig.Key(), export.Namespace, export.Name, err)
		return nil, err
	}

	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		msg := fmt.Sprintf("[%s] ExternalName service %s/%s cannot be exported", c.connectorConfig.Key(), export.Namespace, export.Name)
		klog.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}

	return svc, nil
}
