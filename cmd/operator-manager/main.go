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
	clusterv1alpha1 "github.com/flomesh-io/traffic-guru/controllers/cluster/v1alpha1"
	gatewayv1alpha2 "github.com/flomesh-io/traffic-guru/controllers/gateway/v1alpha2"
	proxyprofilev1alpha1 "github.com/flomesh-io/traffic-guru/controllers/proxyprofile/v1alpha1"
	flomeshadmission "github.com/flomesh-io/traffic-guru/pkg/admission"
	"github.com/flomesh-io/traffic-guru/pkg/certificate"
	certificateconfig "github.com/flomesh-io/traffic-guru/pkg/certificate/config"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/config"
	cfghandler "github.com/flomesh-io/traffic-guru/pkg/config"
	"github.com/flomesh-io/traffic-guru/pkg/injector"
	"github.com/flomesh-io/traffic-guru/pkg/kube"
	"github.com/flomesh-io/traffic-guru/pkg/version"
	"github.com/flomesh-io/traffic-guru/pkg/webhooks"
	clusterwh "github.com/flomesh-io/traffic-guru/pkg/webhooks/cluster"
	cmwh "github.com/flomesh-io/traffic-guru/pkg/webhooks/cm"
	gatewaywh "github.com/flomesh-io/traffic-guru/pkg/webhooks/gateway"
	gatewayclasswh "github.com/flomesh-io/traffic-guru/pkg/webhooks/gatewayclass"
	httproutewh "github.com/flomesh-io/traffic-guru/pkg/webhooks/httproute"
	pfwh "github.com/flomesh-io/traffic-guru/pkg/webhooks/proxyprofile"
	referencepolicywh "github.com/flomesh-io/traffic-guru/pkg/webhooks/referencepolicy"
	tcproutewh "github.com/flomesh-io/traffic-guru/pkg/webhooks/tcproute"
	tlsroutewh "github.com/flomesh-io/traffic-guru/pkg/webhooks/tlsroute"
	udproutewh "github.com/flomesh-io/traffic-guru/pkg/webhooks/udproute"
	"github.com/spf13/pflag"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type startArgs struct {
	managerConfigFile string
	repoAddr          string
	aggregatorPort    int
}

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;delete

