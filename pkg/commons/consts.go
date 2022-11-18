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

package commons

import (
	"text/template"
	"time"
)

const (
	// Global constants

	DefaultFsmNamespace           = "flomesh"
	DefaultCABundleName           = "flomesh-ca-bundle"
	RootCACertName                = "ca.crt"
	RootCAPrivateKeyName          = "ca.key"
	TLSCertName                   = "tls.crt"
	TLSPrivateKeyName             = "tls.key"
	WebhookServerServingCertsPath = "/tmp/k8s-webhook-server/serving-certs"
	DefaultCAValidityPeriod       = 24 * 365 * 10 * time.Hour
	DefaultCACommonName           = "flomesh.io"
	DefaultCACountry              = "CN"
	DefaultCALocality             = "Dalian"
	DefaultCAOrganization         = "flomesh.io"
	ManagerDeploymentName         = "fsm-manager"
	MeshConfigName                = "fsm-mesh-config"
	MeshConfigJsonName            = "mesh_config.json"
	DefaultPipyRepoPath           = "/repo"
	DefaultPipyRepoApiPath        = "/api/v1/repo"

	// Proxy CRD

	DeploymentNameSuffix = "-fsmdp"
	ServiceNameSuffix    = "-fsmsvc"
	ConfigMapNameSuffix  = "-fsmcm"
	DaemonSetNameSuffix  = "-fsmds"
	VolumeNameSuffix     = "-fsmvlm"

	// Webhooks

	DefaultWebhookServiceName                 = "fsm-webhook-service"
	DefaultMutatingWebhookConfigurationName   = "flomesh-mutating-webhook-configuration"
	DefaultValidatingWebhookConfigurationName = "flomesh-validating-webhook-configuration"
	ProxyInjectorWebhookPath                  = "/proxy-injector-flomesh-io-v1alpha1"
	ProxyProfileMutatingWebhookPath           = "/mutate-flomesh-io-v1alpha1-proxyprofile"
	ProxyProfileValidatingWebhookPath         = "/validate-flomesh-io-v1alpha1-proxyprofile"
	ConfigMapMutatingWebhookPath              = "/mutate-core-v1-configmap"
	ConfigMapValidatingWebhookPath            = "/validate-core-v1-configmap"
	ClusterMutatingWebhookPath                = "/mutate-flomesh-io-v1alpha1-cluster"
	ClusterValidatingWebhookPath              = "/validate-flomesh-io-v1alpha1-cluster"
	NamespacedIngressMutatingWebhookPath      = "/mutate-flomesh-io-v1alpha1-namespacedingress"
	NamespacedIngressValidatingWebhookPath    = "/validate-flomesh-io-v1alpha1-namespacedingress"
	GatewayMutatingWebhookPath                = "/mutate-gateway-networking-k8s-io-v1alpha2-gateway"
	GatewayValidatingWebhookPath              = "/validate-gateway-networking-k8s-io-v1alpha2-gateway"
	GatewayClassMutatingWebhookPath           = "/mutate-gateway-networking-k8s-io-v1alpha2-gatewayclass"
	GatewayClassValidatingWebhookPath         = "/validate-gateway-networking-k8s-io-v1alpha2-gatewayclass"
	HTTPRouteMutatingWebhookPath              = "/mutate-gateway-networking-k8s-io-v1alpha2-httproute"
	HTTPRouteValidatingWebhookPath            = "/validate-gateway-networking-k8s-io-v1alpha2-httproute"
	ReferencePolicyMutatingWebhookPath        = "/mutate-gateway-networking-k8s-io-v1alpha2-referencepolicy"
	ReferencePolicyValidatingWebhookPath      = "/validate-gateway-networking-k8s-io-v1alpha2-referencepolicy"
	TCPRouteMutatingWebhookPath               = "/mutate-gateway-networking-k8s-io-v1alpha2-tcproute"
	TCPRouteValidatingWebhookPath             = "/validate-gateway-networking-k8s-io-v1alpha2-tcproute"
	TLSRouteMutatingWebhookPath               = "/mutate-gateway-networking-k8s-io-v1alpha2-tlsroute"
	TLSRouteValidatingWebhookPath             = "/validate-gateway-networking-k8s-io-v1alpha2-tlsroute"
	UDPRouteMutatingWebhookPath               = "/mutate-gateway-networking-k8s-io-v1alpha2-udproute"
	UDPRouteValidatingWebhookPath             = "/validate-gateway-networking-k8s-io-v1alpha2-udproute"

	// Sidecar constants

	AnnotationPrefix                  = "flomesh.io"
	ProxyInjectIndicator              = AnnotationPrefix + "/inject"
	FlomeshControlPlaneLabel          = AnnotationPrefix + "/control-plane"
	ProxyInjectAnnotation             = ProxyInjectIndicator
	ProxyInjectNamespaceLabel         = ProxyInjectIndicator
	ProxyInjectStatusAnnotation       = AnnotationPrefix + "/inject-status"
	MatchedProxyProfile               = AnnotationPrefix + "/proxy-profile"
	ConfigHashAnnotation              = AnnotationPrefix + "/config-hash"
	SpecHashAnnotation                = AnnotationPrefix + "/spec-hash"
	ProxyProfileLastUpdated           = AnnotationPrefix + "/last-updated"
	ProxyProfileLastUpdatedTimeFormat = "20060102-150405.0000"
	InjectorAnnotationPrefix          = "sidecar.flomesh.io"
	ProxyServiceNameAnnotation        = InjectorAnnotationPrefix + "/service-name"
	ProxyDefaultProxyProfileLabel     = InjectorAnnotationPrefix + "/is-default-proxyprofile"
	ProxyProfileLabel                 = MatchedProxyProfile
	ProxyInjectEnabled                = "true"
	ProxyInjectDisabled               = "false"
	ProxyInjectdStatus                = "injected"
	ProxySharedResourceVolumeName     = "shared-proxy-res"
	ProxySharedResoueceMountPath      = "/sidecar"
	ProxyProfileConfigMapMountPath    = "/config"
	PipyProxyConfigFileEnvName        = "PIPY_CONFIG_FILE"
	ProxyProfileConfigWorkDirEnvName  = "_SIDECAR_CONFIG_PATH_"
	CRDTypeLabel                      = AnnotationPrefix + "/crd"
	CRDVersionLabel                   = AnnotationPrefix + "/crd-version"
	ProxyParentPathEnvName            = "PROXY_PARENT_PATH"
	ProxyRepoBaseUrlEnvName           = "PROXY_REPO_BASE_URL"
	ProxyRepoApiBaseUrlEnvName        = "PROXY_REPO_API_BASE_URL"
	MatchedProxyProfileEnvName        = "MATCHED_PROXY_PROFILE"
	DefaultServicePathTpl             = "/" + ClusterTpl + "/services"
	DefaultIngressPathTpl             = "/" + ClusterTpl + "/ingress"
	DefaultNamespacedIngressPathTpl   = "/" + ClusterTpl + "/nsig/{{ .Namespace }}"
	DefaultProxyProfileParentPathTpl  = DefaultServicePathTpl
	DefaultProxyProfilePathTpl        = "/" + ClusterTpl + "/pf/{{ .ProxyProfile }}"
	DefaultSidecarPathTpl             = "/" + ClusterTpl + "/sidecars/{{ .ProxyProfile }}/{{ .Sidecar }}"
	DefaultServiceBasePath            = "/base/services"
	DefaultIngressBasePath            = "/base/ingress"

	DefaultHttpSchema = "http"

	// Cluster constants

	MultiClustersPrefix       = "cluster.flomesh.io"
	MultiClustersExported     = MultiClustersPrefix + "/export"
	MultiClustersExportedName = MultiClustersPrefix + "/export-name"
	ClusterTpl                = "{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}"

	// FLB constants

	FlbPrefix                = "flb.flomesh.io"
	FlbEnabledAnnotation     = FlbPrefix + "/enabled"
	FlbClusterAnnotation     = FlbPrefix + "/cluster"
	FlbAddressPoolAnnotation = FlbPrefix + "/address-pool"
	FlbDesiredIPAnnotation   = FlbPrefix + "/desired-ip"
)

const AppVersionTemplate = `

===========================================================
- Version: %s
- ImageVersion: %s
- GitVersion: %s
- GitCommit: %s
- BuildDate: %s
============================================================

`

var (
	ClusterIDTemplate              = template.Must(template.New("ClusterIDTemplate").Parse(ClusterTpl))
	ProxyProfileParentPathTemplate = template.Must(template.New("ProxyProfileParentPathTemplate").Parse(DefaultProxyProfileParentPathTpl))
	ProxyProfilePathTemplate       = template.Must(template.New("ProxyProfilePathTemplate").Parse(DefaultProxyProfilePathTpl))
	SidecarPathTemplate            = template.Must(template.New("SidecarPathTemplate").Parse(DefaultSidecarPathTpl))
	IngressPathTemplate            = template.Must(template.New("IngressPathTemplate").Parse(DefaultIngressPathTpl))
	NamespacedIngressPathTemplate  = template.Must(template.New("NamespacedIngressPathTemplate").Parse(DefaultNamespacedIngressPathTpl))
	ServicePathTemplate            = template.Must(template.New("ServicePathTemplate").Parse(DefaultServicePathTpl))
)
