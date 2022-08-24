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
	"github.com/flomesh-io/fsm/pkg/certificate"
	certificateconfig "github.com/flomesh-io/fsm/pkg/certificate/config"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/spf13/pflag"
	"github.com/tidwall/sjson"
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
	certMgr := issueCA(k8sApi, mc)

	// 2. upload init scripts to pipy repo
	repoClient := repo.NewRepoClientWithApiBaseUrl(mc.RepoApiBaseURL())
	initRepo(repoClient)
	if mc.Ingress.TLS {
		if mc.Ingress.TLSOffload && mc.Ingress.SSLPassthrough {
			klog.Errorf("Both TLSOffload and SSLPassthrough are enabled, they are mutual exclusive, please check MeshConfig.")
			os.Exit(1)
		}

		if mc.Ingress.TLSOffload {
			issueCertForIngress(repoClient, certMgr, mc)
		}

		if mc.Ingress.SSLPassthrough {
			enableSSLPassthrough(repoClient, mc)
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

func issueCA(k8sApi *kube.K8sAPI, mc *config.MeshConfig) certificate.Manager {
	certCfg := certificateconfig.NewConfig(k8sApi, certificate.CertificateManagerType(mc.Certificate.Manager))
	mgr, err := certCfg.GetCertificateManager()
	if err != nil {
		klog.Errorf("get certificate manager, %s", err.Error())
		os.Exit(1)
	}

	return mgr
}

func initRepo(repoClient *repo.PipyRepoClient) {
	// wait until pipy repo is up
	wait.PollImmediate(5*time.Second, 60*time.Second, func() (bool, error) {
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

func issueCertForIngress(repoClient *repo.PipyRepoClient, certMgr certificate.Manager, mc *config.MeshConfig) {
	// 1. issue cert
	cert, err := certMgr.IssueCertificate("ingress-pipy", commons.DefaultCAValidityPeriod, []string{})
	if err != nil {
		klog.Errorf("Issue certificate for ingress-pipy error: %s", err)
		os.Exit(1)
	}

	// 2. get main.json
	path := "/base/ingress/config/main.json"
	json, err := repoClient.GetFile(path)
	if err != nil {
		klog.Errorf("Get %q from pipy repo error: %s", path, err)
		os.Exit(1)
	}

	// 3. update CertificateChain
	newJson, err := sjson.Set(json, "certificates.cert", string(cert.CrtPEM))
	if err != nil {
		klog.Errorf("Failed to update certificates.cert: %s", err)
		os.Exit(1)
	}
	// 4. update Private Key
	newJson, err = sjson.Set(newJson, "certificates.key", string(cert.KeyPEM))
	if err != nil {
		klog.Errorf("Failed to update certificates.key: %s", err)
		os.Exit(1)
	}

	// 5. update CA
	//newJson, err = sjson.Set(newJson, "certificates.ca", string(cert.CA))
	//if err != nil {
	//	klog.Errorf("Failed to update certificates.key: %s", err)
	//	os.Exit(1)
	//}

	// 6. update main.json
	batch := repo.Batch{
		Basepath: "/base/ingress",
		Items: []repo.BatchItem{
			{
				Path:     "/config",
				Filename: "main.json",
				Content:  newJson,
			},
		},
	}
	if err := repoClient.Batch([]repo.Batch{batch}); err != nil {
		klog.Errorf("Failed to update %q: %s", path, err)
		os.Exit(1)
	}
}

func enableSSLPassthrough(repoClient *repo.PipyRepoClient, mc *config.MeshConfig) {
	// 1. get main.json
	path := "/base/ingress/config/main.json"
	json, err := repoClient.GetFile(path)
	if err != nil {
		klog.Errorf("Get %q from pipy repo error: %s", path, err)
		os.Exit(1)
	}

	// 2. update ssl passthrough config
	newJson, err := sjson.Set(json, "sslPassthrough", mc.Ingress.SSLPassthrough)
	if err != nil {
		klog.Errorf("Failed to update sslPassthrough: %s", err)
		os.Exit(1)
	}

	// 3. update main.json
	batch := repo.Batch{
		Basepath: "/base/ingress",
		Items: []repo.BatchItem{
			{
				Path:     "/config",
				Filename: "main.json",
				Content:  newJson,
			},
		},
	}
	if err := repoClient.Batch([]repo.Batch{batch}); err != nil {
		klog.Errorf("Failed to update %q: %s", path, err)
		os.Exit(1)
	}
}
