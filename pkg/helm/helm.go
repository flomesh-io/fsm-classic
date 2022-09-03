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
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/util"
	"helm.sh/helm/v3/pkg/action"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"io"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

func RenderChart(
	name string,
	object metav1.Object,
	chartSource []byte,
	mc *config.MeshConfig,
	client client.Client,
	scheme *runtime.Scheme,
	resolveValues func(metav1.Object, *config.MeshConfig) (map[string]interface{}, error),
) (ctrl.Result, error) {
	installClient := helmClient(name, object.GetNamespace())
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

func helmClient(name, namespace string) *helm.Install {
	configFlags := &genericclioptions.ConfigFlags{Namespace: &namespace}

	klog.V(5).Infof("[HELM UTIL] Initializing Helm Action Config ...")
	actionConfig := new(action.Configuration)
	_ = actionConfig.Init(configFlags, namespace, "secret", func(format string, v ...interface{}) {})

	klog.V(5).Infof("[HELM UTIL] Creating Helm Install Client ...")
	installClient := helm.NewInstall(actionConfig)
	installClient.ReleaseName = fmt.Sprintf("%s-%s", name, namespace)
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
				return ctrl.Result{RequeueAfter: 2 * time.Second}, err
			}
		}

		klog.V(5).Infof("[HELM UTIL] Processing YAML : \n\n%s\n\n", string(buf))
		obj, err := util.DecodeYamlToUnstructured(buf)
		if err != nil {
			klog.Errorf("Error decoding YAML to Unstructured object: %s", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, err
		}
		klog.V(5).Infof("[HELM UTIL] Unstructured Object = \n\n%v\n\n", obj)

		if owner.GetNamespace() == obj.GetNamespace() {
			if err = ctrl.SetControllerReference(owner, obj, scheme); err != nil {
				klog.Errorf("Error setting controller reference: %s", err)
				return ctrl.Result{RequeueAfter: 2 * time.Second}, err
			}

			klog.V(5).Infof("[HELM UTIL] Resource %s/%s, Owner: %#v", obj.GetNamespace(), obj.GetName(), obj.GetOwnerReferences())
		}

		result, err := createOrUpdateUnstructured(context.TODO(), client, obj)
		if err != nil {
			klog.Errorf("Error creating/updating object: %s", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, err
		}

		klog.V(5).Infof("[HELM UTIL] Successfully %s object: %#v", result, obj)
	}

	return ctrl.Result{}, nil
}

func createOrUpdateUnstructured(ctx context.Context, c client.Client, obj *unstructured.Unstructured) (controllerutil.OperationResult, error) {
	// a copy of new object
	modifiedObj := obj.DeepCopyObject().(client.Object)
	klog.V(5).Infof("Modified: %#v", modifiedObj)

	key := client.ObjectKeyFromObject(obj)
	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("Get Object %s err: %s", key, err)
			return controllerutil.OperationResultNone, err
		}
		klog.V(5).Infof("Creating Object %s ...", key)
		if err := c.Create(ctx, obj); err != nil {
			klog.Errorf("Create Object %s err: %s", key, err)
			return controllerutil.OperationResultNone, err
		}

		klog.V(5).Infof("Object %s is created successfully.", key)
		return controllerutil.OperationResultCreated, nil
	}
	klog.V(5).Infof("Found Object %s: %#v", key, obj)

	result := controllerutil.OperationResultNone
	if !reflect.DeepEqual(obj, modifiedObj) {
		klog.V(5).Infof("Patching Object %s ...", key)
		patchData, err := client.Apply.Data(modifiedObj)
		if err != nil {
			klog.Errorf("Create ApplyPatch err: %s", err)
			return controllerutil.OperationResultNone, err
		}

		// Only issue a Patch if the before and after resources differ
		objPatch := client.RawPatch(types.ApplyPatchType, patchData)
		opts := &client.PatchOptions{FieldManager: "fsm", Force: pointer.Bool(true)}
		if err := c.Patch(ctx, obj, objPatch, opts); err != nil {
			klog.Errorf("Patch Object %s err: %s", key, err)
			return result, err
		}
		result = controllerutil.OperationResultUpdated
	}

	klog.V(5).Infof("Object %s is %s successfully.", key, result)
	return result, nil
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
