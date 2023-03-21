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

package flb

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

var _ handler.EventHandler = &endpointsEventHandler{}

//var endpointsPredicates = predicate.Funcs{
//	CreateFunc: func(e event.CreateEvent) bool {
//		return injector.IsProxyInjectLabelEnabled(e.Object.GetLabels())
//	},
//
//	UpdateFunc: func(e event.UpdateEvent) bool {
//		nsOld := e.ObjectOld.(*corev1.Namespace)
//		nsNew := e.ObjectNew.(*corev1.Namespace)
//		if nsOld.ResourceVersion == nsNew.ResourceVersion {
//			return false
//		}
//		klog.V(3).Infof("Received Namespace %s UpdateEvent", nsNew.GetName())
//
//		oldProxyInjectLabel := getProxyInjectNamespaceLabel(nsOld)
//		newProxyInjectLabel := getProxyInjectNamespaceLabel(nsNew)
//
//		return oldProxyInjectLabel != newProxyInjectLabel
//	},
//
//	DeleteFunc: func(e event.DeleteEvent) bool {
//		return true
//	},
//}

type endpointsEventHandler struct {
	client.Client
}

func (e *endpointsEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	klog.V(7).Infof("endpointsEventHandler - Create(), event=%#v", evt.Object)

	e.enqueueObject(evt.Object, q)
}

func (e *endpointsEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	klog.V(7).Infof("endpointsEventHandler - Update(), event=%#v", evt)

	e.enqueueObject(evt.ObjectNew, q)
}

func (e *endpointsEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	klog.V(7).Infof("endpointsEventHandler - Delete(), event=%#v", evt)

	e.enqueueObject(evt.Object, q)
}

func (e *endpointsEventHandler) enqueueObject(obj client.Object, q workqueue.RateLimitingInterface) {
	if svc := e.getServiceByEndpoints(obj); svc != nil && isFlbEnabled(context.TODO(), e.Client, svc) {
		q.AddAfter(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			},
		}, 1*time.Second)
	}
}

func (e *endpointsEventHandler) getServiceByEndpoints(ep client.Object) *corev1.Service {
	svc := &corev1.Service{}
	if err := e.Client.Get(
		context.TODO(),
		client.ObjectKey{Namespace: ep.GetNamespace(), Name: ep.GetName()},
		svc,
	); err != nil {
		klog.Errorf("failed to get service %s/%s: %s", ep.GetNamespace(), ep.GetName(), err)
		return nil
	}

	return svc
}

func (e *endpointsEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {

}
