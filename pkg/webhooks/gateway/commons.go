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

package gateway

//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-gateway,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gateways,verbs=create;update,versions=v1alpha2,name=mgateway.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-gatewayclass,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=create;update,versions=v1alpha2,name=mgatewayclass.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-httproute,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=referencepolices,verbs=create;update,versions=v1alpha2,name=mreferencepolicy.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-referencepolicy,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=httproutes,verbs=create;update,versions=v1alpha2,name=mhttproute.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-tcproute,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=tcproutes,verbs=create;update,versions=v1alpha2,name=mtcproute.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-tlsroute,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=tlsroutes,verbs=create;update,versions=v1alpha2,name=mtlsroute.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-gateway-networking-k8s-io-v1alpha2-udproute,mutating=true,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=udproutes,verbs=create;update,versions=v1alpha2,name=mudproute.gateway.networking.k8s.io,admissionReviewVersions=v1

//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gateways,verbs=create;update,versions=v1alpha2,name=vgateway.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-gatewayclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=create;update,versions=v1alpha2,name=mgatewayclass.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-httproute,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=referencepolices,verbs=create;update,versions=v1alpha2,name=mreferencepolicy.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-referencepolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=httproutes,verbs=create;update,versions=v1alpha2,name=mhttproute.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-tcproute,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=tcproutes,verbs=create;update,versions=v1alpha2,name=mtcproute.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-tlsroute,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=tlsroutes,verbs=create;update,versions=v1alpha2,name=mtlsroute.gateway.networking.k8s.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-udproute,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=udproutes,verbs=create;update,versions=v1alpha2,name=mudproute.gateway.networking.k8s.io,admissionReviewVersions=v1
