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
	"github.com/flomesh-io/fsm/pkg/cache/controller"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress"
	"github.com/flomesh-io/fsm/pkg/kube"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/events"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	utilcache "k8s.io/kubernetes/pkg/proxy/util"
)

type BaseServiceInfo struct {
	address  string
	port     int
	portName string
	protocol corev1.Protocol
	//exportName string
	//export     bool
	//sessionAffinityType      corev1.ServiceAffinity
}

var _ ServicePort = &BaseServiceInfo{}

func (info *BaseServiceInfo) String() string {
	return fmt.Sprintf("%s:%d/%s", info.address, info.port, info.protocol)
}

func (info *BaseServiceInfo) Address() string {
	return info.address
}

func (info *BaseServiceInfo) Port() int {
	return info.port
}

func (info *BaseServiceInfo) Protocol() corev1.Protocol {
	return info.protocol
}

//func (info *BaseServiceInfo) Export() bool {
//	return info.export
//}
//
//func (info *BaseServiceInfo) ExportName() string {
//	return info.exportName
//}

type enrichServiceInfoFunc func(*corev1.ServicePort, *corev1.Service, *BaseServiceInfo) ServicePort

type serviceChange struct {
	previous ServiceMap
	current  ServiceMap
}

type ServiceChangeTracker struct {
	lock              sync.Mutex
	items             map[types.NamespacedName]*serviceChange
	enrichServiceInfo enrichServiceInfoFunc
	recorder          events.EventRecorder
	controllers       *controller.LocalControllers
	k8sAPI            *kube.K8sAPI
}

type ServiceMap map[ServicePortName]ServicePort

type serviceInfo struct {
	*BaseServiceInfo
	svcName types.NamespacedName
	Type    corev1.ServiceType
}

func (sct *ServiceChangeTracker) newBaseServiceInfo(port *corev1.ServicePort, service *corev1.Service) *BaseServiceInfo {
	klog.V(5).Infof("Service %s/%s, Type: %q, Port %s", service.Namespace, service.Name, service.Spec.Type, port.String())
	switch service.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		// ONLY supports IPv4 for now
		clusterIP := utilcache.GetClusterIPByFamily(corev1.IPv4Protocol, service)
		info := &BaseServiceInfo{
			//address:  netutils.ParseIPSloppy(clusterIP),
			address:  clusterIP,
			port:     int(port.Port),
			portName: port.Name,
			protocol: port.Protocol,
			//sessionAffinityType:   service.Spec.SessionAffinity,
		}

		return info
	case corev1.ServiceTypeExternalName:
		externalName := service.Spec.ExternalName

		if externalName == "localhost" {
			klog.Errorf("Use localhost name %s as External Name in %s/%s", externalName, service.Namespace, service.Name)
			return nil
		}

		ip := net.ParseIP(externalName)
		if ip != nil && ip.IsLoopback() {
			klog.Errorf("External Name %s is resolved to Loopback IP in %s/%s", externalName, service.Namespace, service.Name)
			return nil
		}

		if ip == nil {
			externalName := strings.TrimSuffix(externalName, ".")
			if errs := validation.IsDNS1123Subdomain(externalName); len(errs) > 0 {
				klog.Errorf("Invalid DNS name %q: %v", service.Spec.ExternalName, errs)
				return nil
			}
		}

		info := &BaseServiceInfo{
			address:  fmt.Sprintf("%s:%d", service.Spec.ExternalName, port.TargetPort.IntValue()),
			port:     int(port.Port),
			portName: port.Name,
			protocol: port.Protocol,
		}

		return info
	case corev1.ServiceTypeNodePort:
		// ignore it
	case corev1.ServiceTypeLoadBalancer:
		// TODO: ignore it? Or is it possible to discover Ingress controller(ONLY ingress-pipy) automatically?
	}

	return nil
}

func NewServiceChangeTracker(enrichServiceInfo enrichServiceInfoFunc, recorder events.EventRecorder, controllers *controller.LocalControllers, api *kube.K8sAPI) *ServiceChangeTracker {
	return &ServiceChangeTracker{
		items:             make(map[types.NamespacedName]*serviceChange),
		enrichServiceInfo: enrichServiceInfo,
		recorder:          recorder,
		controllers:       controllers,
		k8sAPI:            api,
	}
}

func (sct *ServiceChangeTracker) Update(previous, current *corev1.Service) bool {
	svc := current
	if svc == nil {
		svc = previous
	}

	if svc == nil {
		return false
	}

	if sct.shouldSkipService(svc) {
		return false
	}

	namespacedName := types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}

	sct.lock.Lock()
	defer sct.lock.Unlock()

	change, exists := sct.items[namespacedName]
	if !exists {
		change = &serviceChange{}
		change.previous = sct.serviceToServiceMap(previous)
		sct.items[namespacedName] = change
	}
	change.current = sct.serviceToServiceMap(current)
	if reflect.DeepEqual(change.previous, change.current) {
		delete(sct.items, namespacedName)
	} else {
		klog.V(2).Infof("Service %s updated: %d ports", namespacedName, len(change.current))
	}

	return len(sct.items) > 0
}

