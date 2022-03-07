/*
 * MIT License
 *
 * Copyright (c) 2022-2022.  flomesh.io
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
)

const (
	// Global constants
	DefaultFlomeshNamespace      = "flomesh"
	OperatorManagerComponentName = "operator-manager"
	OperatorConfigName           = "operator-config"
	OperatorConfigJsonName       = "operator_config.json"
	DefaultPipyImage             = "flomesh/pipy-pjs:0.4.0-334"
	DefaultPipyRepoPath          = "/repo"
	DefaultPipyRepoApiPath       = "/api/v1/repo"

	// Proxy CRD

	DeploymentNameSuffix = "-fsmdp"
	ServiceNameSuffix    = "-fsmsvc"
	ConfigMapNameSuffix  = "-fsmcm"
	DaemonSetNameSuffix  = "-fsmds"
	VolumeNameSuffix     = "-fsmvlm"

	// Sidecar constants
	DefaultProxyImage                 = DefaultPipyImage
	DefaultProxyInitImage             = "flomesh/proxy-init:latest"
	ProxyInjectorWebhookPath          = "/proxy-injector-flomesh-io-v1alpha1"
	AnnotationPrefix                  = "flomesh.io"
	ProxyInjectIndicator              = AnnotationPrefix + "/inject"
	ProxyInjectAnnotation             = ProxyInjectIndicator
	ProxyInjectNamespaceLabel         = ProxyInjectIndicator
	ProxyInjectStatusAnnotation       = AnnotationPrefix + "/inject-status"
	MatchedProxyProfileAnnotation     = AnnotationPrefix + "/proxy-profile"
	ConfigHashAnnotation              = AnnotationPrefix + "/config-hash"
	SpecHashAnnotation                = AnnotationPrefix + "/spec-hash"
	ProxySpecHashAnnotation           = AnnotationPrefix + "/proxy-hash"
	InjectorAnnotationPrefix          = "sidecar.flomesh.io"
	ProxyServiceNameAnnotation        = InjectorAnnotationPrefix + "/service-name"
	ProxyDefaultProxyProfileLabel     = InjectorAnnotationPrefix + "/is-default-proxyprofile"
	ProxyProfileLabel                 = MatchedProxyProfileAnnotation
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
	DefaultProxyParentCodebasePathTpl = "/{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/services/"
	DefaultProxyCodebasePathTpl       = "/{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/sidecars/{{ .Namespace }}/{{ .Service }}/"
	ProxyCodebasePathsEnvName         = "PROXY_CODEBASE_PATHS"
	ProxyRepoBaseUrlEnvName           = "PROXY_REPO_BASE_URL"
	ProxyRepoApiBaseUrlEnvName        = "PROXY_REPO_API_BASE_URL"

	// DefaultHttpSchema, default http schema
	DefaultHttpSchema = "http"

	// Cluster constants
	MultiClustersPrefix                   = "cluster.flomesh.io"
	MultiClustersClusterName              = MultiClustersPrefix + "/name"
	MultiClustersRegion                   = MultiClustersPrefix + "/region"
	MultiClustersZone                     = MultiClustersPrefix + "/zone"
	MultiClustersGroup                    = MultiClustersPrefix + "/group"
	MultiClustersExported                 = MultiClustersPrefix + "/export"
	MultiClustersExportedName             = MultiClustersPrefix + "/export-name"
	MultiClustersSecretType               = MultiClustersPrefix + "/kubeconfig"
	KubeConfigEnvName                     = "KUBECONFIG"
	KubeConfigKey                         = "kubeconfig"
	ReservedInClusterClusterName          = "local"
	ClusterNameEnvName                    = "FLOMESH_CLUSTER_NAME"
	ClusterRegionEnvName                  = "FLOMESH_CLUSTER_REGION"
	ClusterZoneEnvName                    = "FLOMESH_CLUSTER_ZONE"
	ClusterGroupEnvName                   = "FLOMESH_CLUSTER_GROUP"
	ClusterGatewayEnvName                 = "FLOMESH_CLUSTER_GATEWAY"
	ClusterConnectorModeEnvName           = "FLOMESH_CLUSTER_CONNECTOR_MODE"
	ClusterControlPlaneRepoRootUrlEnvName = "FLOMESH_CLUSTER_CONTROL_PLANE_REPO_ROOT_URL"
	ClusterControlPlaneRepoPathEnvName    = "FLOMESH_CLUSTER_CONTROL_PLANE_REPO_PATH"
	ClusterControlPlaneRepoApiPathEnvName = "FLOMESH_CLUSTER_CONTROL_PLANE_REPO_API_PATH"
	FlomeshRepoServiceAddressEnvName      = "FLOMESH_REPO_SERVICE_ADDRESS"
	FlomeshServiceCollectorAddressEnvName = "FLOMESH_SERVICE_COLLECTOR_ADDRESS"
	ClusterConnectorDeploymentPrefix      = "cluster-connector-"
	ClusterConnectorSecretVolumeName      = "kubeconfig"
	ClusterConnectorConfigmapVolumeName   = "connector-config"
	ClusterConnectorSecretNamePrefix      = "cluster-credentials-"
	ClusterConnectorSecretNameTpl         = ClusterConnectorSecretNamePrefix + "%s"
	DefaultClusterConnectorImage          = "flomesh/cluster-connector:latest"
	ClusterIDTpl                          = "{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}"
)

const AppVersionTemplate = `

===========================================================
- Version: %s
- ImageVersion: %s
- GitVersion: %s
- GitCommit: %s
============================================================

`

var (
	DefaultWatchedConfigMaps = sets.String{}
	ClusterIDTemplate        = template.Must(template.New("ClusterIDTemplate").Parse(ClusterIDTpl))
)

func init() {
	DefaultWatchedConfigMaps.Insert(OperatorConfigName)
}
