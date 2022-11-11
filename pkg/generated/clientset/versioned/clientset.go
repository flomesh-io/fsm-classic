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

package versioned

import (
	"fmt"
	"net/http"

	clusterv1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/cluster/v1alpha1"
	globaltrafficpolicyv1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/globaltrafficpolicy/v1alpha1"
	multiclusterendpointv1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/multiclusterendpoint/v1alpha1"
	namespacedingressv1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/namespacedingress/v1alpha1"
	proxyprofilev1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/proxyprofile/v1alpha1"
	serviceexportv1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/serviceexport/v1alpha1"
	serviceimportv1alpha1 "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/typed/serviceimport/v1alpha1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	ClusterV1alpha1() clusterv1alpha1.ClusterV1alpha1Interface
	GlobaltrafficpolicyV1alpha1() globaltrafficpolicyv1alpha1.GlobaltrafficpolicyV1alpha1Interface
	MulticlusterendpointV1alpha1() multiclusterendpointv1alpha1.MulticlusterendpointV1alpha1Interface
	NamespacedingressV1alpha1() namespacedingressv1alpha1.NamespacedingressV1alpha1Interface
	ProxyprofileV1alpha1() proxyprofilev1alpha1.ProxyprofileV1alpha1Interface
	ServiceexportV1alpha1() serviceexportv1alpha1.ServiceexportV1alpha1Interface
	ServiceimportV1alpha1() serviceimportv1alpha1.ServiceimportV1alpha1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	clusterV1alpha1              *clusterv1alpha1.ClusterV1alpha1Client
	globaltrafficpolicyV1alpha1  *globaltrafficpolicyv1alpha1.GlobaltrafficpolicyV1alpha1Client
	multiclusterendpointV1alpha1 *multiclusterendpointv1alpha1.MulticlusterendpointV1alpha1Client
	namespacedingressV1alpha1    *namespacedingressv1alpha1.NamespacedingressV1alpha1Client
	proxyprofileV1alpha1         *proxyprofilev1alpha1.ProxyprofileV1alpha1Client
	serviceexportV1alpha1        *serviceexportv1alpha1.ServiceexportV1alpha1Client
	serviceimportV1alpha1        *serviceimportv1alpha1.ServiceimportV1alpha1Client
}

// ClusterV1alpha1 retrieves the ClusterV1alpha1Client
func (c *Clientset) ClusterV1alpha1() clusterv1alpha1.ClusterV1alpha1Interface {
	return c.clusterV1alpha1
}

// GlobaltrafficpolicyV1alpha1 retrieves the GlobaltrafficpolicyV1alpha1Client
func (c *Clientset) GlobaltrafficpolicyV1alpha1() globaltrafficpolicyv1alpha1.GlobaltrafficpolicyV1alpha1Interface {
	return c.globaltrafficpolicyV1alpha1
}

// MulticlusterendpointV1alpha1 retrieves the MulticlusterendpointV1alpha1Client
func (c *Clientset) MulticlusterendpointV1alpha1() multiclusterendpointv1alpha1.MulticlusterendpointV1alpha1Interface {
	return c.multiclusterendpointV1alpha1
}

// NamespacedingressV1alpha1 retrieves the NamespacedingressV1alpha1Client
func (c *Clientset) NamespacedingressV1alpha1() namespacedingressv1alpha1.NamespacedingressV1alpha1Interface {
	return c.namespacedingressV1alpha1
}

// ProxyprofileV1alpha1 retrieves the ProxyprofileV1alpha1Client
func (c *Clientset) ProxyprofileV1alpha1() proxyprofilev1alpha1.ProxyprofileV1alpha1Interface {
	return c.proxyprofileV1alpha1
}

// ServiceexportV1alpha1 retrieves the ServiceexportV1alpha1Client
func (c *Clientset) ServiceexportV1alpha1() serviceexportv1alpha1.ServiceexportV1alpha1Interface {
	return c.serviceexportV1alpha1
}

// ServiceimportV1alpha1 retrieves the ServiceimportV1alpha1Client
func (c *Clientset) ServiceimportV1alpha1() serviceimportv1alpha1.ServiceimportV1alpha1Interface {
	return c.serviceimportV1alpha1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	// share the transport between all clients
	httpClient, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return NewForConfigAndClient(&configShallowCopy, httpClient)
}

// NewForConfigAndClient creates a new Clientset for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfigAndClient will generate a rate-limiter in configShallowCopy.
func NewForConfigAndClient(c *rest.Config, httpClient *http.Client) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}

	var cs Clientset
	var err error
	cs.clusterV1alpha1, err = clusterv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.globaltrafficpolicyV1alpha1, err = globaltrafficpolicyv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.multiclusterendpointV1alpha1, err = multiclusterendpointv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.namespacedingressV1alpha1, err = namespacedingressv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.proxyprofileV1alpha1, err = proxyprofilev1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.serviceexportV1alpha1, err = serviceexportv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.serviceimportV1alpha1, err = serviceimportv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	cs, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.clusterV1alpha1 = clusterv1alpha1.New(c)
	cs.globaltrafficpolicyV1alpha1 = globaltrafficpolicyv1alpha1.New(c)
	cs.multiclusterendpointV1alpha1 = multiclusterendpointv1alpha1.New(c)
	cs.namespacedingressV1alpha1 = namespacedingressv1alpha1.New(c)
	cs.proxyprofileV1alpha1 = proxyprofilev1alpha1.New(c)
	cs.serviceexportV1alpha1 = serviceexportv1alpha1.New(c)
	cs.serviceimportV1alpha1 = serviceimportv1alpha1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
