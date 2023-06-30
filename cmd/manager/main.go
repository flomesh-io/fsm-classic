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
	nsigv1alpha1 "github.com/flomesh-io/fsm-classic/apis/namespacedingress/v1alpha1"
	pfv1alpha1 "github.com/flomesh-io/fsm-classic/apis/proxyprofile/v1alpha1"
	pfhelper "github.com/flomesh-io/fsm-classic/apis/proxyprofile/v1alpha1/helper"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/event"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/repo"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	"github.com/flomesh-io/fsm-classic/pkg/util/tls"
	"github.com/flomesh-io/fsm-classic/pkg/version"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"math/rand"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	flomeshscheme "github.com/flomesh-io/fsm-classic/pkg/generated/clientset/versioned/scheme"
	"github.com/go-co-op/gocron"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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
	mc.Cluster.UID = getClusterUID(k8sApi, mc)
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
	initRepo(repoClient)

	// setup HTTP
	setupHTTP(repoClient, mc)

	// setup TLS config
	setupTLS(certMgr, repoClient, mc)

	// setup Logging
	setupLogging(k8sApi, repoClient, mc)

	// create a new manager for controllers
	mgr := newManager(kubeconfig, options)

	stopCh := util.RegisterOSExitHandlers()
	broker := event.NewBroker(stopCh)

	// create mutating and validating webhook configurations
	createWebhookConfigurations(k8sApi, controlPlaneConfigStore, certMgr)

	// register webhooks
	registerToWebhookServer(mgr, k8sApi, controlPlaneConfigStore)

	// register Reconcilers
	registerReconcilers(mgr, k8sApi, controlPlaneConfigStore, certMgr, broker)

	registerEventHandler(mgr, k8sApi, controlPlaneConfigStore, certMgr, repoClient)

	// add endpoints for Liveness and Readiness check
	addLivenessAndReadinessCheck(mgr)
	//+kubebuilder:scaffold:builder

	// start the controller manager
	startManager(mgr, mc, repoClient)
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

func getClusterUID(api *kube.K8sAPI, mc *config.MeshConfig) string {
	ns, err := api.Client.CoreV1().Namespaces().Get(context.TODO(), mc.GetMeshNamespace(), metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get fsm namespace: %s", err)
		os.Exit(1)
	}

	return string(ns.UID)
}

func addLivenessAndReadinessCheck(mgr manager.Manager) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
}

func startManager(mgr manager.Manager, mc *config.MeshConfig, repoClient *repo.PipyRepoClient) {
	rebuildRepoJob := func(repoClient *repo.PipyRepoClient, client client.Client, mc *config.MeshConfig) error {
		if err := wait.PollImmediate(1*time.Second, 60*5*time.Second, func() (bool, error) {
			if repoClient.IsRepoUp() {
				klog.V(2).Info("Repo is READY!")
				return true, nil
			}

			klog.V(2).Info("Repo is not up, sleeping ...")
			return false, nil
		}); err != nil {
			klog.Errorf("Error happened while waiting for repo up, %s", err)
		}

		// initialize the repo
		if err := repoClient.Batch([]repo.Batch{ingressBatch(), servicesBatch()}); err != nil {
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

		return nil
	}

	if err := mgr.Add(manager.RunnableFunc(func(context.Context) error {
		s := gocron.NewScheduler(time.Local)
		s.SingletonModeAll()
		if _, err := s.Every(mc.Repo.RecoverIntervalInSeconds).Seconds().Do(rebuildRepoJob, repoClient, mgr.GetClient(), mc); err != nil {
			klog.Errorf("Error happened while rebuilding repo: %s", err)
		}
		s.StartAsync()

		return nil
	})); err != nil {
		klog.Errorf("unable add re-initializing repo task to the manager")
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("problem running manager, %s", err.Error())
		os.Exit(1)
	}
}
