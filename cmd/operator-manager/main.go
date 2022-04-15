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
	clusterv1alpha1 "github.com/flomesh-io/traffic-guru/apis/cluster/v1alpha1"
	"github.com/flomesh-io/traffic-guru/controllers/cluster"
	"github.com/flomesh-io/traffic-guru/controllers/gateway"
	"github.com/flomesh-io/traffic-guru/controllers/proxyprofile"
	"github.com/flomesh-io/traffic-guru/pkg/aggregator"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	cfghandler "github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/injector"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	"github.com/flomesh-io/traffic-guru/pkg/version"
	"github.com/flomesh-io/traffic-guru/pkg/webhooks"
	cmwh "github.com/flomesh-io/traffic-guru/pkg/webhooks/cm"
	pfwh "github.com/flomesh-io/traffic-guru/pkg/webhooks/proxyprofile"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"math/rand"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	flomeshscheme "github.com/flomesh-io/traffic-guru/pkg/generated/clientset/versioned/scheme"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	gwv1alpha2schema "sigs.k8s.io/gateway-api/pkg/client/clientset/gateway/versioned/scheme"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
	//setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(flomeshscheme.AddToScheme(scheme))
	utilruntime.Must(gwv1alpha2schema.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	// process CLI arguments and parse them to flags
	managerConfigFile, repoAddr, aggregatorPort := processFlags()
	options := loadManagerOptions(managerConfigFile)

	klog.Infof(commons.AppVersionTemplate, version.Version, version.ImageVersion, version.GitVersion, version.GitCommit, version.BuildDate)

	// create a new manager for controllers
	kubeconfig := ctrl.GetConfigOrDie()
	k8sApi := newK8sAPI(kubeconfig)
	if !version.IsSupportedK8sVersion(k8sApi) {
		klog.Error(fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String()))
		os.Exit(1)
	}

	controlPlaneConfigStore := config.NewStore(k8sApi)
	mgr := newManager(kubeconfig, options)

	// register CRDs
	registerCRDs(mgr, k8sApi, controlPlaneConfigStore)

	// register webhooks
	registerWebhooks(mgr, k8sApi, controlPlaneConfigStore)

	registerEventHandler(mgr, k8sApi, controlPlaneConfigStore)

	// add endpoints for Liveness and Readiness check
	addLivenessAndReadinessCheck(mgr)
	//+kubebuilder:scaffold:builder

	// start the controller manager
	startManager(mgr, repoAddr, aggregatorPort)
}

