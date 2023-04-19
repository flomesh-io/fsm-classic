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
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
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
	fsmNamespace string
}

type gateway struct {
	k8sApi *kube.K8sAPI
	mc     *config.MeshConfig
}

func (gw *gateway) codebase() string {
	return fmt.Sprintf("%s%s/", gw.mc.RepoBaseURL(), gw.mc.GatewayCodebasePath(config.GetFsmPodNamespace()))
}

func (gw *gateway) calcPipySpawn() int64 {
	panic("implement it")
}

func main() {
	// process CLI arguments and parse them to flags
	args := processFlags()

	klog.Infof(commons.AppVersionTemplate, version.Version, version.ImageVersion, version.GitVersion, version.GitCommit, version.BuildDate)

	kubeconfig := ctrl.GetConfigOrDie()
	k8sApi := newK8sAPI(kubeconfig, args)
	if !version.IsSupportedK8sVersionForGatewayAPI(k8sApi) {
		klog.Error(fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersionForGatewayAPI.String()))
		os.Exit(1)
	}

	configStore := config.NewStore(k8sApi)
	mc := configStore.MeshConfig.GetConfig()

	if !mc.IsGatewayApiEnabled() {
		klog.Errorf("GatewayAPI is not enabled, FSM doesn't support Ingress and GatewayAPI are both enabled.")
		os.Exit(1)
	}

	if !version.IsSupportedK8sVersionForGatewayAPI(k8sApi) {
		klog.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersionForGatewayAPI.String())
		os.Exit(1)
	}

	gw := &gateway{k8sApi: k8sApi, mc: mc}

	// codebase URL
	url := gw.codebase()
	klog.Infof("Gateway Repo = %q", url)

	// calculate pipy spawn
	spawn := gw.calcPipySpawn()
	klog.Infof("PIPY SPAWN = %d", spawn)

	startPipy(spawn, url)

	startHealthAndReadyProbeServer()
}

func processFlags() *startArgs {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctrl.SetLogger(klogr.New())

	return &startArgs{
		fsmNamespace: config.GetFsmNamespace(),
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

func startPipy(spawn int64, url string) {
	args := []string{url}
	if spawn > 1 {
		args = append([]string{"--reuse-port", fmt.Sprintf("--threads=%d", spawn)}, args...)
	}

	cmd := exec.Command("pipy", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	klog.Infof("cmd = %v", cmd)

	if err := cmd.Start(); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
}

func startHealthAndReadyProbeServer() {
	router := gin.Default()
	router.GET(HealthPath, health)
	router.GET(ReadyPath, health)
	if err := router.Run(":8081"); err != nil {
		klog.Errorf("Failed to start probe server: %s", err)
		os.Exit(1)
	}
}

func health(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}