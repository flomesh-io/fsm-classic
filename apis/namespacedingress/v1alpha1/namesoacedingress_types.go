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

// NamespacedIngressSpec defines the desired state of NamespacedIngress
type NamespacedIngressSpec struct {
	// ServiceType determines how the Ingress is exposed. For an Ingress
	//   the most used types are NodePort, and LoadBalancer
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// ServiceAnnotations, those annotations are applied to Ingress Service
	// +optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=5

	// The list of ports that are exposed by this ingress service.
	Ports []ServicePort `json:"ports,omitempty"`

	// List of environment variables to set in the ingress container.
	// Cannot be updated.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Compute Resources required by Ingress container.
	// Cannot be updated.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:default=fsm-namespaced-ingress

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// ServicePort contains information on service's port.
type ServicePort struct {
	// The name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. When considering
	// the endpoints for a Service, this must match the 'name' field in the
	// EndpointPort.
	// Optional if only one ServicePort is defined on this service.
	// +optional
	Name string `json:"name,omitempty"`

	// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	// +default="TCP"
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// The application protocol for this port.
	// This field follows standard Kubernetes label syntax.
	// +optional
	AppProtocol *string `json:"appProtocol,omitempty"`

	// The port that will be exposed by this service.
	Port int32 `json:"port"`

	// The port on each node on which this service is exposed when type is
	// NodePort or LoadBalancer.  Usually assigned by the system. If a value is
	// specified, in-range, and not in use it will be used, otherwise the
	// operation will fail.  If not specified, a port will be allocated if this
	// Service requires one.  If this field is specified when creating a
	// Service which does not need it, creation will fail. This field will be
	// wiped when updating a Service to no longer need it (e.g. changing type
	// from NodePort to ClusterIP).
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`
}

// NamespacedIngressStatus defines the observed state of NamespacedIngress
type NamespacedIngressStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=nsig,scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"

// NamespacedIngress is the Schema for the NamespacedIngresss API
type NamespacedIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespacedIngressSpec   `json:"spec,omitempty"`
	Status NamespacedIngressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedIngressList contains a list of NamespacedIngress
type NamespacedIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespacedIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespacedIngress{}, &NamespacedIngressList{})
}