func main() {
	// process CLI arguments and parse them to flags
	args := processFlags()
	options := loadManagerOptions(args.managerConfigFile)

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
	certMgr := getCertificateManager(k8sApi, controlPlaneConfigStore)

	// create mutating and validating webhook configurations
	createWebhookConfigurations(k8sApi, controlPlaneConfigStore, certMgr)

	// register CRDs
	registerCRDs(mgr, k8sApi, controlPlaneConfigStore)

	// register webhooks
	registerToWebhookServer(mgr, k8sApi, controlPlaneConfigStore)

	registerEventHandler(mgr, k8sApi, controlPlaneConfigStore)

	// add endpoints for Liveness and Readiness check
	addLivenessAndReadinessCheck(mgr)
	//+kubebuilder:scaffold:builder

	// start the controller manager
	startManager(mgr)
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

func newK8sAPI(kubeconfig *rest.Config) *kube.K8sAPI {
	api, err := kube.NewAPIForConfig(kubeconfig, 30*time.Second)
	if err != nil {
		klog.Error(err, "unable to create k8s client")
		os.Exit(1)
	}

	return api
}

func getCertificateManager(k8sApi *kube.K8sAPI, controlPlaneConfigStore *cfghandler.Store) certificate.Manager {
	mc := controlPlaneConfigStore.MeshConfig
	certCfg := certificateconfig.NewConfig(k8sApi, certificate.CertificateManagerType(mc.Certificate.Manager))
	certMgr, err := certCfg.GetCertificateManager()
	if err != nil {
		klog.Errorf("get certificate manager, %s", err.Error())
		os.Exit(1)
	}

	return certMgr
}

func createWebhookConfigurations(k8sApi *kube.K8sAPI, configStore *cfghandler.Store, certMgr certificate.Manager) {
	cert, err := issueCertForWebhook(certMgr)
	if err != nil {
		os.Exit(1)
	}

	ns := commons.DefaultFlomeshNamespace
	svcName := commons.DefaultWebhookServiceName
	caBundle := cert.CA
	webhooks.RegisterWebhooks(ns, svcName, caBundle)
	if configStore.MeshConfig.GatewayApiEnabled {
		webhooks.RegisterGatewayApiWebhooks(ns, svcName, caBundle)
	}

	// Mutating
	mwc := flomeshadmission.NewMutatingWebhookConfiguration()
	mutating := k8sApi.Client.
		AdmissionregistrationV1().
		MutatingWebhookConfigurations()
	if _, err = mutating.Create(context.Background(), mwc, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			existingMwc, err := mutating.Get(context.Background(), mwc.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Unable to get MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
				os.Exit(1)
			}

			existingMwc.Webhooks = mwc.Webhooks
			_, err = mutating.Update(context.Background(), existingMwc, metav1.UpdateOptions{})
			if err != nil {
				// Should be not conflict for a leader-election manager, error is error
				klog.Errorf("Unable to update MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
				os.Exit(1)
			}
		} else {
			klog.Errorf("Unable to create MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
			os.Exit(1)
		}
	}

	// Validating
	vmc := flomeshadmission.NewValidatingWebhookConfiguration()
	validating := k8sApi.Client.
		AdmissionregistrationV1().
		ValidatingWebhookConfigurations()
	if _, err = validating.Create(context.Background(), vmc, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			existingVmc, err := validating.Get(context.Background(), vmc.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Unable to get ValidatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
				os.Exit(1)
			}

			existingVmc.Webhooks = vmc.Webhooks
			_, err = validating.Update(context.Background(), existingVmc, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Unable to update ValidatingWebhookConfigurations %q, %s", vmc.Name, err.Error())
				os.Exit(1)
			}
		} else {
			klog.Errorf("Unable to create ValidatingWebhookConfigurations %q, %s", vmc.Name, err.Error())
			os.Exit(1)
		}
	}
}

func issueCertForWebhook(certMgr certificate.Manager) (*certificate.Certificate, error) {
	// TODO: refactoring it later, configurable CN and dns names
	cert, err := certMgr.IssueCertificate(
		"webhook-service",
		commons.DefaultCAValidityPeriod,
		[]string{
			"webhook-service",
			"webhook-service.flomesh.svc",
			"webhook-service.flomesh.svc.cluster.local",
		},
	)
	if err != nil {
		klog.Error("Error issuing certificate, ", err)
		return nil, err
	}

	// write ca.crt, tls.crt & tls.key to file
	if err := os.MkdirAll(commons.WebhookServerServingCertsPath, 755); err != nil {
		klog.Errorf("error creating dir %q, %s", commons.WebhookServerServingCertsPath, err.Error())
		return nil, err
	}

	certFiles := map[string][]byte{
		commons.RootCACertName:    cert.CA,
		commons.TLSCertName:       cert.CrtPEM,
		commons.TLSPrivateKeyName: cert.KeyPEM,
	}

	for file, data := range certFiles {
		fileName := fmt.Sprintf("%s/%s", commons.WebhookServerServingCertsPath, file)
		if err := ioutil.WriteFile(
			fileName,
			data,
			420); err != nil {
			klog.Errorf("error writing file %q, %s", fileName, err.Error())
			return nil, err
		}
	}

	return cert, nil
}

func registerCRDs(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	registerProxyProfileCRD(mgr, api, controlPlaneConfigStore)
	registerClusterCRD(mgr, api, controlPlaneConfigStore)

	mc := controlPlaneConfigStore.MeshConfig
	if mc.GatewayApiEnabled {
		registerGatewayAPICRDs(mgr, api, controlPlaneConfigStore)
	}
}

func registerProxyProfileCRD(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&proxyprofilev1alpha1.ProxyProfileReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ProxyProfile"),
		K8sApi:                  api,
		ControlPlaneConfigStore: controlPlaneConfigStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ProxyProfile")
		os.Exit(1)
	}
}