func processFlags() (string, string, int) {
	var configFile, repoAddr string
	var aggregatorPort int
	flag.StringVar(&configFile, "config", "manager_config.yaml",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.IntVar(&aggregatorPort, "aggregator-port", 6767,
		"The listening port of service aggregator.")
	flag.StringVar(&repoAddr, "repo-addr", "repo-service.flomesh.svc:6060",
		"The address of repo service.")

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctrl.SetLogger(klogr.New())

	return configFile, repoAddr, aggregatorPort
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

func getManagerEnvConfig() config.ManagerEnvironmentConfiguration {
	var cfg config.ManagerEnvironmentConfiguration

	err := envconfig.Process("FLOMESH", &cfg)
	if err != nil {
		klog.Error(err, "unable to load the configuration from environment")
		os.Exit(1)
	}

	return cfg
}

func newManager(kubeconfig *rest.Config, options ctrl.Options) manager.Manager {
	mgr, err := ctrl.NewManager(kubeconfig, options)
	if err != nil {
		klog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	return mgr
}

func newK8sAPI(kubeconfig *rest.Config) *kube.K8sAPI {
	api, err := kube.NewAPIForConfig(kubeconfig, 30*time.Second)
	if err != nil {
		klog.Error(err, "unable to create k8s client")
		os.Exit(1)
	}

	return api
}

func registerCRDs(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	registerProxyProfileCRD(mgr, api, controlPlaneConfigStore)
	registerClusterCRD(mgr, api, controlPlaneConfigStore)

	if controlPlaneConfigStore.MeshConfig.GatewayApiEnabled {
		registerGatewayAPI(mgr, api, controlPlaneConfigStore)
	}
}

func registerProxyProfileCRD(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&proxyprofile.ProxyProfileReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ProxyProfile"),
		K8sApi:                  api,
		ControlPlaneConfigStore: controlPlaneConfigStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ProxyProfile")
		os.Exit(1)
	}
	//if err := (&v1alpha1.ProxyProfile{}).SetupWebhookWithManager(mgr); err != nil {
	//	klog.Fatal(err, "unable to create webhook", "webhook", "ProxyProfile")
	//	os.Exit(1)
	//}
}

func registerClusterCRD(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&cluster.ClusterReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  api,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("Cluster"),
		ControlPlaneConfigStore: controlPlaneConfigStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	if err := (&clusterv1alpha1.Cluster{}).SetupWebhookWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create webhook", "webhook", "Cluster")
		os.Exit(1)
	}
}

func registerGatewayAPI(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&gateway.GatewayReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("Gateway"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Gateway")
		os.Exit(1)
	}

	if err := (&gateway.GatewayClassReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("GatewayClass"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "GatewayClass")
		os.Exit(1)
	}

	if err := (&gateway.ReferencePolicyReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("ReferencePolicy"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ReferencePolicy")
		os.Exit(1)
	}

	if err := (&gateway.HTTPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("HTTPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "HTTPRoute")
		os.Exit(1)
	}

	if err := (&gateway.TLSRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("TLSRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "TLSRoute")
		os.Exit(1)
	}

	if err := (&gateway.TCPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("TCPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "TCPRoute")
		os.Exit(1)
	}

	if err := (&gateway.UDPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("UDPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "UDPRoute")
		os.Exit(1)
	}
}

func registerWebhooks(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *cfghandler.Store) {
	hookServer := mgr.GetWebhookServer()
	mc := controlPlaneConfigStore.MeshConfig
	klog.Infof("Parameters: proxy-image=%s, proxy-init-image=%s", mc.DefaultPipyImage, mc.ProxyInitImage)
	hookServer.Register(commons.ProxyInjectorWebhookPath,
		&webhook.Admission{
			Handler: &injector.ProxyInjector{
				Client:         mgr.GetClient(),
				ProxyImage:     mc.DefaultPipyImage,
				ProxyInitImage: mc.ProxyInitImage,
				Recorder:       mgr.GetEventRecorderFor("ProxyInjector"),
				ConfigStore:    controlPlaneConfigStore,
				K8sAPI:         api,
			},
		},
	)
	hookServer.Register("/mutate-flomesh-io-v1alpha1-proxyprofile",
		webhooks.DefaultingWebhookFor(pfwh.NewProxyProfileDefaulter(api)),
	)
	hookServer.Register("/validate-flomesh-io-v1alpha1-proxyprofile",
		webhooks.ValidatingWebhookFor(pfwh.NewProxyProfileValidator(api)),
	)
	hookServer.Register("/mutate-core-v1-configmap",
		webhooks.DefaultingWebhookFor(cmwh.NewConfigMapDefaulter(api)),
	)
	hookServer.Register("/validate-core-v1-configmap",
		webhooks.ValidatingWebhookFor(cmwh.NewConfigMapValidator(api)),
	)
}

func registerEventHandler(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *cfghandler.Store) {

	// FIXME: make it configurable
	resyncPeriod := 15 * time.Minute

	configmapInformer, err := mgr.GetCache().GetInformer(context.TODO(), &corev1.ConfigMap{})

	if err != nil {
		klog.Error(err, "unable to get informer for ConfigMap")
		os.Exit(1)
	}

	cfghandler.RegisterConfigurationHanlder(
		cfghandler.NewFlomeshConfigurationHandler(
			mgr.GetClient(),
			api,
			controlPlaneConfigStore,
		),
		configmapInformer,
		resyncPeriod,
	)
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

func startManager(mgr manager.Manager, repoAddr string, aggregatorPort int) {
	err := mgr.Add(manager.RunnableFunc(func(context.Context) error {
		return aggregator.NewAggregator(
			fmt.Sprintf(":%d", aggregatorPort),
			repoAddr,
		).Run()
	}))
	if err != nil {
		klog.Error(err, "unable add aggregator server to the manager")
		os.Exit(1)
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatal(err, "problem running manager")
		os.Exit(1)
	}
}
