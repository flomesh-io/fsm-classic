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

package cluster

import (
	"fmt"
	clusterv1alpha1 "github.com/flomesh-io/traffic-guru/apis/cluster/v1alpha1"
	flomeshadmission "github.com/flomesh-io/traffic-guru/pkg/admission"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/klog/v2"
)

const (
	kind      = "Cluster"
	groups    = "flomesh.io"
	resources = "clusters"
	versions  = "v1alpha1"

	mwPath = commons.ClusterMutatingWebhookPath
	mwName = "mcluster.kb.flomesh.io"
	vwPath = commons.ClusterValidatingWebhookPath
	vwName = "vcluster.kb.flomesh.io"
)

func RegisterWebhooks(webhookSvcNs, webhookSvcName string, caBundle []byte) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{groups},
		[]string{versions},
		[]string{resources},
	)

	mutatingWebhook := flomeshadmission.NewMutatingWebhook(
		mwName,
		webhookSvcNs,
		webhookSvcName,
		mwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	validatingWebhook := flomeshadmission.NewValidatingWebhook(
		vwName,
		webhookSvcNs,
		webhookSvcName,
		vwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type ClusterDefaulter struct {
	k8sAPI *kube.K8sAPI
}

func NewDefaulter(k8sAPI *kube.K8sAPI) *ClusterDefaulter {
	return &ClusterDefaulter{
		k8sAPI: k8sAPI,
	}
}

func (w *ClusterDefaulter) Kind() string {
	return kind
}

func (w *ClusterDefaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", c.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", c.Spec)

	meshConfig := config.GetMeshConfig(w.k8sAPI)

	if meshConfig == nil {
		return
	}

	if c.Spec.Mode == "" {
		c.Spec.Mode = clusterv1alpha1.InCluster
	}
	// for InCluster connector, it's name is always 'local'
	if c.Spec.Mode == clusterv1alpha1.InCluster {
		c.Name = "local"
		// TODO: checks if need to set r.Spec.ControlPlaneRepoRootUrl
	}

	if c.Spec.Replicas == nil {
		c.Spec.Replicas = defaultReplicas()
	}

	klog.V(4).Infof("After setting default values, spec=%#v", c.Spec)
}

func defaultReplicas() *int32 {
	var r int32 = 1
	return &r
}

type ClusterValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ClusterValidator) Kind() string {
	return kind
}

func (w *ClusterValidator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *ClusterValidator) ValidateUpdate(oldObj, obj interface{}) error {
	return doValidation(obj)
}

func (w *ClusterValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI) *ClusterValidator {
	return &ClusterValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	c, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return nil
	}

	switch c.Spec.Mode {
	case clusterv1alpha1.OutCluster:
		if c.Spec.Gateway == "" {
			return fmt.Errorf("gateway must be set in OutCluster mode")
		}

		if c.Spec.Kubeconfig == "" {
			return fmt.Errorf("kubeconfig must be set in OutCluster mode")
		}

		if c.Name == "local" {
			return fmt.Errorf("'local' is reserved for InCluster Mode ONLY, please change the cluster name")
		}

		if c.Spec.ControlPlaneRepoRootUrl == "" {
			return fmt.Errorf("controlPlaneRepoBaseUrl must be set in OutCluster mode")
		}
	case clusterv1alpha1.InCluster:
		return nil
	}

	return nil
}
