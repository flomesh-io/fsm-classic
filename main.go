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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/route"
	"k8s.io/utils/pointer"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func main() {
	listeners := []route.Listener{
		{
			Protocol: "HTTP",
			Port:     80,
		},
		{
			Protocol: "HTTPS",
			Port:     443,
			TLS: &route.TLS{
				TLSModeType: "Terminate",
				MTLS:        false,
				Certificates: []route.Certificate{
					{
						CertChain:  "CertChain",
						PrivateKey: "PrivateKey",
						IssuingCA:  "IssuingCA",
					},
				},
			},
		},
		{
			Protocol: "TCP",
			Port:     1000,
		},
	}

	rules := make(map[int32]route.RouteRule)

	l7rules := route.L7RouteRule{
		"abc.com": route.HTTPRouteRuleSpec{
			RouteType: "HTTP",
			Matches: []route.HTTPTrafficMatch{
				{
					Path: &route.Path{
						MatchType: "Prefix",
						Path:      "/path",
					},
					Headers: []route.Headers{
						{
							MatchType: "Exact",
							Headers: map[string]string{
								"abc": "1",
							},
						},
					},
					RequestParams: []route.RequestParams{
						{
							MatchType: "Exact",
							RequestParams: map[string]string{
								"abc": "1",
							},
						},
					},
					Methods: []string{"GET", "POST"},
					BackendService: map[string]int32{
						"bookstore/bookstore-v1:14001": 100,
					},
				},
			},
		},
		"xyz.com": route.GRPCRouteRuleSpec{
			RouteType: "GRPC",
			Matches: []route.GRPCTrafficMatch{
				{
					Headers: []route.Headers{
						{
							MatchType: "Exact",
							Headers: map[string]string{
								"abc": "1",
							},
						},
						{
							MatchType: "Regex",
							Headers: map[string]string{
								"xxx": "^a",
							},
						},
					},
					Method: &route.GRPCMethod{
						MatchType: "Exact",
						Service:   pointer.String("com.example.GreetingService"),
						Method:    pointer.String("Hello"),
					},
					BackendService: map[string]int32{
						"bookstore/bookstore-v1:14001": 100,
					},
				},
			},
		},
	}
	rules[443] = l7rules

	tcpRules := route.TCPRouteRule{
		"bookstore/bookstore-v1:14001": 100,
	}
	rules[1000] = tcpRules

	tlsTerminateRules := route.TLSTerminateRouteRule{
		"test.com": route.TLSBackendService{
			"bookstore/bookstore-v1:14001": 100,
		},
		"demo.com": route.TLSBackendService{
			"bookstore/bookstore-v1:14001": 100,
		},
	}
	rules[8443] = tlsTerminateRules

	tlsPassthroughRules := route.TLSPassthroughRouteRule{
		"abc.com":  "123.com",
		"test.com": "7789.com:8443",
		"xyz.com":  "456.com:443",
	}
	rules[9999] = tlsPassthroughRules

	services := make(map[string]route.ServiceConfig)
	services["bookstore/bookstore-v1:14001"] = route.ServiceConfig{
		Endpoints: map[string]route.Endpoint{
			"10.0.0.5:8080": {
				Weight: 50,
				Tags: map[string]string{
					"Cluster": "local",
				},
			},
			"10.0.0.6:8080": {
				Weight: 50,
				Tags: map[string]string{
					"Cluster": "local",
				},
			},
			"192.168.1.77:80": {
				Weight: 50,
				Tags: map[string]string{
					"Cluster": "cluster1",
				},
				UpstreamCert: &route.UpstreamCert{
					CertChain:  "CertChain",
					PrivateKey: "PrivateKey",
					IssuingCA:  "IssuingCA",
				},
			},
		},

		Filters: []route.Filter{
			gwv1beta1.HTTPRouteFilter{
				Type: "RequestHeaderModifier",
				RequestHeaderModifier: &gwv1beta1.HTTPHeaderFilter{
					Set: []gwv1beta1.HTTPHeader{
						{
							Name:  "avb",
							Value: "123,456",
						},
					},
					Add: []gwv1beta1.HTTPHeader{
						{
							Name:  "xyz",
							Value: "123",
						},
					},
					Remove: []string{"help"},
				},
			},
			gwv1beta1.HTTPRouteFilter{
				Type: "RequestRedirect",
				RequestRedirect: &gwv1beta1.HTTPRequestRedirectFilter{
					Path: &gwv1beta1.HTTPPathModifier{
						Type:               gwv1beta1.PrefixMatchHTTPPathModifier,
						ReplacePrefixMatch: pointer.String("/foo"),
					},
					StatusCode: pointer.Int(301),
				},
			},
		},

		RetryPolicy: &route.RetryPolicy{
			RetryOn:                  "yes",
			PerTryTimeout:            100,
			NumRetries:               3,
			RetryBackoffBaseInterval: 15,
		},

		UpstreamCert: &route.UpstreamCert{
			CertChain:  "CertChain",
			PrivateKey: "PrivateKey",
			IssuingCA:  "IssuingCA",
		},
	}

	services["test/test-v1:14001"] = route.ServiceConfig{
		Endpoints: map[string]route.Endpoint{
			"10.0.1.5:8080": {
				Weight: 50,
				Tags: map[string]string{
					"Cluster": "local",
				},
			},
			"10.0.1.6:8080": {
				Weight: 50,
				Tags: map[string]string{
					"Cluster": "local",
				},
			},
			"192.168.2.77:80": {
				Weight: 50,
				Tags: map[string]string{
					"Cluster": "cluster1",
				},
				UpstreamCert: &route.UpstreamCert{
					CertChain:  "CertChain",
					PrivateKey: "PrivateKey",
					IssuingCA:  "IssuingCA",
				},
			},
		},

		Filters: []route.Filter{
			gwv1alpha2.GRPCRouteFilter{
				Type: "RequestHeaderModifier",
				RequestHeaderModifier: &gwv1alpha2.HTTPHeaderFilter{
					Set: []gwv1alpha2.HTTPHeader{
						{
							Name:  "avb",
							Value: "123,456",
						},
					},
					Add: []gwv1alpha2.HTTPHeader{
						{
							Name:  "xyz",
							Value: "123",
						},
					},
					Remove: []string{"help"},
				},
			},
			//gwv1alpha2.GRPCRouteFilter{
			//    Type: "RequestMirror",
			//    RequestMirror: &gwv1alpha2.HTTPRequestMirrorFilter{
			//        BackendRef: gwv1alpha2.BackendObjectReference{
			//            Name: gwv1alpha2.ObjectName(""),
			//            Namespace: *gwv1alpha2.Namespace(""),
			//        },
			//    },
			//},
		},

		RetryPolicy: &route.RetryPolicy{
			RetryOn:                  "yes",
			PerTryTimeout:            100,
			NumRetries:               3,
			RetryBackoffBaseInterval: 15,
		},

		UpstreamCert: &route.UpstreamCert{
			CertChain:  "CertChain",
			PrivateKey: "PrivateKey",
			IssuingCA:  "IssuingCA",
		},
	}

	config := &route.ConfigSpec{
		Listeners: listeners,
		Certificate: &route.Certificate{
			CertChain:  "CertChain",
			PrivateKey: "PrivateKey",
			IssuingCA:  "IssuingCA",
		},
		RouteRules: rules,
		Services:   services,
		Chains:     chains(),
	}
	e, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(e))
}

