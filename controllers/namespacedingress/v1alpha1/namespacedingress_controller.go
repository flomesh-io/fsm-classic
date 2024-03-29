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
	"context"
	_ "embed"
	"fmt"
	nsigv1alpha1 "github.com/flomesh-io/fsm-classic/apis/namespacedingress/v1alpha1"
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/config/utils"
	"github.com/flomesh-io/fsm-classic/pkg/helm"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/repo"
	ghodssyaml "github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	//go:embed chart.tgz
	chartSource []byte
)

// NamespacedIngressReconciler reconciles a NamespacedIngress object
type NamespacedIngressReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
	CertMgr                 certificate.Manager
}

type namespacedIngressValues struct {
	NamespacedIngress *nsigv1alpha1.NamespacedIngress `json:"nsig,omitempty"`
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the NamespacedIngress closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespacedIngress object against the actual NamespacedIngress state, and then
// perform operations to make the NamespacedIngress state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *NamespacedIngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()

	klog.Infof("[NSIG] Ingress Enabled = %t, Namespaced Ingress = %t", mc.Ingress.Enabled, mc.Ingress.Namespaced)
	if !mc.Ingress.Enabled || !mc.Ingress.Namespaced {
		klog.Warning("Ingress is not enabled or Ingress mode is not Namespace, ignore processing NamespacedIngress...")
		return ctrl.Result{}, nil
	}

	nsig := &nsigv1alpha1.NamespacedIngress{}
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name, Namespace: req.Namespace},
		nsig,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("[NSIG] NamespacedIngress resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get NamespacedIngress, %#v", err)
		return ctrl.Result{}, err
	}

	ctrlResult, err := r.deriveCodebases(nsig, mc)
	if err != nil {
		return ctrlResult, err
	}

	ctrlResult, err = r.updateConfig(nsig, mc)
	if err != nil {
		return ctrlResult, err
	}

	releaseName := fmt.Sprintf("namespaced-ingress-%s", nsig.Namespace)
	if ctrlResult, err = helm.RenderChart(releaseName, nsig, chartSource, mc, r.Client, r.Scheme, resolveValues); err != nil {
		return ctrlResult, err
	}

	return ctrl.Result{}, nil
}

func resolveValues(object metav1.Object, mc *config.MeshConfig) (map[string]interface{}, error) {
	nsig, ok := object.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of nsigv1alpha1.NamespacedIngress", object)
	}

	klog.V(5).Infof("[NSIG] Resolving Values ...")

	nsigBytes, err := ghodssyaml.Marshal(&namespacedIngressValues{NamespacedIngress: nsig})
	if err != nil {
		return nil, fmt.Errorf("convert NamespacedIngress to yaml, err = %#v", err)
	}
	klog.V(5).Infof("\n\nNSIG VALUES YAML:\n\n\n%s\n\n", string(nsigBytes))
	nsigValues, err := chartutil.ReadValues(nsigBytes)
	if err != nil {
		return nil, err
	}

	finalValues := nsigValues.AsMap()

	overrides := []string{
		"fsm.ingress.namespaced=true",
		fmt.Sprintf("fsm.image.repository=%s", mc.Images.Repository),
		fmt.Sprintf("fsm.namespace=%s", mc.GetMeshNamespace()),
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func (r *NamespacedIngressReconciler) deriveCodebases(nsig *nsigv1alpha1.NamespacedIngress, mc *config.MeshConfig) (ctrl.Result, error) {
	repoClient := repo.NewRepoClient(mc.RepoRootURL())

	ingressPath := mc.NamespacedIngressCodebasePath(nsig.Namespace)
	parentPath := mc.IngressCodebasePath()
	if _, err := repoClient.DeriveCodebase(ingressPath, parentPath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespacedIngressReconciler) updateConfig(nsig *nsigv1alpha1.NamespacedIngress, mc *config.MeshConfig) (ctrl.Result, error) {
	if mc.Ingress.Namespaced && nsig.Spec.TLS.Enabled {
		repoClient := repo.NewRepoClient(mc.RepoRootURL())
		basepath := mc.NamespacedIngressCodebasePath(nsig.Namespace)

		if nsig.Spec.TLS.SSLPassthrough.Enabled {
			// SSL passthrough
			err := utils.UpdateSSLPassthrough(
				basepath,
				repoClient,
				nsig.Spec.TLS.SSLPassthrough.Enabled,
				*nsig.Spec.TLS.SSLPassthrough.UpstreamPort,
			)
			if err != nil {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
		} else {
			// TLS offload
			err := utils.IssueCertForIngress(basepath, repoClient, r.CertMgr, mc)
			if err != nil {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespacedIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nsigv1alpha1.NamespacedIngress{}).
		//Owns(&corev1.Service{}).
		//Owns(&appv1.Deployment{}).
		//Owns(&corev1.ServiceAccount{}).
		//Owns(&rbacv1.Role{}).
		//Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
