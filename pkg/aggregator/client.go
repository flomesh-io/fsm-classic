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

package aggregator

import (
	"fmt"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/route"
	"github.com/go-resty/resty/v2"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

type AggregatorClient struct {
	host             string
	port             int
	baseUrl          string
	defaultTransport *http.Transport
	httpClient       *resty.Client
}

//var _ Repo = &AggregatorClient{}

func NewAggregatorClient(clusterCfg *config.Store) *AggregatorClient {
	return NewAggregatorClientWithTransport(
		clusterCfg,
		&http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		})
}

func NewAggregatorClientWithTransport(clusterCfg *config.Store, transport *http.Transport) *AggregatorClient {
	baseUrl := fmt.Sprintf(BaseUrlTemplate, commons.DefaultHttpSchema, clusterCfg.MeshConfig.ServiceAggregatorAddr)

	client := &AggregatorClient{
		baseUrl:          baseUrl,
		defaultTransport: transport,
	}

	client.httpClient = resty.New().
		SetTransport(client.defaultTransport).
		SetScheme(commons.DefaultHttpSchema).
		SetBaseURL(client.baseUrl).
		SetTimeout(5 * time.Second).
		SetDebug(true).
		EnableTrace()

	return client
}

func (c *AggregatorClient) PostIngresses(routes route.IngressRoute) {
	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(routes).
		SetResult(&commons.Response{}).
		Post(IngressPath)

	if err != nil {
		klog.Errorf("error happened while trying to send ingress routes to aggregator, %s", err.Error())
	}

	if resp.StatusCode() != http.StatusOK {
		result := resp.Result().(*commons.Response)
		klog.Errorf("aggregator server responsed with error: %s", result.Result)
	}
}

func (c *AggregatorClient) PostServices(routes route.ServiceRoute) {
	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(routes).
		SetResult(&commons.Response{}).
		Post(ServicePath)

	if err != nil {
		klog.Errorf("error happened while trying to send service routes to aggregator, %s", err.Error())
	}

	if resp.StatusCode() != http.StatusOK {
		result := resp.Result().(*commons.Response)
		klog.Errorf("aggregator server responsed with error: %s", result.Result)
	}
}
