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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EndpointSlicesProcessor struct {
}

func (p *EndpointSlicesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	eps, ok := obj.(*discovery.EndpointSlice)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	owner := metav1.GetControllerOf(eps)
	if owner == nil {
		return false
	}

	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: owner.Name}
	_, found := cache.endpointslices[svcKey]
	if !found {
		cache.endpointslices[svcKey] = make(map[client.ObjectKey]bool)
	}
	cache.endpointslices[svcKey][objectKey(eps)] = true

	return cache.isRoutableService(svcKey)
}

func (p *EndpointSlicesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	eps, ok := obj.(*discovery.EndpointSlice)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	owner := metav1.GetControllerOf(eps)
	if owner == nil {
		return false
	}

	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: owner.Name}
	slices, found := cache.endpointslices[svcKey]
	if !found {
		return false
	}

	sliceKey := objectKey(eps)
	_, found = slices[sliceKey]
	delete(cache.endpointslices[svcKey], sliceKey)

	if len(cache.endpointslices[svcKey]) == 0 {
		delete(cache.endpointslices, svcKey)
	}

	return found
}
