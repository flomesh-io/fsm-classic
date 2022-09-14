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
	svcexpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/event"
)

func (c *Cache) OnServiceExportAdd(export *svcexpv1alpha1.ServiceExport) {
	c.broker.GetQueue().Add(
		event.NewServiceExportMessage(
			event.ServiceExportCreated,
			event.GeoInfo{
				Region:  c.connectorConfig.ClusterRegion,
				Zone:    c.connectorConfig.ClusterZone,
				Group:   c.connectorConfig.ClusterGroup,
				Cluster: c.connectorConfig.ClusterName,
			},
			export,
		))

}

func (c *Cache) OnServiceExportUpdate(oldClass, export *svcexpv1alpha1.ServiceExport) {
	if oldClass.ResourceVersion == export.ResourceVersion {
		return
	}

	// ServiceExport doesn't have spec, no need to handle update?
	// Probably should take care of status update???

	c.broker.GetQueue().Add(
		event.NewServiceExportMessage(
			event.ServiceExportCreated,
			event.GeoInfo{
				Region:  c.connectorConfig.ClusterRegion,
				Zone:    c.connectorConfig.ClusterZone,
				Group:   c.connectorConfig.ClusterGroup,
				Cluster: c.connectorConfig.ClusterName,
			},
			export,
		))
}

func (c *Cache) OnServiceExportDelete(export *svcexpv1alpha1.ServiceExport) {
	c.broker.GetQueue().Add(
		event.NewServiceExportMessage(
			event.ServiceExportDeleted,
			event.GeoInfo{
				Region:  c.connectorConfig.ClusterRegion,
				Zone:    c.connectorConfig.ClusterZone,
				Group:   c.connectorConfig.ClusterGroup,
				Cluster: c.connectorConfig.ClusterName,
			},
			export,
		))
}

func (c *Cache) OnServiceExportSynced() {
	c.mu.Lock()
	c.serviceExportSynced = true
	c.setInitialized(c.servicesSynced && c.endpointsSynced && c.ingressesSynced)
	c.mu.Unlock()

	c.syncRoutes()
}
