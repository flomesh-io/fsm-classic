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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// +kubebuilder:default=false

	// IsInCluster, it indicates the Cluster resource is used for connecting local cluster
	//  or a remote cluster.
	IsInCluster bool `json:"isInCluster,omitempty"`

	// +kubebuilder:default=default
	// +optional

	// Region, the locality information of this cluster
	Region string `json:"region,omitempty"`

	// +kubebuilder:default=default
	// +optional

	// Zone, the locality information of this cluster
	Zone string `json:"zone,omitempty"`

	// +kubebuilder:default=default
	// +optional

	// Group, the locality information of this cluster
	Group string `json:"group,omitempty"`

	// The ingress address of this cluster
	// +optional

	// Gateway, the address of the gateway/ingress of this cluster
	Gateway string `json:"gateway,omitempty"`

	// +optional

	// ControlPlaneRepoRootUrl, for Remote cluster mode, the linked clusters need to pull
	//  scripts/config from control panel cluster, it's the full external URL of pipy repo of
	// control panel which is accessed from outside
	ControlPlaneRepoRootUrl string `json:"controlPlaneRepoRootUrl,omitempty"`

	// +kubebuilder:default=1
	// +optional

	// Replicas, how many replicas of the cluster-connector will be running for this cluster,
	//  it's in active-standby mode
	Replicas *int32 `json:"replicas,omitempty"`

	// FIXME: temp solution, should NOT store this as plain text.
	//  consider use cli to add cluster to control plane, import kubeconfig
	//  and create a Secret with proper SA to store it as bytes

	// +optional

	// Kubeconfig, The kubeconfig of the cluster you want to connnect to
	// This's not needed if ClusterMode is InCluster, it will use InCluster
	// config
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// List of environment variables to set in the cluster-connector container.
	// Cannot be updated.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Compute Resources required by cluster-connector container.
	// Cannot be updated.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10

	// LogLevel is the log level of this ingress controller pod.
	// +optional
	LogLevel int `json:"logLevel,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Secret string `json:"secret"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Region",type="string",priority=0,JSONPath=".spec.region"
// +kubebuilder:printcolumn:name="Zone",type="string",priority=0,JSONPath=".spec.zone"
// +kubebuilder:printcolumn:name="Group",type="string",priority=0,JSONPath=".spec.group"
// +kubebuilder:printcolumn:name="Gateway",type="string",priority=0,JSONPath=".spec.gateway"
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
