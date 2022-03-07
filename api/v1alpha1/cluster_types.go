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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// +kubebuilder:default=InCLuster
	Mode ClusterMode `json:"mode,omitempty"`

	// +kubebuilder:default=default
	// +optional
	Region string `json:"region,omitempty"`

	// +kubebuilder:default=default
	// +optional
	Zone string `json:"zone,omitempty"`

	// +kubebuilder:default=default
	// +optional
	Group string `json:"group,omitempty"`

	// The ingress address of this cluster
	// +optional
	Gateway string `json:"gateway,omitempty"`

	// +optional
	ControlPlaneRepoRootUrl string `json:"controlPlaneRepoRootUrl,omitempty"`

	// +optional
	// +kubebuilder:default=/repo
	ControlPlaneRepoPath string `json:"controlPlaneRepoPath,omitempty"`

	// +optional
	// +kubebuilder:default=/api/v1/repo
	ControlPlaneRepoApiPath string `json:"controlPlaneRepoApiPath,omitempty"`

	// FIXME: temp solution, should NOT store this as plain text.
	//  consider use cli to add cluster to control plane, import kubeconfig
	//  and create a Secret with proper SA to store it as bytes
	// Kubeconfig, The kubeconfig of the cluster you want to connnect to
	// This's not needed if ClusterMode is InCluster, it will use InCluster
	// config
	// +optional
	Kubeconfig string `json:"kubeconfig,omitempty"`
}

type ClusterMode string

const (
	InCluster  ClusterMode = "InCluster"
	OutCluster ClusterMode = "OutCluster"
)

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Secret string `json:"secret"`
}

// +genclient
// +genclient:nonNamespaced
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

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