func (sct *ServiceChangeTracker) NotifyIngressChange(previous, current *corev1.Service) {
	svc := current
	if svc == nil {
		svc = previous
	}

	if svc == nil {
		return
	}

	if sct.shouldSkipService(svc) {
		return
	}

	// trigger ingress to be updated
	ingresses, err := sct.controllers.Ingressv1.Lister.Ingresses(svc.Namespace).List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all ingresses in namespace %q: %s", svc.Namespace, err)
	}

	for _, ing := range ingresses {
		if !ingresspipy.IsValidPipyIngress(ing) {
			continue
		}

		if sct.isReferencedByIngress(ing, svc) {
			if ing.Annotations == nil {
				ing.Annotations = map[string]string{}
			}

			ing.Annotations["fsm.flomesh.io/lastUpdated"] = time.Now().Format("20060102-150405.0000")

			if _, err := sct.k8sAPI.Client.
				NetworkingV1().
				Ingresses(svc.Namespace).
				Update(context.TODO(), ing, metav1.UpdateOptions{}); err != nil {
				klog.Errorf("Failed to update annotation of Ingress %s/%s: %s", svc.Namespace, svc.Name, err)
			}
		}
	}
}

func (sct *ServiceChangeTracker) isReferencedByIngress(ing *networkingv1.Ingress, svc *corev1.Service) bool {
	for _, r := range ing.Spec.Rules {
		if r.HTTP == nil {
			continue
		}

		for _, path := range r.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}

			if path.Backend.Service.Name == svc.Name {
				return true
			}
		}
	}

	return false
}

func (sm *ServiceMap) Update(changes *ServiceChangeTracker) {
	sm.apply(changes)
}

func (sct *ServiceChangeTracker) serviceToServiceMap(service *corev1.Service) ServiceMap {
	if service == nil {
		return nil
	}

	clusterIP := utilcache.GetClusterIPByFamily(corev1.IPv4Protocol, service)
	if clusterIP == "" {
		return nil
	}

	serviceMap := make(ServiceMap)
	svcName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		svcPortName := ServicePortName{NamespacedName: svcName, Port: servicePort.Name, Protocol: servicePort.Protocol}
		baseSvcInfo := sct.newBaseServiceInfo(servicePort, service)
		if baseSvcInfo == nil {
			// nil means we cannot handle such type of service
			continue
		}
		if sct.enrichServiceInfo != nil {
			serviceMap[svcPortName] = sct.enrichServiceInfo(servicePort, service, baseSvcInfo)
		} else {
			serviceMap[svcPortName] = baseSvcInfo
		}
	}
	return serviceMap
}

func (sct *ServiceChangeTracker) shouldSkipService(svc *corev1.Service) bool {
	if svc == nil {
		return true
	}

	// Checks if ServiceImport with the same name exists
	// If true, the Service and ServiceImport are aggregated
	//if exists := sct.serviceImportExists(svc); exists {
	//	return true
	//}

	switch svc.Spec.Type {
	// ignore NodePort and LoadBalancer service
	case corev1.ServiceTypeNodePort, corev1.ServiceTypeLoadBalancer:
		klog.V(2).Infof("Service %s/%s is ignored due to type is %q", svc.Namespace, svc.Name, svc.Spec.Type)
		return true
	}

	// TODO: add ignore namespace list to filter

	return false
}

func (sct *ServiceChangeTracker) serviceImportExists(svc *corev1.Service) bool {
	_, err := sct.controllers.ServiceImport.Lister.
		ServiceImports(svc.Namespace).
		Get(svc.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			// do nothing, not exists, go ahead and check svc
			klog.V(5).Infof("ServiceImport %s/%s doesn't exist", svc.Namespace, svc.Name)
			return false
		}

		klog.Warningf("Failed to get ServiceImport %s/%s, %s", svc.Namespace, svc.Name, err)

		return false
	}

	return true
}

func (sm *ServiceMap) apply(changes *ServiceChangeTracker) {
	changes.lock.Lock()
	defer changes.lock.Unlock()
	for _, change := range changes.items {
		sm.merge(change.current)
		change.previous.filter(change.current)
		sm.unmerge(change.previous)
	}
	changes.items = make(map[types.NamespacedName]*serviceChange)
}

func (sm *ServiceMap) merge(other ServiceMap) sets.String {
	existingPorts := sets.NewString()
	for svcPortName, info := range other {
		existingPorts.Insert(svcPortName.String())
		_, exists := (*sm)[svcPortName]
		if !exists {
			klog.V(1).Infof("Adding new service port %q at %s", svcPortName, info.String())
		} else {
			klog.V(1).Infof("Updating existing service port %q at %s", svcPortName, info.String())
		}
		(*sm)[svcPortName] = info
	}
	return existingPorts
}

func (sm *ServiceMap) filter(other ServiceMap) {
	for svcPortName := range *sm {
		if _, ok := other[svcPortName]; ok {
			delete(*sm, svcPortName)
		}
	}
}

func (sm *ServiceMap) unmerge(other ServiceMap) {
	for svcPortName := range other {
		_, exists := (*sm)[svcPortName]
		if exists {
			klog.V(1).Infof("Removing service port %q", svcPortName)
			delete(*sm, svcPortName)
		} else {
			klog.Errorf("Service port %q doesn't exists", svcPortName)
		}
	}
}

func enrichServiceInfo(port *corev1.ServicePort, service *corev1.Service, baseInfo *BaseServiceInfo) ServicePort {
	//annotations := service.GetAnnotations()
	//if annotations != nil && annotations[commons.MultiClustersExported] == "true" {
	//	baseInfo.export = true
	//	exportedName := annotations[commons.MultiClustersExportedName]
	//	if exportedName != "" {
	//		baseInfo.exportName = exportedName
	//	} else {
	//		baseInfo.exportName = service.Name
	//	}
	//}

	info := &serviceInfo{BaseServiceInfo: baseInfo}

	svcName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
	info.svcName = svcName
	info.Type = service.Spec.Type

	return info
}