func chains() route.Chains {
	return route.Chains{
		InboundHTTP: []string{
			"modules/inbound-tls-termination.js",
			"modules/inbound-http-routing.js",
			"plugins/inbound-http-default-routing.js",
			"modules/inbound-metrics-http.js",
			"modules/inbound-tracing-http.js",
			"modules/inbound-logging-http.js",
			"modules/inbound-throttle-service.js",
			"modules/inbound-throttle-route.js",
			"modules/inbound-http-load-balancing.js",
			"modules/inbound-http-default.js",
		},
		InboundTCP: []string{
			"modules/inbound-tls-termination.js",
			"modules/inbound-tcp-routing.js",
			"modules/inbound-tcp-load-balancing.js",
			"modules/inbound-tcp-default.js",
		},
		OutboundHTTP: []string{
			"modules/outbound-http-routing.js",
			"plugins/outbound-http-default-routing.js",
			"modules/outbound-metrics-http.js",
			"modules/outbound-tracing-http.js",
			"modules/outbound-logging-http.js",
			"modules/outbound-circuit-breaker.js",
			"modules/outbound-http-load-balancing.js",
			"modules/outbound-http-default.js",
		},
		OutboundTCP: []string{
			"modules/outbound-tcp-routing.js",
			"modules/outbound-tcp-load-balancing.js",
			"modules/outbound-tcp-default.js",
		},
	}
}
