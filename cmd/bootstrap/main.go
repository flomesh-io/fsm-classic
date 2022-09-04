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
	"flag"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/aggregator"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/util/tls"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/spf13/pflag"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"math/rand"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

const (
	ScriptsRoot = "/repo/scripts"
	HealthPath  = "/healthz"
	ReadyPath   = "/readyz"
)

type startArgs struct {
	namespace string
}

func main() {
	args := processFlags()

	klog.Infof(commons.AppVersionTemplate, version.Version, version.ImageVersion, version.GitVersion, version.GitCommit, version.BuildDate)

	kubeconfig := ctrl.GetConfigOrDie()
	k8sApi := newK8sAPI(kubeconfig, args)
	if !version.IsSupportedK8sVersion(k8sApi) {
		klog.Error(fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String()))
		os.Exit(1)
	}

	configStore := config.NewStore(k8sApi)
	mc := configStore.MeshConfig.GetConfig()

	// 1. generate certificate and store it in k8s secret flomesh-ca-bundle
	certMgr, err := tls.GetCertificateManager(k8sApi, mc)
	if err != nil {
		os.Exit(1)
	}

	// 2. upload init scripts to pipy repo
	repoClient := repo.NewRepoClientWithApiBaseUrl(mc.RepoApiBaseURL())
	initRepo(repoClient)
	klog.V(5).Infof("mc.Ingress.TLS=%#v", mc.Ingress.TLS)
	if mc.Ingress.TLS.Enabled {
		if mc.Ingress.TLS.SSLPassthrough.Enabled {
			// SSL Passthrough
			err = tls.UpdateSSLPassthrough(
				commons.DefaultIngressBasePath,
				repoClient,
				mc.Ingress.TLS.SSLPassthrough.Enabled,
				mc.Ingress.TLS.SSLPassthrough.UpstreamPort,
			)
			if err != nil {
				os.Exit(1)
			}
		} else {
			// TLS Offload
			err = tls.IssueCertForIngress(commons.DefaultIngressBasePath, repoClient, certMgr, mc)
			if err != nil {
				os.Exit(1)
			}
		}
	}

	// 3. start aggregator
	startAggregator(mc)

	// 4. health check
	//go startHealthAndReadyProbeServer()
}

func processFlags() *startArgs {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctrl.SetLogger(klogr.New())

	return &startArgs{
		namespace: config.GetFsmNamespace(),
	}
}

func newK8sAPI(kubeconfig *rest.Config, args *startArgs) *kube.K8sAPI {
	api, err := kube.NewAPIForConfig(kubeconfig, 30*time.Second)
	if err != nil {
		klog.Error(err, "unable to create k8s client")
		os.Exit(1)
	}

	return api
}

func initRepo(repoClient *repo.PipyRepoClient) {
	// wait until pipy repo is up or timeout after 5 minutes
	wait.PollImmediate(5*time.Second, 60*5*time.Second, func() (bool, error) {
		if repoClient.IsRepoUp() {
			klog.V(2).Info("Repo is READY!")
			return true, nil
		}

		klog.V(2).Info("Repo is not up, sleeping ...")
		return false, nil
	})

	// initialize the repo
	if err := repoClient.Batch([]repo.Batch{ingressBatch(), servicesBatch()}); err != nil {
		os.Exit(1)
	}
}

func ingressBatch() repo.Batch {
	return createBatch("/base/ingress", fmt.Sprintf("%s/ingress", ScriptsRoot))
}

func servicesBatch() repo.Batch {
	return createBatch("/base/services", fmt.Sprintf("%s/services", ScriptsRoot))
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

func startAggregator(mc *config.MeshConfig) {
	aggregatorAddr := fmt.Sprintf(":%s", mc.AggregatorPort())

	aggregator.NewAggregator(aggregatorAddr, mc.RepoAddr()).Run()
}
