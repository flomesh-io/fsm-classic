/*
 * MIT License
 *
 * Copyright (c) 2021-2022.  flomesh.io
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

package v1alpha1

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ProxyProfileSpec defines the desired state of ProxyProfile
type ProxyProfileSpec struct {
	// If this ProxyProfile is Disabled. A disabled ProxyProfile doesn't participate
	// the sidecar injection process.
	// +kubebuilder:default=false
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// ConfigMode tells where the sidecar loads the scripts/config from, Local means from local files mounted by configmap,
	//  Remote means loads from remote pipy repo. Default value is Remote
	// +kubebuilder:default=Remote
	// +optional
	ConfigMode ProxyConfigMode `json:"mode,omitempty"`

	// Selector is a label query over pods that should be injected
	// This's optional, please NOTE a nil or empty Selector match
	// nothing not everything.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Namespace ProxyProfile will only match the pods in the namespace
	// otherwise, match pods in all namespaces(in cluster)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Config contains the configuration data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// Values with non-UTF-8 byte sequences must use the BinaryData field.
	// This option is mutually exclusive with RepoBaseUrl option, you can only
	// have either one.
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// Url without script name from where pipy Config will be pulled.
	// This option is mutually exclusive with Config option, you can only
	// have either one
	// The valid format is protocol://IP:port/context
	// - protocol is http/https
	// - IP is either IP address or valid DNS name
	// - port is the port number, which is between 1 and 65535
	// +optional
	RepoBaseUrl string `json:"repoBaseUrl,omitempty"`

	// ParentCodebasePath,
	// if provides, the sidecar derives the Parent Codebase to have common config and
	// plugins
	// +optional
	ParentCodebasePath string `json:"parentCodebasePath,omitempty"`

	// List of environment variables to set in each of the service containers.
	// Cannot be updated.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	ServiceEnv []corev1.EnvVar `json:"serviceEnv,omitempty"`

	// RestartPolicy indicates if ProxyProfile is updated, those already injected PODs
	// should be updated or not. Default value is Never, it only has impact to new created
	// PODs, existing PODs will not be updated.
	// +kubebuilder:default=Never
	// +optional
	RestartPolicy ProxyRestartPolicy `json:"restartPolicy,omitempty"`

	// RestartScope takes effect when RestartPolicy is Always, it tells if we can restart
	// the entire POD to apply the changes or only the sidecar containers inside the POD.
	// Default value is Pod.
	// +kubebuilder:default=Pod
	// +optional
	RestartScope ProxyRestartScope `json:"restartScope,omitempty"`

	// List of sidecars, will be injected into POD.
	// +kubebuilder:validation:MinItems=1
	Sidecars []Sidecar `json:"sidecars,omitempty"`
}

type Sidecar struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string `json:"name"`

	// The file name of entrypoint script for starting the PIPY instance.
	// If not provided, the default value is the value of Name field with surfix .js.
	// For example, if the Name of the sidecar is proxy, it looks up proxy.js in config folder.
	// It only works in local config mode, if pulls scripts from remote repo, the repo server
	// returns the name of startup script.
	// +optional
	StartupScriptName string `json:"startupScriptName,omitempty"`

	// Docker image name.
	// This field is optional to allow higher level config management to default or override
	// container images in workload controllers like Deployments and StatefulSets.
	// +optional
	Image string `json:"image,omitempty"`

	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// List of environment variables to set in the sidecar container.
	// Cannot be updated.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Entrypoint array. Not executed within a shell.
	// The docker image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// +optional
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// +optional
	Args []string `json:"args,omitempty"`

	// ParentCodebasePath, same meaning as the spec.ParentCodebasePath, it overrides the parant's value
	//   for this sidecar.
	// +optional
	ParentCodebasePath string `json:"parentCodebasePath,omitempty"`

	// CodebasePath, codebase for this sidecar. Can be either an absolute path or a go templates
	// /[region]/[zone]/[group]/[cluster]/sidecars/[namespace]/[service-name]
	// /default/default/default/cluster1/sidecars/test/service1
	// +optional
	CodebasePath string `json:"codebasePath,omitempty"`
}

// ProxyProfileStatus defines the observed state of ProxyProfile
type ProxyProfileStatus struct {
	// All associated config maps, key is namespace and value is the name of configmap
	ConfigMaps map[string]string `json:"configMaps"`
}

type ProxyRestartPolicy string

const (
	ProxyRestartPolicyNever  ProxyRestartPolicy = "Never"
	ProxyRestartPolicyAlways ProxyRestartPolicy = "Always"
)

type ProxyRestartScope string

const (
	ProxyRestartScopePod     ProxyRestartScope = "Pod"
	ProxyRestartScopeSidecar ProxyRestartScope = "Sidecar"
	ProxyRestartScopeOwner   ProxyRestartScope = "Owner"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=pf,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type="string",priority=0,JSONPath=".spec.namespace"
// +kubebuilder:printcolumn:name="Disabled",type="boolean",priority=0,JSONPath=".spec.disabled"
// +kubebuilder:printcolumn:name="Selector",type="string",priority=1,JSONPath=".spec.selector"
// +kubebuilder:printcolumn:name="Config",type="string",priority=1,JSONPath=".status.configMaps"
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"

// ProxyProfile is the Schema for the proxyprofiles API
type ProxyProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProxyProfileSpec   `json:"spec,omitempty"`
	Status ProxyProfileStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProxyProfileList contains a list of ProxyProfile
type ProxyProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProxyProfile{}, &ProxyProfileList{})
}

func (pf *ProxyProfile) ConfigHash() string {
	//configBytes, err := util.GetBytes(pf.Spec.Config)
	//if err != nil {
	//	klog.Errorf("Not able convert ProxyProfile Config to bytes, ProxyProfile: %s, error: %#v", pf.Name, err)
	//	return "", err
	//}
	//
	//return util.Hash(configBytes), nil
	return util.SimpleHash(pf.Spec.Config)
}

func (pf *ProxyProfile) SpecHash() string {
	return util.SimpleHash(pf.Spec)
}

func (pf *ProxyProfile) ConstructLabelSelector() labels.Selector {
	return labels.SelectorFromSet(pf.ConstructLabels())
}

func (pf *ProxyProfile) ConstructLabels() map[string]string {
	return map[string]string{
		commons.ProxyProfileLabel: pf.Name,
		commons.CRDTypeLabel:      pf.Kind,
		commons.CRDVersionLabel:   pf.GroupVersionKind().Version,
	}
}

func (pf *ProxyProfile) GenerateConfigMapName(namespace string) string {
	return fmt.Sprintf("%s-%s-%s",
		pf.Name+commons.ConfigMapNameSuffix,
		util.HashFNV(fmt.Sprintf("%s/%s", namespace, pf.Name)),
		util.GenerateRandom(4),
	)
}

func (pf *ProxyProfile) GetConfigMode() ProxyConfigMode {
	//if pf.isRemoteMode() {
	//	return Remote
	//} else if pf.isLocalMode() {
	//	return Local
	//} else {
	//	return Unknown
	//}

	return pf.Spec.ConfigMode
}

func (pf *ProxyProfile) isRemoteMode() bool {
	//return pf.Spec.RepoBaseUrl != "" && len(pf.Spec.Config) == 0

	return pf.Spec.ConfigMode == ProxyConfigModeRemote
}

func (pf *ProxyProfile) isLocalMode() bool {
	//return len(pf.Spec.Config) > 0 && pf.Spec.RepoBaseUrl == ""

	return pf.Spec.ConfigMode == ProxyConfigModeLocal
}

//func (pf *ProxyProfile) IsEnabled() bool {
//	switch strings.ToLower(pf.Spec.Enabled) {
//	case "true", "yes", "t", "y":
//		return true
//	default:
//		return false
//	}
//}
