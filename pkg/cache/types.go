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
	"fmt"
	"github.com/flomesh-io/traffic-guru/pkg/controller"
	gwcontrollerv1alpha2 "github.com/flomesh-io/traffic-guru/pkg/controller/gateway/v1alpha2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Route , Ingress Route interface
type Route interface {
	String() string
	Headers() map[string]string
	Host() string
	Path() string
	Backend() ServicePortName
	Rewrite() []string
}

type ServicePortName struct {
	types.NamespacedName
	Port     string
	Protocol v1.Protocol
}

func (spn ServicePortName) String() string {
	return fmt.Sprintf("%s%s", spn.NamespacedName.String(), fmtPortName(spn.Port))
}

func fmtPortName(in string) string {
	if in == "" {
		return ""
	}
	return fmt.Sprintf(":%s", in)
}

type ServicePort interface {
	String() string
	Address() string
	Port() int
	Protocol() v1.Protocol
	Export() bool
	ExportName() string
}

type Endpoint interface {
	String() string
	IP() string
	Port() (int, error)
	NodeName() string
	HostName() string
	Equal(Endpoint) bool
}

type ServiceEndpoint struct {
	Endpoint        string
	ServicePortName ServicePortName
}

type Controllers struct {
	Service        *controller.ServiceController
	Endpoints      *controller.EndpointsController
	Ingressv1      *controller.Ingressv1Controller
	IngressClassv1 *controller.IngressClassv1Controller
	//ConfigMap      *ConfigMapController
	GatewayApi *GatewayApiControllers
}

type GatewayApiControllers struct {
	V1alpha2 *GatewayApiV1alpha2Controllers
}

type GatewayApiV1alpha2Controllers struct {
	Gateway         *gwcontrollerv1alpha2.GatewayController
	GatewayClass    *gwcontrollerv1alpha2.GatewayClassController
	HTTPRoute       *gwcontrollerv1alpha2.HTTPRouteController
	ReferencePolicy *gwcontrollerv1alpha2.ReferencePolicyController
	TCPRoute        *gwcontrollerv1alpha2.TCPRouteController
	TLSRoute        *gwcontrollerv1alpha2.TLSRouteController
	UDPRoute        *gwcontrollerv1alpha2.UDPRouteController
}
