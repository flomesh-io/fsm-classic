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
	"k8s.io/apimachinery/pkg/util/sets"
	"text/template"
	"time"
)

const (
	// Global constants

	DefaultFlomeshNamespace       = "flomesh"
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
	OperatorManagerComponentName  = "operator-manager"
	MeshConfigName                = "mesh-config"
	MeshConfigJsonName            = "mesh_config.json"
	DefaultPipyImage              = "flomesh/pipy:0.30.0-100"
	DefaultPipyRepoPath           = "/repo"
	DefaultPipyRepoApiPath        = "/api/v1/repo"

	// Proxy CRD

	DeploymentNameSuffix = "-fsmdp"
	ServiceNameSuffix    = "-fsmsvc"
	ConfigMapNameSuffix  = "-fsmcm"
	DaemonSetNameSuffix  = "-fsmds"
	VolumeNameSuffix     = "-fsmvlm"

	// Webhooks

	DefaultWebhookServiceName                 = "webhook-service"
	DefaultMutatingWebhookConfigurationName   = "flomesh-mutating-webhook-configuration"
	DefaultValidatingWebhookConfigurationName = "flomesh-validating-webhook-configuration"
	ProxyInjectorWebhookPath                  = "/proxy-injector-flomesh-io-v1alpha1"
	ProxyProfileMutatingWebhookPath           = "/mutate-flomesh-io-v1alpha1-proxyprofile"
	ProxyProfileValidatingWebhookPath         = "/validate-flomesh-io-v1alpha1-proxyprofile"
	ConfigMapMutatingWebhookPath              = "/mutate-core-v1-configmap"
	ConfigMapValidatingWebhookPath            = "/validate-core-v1-configmap"
	ClusterMutatingWebhookPath                = "/mutate-flomesh-io-v1alpha1-cluster"
	ClusterValidatingWebhookPath              = "/validate-flomesh-io-v1alpha1-cluster"
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

	DefaultProxyImage                 = DefaultPipyImage
	DefaultProxyInitImage             = "flomesh/proxy-init:latest"
	AnnotationPrefix                  = "flomesh.io"
	ProxyInjectIndicator              = AnnotationPrefix + "/inject"
	FlomeshControlPlaneLabel          = AnnotationPrefix + "/control-plane"
	ProxyInjectAnnotation             = ProxyInjectIndicator
	ProxyInjectNamespaceLabel         = ProxyInjectIndicator
	ProxyInjectStatusAnnotation       = AnnotationPrefix + "/inject-status"
	MatchedProxyProfileLabel          = AnnotationPrefix + "/proxy-profile"
	ConfigHashAnnotation              = AnnotationPrefix + "/config-hash"
	SpecHashAnnotation                = AnnotationPrefix + "/spec-hash"
	ProxySpecHashAnnotation           = AnnotationPrefix + "/proxy-hash"
	ProxyProfileLastUpdatedAnnotation = AnnotationPrefix + "/last-updated"
	InjectorAnnotationPrefix          = "sidecar.flomesh.io"
	ProxyServiceNameAnnotation        = InjectorAnnotationPrefix + "/service-name"
	ProxyDefaultProxyProfileLabel     = InjectorAnnotationPrefix + "/is-default-proxyprofile"
	ProxyProfileLabel                 = MatchedProxyProfileLabel
	ProxyInjectEnabled                = "true"
	ProxyInjectDisabled               = "false"
	ProxyInjectdStatus                = "injected"
	ProxySharedResourceVolumeName     = "shared-proxy-res"
	ProxySharedResoueceMountPath      = "/sidecar"
	ProxyProfileConfigMapMountPath    = "/config"
	ProxyConfigWorkDir                = "/etc/pipy/proxy"
	PipyProxyConfigFileEnvName        = "PIPY_CONFIG_FILE"
	PipyProxyPortEnvName              = "_PIPY_LISTEN_PORT_"
	ProxyProfileConfigWorkDirEnvName  = "_SIDECAR_CONFIG_PATH_"
	DefaultProxyStartupScriptName     = "config.js"
	ProxyCRDLabel                     = AnnotationPrefix + "/proxy"
	ProxyCRDAnnotation                = ProxyCRDLabel
	ProxyModeLabel                    = AnnotationPrefix + "/proxy-mode"
	CRDTypeLabel                      = AnnotationPrefix + "/crd"
	CRDVersionLabel                   = AnnotationPrefix + "/crd-version"
	ProxyParentPathEnvName            = "PROXY_PARENT_PATH"
	ProxyPathsEnvName                 = "PROXY_PATHS"
	ProxyRepoBaseUrlEnvName           = "PROXY_REPO_BASE_URL"
	ProxyRepoApiBaseUrlEnvName        = "PROXY_REPO_API_BASE_URL"
	MatchedProxyProfileEnvName        = "MATCHED_PROXY_PROFILE"
	DefaultServicePathTpl             = "/" + ClusterTpl + "/services"
	DefaultIngressPathTpl             = "/" + ClusterTpl + "/ingress"
	DefaultProxyProfileParentPathTpl  = DefaultServicePathTpl
	DefaultProxyProfilePathTpl        = "/" + ClusterTpl + "/pf/{{ .ProxyProfile }}"
	DefaultSidecarPathTpl             = "/" + ClusterTpl + "/sidecars/{{ .ProxyProfile }}/{{ .Sidecar }}"
	DefaultServiceBasePath            = "/base/services"
	DefaultIngressBasePath            = "/base/ingress"

	// DefaultHttpSchema, default http schema
	DefaultHttpSchema = "http"

	// Cluster constants

	MultiClustersPrefix                    = "cluster.flomesh.io"
	MultiClustersClusterName               = MultiClustersPrefix + "/name"
	MultiClustersRegion                    = MultiClustersPrefix + "/region"
	MultiClustersZone                      = MultiClustersPrefix + "/zone"
	MultiClustersGroup                     = MultiClustersPrefix + "/group"
	MultiClustersExported                  = MultiClustersPrefix + "/export"
	MultiClustersExportedName              = MultiClustersPrefix + "/export-name"
	MultiClustersSecretType                = MultiClustersPrefix + "/kubeconfig"
	KubeConfigEnvName                      = "KUBECONFIG"
	KubeConfigKey                          = "kubeconfig"
	ReservedInClusterClusterName           = "local"
	ClusterNameEnvName                     = "FLOMESH_CLUSTER_NAME"
	ClusterRegionEnvName                   = "FLOMESH_CLUSTER_REGION"
	ClusterZoneEnvName                     = "FLOMESH_CLUSTER_ZONE"
	ClusterGroupEnvName                    = "FLOMESH_CLUSTER_GROUP"
	ClusterGatewayEnvName                  = "FLOMESH_CLUSTER_GATEWAY"
	ClusterConnectorModeEnvName            = "FLOMESH_CLUSTER_CONNECTOR_MODE"
	ClusterControlPlaneRepoRootUrlEnvName  = "FLOMESH_CLUSTER_CONTROL_PLANE_REPO_ROOT_URL"
	ClusterControlPlaneRepoPathEnvName     = "FLOMESH_CLUSTER_CONTROL_PLANE_REPO_PATH"
	ClusterControlPlaneRepoApiPathEnvName  = "FLOMESH_CLUSTER_CONTROL_PLANE_REPO_API_PATH"
	FlomeshRepoServiceAddressEnvName       = "FLOMESH_REPO_SERVICE_ADDRESS"
	FlomeshServiceAggregatorAddressEnvName = "FLOMESH_SERVICE_AGGREGATOR_ADDRESS"
	ClusterConnectorDeploymentPrefix       = "cluster-connector-"
	ClusterConnectorSecretVolumeName       = "kubeconfig"
	ClusterConnectorConfigmapVolumeName    = "connector-config"
	ClusterConnectorSecretNamePrefix       = "cluster-credentials-"
	ClusterConnectorSecretNameTpl          = ClusterConnectorSecretNamePrefix + "%s"
	DefaultClusterConnectorImage           = "flomesh/cluster-connector:latest"
	ClusterTpl                             = "{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}"
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
	DefaultWatchedConfigMaps       = sets.String{}
	ClusterIDTemplate              = template.Must(template.New("ClusterIDTemplate").Parse(ClusterTpl))
	ProxyProfileParentPathTemplate = template.Must(template.New("ProxyProfileParentPathTemplate").Parse(DefaultProxyProfileParentPathTpl))
	ProxyProfilePathTemplate       = template.Must(template.New("ProxyProfilePathTemplate").Parse(DefaultProxyProfilePathTpl))
	SidecarPathTemplate            = template.Must(template.New("SidecarPathTemplate").Parse(DefaultSidecarPathTpl))
	IngressPathTemplate            = template.Must(template.New("IngressPathTemplate").Parse(DefaultIngressPathTpl))
	ServicePathTemplate            = template.Must(template.New("ServicePathTemplate").Parse(DefaultServicePathTpl))
)

func init() {
	DefaultWatchedConfigMaps.Insert(MeshConfigName)
}
