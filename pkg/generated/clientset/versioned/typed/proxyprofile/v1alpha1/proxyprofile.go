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

	v1alpha1 "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ProxyProfilesGetter has a method to return a ProxyProfileInterface.
// A group's client should implement this interface.
type ProxyProfilesGetter interface {
	ProxyProfiles() ProxyProfileInterface
}

// ProxyProfileInterface has methods to work with ProxyProfile resources.
type ProxyProfileInterface interface {
	Create(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile, opts v1.CreateOptions) (*v1alpha1.ProxyProfile, error)
	Update(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile, opts v1.UpdateOptions) (*v1alpha1.ProxyProfile, error)
	UpdateStatus(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile, opts v1.UpdateOptions) (*v1alpha1.ProxyProfile, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ProxyProfile, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ProxyProfileList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ProxyProfile, err error)
	ProxyProfileExpansion
}

// proxyProfiles implements ProxyProfileInterface
type proxyProfiles struct {
	client rest.Interface
}

// newProxyProfiles returns a ProxyProfiles
func newProxyProfiles(c *ProxyprofileV1alpha1Client) *proxyProfiles {
	return &proxyProfiles{
		client: c.RESTClient(),
	}
}

// Get takes name of the proxyProfile, and returns the corresponding proxyProfile object, and an error if there is any.
func (c *proxyProfiles) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ProxyProfile, err error) {
	result = &v1alpha1.ProxyProfile{}
	err = c.client.Get().
		Resource("proxyprofiles").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ProxyProfiles that match those selectors.
func (c *proxyProfiles) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ProxyProfileList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ProxyProfileList{}
	err = c.client.Get().
		Resource("proxyprofiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested proxyProfiles.
func (c *proxyProfiles) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("proxyprofiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a proxyProfile and creates it.  Returns the server's representation of the proxyProfile, and an error, if there is any.
func (c *proxyProfiles) Create(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile, opts v1.CreateOptions) (result *v1alpha1.ProxyProfile, err error) {
	result = &v1alpha1.ProxyProfile{}
	err = c.client.Post().
		Resource("proxyprofiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(proxyProfile).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a proxyProfile and updates it. Returns the server's representation of the proxyProfile, and an error, if there is any.
func (c *proxyProfiles) Update(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile, opts v1.UpdateOptions) (result *v1alpha1.ProxyProfile, err error) {
	result = &v1alpha1.ProxyProfile{}
	err = c.client.Put().
		Resource("proxyprofiles").
		Name(proxyProfile.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(proxyProfile).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *proxyProfiles) UpdateStatus(ctx context.Context, proxyProfile *v1alpha1.ProxyProfile, opts v1.UpdateOptions) (result *v1alpha1.ProxyProfile, err error) {
	result = &v1alpha1.ProxyProfile{}
	err = c.client.Put().
		Resource("proxyprofiles").
		Name(proxyProfile.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(proxyProfile).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the proxyProfile and deletes it. Returns an error if one occurs.
func (c *proxyProfiles) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("proxyprofiles").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *proxyProfiles) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("proxyprofiles").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched proxyProfile.
func (c *proxyProfiles) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ProxyProfile, err error) {
	result = &v1alpha1.ProxyProfile{}
	err = c.client.Patch(pt).
		Resource("proxyprofiles").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
