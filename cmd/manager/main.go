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
	"github.com/flomesh-io/fsm/cmd/manager/basic"
	"github.com/flomesh-io/fsm/cmd/manager/connector"
	"github.com/flomesh-io/fsm/cmd/manager/handler"
	"github.com/flomesh-io/fsm/cmd/manager/health"
	"github.com/flomesh-io/fsm/cmd/manager/reconciler"
	mrepo "github.com/flomesh-io/fsm/cmd/manager/repo"
	"github.com/flomesh-io/fsm/cmd/manager/webhook"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/kube"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/util"
	"github.com/flomesh-io/fsm/pkg/util/tls"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"math/rand"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	flomeshscheme "github.com/flomesh-io/fsm/pkg/generated/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	gwschema "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
	//setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(flomeshscheme.AddToScheme(scheme))
	utilruntime.Must(gwschema.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type startArgs struct {
	managerConfigFile string
	namespace         string
}

func main() {
	// process CLI arguments and parse them to flags
	args := processFlags()
	options := loadManagerOptions(args.managerConfigFile)

	klog.Infof(commons.AppVersionTemplate, version.Version, version.ImageVersion, version.GitVersion, version.GitCommit, version.BuildDate)

	kubeconfig := ctrl.GetConfigOrDie()
	k8sApi := newK8sAPI(kubeconfig, args)
	if !version.IsSupportedK8sVersion(k8sApi) {
		klog.Error(fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String()))
		os.Exit(1)
	}

	// fsm configurations
	controlPlaneConfigStore := config.NewStore(k8sApi)
	mcClient := controlPlaneConfigStore.MeshConfig
	mc := mcClient.GetConfig()
	mc.Cluster.UID = getClusterUID(k8sApi)
	mc, err := mcClient.UpdateConfig(mc)
	if err != nil {
		os.Exit(1)
	}

	// generate certificate and store it in k8s secret flomesh-ca-bundle
	certMgr, err := tls.GetCertificateManager(k8sApi, mc)
	if err != nil {
		os.Exit(1)
	}

	// upload init scripts to pipy repo
	repoClient := repo.NewRepoClient(mc.RepoRootURL())
	// create a new manager for controllers
	mgr := newManager(kubeconfig, options)
	stopCh := util.RegisterOSExitHandlers()
	broker := mcsevent.NewBroker(stopCh)

	ctx := &fctx.FsmContext{
		Manager:            mgr,
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		K8sAPI:             k8sApi,
		ConfigStore:        controlPlaneConfigStore,
		CertificateManager: certMgr,
		RepoClient:         repoClient,
		Broker:             broker,
		StopCh:             stopCh,
	}

	if mc.IsIngressEnabled() {
		ctx.Connector = connector.GetLocalConnector(ctx)
	}

	if mc.IsGatewayApiEnabled() {
		if !version.IsSupportedK8sVersionForGatewayAPI(k8sApi) {
			klog.Errorf("kubernetes server version %s is not supported, requires at least %s",
				version.ServerVersion.String(), version.MinK8sVersionForGatewayAPI.String())
			os.Exit(1)
		}

		ctx.EventHandler = handler.GetResourceEventHandler(ctx)
	}

	for _, f := range []func(*fctx.FsmContext) error{
		mrepo.InitRepo,
		basic.SetupHTTP,
		basic.SetupTLS,
		//logging.SetupLogging,
		webhook.RegisterWebHooks,
		handler.RegisterEventHandlers,
		reconciler.RegisterReconcilers,
		health.AddLivenessAndReadinessCheck,
		StartManager,
	} {
		if err := f(ctx); err != nil {
			klog.Errorf("Failed to startup: %s", err)
			os.Exit(1)
		}
	}
}

func processFlags() *startArgs {
	var configFile string
	flag.StringVar(&configFile, "config", "manager_config.yaml",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctrl.SetLogger(klogr.New())

	return &startArgs{
		managerConfigFile: configFile,
		namespace:         config.GetFsmNamespace(),
	}
}

func loadManagerOptions(configFile string) ctrl.Options {
	var err error
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			klog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	return options
}

func newManager(kubeconfig *rest.Config, options ctrl.Options) manager.Manager {
	mgr, err := ctrl.NewManager(kubeconfig, options)
	if err != nil {
		klog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	return mgr
}

func newK8sAPI(kubeconfig *rest.Config, args *startArgs) *kube.K8sAPI {
	api, err := kube.NewAPIForConfig(kubeconfig, 30*time.Second)
	if err != nil {
		klog.Error(err, "unable to create k8s client")
		os.Exit(1)
	}

	return api
}

func getClusterUID(api *kube.K8sAPI) string {
	ns, err := api.Client.CoreV1().Namespaces().Get(context.TODO(), config.GetFsmNamespace(), metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get fsm namespace: %s", err)
		os.Exit(1)
	}

	return string(ns.UID)
}

func StartManager(ftx *fctx.FsmContext) error {
	if ftx.Connector != nil {
		if err := ftx.Manager.Add(manager.RunnableFunc(func(ctx context.Context) error {
			return ftx.Connector.Run(ftx.StopCh)
		})); err != nil {
			return err
		}
	}

	if ftx.EventHandler != nil {
		if err := ftx.Manager.Add(ftx.EventHandler); err != nil {
			return err
		}
	}

	klog.Info("starting manager")
	if err := ftx.Manager.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("problem running manager, %s", err)
		return err
	}

	return nil
}
