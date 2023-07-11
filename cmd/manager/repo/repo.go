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

package main

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/repo"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	ScriptsRoot = "/repo/scripts"
)

func InitRepo(ctx *fctx.FsmContext) error {
	klog.Infof("[MGR] Initializing PIPY Repo ...")
	// wait until pipy repo is up or timeout after 5 minutes
	repoClient := ctx.RepoClient

	if err := wait.PollImmediate(5*time.Second, 60*5*time.Second, func() (bool, error) {
		if repoClient.IsRepoUp() {
			klog.V(2).Info("Repo is READY!")
			return true, nil
		}

		klog.V(2).Info("Repo is not up, sleeping ...")
		return false, nil
	}); err != nil {
		klog.Errorf("Error happened while waiting for repo up, %s", err)
		return err
	}

	mc := ctx.ConfigStore.MeshConfig.GetConfig()
	// initialize the repo
	if err := repoClient.Batch(getBatches(mc)); err != nil {
		return err
	}

	// derive codebase
	// Services
	defaultServicesPath := mc.GetDefaultServicesPath()
	if err := repoClient.DeriveCodebase(defaultServicesPath, commons.DefaultServiceBasePath); err != nil {
		return err
	}

	// Ingress
	if mc.IsIngressEnabled() {
		defaultIngressPath := mc.GetDefaultIngressPath()
		if err := repoClient.DeriveCodebase(defaultIngressPath, commons.DefaultIngressBasePath); err != nil {
			return err
		}
	}

	// GatewayAPI
	if mc.IsGatewayApiEnabled() {
		defaultGatewaysPath := mc.GetDefaultGatewaysPath()
		if err := repoClient.DeriveCodebase(defaultGatewaysPath, commons.DefaultGatewayBasePath); err != nil {
			return err
		}
	}

	return nil
}

func getBatches(mc *config.MeshConfig) []repo.Batch {
	batches := []repo.Batch{servicesBatch()}

	if mc.IsIngressEnabled() {
		batches = append(batches, ingressBatch())
	}

	if mc.IsGatewayApiEnabled() {
		batches = append(batches, gatewaysBatch())
	}

	return batches
}

func ingressBatch() repo.Batch {
	return createBatch(commons.DefaultIngressBasePath, fmt.Sprintf("%s/ingress", ScriptsRoot))
}

func servicesBatch() repo.Batch {
	return createBatch(commons.DefaultServiceBasePath, fmt.Sprintf("%s/services", ScriptsRoot))
}

func gatewaysBatch() repo.Batch {
	return createBatch(commons.DefaultGatewayBasePath, fmt.Sprintf("%s/gateways", ScriptsRoot))
}

func createBatch(repoPath, scriptsDir string) repo.Batch {
	batch := repo.Batch{
		Basepath: repoPath,
		Items:    []repo.BatchItem{},
	}

	for _, file := range listFiles(scriptsDir) {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}

		balancerItem := repo.BatchItem{
			Path:     strings.TrimPrefix(filepath.Dir(file), scriptsDir),
			Filename: filepath.Base(file),
			Content:  string(content),
		}
		batch.Items = append(batch.Items, balancerItem)
	}

	return batch
}

func listFiles(root string) (files []string) {
	err := filepath.Walk(root, visit(&files))

	if err != nil {
		panic(err)
	}

	return files
}

func visit(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			klog.Errorf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			*files = append(*files, path)
		}

		return nil
	}
}

func rebuildRepoJob(repoClient *repo.PipyRepoClient, client client.Client, mc *config.MeshConfig) error {
	klog.Infof("<<<<<< rebuilding repo - start >>>>>> ")

	if !repoClient.IsRepoUp() {
		klog.V(2).Info("Repo is not up, sleeping ...")
		return nil
	}

	// initialize the repo
	batches := make([]repo.Batch, 0)
	if !repoClient.CodebaseExists(commons.DefaultIngressBasePath) {
		batches = append(batches, ingressBatch())
	}
	if !repoClient.CodebaseExists(commons.DefaultServiceBasePath) {
		batches = append(batches, servicesBatch())
	}
	if len(batches) > 0 {
		if err := repoClient.Batch(batches); err != nil {
			klog.Errorf("Failed to write config to repo: %s", err)
			return err
		}

		defaultIngressPath := mc.GetDefaultIngressPath()
		if _, err := repoClient.DeriveCodebase(defaultIngressPath, commons.DefaultIngressBasePath); err != nil {
			klog.Errorf("%q failed to derive codebase %q: %s", defaultIngressPath, commons.DefaultIngressBasePath, err)
			return err
		}

		defaultServicesPath := mc.GetDefaultServicesPath()
		if _, err := repoClient.DeriveCodebase(defaultServicesPath, commons.DefaultServiceBasePath); err != nil {
			klog.Errorf("%q failed to derive codebase %q: %s", defaultServicesPath, commons.DefaultServiceBasePath, err)
			return err
		}

		if mc.Ingress.Enabled && mc.Ingress.Namespaced {
			nsigList := &nsigv1alpha1.NamespacedIngressList{}
			if err := client.List(context.TODO(), nsigList); err != nil {
				return err
			}

			for _, nsig := range nsigList.Items {
				ingressPath := mc.NamespacedIngressCodebasePath(nsig.Namespace)
				parentPath := mc.IngressCodebasePath()
				if _, err := repoClient.DeriveCodebase(ingressPath, parentPath); err != nil {
					klog.Errorf("Codebase of NamespaceIngress %q failed to derive codebase %q: %s", ingressPath, parentPath, err)
					return err
				}
			}
		}

		pfList := &pfv1alpha1.ProxyProfileList{}
		if err := client.List(context.TODO(), pfList); err != nil {
			return err
		}

		for _, pf := range pfList.Items {
			// ProxyProfile codebase derives service codebase
			pfPath := pfhelper.GetProxyProfilePath(pf.Name, mc)
			pfParentPath := pfhelper.GetProxyProfileParentPath(mc)
			klog.V(5).Infof("Deriving service codebase of ProxyProfile %q", pf.Name)
			if _, err := repoClient.DeriveCodebase(pfPath, pfParentPath); err != nil {
				klog.Errorf("Deriving service codebase of ProxyProfile %q error: %#v", pf.Name, err)
				return err
			}

			// sidecar codebase derives ProxyProfile codebase
			for _, sidecar := range pf.Spec.Sidecars {
				sidecarPath := pfhelper.GetSidecarPath(pf.Name, sidecar.Name, mc)
				klog.V(5).Infof("Deriving codebase of sidecar %q of ProxyProfile %q", sidecar.Name, pf.Name)
				if _, err := repoClient.DeriveCodebase(sidecarPath, pfPath); err != nil {
					klog.Errorf("Deriving codebase of sidecar %q of ProxyProfile %q error: %#v", sidecar.Name, pf.Name, err)
					return err
				}
			}
		}

		if err := utils.UpdateMainVersion(commons.DefaultIngressBasePath, repoClient, mc); err != nil {
			klog.Errorf("Failed to update version of main.json: %s", err)
			return err
		}
	}

	klog.Infof("<<<<<< rebuilding repo - end >>>>>> ")
	return nil
}