func registerClusterCRD(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&clusterv1alpha1.ClusterReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  api,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("Cluster"),
		ControlPlaneConfigStore: controlPlaneConfigStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
}

func registerGatewayAPICRDs(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&gatewayv1alpha2.GatewayReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("Gateway"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Gateway")
		os.Exit(1)
	}

	if err := (&gatewayv1alpha2.GatewayClassReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("GatewayClass"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "GatewayClass")
		os.Exit(1)
	}

	if err := (&gatewayv1alpha2.ReferencePolicyReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("ReferencePolicy"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ReferencePolicy")
		os.Exit(1)
	}

	if err := (&gatewayv1alpha2.HTTPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("HTTPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "HTTPRoute")
		os.Exit(1)
	}

	if err := (&gatewayv1alpha2.TLSRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("TLSRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "TLSRoute")
		os.Exit(1)
	}

	if err := (&gatewayv1alpha2.TCPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("TCPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "TCPRoute")
		os.Exit(1)
	}

	if err := (&gatewayv1alpha2.UDPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("UDPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "UDPRoute")
		os.Exit(1)
	}
}

func registerToWebhookServer(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *cfghandler.Store) {
	hookServer := mgr.GetWebhookServer()
	mc := controlPlaneConfigStore.MeshConfig

	// Proxy Injector
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

	// Cluster
	hookServer.Register(commons.ClusterMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(clusterwh.NewDefaulter(api)),
	)
	hookServer.Register(commons.ClusterValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(clusterwh.NewValidator(api)),
	)

	// ProxyProfile
	hookServer.Register(commons.ProxyProfileMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(pfwh.NewDefaulter(api)),
	)
	hookServer.Register(commons.ProxyProfileValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(pfwh.NewValidator(api)),
	)

	// core ConfigMap
	hookServer.Register(commons.ConfigMapMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(cmwh.NewDefaulter(api)),
	)
	hookServer.Register(commons.ConfigMapValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(cmwh.NewValidator(api)),
	)

	// Gateway API
	if mc.GatewayApiEnabled {
		registerGatewayApiToWebhookServer(mgr, api, controlPlaneConfigStore)
	}
}

func registerGatewayApiToWebhookServer(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *cfghandler.Store) {
	hookServer := mgr.GetWebhookServer()

	// Gateway
	hookServer.Register(commons.GatewayMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(gatewaywh.NewDefaulter(api)),
	)
	hookServer.Register(commons.GatewayValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(gatewaywh.NewValidator(api)),
	)

	// GatewayClass
	hookServer.Register(commons.GatewayClassMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(gatewayclasswh.NewDefaulter(api)),
	)
	hookServer.Register(commons.GatewayClassValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(gatewayclasswh.NewValidator(api)),
	)

	// ReferencePolicy
	hookServer.Register(commons.ReferencePolicyMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(referencepolicywh.NewDefaulter(api)),
	)
	hookServer.Register(commons.ReferencePolicyValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(referencepolicywh.NewValidator(api)),
	)

	// HTTPRoute
	hookServer.Register(commons.HTTPRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(httproutewh.NewDefaulter(api)),
	)
	hookServer.Register(commons.HTTPRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(httproutewh.NewValidator(api)),
	)

	// TCPRoute
	hookServer.Register(commons.TCPRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(tcproutewh.NewDefaulter(api)),
	)
	hookServer.Register(commons.TCPRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(tcproutewh.NewValidator(api)),
	)

	// TLSRoute
	hookServer.Register(commons.TLSRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(tlsroutewh.NewDefaulter(api)),
	)
	hookServer.Register(commons.TLSRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(tlsroutewh.NewValidator(api)),
	)

	// UDPRoute
	hookServer.Register(commons.UDPRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(udproutewh.NewDefaulter(api)),
	)
	hookServer.Register(commons.UDPRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(udproutewh.NewValidator(api)),
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

func startManager(mgr manager.Manager) {
	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("problem running manager, %s", err.Error())
		os.Exit(1)
	}
}
