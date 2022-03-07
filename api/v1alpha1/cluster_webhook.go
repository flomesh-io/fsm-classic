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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
//var clusterlog = logf.Log.WithName("cluster-resource")

func (r *Cluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-flomesh-io-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=flomesh.io,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.flomesh.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Cluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Cluster) Default() {
	klog.Info("default", "name", r.Name)
	if r.Spec.Mode == "" {
		r.Spec.Mode = InCluster
	}
	// for InCluster connector, it's name is always 'local'
	if r.Spec.Mode == InCluster {
		r.Name = "local"
		// TODO: checks if need to set r.Spec.ControlPlaneRepoRootUrl
	}

	if r.Spec.ControlPlaneRepoPath == "" {
		r.Spec.ControlPlaneRepoPath = commons.DefaultPipyRepoPath
	}

	if r.Spec.ControlPlaneRepoApiPath == "" {
		r.Spec.ControlPlaneRepoApiPath = commons.DefaultPipyRepoApiPath
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-flomesh-io-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=flomesh.io,resources=clusters,verbs=create;update,versions=v1alpha1,name=vcluster.kb.flomesh.io,admissionReviewVersions=v1

var _ webhook.Validator = &Cluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateCreate() error {
	klog.Info("validate create", "name", r.Name)

	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateUpdate(old runtime.Object) error {
	klog.Info("validate update", "name", r.Name)

	return r.validate()
}

func (r *Cluster) validate() error {
	switch r.Spec.Mode {
	case OutCluster:
		if r.Spec.Gateway == "" {
			return fmt.Errorf("gateway must be set in OutCluster mode")
		}

		if r.Spec.Kubeconfig == "" {
			return fmt.Errorf("kubeconfig must be set in OutCluster mode")
		}

		if r.Name == "local" {
			return fmt.Errorf("'local' is reserved for InCluster Mode ONLY, please change the cluster name")
		}

		if r.Spec.ControlPlaneRepoRootUrl == "" {
			return fmt.Errorf("controlPlaneRepoBaseUrl must be set in OutCluster mode")
		}
	case InCluster:
		return nil
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateDelete() error {
	klog.Info("validate delete", "name", r.Name)

	return nil
}
