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

package helm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	"helm.sh/helm/v3/pkg/action"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func RenderChart(
	releaseName string,
	object metav1.Object,
	chartSource []byte,
	mc *config.MeshConfig,
	client client.Client,
	scheme *runtime.Scheme,
	resolveValues func(metav1.Object, *config.MeshConfig) (map[string]interface{}, error),
) (ctrl.Result, error) {
	installClient := helmClient(releaseName, object.GetNamespace())
	chart, err := loader.LoadArchive(bytes.NewReader(chartSource))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error loading chart for installation: %s", err)
	}
	klog.V(5).Infof("[HELM UTIL] Chart = %#v", chart)

	values, err := resolveValues(object, mc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error resolve values for installation: %s", err)
	}
	klog.V(5).Infof("[HELM UTIL] Values = %s", values)

	rel, err := installClient.Run(chart, values)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error install %s/%s: %s", object.GetNamespace(), object.GetName(), err)
	}
	klog.V(5).Infof("[HELM UTIL] Manifest = \n%s\n", rel.Manifest)

	if result, err := applyChartYAMLs(object, rel, client, scheme); err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func helmClient(releaseName, namespace string) *helm.Install {
	configFlags := &genericclioptions.ConfigFlags{Namespace: &namespace}

	klog.V(5).Infof("[HELM UTIL] Initializing Helm Action Config ...")
	actionConfig := new(action.Configuration)
	_ = actionConfig.Init(configFlags, namespace, "secret", func(format string, v ...interface{}) {})

	klog.V(5).Infof("[HELM UTIL] Creating Helm Install Client ...")
	installClient := helm.NewInstall(actionConfig)
	installClient.ReleaseName = releaseName
	installClient.Namespace = namespace
	installClient.CreateNamespace = false
	installClient.DryRun = true
	installClient.ClientOnly = true

	return installClient
}

func applyChartYAMLs(owner metav1.Object, rel *release.Release, client client.Client, scheme *runtime.Scheme) (ctrl.Result, error) {
	yamlReader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader([]byte(rel.Manifest))))
	for {
		buf, err := yamlReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				klog.Errorf("Error reading yaml: %s", err)
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
		}

		klog.V(5).Infof("[HELM UTIL] Processing YAML : \n\n%s\n\n", string(buf))
		obj, err := util.DecodeYamlToUnstructured(buf)
		if err != nil {
			klog.Errorf("Error decoding YAML to Unstructured object: %s", err)
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}
		klog.V(5).Infof("[HELM UTIL] Unstructured Object = \n\n%v\n\n", obj)

		if isValidOwner(owner, obj) {
			if err = ctrl.SetControllerReference(owner, obj, scheme); err != nil {
				klog.Errorf("Error setting controller reference: %s", err)
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
			klog.V(5).Infof("[HELM UTIL] Resource %s/%s, Owner: %#v", obj.GetNamespace(), obj.GetName(), obj.GetOwnerReferences())
		}

		result, err := util.CreateOrUpdate(context.TODO(), client, obj)
		if err != nil {
			klog.Errorf("Error creating/updating object: %s", err)
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}

		klog.V(5).Infof("[HELM UTIL] Successfully %s object: %#v", result, obj)
	}

	return ctrl.Result{}, nil
}

func isValidOwner(owner, object metav1.Object) bool {
	ownerNs := owner.GetNamespace()
	if ownerNs != "" {
		objNs := object.GetNamespace()
		if objNs == "" {
			klog.Warningf("cluster-scoped resource must not have a namespace-scoped owner, owner's namespace %s", ownerNs)
			return false
		}
		if ownerNs != objNs {
			klog.Warningf("cross-namespace owner references are disallowed, owner's namespace %s, obj's namespace %s", owner.GetNamespace(), object.GetNamespace())
			return false
		}
	}

	return true
}

func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
