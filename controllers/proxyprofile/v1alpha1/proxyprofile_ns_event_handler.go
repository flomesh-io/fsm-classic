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
	pfv1alpha1 "github.com/flomesh-io/traffic-guru/apis/proxyprofile/v1alpha1"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/injector"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

var _ handler.EventHandler = &NamespaceEventHandler{}

var namespacePredicates = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return injector.IsProxyInjectLabelEnabled(e.Object.GetLabels())
	},

	UpdateFunc: func(e event.UpdateEvent) bool {
		nsOld := e.ObjectOld.(*corev1.Namespace)
		nsNew := e.ObjectNew.(*corev1.Namespace)
		if nsOld.ResourceVersion == nsNew.ResourceVersion {
			return false
		}
		klog.V(3).Infof("Received Namespace %s UpdateEvent", nsNew.GetName())

		oldProxyInjectLabel := getProxyInjectNamespaceLabel(nsOld)
		newProxyInjectLabel := getProxyInjectNamespaceLabel(nsNew)

		return oldProxyInjectLabel != newProxyInjectLabel
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		return true
	},
}

type NamespaceEventHandler struct {
	client.Client
}

func (p *NamespaceEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	klog.V(7).Infof("NamespaceEventHandler - Create(), event=%#v", evt.Object)

	namespace := evt.Object.GetName()
	p.notifyProxyProfileReconciler(namespace, q)
}

func (p *NamespaceEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	klog.V(7).Infof("NamespaceEventHandler - Update(), event=%#v", evt)

	nsNew := evt.ObjectNew.(*corev1.Namespace)
	p.notifyProxyProfileReconciler(nsNew.GetName(), q)
}

func (p *NamespaceEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	klog.V(7).Infof("NamespaceEventHandler - Delete(), event=%#v", evt)
	// need ProxyProfile reconciler to update status
	p.notifyProxyProfileReconciler(evt.Object.GetName(), q)
}

func (p *NamespaceEventHandler) notifyProxyProfileReconciler(namespace string, q workqueue.RateLimitingInterface) {
	// 1. list all existing ProxyProfiles
	profiles := &pfv1alpha1.ProxyProfileList{}
	if err := p.List(context.TODO(), profiles); err != nil {
		klog.Errorf("Not able to list all ProxyProfiles in Namespace %s", namespace)
		// skip creating cm
		return
	}

	// 2. create configmap for each ProxyProfile in the new namespace if the namespace is matched
	items := make(map[string]bool, 0)
	for _, pf := range profiles.Items {
		klog.V(5).Infof("ProxyProfile %s, pf.Spec.Namespace = %s", pf.Name, pf.Spec.Namespace)

		switch pf.GetConfigMode() {
		case pfv1alpha1.ProxyConfigModeLocal:
			// Enqueue the request instead of invoking controller directly
			if pf.Spec.Namespace == "" || pf.Spec.Namespace == namespace {
				klog.V(3).Infof("Namespace %s is created/updated and ProxyProfile %s is capable of creating/updating ConfigMap in it.", namespace, pf.Name)
				items[pf.Name] = true
			}
		case pfv1alpha1.ProxyConfigModeRemote:
			// do nothing
			// NOTE: no need to change for new ProxyProfile spec (Feb-13, 2022).
		default:
			// do nothing
		}
	}

	// ensure each ProxyProfile is notified only once
	for pfName := range items {
		klog.V(3).Infof("Adding ProxyProfile %s to workqueue", pfName)
		q.AddAfter(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: pfName,
			},
		}, 1*time.Second)
	}
}

func (p *NamespaceEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {

}

func getProxyInjectNamespaceLabel(ns *corev1.Namespace) string {
	labels := ns.GetLabels()
	if len(labels) == 0 {
		return ""
	}

	return labels[commons.ProxyInjectNamespaceLabel]
}
