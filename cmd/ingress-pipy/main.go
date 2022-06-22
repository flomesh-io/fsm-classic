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
	"context"
	"flag"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

const (
	HealthPath = "/healthz"
	ReadyPath  = "/readyz"
)

type startArgs struct {
	namespace string
}

type ingress struct {
	k8sApi    *kube.K8sAPI
	namespace string
}

func main() {
	// process CLI arguments and parse them to flags
	args := processFlags()

	klog.Infof(commons.AppVersionTemplate, version.Version, version.ImageVersion, version.GitVersion, version.GitCommit, version.BuildDate)

	kubeconfig := ctrl.GetConfigOrDie()
	k8sApi := newK8sAPI(kubeconfig, args)
	if !version.IsSupportedK8sVersion(k8sApi) {
		klog.Error(fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String()))
		os.Exit(1)
	}

	ing := &ingress{
		namespace: args.namespace,
		k8sApi:    k8sApi,
	}

	configStore := config.NewStore(k8sApi)
	mc := configStore.MeshConfig.GetConfig()
	ingressRepoUrl := fmt.Sprintf("%s%s", mc.RepoBaseURL(), mc.IngressCodebasePath())
	klog.Infof("Ingress Repo = %q", ingressRepoUrl)

	cpuLimits, err := ing.getIngressCpuLimitsQuota()
	if err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
	klog.Infof("CPU Limits = %#v", cpuLimits)

	spawn := int64(1)
	if cpuLimits.Value() > 0 {
		spawn = cpuLimits.Value()
	}
	klog.Infof("PIPY SPAWN = %d", spawn)

	// start pipy
	for i := int64(0); i < spawn; i++ {
		klog.Infof("starting pipy(index=%d) ...", i)
		startPipy(ingressRepoUrl, true)
	}

	startHealthAndReadyProbeServer()
}

func startPipy(ingressRepoUrl string, background bool) {
	args := []string{"--reuse-port", ingressRepoUrl}
	if background {
		args = append(args, "&")
	}

	cmd := exec.Command("pipy", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
}

func processFlags() *startArgs {
	var namespace string
	flag.StringVar(&namespace, "fsm-namespace", commons.DefaultFsmNamespace,
		"The namespace of FSM.")

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctrl.SetLogger(klogr.New())
	config.SetFsmNamespace(namespace)

	return &startArgs{
		namespace: namespace,
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

func startHealthAndReadyProbeServer() {
	router := gin.Default()
	router.GET(HealthPath, health)
	router.GET(ReadyPath, health)
	router.Run(":8081")
}

func health(c *gin.Context) {
	// TODO: check pipy and returns status accordingly
	c.String(http.StatusOK, "OK")
}

func (i *ingress) getIngressCpuLimitsQuota() (*resource.Quantity, error) {
	podName := os.Getenv("INGRESS_POD_NAME")
	if podName == "" {
		return nil, errors.New("INGRESS_POD_NAME env variable cannot be empty")
	}

	pod, err := i.k8sApi.Client.CoreV1().Pods(i.namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error retrieving ingress-pipy pod %s", podName)
		return nil, err
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == "ingress" {
			return c.Resources.Limits.Cpu(), nil
		}
	}

	return nil, errors.Errorf("No container named 'ingress' in POD %q", podName)
}
