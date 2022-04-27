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
	"fmt"
	"github.com/flomesh-io/traffic-guru/pkg/aggregator"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	cachectrl "github.com/flomesh-io/traffic-guru/pkg/controller"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	routepkg "github.com/flomesh-io/traffic-guru/pkg/route"
	"github.com/flomesh-io/traffic-guru/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/async"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Cache struct {
	connectorConfig config.ConnectorConfig
	k8sAPI          *kube.K8sAPI
	recorder        events.EventRecorder
	clusterCfg      *config.Store

	serviceChanges   *ServiceChangeTracker
	endpointsChanges *EndpointChangeTracker
	ingressChanges   *IngressChangeTracker

	serviceMap   ServiceMap
	endpointsMap EndpointsMap
	ingressMap   IngressMap

	mu sync.Mutex

	endpointsSynced      bool
	servicesSynced       bool
	ingressesSynced      bool
	ingressClassesSynced bool
	initialized          int32

	syncRunner       *async.BoundedFrequencyRunner
	aggregatorClient *aggregator.AggregatorClient

	controllers *Controllers
	broadcaster events.EventBroadcaster

	ingressRoutesVersion string
	serviceRoutesVersion string
}

func NewCache(connectorConfig config.ConnectorConfig, api *kube.K8sAPI, resyncPeriod time.Duration) *Cache {
	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: api.Client.EventsV1()})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, "cluster-connector")
	clusterCfg := config.NewStore(api)

	c := &Cache{
		connectorConfig:  connectorConfig,
		k8sAPI:           api,
		recorder:         recorder,
		clusterCfg:       clusterCfg,
		serviceChanges:   NewServiceChangeTracker(enrichServiceInfo, recorder),
		endpointsChanges: NewEndpointChangeTracker(nil, recorder),
		serviceMap:       make(ServiceMap),
		endpointsMap:     make(EndpointsMap),
		ingressMap:       make(IngressMap),
		aggregatorClient: aggregator.NewAggregatorClient(clusterCfg),
		broadcaster:      eventBroadcaster,
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(api.Client, resyncPeriod)
	serviceController := cachectrl.NewServiceControllerWithEventHandler(
		informerFactory.Core().V1().Services(),
		resyncPeriod,
		c,
	)
	endpointsController := cachectrl.NewEndpointsControllerWithEventHandler(
		informerFactory.Core().V1().Endpoints(),
		resyncPeriod,
		c,
	)
	ingressClassV1Controller := cachectrl.NewIngressClassv1ControllerWithEventHandler(
		informerFactory.Networking().V1().IngressClasses(),
		resyncPeriod,
		c,
	)
	ingressV1Controller := cachectrl.NewIngressv1ControllerWithEventHandler(
		informerFactory.Networking().V1().Ingresses(),
		resyncPeriod,
		c,
	)

	// For ConfigMaps, we only cencern flomesh related configs for now, need to narrow the scope of watching
	//infFactoryConfigmaps := informers.NewSharedInformerFactoryWithOptions(api.Client, resyncPeriod,
	//	informers.WithNamespace("flomesh"),
	//)
	//configmapController := cachectrl.NewConfigMapControllerWithEventHandler(
	//	infFactoryConfigmaps.Core().V1().ConfigMaps(),
	//	resyncPeriod,
	//	c,
	//	config.DefaultConfigurationFilter,
	//)

	c.controllers = &Controllers{
		Service:        serviceController,
		Endpoints:      endpointsController,
		Ingressv1:      ingressV1Controller,
		IngressClassv1: ingressClassV1Controller,
		//ConfigMap:      configmapController,
	}

	c.ingressChanges = NewIngressChangeTracker(api, c.controllers, recorder, enrichIngressInfo)

	// FIXME: make it configurable
	minSyncPeriod := 3 * time.Second
	syncPeriod := 30 * time.Second
	burstSyncs := 2
	c.syncRunner = async.NewBoundedFrequencyRunner("sync-runner", c.syncRoutes, minSyncPeriod, syncPeriod, burstSyncs)

	return c
}

func (c *Cache) GetControllers() *Controllers {
	return c.controllers
}

func (c *Cache) GetBroadcaster() events.EventBroadcaster {
	return c.broadcaster
}

func (c *Cache) setInitialized(value bool) {
	var initialized int32
	if value {
		initialized = 1
	}
	atomic.StoreInt32(&c.initialized, initialized)
}

func (c *Cache) isInitialized() bool {
	return atomic.LoadInt32(&c.initialized) > 0
}

func (c *Cache) Sync() {
	c.syncRunner.Run()
}

// SyncLoop runs periodic work.  This is expected to run as a goroutine or as the main loop of the app.  It does not return.
func (c *Cache) SyncLoop() {
	c.syncRunner.Loop(wait.NeverStop)
}

