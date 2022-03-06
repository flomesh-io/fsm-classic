/*
 * The NEU License
 *
 * Copyright (c) 2022.  flomesh.io
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is furnished to do
 * so, subject to the following conditions:
 *
 * (1)The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * (2)If the software or part of the code will be directly used or used as a
 * component for commercial purposes, including but not limited to: public cloud
 *  services, hosting services, and/or commercial software, the logo as following
 *  shall be displayed in the eye-catching position of the introduction materials
 * of the relevant commercial services or products (such as website, product
 * publicity print), and the logo shall be linked or text marked with the
 * following URL.
 *
 * LOGO : http://flomesh.cn/assets/flomesh-logo.png
 * URL : https://github.com/flomesh-io
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package collector

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/route"
	"github.com/go-resty/resty/v2"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

type CollectorClient struct {
	host             string
	port             int
	baseUrl          string
	defaultTransport *http.Transport
	httpClient       *resty.Client
}

//var _ Repo = &CollectorClient{}

func NewCollectorClient(connectorConfig config.ConnectorConfig) *CollectorClient {
	return NewCollectorClientWithTransport(
		connectorConfig,
		&http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		})
}

func NewCollectorClientWithTransport(connectorConfig config.ConnectorConfig, transport *http.Transport) *CollectorClient {
	baseUrl := fmt.Sprintf(BaseUrlTemplate, commons.DefaultHttpSchema, connectorConfig.ServiceCollectorAddress)

	client := &CollectorClient{
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

func (c *CollectorClient) PostIngresses(routes route.IngressRoute) {
	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(routes).
		SetResult(&commons.Response{}).
		Post(IngressPath)

	if err != nil {
		klog.Errorf("error happened while trying to send ingress routes to collector, %s", err.Error())
	}

	if resp.StatusCode() != http.StatusOK {
		result := resp.Result().(*commons.Response)
		klog.Errorf("collector server responsed with error: %s", result.Result)
	}
}

func (c *CollectorClient) PostServices(routes route.ServiceRoute) {
	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(routes).
		SetResult(&commons.Response{}).
		Post(ServicePath)

	if err != nil {
		klog.Errorf("error happened while trying to send service routes to collector, %s", err.Error())
	}

	if resp.StatusCode() != http.StatusOK {
		result := resp.Result().(*commons.Response)
		klog.Errorf("collector server responsed with error: %s", result.Result)
	}
}
