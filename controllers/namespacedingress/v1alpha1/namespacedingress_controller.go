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
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	nsigv1alpha1 "github.com/flomesh-io/fsm/apis/namespacedingress/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/mitchellh/mapstructure"
	pkgerr "github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	"io"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	//go:embed chart.tgz
	chartSource []byte

	//go:embed values.yaml
	valuesSource []byte

	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
)

// NamespacedIngressReconciler reconciles a NamespacedIngress object
type NamespacedIngressReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
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

	configFlags := &genericclioptions.ConfigFlags{Namespace: &nsig.Namespace}
	installClient := r.helmClient(nsig, configFlags)
	chart, err := loader.LoadArchive(bytes.NewReader(chartSource))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error loading chart for installation: %s", err)
	}
	klog.V(5).Infof("[NSIG] Chart = %#v", chart)

	values, err := r.resolveValues(nsig, mc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error resolve values for installation: %s", err)
	}
	klog.V(5).Infof("[NSIG] Values = %s", values)

	rel, err := installClient.Run(chart, values)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error install NamespacedIngress %s/%s: %s", nsig.Namespace, nsig.Name, err)
	}
	klog.V(5).Infof("[NSIG] Manifest = \n%s\n", rel.Manifest)

	yamlReader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader([]byte(rel.Manifest))))
	for {
		buf, err := yamlReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				klog.Errorf("Error reading yaml: %s", err)
				return ctrl.Result{RequeueAfter: 2 * time.Second}, err
			}
		}

		klog.V(5).Infof("[NSIG] Processing YAML : \n\n%s\n\n", string(buf))
		obj, err := r.decodeYamlToUnstructured(buf)
		if err != nil {
			klog.Errorf("Error decoding YAML to Unstructured object: %s", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, err
		}

		if nsig.Namespace == obj.GetNamespace() {
			if err = ctrl.SetControllerReference(nsig, obj, r.Scheme); err != nil {
				klog.Errorf("Error setting controller reference: %s", err)
				return ctrl.Result{RequeueAfter: 2 * time.Second}, err
			}

			klog.V(5).Infof("[NSIG] Resource %s/%s, Owner: %#v", obj.GetNamespace(), obj.GetName(), obj.GetOwnerReferences())
		}

		result, err := ctrl.CreateOrUpdate(context.TODO(), r.Client, obj, func() error { return nil })
		if err != nil {
			klog.Errorf("Error creating/updating object: %s", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, err
		}

		klog.V(5).Infof("[NSIG] Successfully %s object: %#v", result, obj)
	}

	return ctrl.Result{}, nil
}

func (r *NamespacedIngressReconciler) helmClient(nsig *nsigv1alpha1.NamespacedIngress, configFlags *genericclioptions.ConfigFlags) *helm.Install {
	klog.V(5).Infof("[NSIG] Initializing Helm Action Config ...")
	actionConfig := new(action.Configuration)
	_ = actionConfig.Init(configFlags, nsig.Namespace, "secret", func(format string, v ...interface{}) {})

	klog.V(5).Infof("[NSIG] Creating Helm Install Client ...")
	installClient := helm.NewInstall(actionConfig)
	installClient.ReleaseName = fmt.Sprintf("ingress-pipy-%s", nsig.Namespace)
	installClient.Namespace = nsig.Namespace
	installClient.CreateNamespace = false
	installClient.DryRun = true
	installClient.ClientOnly = true

	return installClient
}

func (r *NamespacedIngressReconciler) resolveValues(nsig *nsigv1alpha1.NamespacedIngress, mc *config.MeshConfig) (map[string]interface{}, error) {
	klog.V(5).Infof("[NSIG] Resolving Values ...")
	rawValues, err := chartutil.ReadValues(valuesSource)
	if err != nil {
		return nil, err
	}

	finalValues := rawValues.AsMap()
	overrides := []string{
		"fsm.ingress.namespaced=true",
		fmt.Sprintf("fsm.image.repository=%s", mc.Images.Repository),
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	var nsigMap map[string]interface{}
	err = mapstructure.Decode(nsig, &nsigMap)
	if err != nil {
		return nil, fmt.Errorf("convert NamespacedIngress to map, err = %#v", err)
	}

	finalValues = mergeMaps(finalValues, nsigMap)

	return finalValues, nil
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func (r *NamespacedIngressReconciler) decodeYamlToUnstructured(data []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	_, _, err := decUnstructured.Decode(data, nil, obj)
	if err != nil {
		return nil, pkgerr.Wrap(err, "Decode YAML to Unstructured failed. ")
	}

	return obj, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespacedIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nsigv1alpha1.NamespacedIngress{}).
		Owns(&corev1.Service{}).
		Owns(&appv1.Deployment{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
