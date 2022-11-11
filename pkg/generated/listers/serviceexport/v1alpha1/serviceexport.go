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
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ServiceExportLister helps list ServiceExports.
// All objects returned here must be treated as read-only.
type ServiceExportLister interface {
	// List lists all ServiceExports in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.ServiceExport, err error)
	// ServiceExports returns an object that can list and get ServiceExports.
	ServiceExports(namespace string) ServiceExportNamespaceLister
	ServiceExportListerExpansion
}

// serviceExportLister implements the ServiceExportLister interface.
type serviceExportLister struct {
	indexer cache.Indexer
}

// NewServiceExportLister returns a new ServiceExportLister.
func NewServiceExportLister(indexer cache.Indexer) ServiceExportLister {
	return &serviceExportLister{indexer: indexer}
}

// List lists all ServiceExports in the indexer.
func (s *serviceExportLister) List(selector labels.Selector) (ret []*v1alpha1.ServiceExport, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ServiceExport))
	})
	return ret, err
}

// ServiceExports returns an object that can list and get ServiceExports.
func (s *serviceExportLister) ServiceExports(namespace string) ServiceExportNamespaceLister {
	return serviceExportNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ServiceExportNamespaceLister helps list and get ServiceExports.
// All objects returned here must be treated as read-only.
type ServiceExportNamespaceLister interface {
	// List lists all ServiceExports in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.ServiceExport, err error)
	// Get retrieves the ServiceExport from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.ServiceExport, error)
	ServiceExportNamespaceListerExpansion
}

// serviceExportNamespaceLister implements the ServiceExportNamespaceLister
// interface.
type serviceExportNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ServiceExports in the indexer for a given namespace.
func (s serviceExportNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ServiceExport, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ServiceExport))
	})
	return ret, err
}

// Get retrieves the ServiceExport from the indexer for a given namespace and name.
func (s serviceExportNamespaceLister) Get(name string) (*v1alpha1.ServiceExport, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("serviceexport"), name)
	}
	return obj.(*v1alpha1.ServiceExport), nil
}