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
	"bytes"
	"context"
	_ "embed"
	"encoding/gob"
	"fmt"
	ingdpv1alpha1 "github.com/flomesh-io/fsm/apis/ingressdeployment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"helm.sh/helm/v3/pkg/action"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:embed chart.tgz
var chartSource []byte

//go:embed values.yaml
var valuesSource []byte

// IngressDeploymentReconciler reconciles a IngressDeployment object
type IngressDeploymentReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the IngressDeployment closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IngressDeployment object against the actual IngressDeployment state, and then
// perform operations to make the IngressDeployment state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IngressDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()

	klog.Infof("Ingress Enabled = %t, Namespaced Ingress = %t", mc.Ingress.Enabled, mc.Ingress.Namespaced)
	if !mc.Ingress.Enabled || !mc.Ingress.Namespaced {
		klog.Warning("Ingress is not enabled or Ingress mode is not Namespace, ignore processing IngressDeployment...")
		return ctrl.Result{}, nil
	}

	igdp := &ingdpv1alpha1.IngressDeployment{}
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name, Namespace: req.Namespace},
		igdp,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("IngressDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get IngressDeployment, %#v", err)
		return ctrl.Result{}, err
	}

	installClient := r.helmClient(igdp)
	chart, err := loader.LoadArchive(bytes.NewReader(chartSource))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error loading chart for installation: %s", err)
	}
	klog.V(5).Infof("[IGDP] Chart = %#v", chart)

	values, err := r.resolveValues(igdp)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error resolve values for installation: %s", err)
	}
	klog.V(5).Infof("[IGDP] Values = %#v", values)

	rel, err := installClient.Run(chart, values)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error install IngressDeployment %s/%s: %s", igdp.Namespace, igdp.Name, err)
	}
	klog.V(5).Infof("[IGDP] Manifest = \n%s\n", rel.Manifest)
	klog.V(5).Infof("[IGDP] RELEASE = \n%#v\n", rel)

	return ctrl.Result{}, nil
}

func (r *IngressDeploymentReconciler) helmClient(igdp *ingdpv1alpha1.IngressDeployment) *helm.Install {
	klog.V(5).Infof("[IGDP] Initializing Helm Action Config ...")
	actionConfig := new(action.Configuration)
	_ = actionConfig.Init(&genericclioptions.ConfigFlags{
		Namespace: &igdp.Namespace,
	}, igdp.Namespace, "secret", func(format string, v ...interface{}) {})

	klog.V(5).Infof("[IGDP] Creating Helm Install Client ...")
	installClient := helm.NewInstall(actionConfig)
	installClient.ReleaseName = fmt.Sprintf("ingress-pipy-%s", igdp.Namespace)
	installClient.Namespace = igdp.Namespace
	installClient.CreateNamespace = false
	installClient.DryRun = true
	installClient.ClientOnly = true

	return installClient
}

func (r *IngressDeploymentReconciler) resolveValues(igdp *ingdpv1alpha1.IngressDeployment) (map[string]interface{}, error) {
	klog.V(5).Infof("[IGDP] Resolving Values ...")
	rawValues, err := chartutil.ReadValues(valuesSource)
	if err != nil {
		return nil, err
	}

	finalValues := rawValues.AsMap()

	if err := strvals.ParseInto("fsm.ingress.namespaced=true", finalValues); err != nil {
		return nil, err
	}

	finalValues["ingressNs"] = igdp.Namespace
	finalValues["ingressDeploymentName"] = igdp.Name
	finalValues["ingressServiceType"] = igdp.Spec.ServiceType

	ports := convertPorts(igdp.Spec.Ports)
	if ports != nil {
		finalValues["ingressPorts"] = ports
	} else {
		return nil, fmt.Errorf("empty ports slice")
	}

	return finalValues, nil
}

func convertPorts(ports []ingdpv1alpha1.ServicePort) []interface{} {
	var result []interface{}

	for _, p := range ports {
		m, err := structToMap(p)
		if err != nil {
			return nil
		}
		result = append(result, m)
	}

	return result
}

func structToMap(obj interface{}) (map[string]interface{}, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(obj)
	if err != nil {
		klog.Errorf("Cannot convert obj to bytes, %#v", err)
		return nil, err
	}

	m := make(map[string]interface{})
	err = yaml.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		klog.Errorf("Cannot unmarshal obj to map, %#v", err)
		return nil, err
	}

	return m, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingdpv1alpha1.IngressDeployment{}).
		Owns(&corev1.Service{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