func (c *Cache) syncRoutes() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.serviceMap.Update(c.serviceChanges)
	klog.V(5).Infof("Service Map: %#v", c.serviceMap)

	c.endpointsMap.Update(c.endpointsChanges)
	klog.V(5).Infof("Endpoints Map: %#v", c.endpointsMap)

	c.ingressMap.Update(c.ingressChanges)
	klog.V(5).Infof("Ingress Map: %#v", c.ingressMap)

	klog.V(3).InfoS("Start syncing rules ...")

	r := routepkg.RouteBase{
		UID:     c.connectorConfig.UID(),
		Region:  c.connectorConfig.ClusterRegion,
		Zone:    c.connectorConfig.ClusterZone,
		Group:   c.connectorConfig.ClusterGroup,
		Cluster: c.connectorConfig.ClusterName,
		Gateway: c.connectorConfig.ClusterGateway,
	}

	ingressRoutes := c.buildIngressRoutes(r)
	klog.V(5).Infof("Ingress Routes:\n %#v", ingressRoutes)
	if c.ingressRoutesVersion != ingressRoutes.Hash {
		klog.V(5).Infof("Ingress Routes changed, old hash=%q, new hash=%q", c.ingressRoutesVersion, ingressRoutes.Hash)
		c.ingressRoutesVersion = ingressRoutes.Hash
		go c.aggregatorClient.PostIngresses(ingressRoutes)
	}

	serviceRoutes := c.buildServiceRoutes(r)
	klog.V(5).Infof("Service Routes:\n %#v", serviceRoutes)
	if c.serviceRoutesVersion != serviceRoutes.Hash {
		klog.V(5).Infof("Service Routes changed, old hash=%q, new hash=%q", c.serviceRoutesVersion, serviceRoutes.Hash)
		c.serviceRoutesVersion = serviceRoutes.Hash
		go c.aggregatorClient.PostServices(serviceRoutes)
	}
}

func (c *Cache) buildIngressRoutes(r routepkg.RouteBase) routepkg.IngressRoute {
	ingressRoutes := routepkg.IngressRoute{
		RouteBase: r,
		//Hash:      hash,
		Routes: []routepkg.IngressRouteEntry{},
	}

	for svcName, route := range c.ingressMap {
		ir := routepkg.IngressRouteEntry{
			Host:        route.Host(),
			Path:        route.Path(),
			ServiceName: svcName.String(),
			Rewrite:     route.Rewrite(),
			Upstreams:   []routepkg.EndpointEntry{},
		}
		for _, e := range c.endpointsMap[svcName] {
			ep, ok := e.(*BaseEndpointInfo)
			if !ok {
				klog.ErrorS(nil, "Failed to cast BaseEndpointInfo", "endpoint", e.String())
				continue
			}

			epIP := ep.IP()
			epPort, err := ep.Port()
			// Error parsing this endpoint has been logged. Skip to next endpoint.
			if epIP == "" || err != nil {
				continue
			}
			entry := routepkg.EndpointEntry{
				IP:   epIP,
				Port: epPort,
				//Protocol: protocol,
			}
			ir.Upstreams = append(ir.Upstreams, entry)
		}

		ingressRoutes.Routes = append(ingressRoutes.Routes, ir)
	}

	ingressRoutes.Hash = util.SimpleHash(ingressRoutes)

	return ingressRoutes
}

func (c *Cache) buildServiceRoutes(r routepkg.RouteBase) routepkg.ServiceRoute {
	// Build  rules for each service.
	serviceRoutes := routepkg.ServiceRoute{
		RouteBase: r,
		//Hash:      hash,
		Routes: []routepkg.ServiceRouteEntry{},
	}

	for svcName, svc := range c.serviceMap {
		svcInfo, ok := svc.(*serviceInfo)
		if !ok {
			klog.ErrorS(nil, "Failed to cast serviceInfo", "svcName", svcName.String())
			continue
		}

		sr := routepkg.ServiceRouteEntry{
			Name:      svcInfo.svcName.Name,
			Namespace: svcInfo.svcName.Namespace,
			Targets:   make([]routepkg.Target, 0),
			//IP:         svcInfo.address.String(),
			//Port:       svcInfo.port,
			PortName:   svcInfo.portName,
			Export:     svcInfo.export,
			ExportName: svcInfo.exportName,
		}

		switch svcInfo.Type {
		case corev1.ServiceTypeClusterIP:
			for _, ep := range c.endpointsMap[svcName] {
				sr.Targets = append(sr.Targets, routepkg.Target{
					Address: ep.String(),
					Tags: map[string]string{
						"Node": ep.NodeName(),
						"Host": ep.HostName(),
					}},
				)
			}
		case corev1.ServiceTypeExternalName:
			sr.Targets = append(sr.Targets, routepkg.Target{
				Address: svcInfo.Address(),
				Tags:    map[string]string{}},
			)
		default:
			continue
		}

		route := c.ingressMap[svcName]
		if route != nil {
			// TODO: for an ingress binds multiple DNS names to route to different services, it's not able to handle it with current implementation.
			//   consider to add annotation to service for hints
			reqPath := strings.TrimSuffix(route.Path(), "*")
			reqPath = strings.TrimSuffix(reqPath, "/")
			sr.ExternalPath = fmt.Sprintf("%s%s", c.connectorConfig.ClusterGateway, reqPath)
		}

		serviceRoutes.Routes = append(serviceRoutes.Routes, sr)
	}

	serviceRoutes.Hash = util.SimpleHash(serviceRoutes)

	return serviceRoutes
}
