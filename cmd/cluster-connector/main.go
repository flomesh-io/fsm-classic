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
	"github.com/flomesh-io/fsm/pkg/cluster"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/util"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/spf13/pflag"

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
	"github.com/kelseyhightower/envconfig"
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
	connectorConfigFile string
	namespace           string
}

func main() {
	// process CLI arguments and parse them to flags
	args := processFlags()
	connectorCfg := getConnectorConfigFromEnv()
	options := loadManagerOptions(args.connectorConfigFile, connectorCfg)

	klog.Infof(commons.AppVersionTemplate, version.Version, version.ImageVersion, version.GitVersion, version.GitCommit, version.BuildDate)
	klog.Infof("Cluster: %#v", connectorCfg)

	// create a new manager for controllers
	kubeconfig := ctrl.GetConfigOrDie()
	connector := newConnector(kubeconfig, connectorCfg)
	mgr := newManager(kubeconfig, options)

	// add endpoints for Liveness and Readiness check
	addLivenessAndReadinessCheck(mgr)
	//+kubebuilder:scaffold:builder

	// start the controller manager
	startManager(mgr, connector)
}

func processFlags() *startArgs {
	var configFile string
	flag.StringVar(&configFile, "config", "connector_config.yaml",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctrl.SetLogger(klogr.New())

	return &startArgs{
		connectorConfigFile: configFile,
		namespace:           config.GetFsmNamespace(),
	}
}

func loadManagerOptions(configFile string, connectorCfg config.ConnectorConfig) ctrl.Options {
	var err error
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			klog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	clusterID := util.EvaluateTemplate(commons.ClusterIDTemplate, struct {
		Region  string
		Zone    string
		Group   string
		Cluster string
	}{
		Region:  connectorCfg.ClusterRegion,
		Zone:    connectorCfg.ClusterZone,
		Group:   connectorCfg.ClusterGroup,
		Cluster: connectorCfg.ClusterName,
	})

	options.LeaderElectionID = fmt.Sprintf("%s.flomesh.io", util.HashFNV(clusterID))

	return options
}

func getConnectorConfigFromEnv() config.ConnectorConfig {
	var cfg config.ConnectorConfig

	err := envconfig.Process("FLOMESH", &cfg)
	if err != nil {
		klog.Error(err, "unable to load the configuration from environment")
		os.Exit(1)
	}

	return cfg
}

func newConnector(kubeconfig *rest.Config, connectorConfig config.ConnectorConfig) *cluster.Connector {
	// FIXME: make it configurable
	connector, err := cluster.NewConnector(kubeconfig, connectorConfig, 15*time.Minute)

	if err != nil {
		klog.Error(err, "unable to create connector")
		os.Exit(1)
	}

	return connector
}

func newManager(kubeconfig *rest.Config, options ctrl.Options) manager.Manager {
	mgr, err := ctrl.NewManager(kubeconfig, options)
	if err != nil {
		klog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	return mgr
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

func startManager(mgr manager.Manager, connector *cluster.Connector) {
	err := mgr.Add(manager.RunnableFunc(func(context.Context) error {
		return connector.Run()
	}))
	if err != nil {
		klog.Error(err, "unable add Cluster Connector to the manager")
		os.Exit(1)
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatal(err, "problem running manager")
		os.Exit(1)
	}
}
