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
	svcimpv1alpha1 "github.com/flomesh-io/fsm-classic/apis/serviceimport/v1alpha1"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	"k8s.io/klog/v2"
)

type ServiceImportsProcessor struct {
}

func (p *ServiceImportsProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*svcimpv1alpha1.ServiceImport)
	if !ok {

		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)
	cache.serviceimports[key] = true

	return cache.isRoutableService(key)
}

func (p *ServiceImportsProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*svcimpv1alpha1.ServiceImport)
	if !ok {

		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)
	_, found := cache.serviceimports[key]
	delete(cache.serviceimports, key)

	return found
}
