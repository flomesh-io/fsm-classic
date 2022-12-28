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
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/flomesh-io/fsm/apis/multiclusterendpoint/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MultiClusterEndpointsGetter has a method to return a MultiClusterEndpointInterface.
// A group's client should implement this interface.
type MultiClusterEndpointsGetter interface {
	MultiClusterEndpoints(namespace string) MultiClusterEndpointInterface
}

// MultiClusterEndpointInterface has methods to work with MultiClusterEndpoint resources.
type MultiClusterEndpointInterface interface {
	Create(ctx context.Context, multiClusterEndpoint *v1alpha1.MultiClusterEndpoint, opts v1.CreateOptions) (*v1alpha1.MultiClusterEndpoint, error)
	Update(ctx context.Context, multiClusterEndpoint *v1alpha1.MultiClusterEndpoint, opts v1.UpdateOptions) (*v1alpha1.MultiClusterEndpoint, error)
	UpdateStatus(ctx context.Context, multiClusterEndpoint *v1alpha1.MultiClusterEndpoint, opts v1.UpdateOptions) (*v1alpha1.MultiClusterEndpoint, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MultiClusterEndpoint, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MultiClusterEndpointList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MultiClusterEndpoint, err error)
	MultiClusterEndpointExpansion
}

// multiClusterEndpoints implements MultiClusterEndpointInterface
type multiClusterEndpoints struct {
	client rest.Interface
	ns     string
}

// newMultiClusterEndpoints returns a MultiClusterEndpoints
func newMultiClusterEndpoints(c *MulticlusterendpointV1alpha1Client, namespace string) *multiClusterEndpoints {
	return &multiClusterEndpoints{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the multiClusterEndpoint, and returns the corresponding multiClusterEndpoint object, and an error if there is any.
func (c *multiClusterEndpoints) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MultiClusterEndpoint, err error) {
	result = &v1alpha1.MultiClusterEndpoint{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MultiClusterEndpoints that match those selectors.
func (c *multiClusterEndpoints) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MultiClusterEndpointList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MultiClusterEndpointList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested multiClusterEndpoints.
func (c *multiClusterEndpoints) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a multiClusterEndpoint and creates it.  Returns the server's representation of the multiClusterEndpoint, and an error, if there is any.
func (c *multiClusterEndpoints) Create(ctx context.Context, multiClusterEndpoint *v1alpha1.MultiClusterEndpoint, opts v1.CreateOptions) (result *v1alpha1.MultiClusterEndpoint, err error) {
	result = &v1alpha1.MultiClusterEndpoint{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(multiClusterEndpoint).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a multiClusterEndpoint and updates it. Returns the server's representation of the multiClusterEndpoint, and an error, if there is any.
func (c *multiClusterEndpoints) Update(ctx context.Context, multiClusterEndpoint *v1alpha1.MultiClusterEndpoint, opts v1.UpdateOptions) (result *v1alpha1.MultiClusterEndpoint, err error) {
	result = &v1alpha1.MultiClusterEndpoint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		Name(multiClusterEndpoint.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(multiClusterEndpoint).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *multiClusterEndpoints) UpdateStatus(ctx context.Context, multiClusterEndpoint *v1alpha1.MultiClusterEndpoint, opts v1.UpdateOptions) (result *v1alpha1.MultiClusterEndpoint, err error) {
	result = &v1alpha1.MultiClusterEndpoint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		Name(multiClusterEndpoint.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(multiClusterEndpoint).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the multiClusterEndpoint and deletes it. Returns an error if one occurs.
func (c *multiClusterEndpoints) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *multiClusterEndpoints) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched multiClusterEndpoint.
func (c *multiClusterEndpoints) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MultiClusterEndpoint, err error) {
	result = &v1alpha1.MultiClusterEndpoint{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("multiclusterendpoints").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
