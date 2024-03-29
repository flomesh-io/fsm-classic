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
	clusterv1alpha1 "github.com/flomesh-io/fsm-classic/controllers/cluster/v1alpha1"
	"github.com/flomesh-io/fsm-classic/controllers/flb"
	gatewayv1beta1 "github.com/flomesh-io/fsm-classic/controllers/gateway/v1beta1"
	nsigv1alpha1 "github.com/flomesh-io/fsm-classic/controllers/namespacedingress/v1alpha1"
	proxyprofilev1alpha1 "github.com/flomesh-io/fsm-classic/controllers/proxyprofile/v1alpha1"
	svcexpv1alpha1 "github.com/flomesh-io/fsm-classic/controllers/serviceexport/v1alpha1"
	svcimpv1alpha1 "github.com/flomesh-io/fsm-classic/controllers/serviceimport/v1alpha1"
	svclb "github.com/flomesh-io/fsm-classic/controllers/servicelb"
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/event"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func registerReconcilers(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store, certMgr certificate.Manager, broker *event.Broker) {
	registerProxyProfile(mgr, api, controlPlaneConfigStore)
	registerCluster(mgr, api, controlPlaneConfigStore, broker, certMgr)
	registerServiceExport(mgr, api, controlPlaneConfigStore, broker)
	registerServiceImport(mgr, api, controlPlaneConfigStore)

	mc := controlPlaneConfigStore.MeshConfig.GetConfig()
	if mc.GatewayApi.Enabled {
		registerGatewayAPIs(mgr, api, controlPlaneConfigStore)
	}

	if mc.Ingress.Namespaced {
		registerNamespacedIngress(mgr, api, controlPlaneConfigStore, certMgr)
	}

	if mc.ServiceLB.Enabled {
		registerServiceLB(mgr, api, controlPlaneConfigStore)
	}

	if mc.FLB.Enabled && mc.ServiceLB.Enabled {
		klog.Errorf("Both FLB and ServiceLB are enabled, they're mutual exclusive.")
		os.Exit(1)
	}

	if mc.ServiceLB.Enabled && !mc.FLB.Enabled {
		klog.V(5).Infof("ServiceLB is enabled")
		registerServiceLB(mgr, api, controlPlaneConfigStore)
	}

	if mc.FLB.Enabled && !mc.ServiceLB.Enabled {
		klog.V(5).Infof("FLB is enabled")
		registerFLB(mgr, api, controlPlaneConfigStore)
	}
}

func registerProxyProfile(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
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

func registerCluster(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store, broker *event.Broker, certMgr certificate.Manager) {
	if err := (clusterv1alpha1.New(
		mgr.GetClient(),
		api,
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("Cluster"),
		controlPlaneConfigStore,
		broker,
		certMgr,
		util.RegisterOSExitHandlers(),
	)).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
}

func registerServiceExport(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store, broker *event.Broker) {
	if err := (&svcexpv1alpha1.ServiceExportReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  api,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceExport"),
		ControlPlaneConfigStore: controlPlaneConfigStore,
		Broker:                  broker,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceExport")
		os.Exit(1)
	}
}

func registerServiceImport(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&svcimpv1alpha1.ServiceImportReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  api,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceImport"),
		ControlPlaneConfigStore: controlPlaneConfigStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceImport")
		os.Exit(1)
	}
}

func registerNamespacedIngress(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store, certMgr certificate.Manager) {
	if err := (&nsigv1alpha1.NamespacedIngressReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  api,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("NamespacedIngress"),
		ControlPlaneConfigStore: controlPlaneConfigStore,
		CertMgr:                 certMgr,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "NamespacedIngress")
		os.Exit(1)
	}
}

func registerGatewayAPIs(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	if err := (&gatewayv1beta1.GatewayReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("Gateway"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Gateway")
		os.Exit(1)
	}

	if err := (&gatewayv1beta1.GatewayClassReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("GatewayClass"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "GatewayClass")
		os.Exit(1)
	}

	if err := (&gatewayv1beta1.HTTPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("HTTPRoute"),
		K8sAPI:   api,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "HTTPRoute")
		os.Exit(1)
	}
}

func registerServiceLB(mgr manager.Manager, api *kube.K8sAPI, store *config.Store) {
	if err := (&svclb.ServiceReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceLB"),
		K8sAPI:                  api,
		ControlPlaneConfigStore: store,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceLB(Service)")
		os.Exit(1)
	}
	if err := (&svclb.NodeReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceLB"),
		K8sAPI:                  api,
		ControlPlaneConfigStore: store,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceLB(Node)")
		os.Exit(1)
	}
}

func registerFLB(mgr manager.Manager, api *kube.K8sAPI, store *config.Store) {
	if err := flb.New(
		mgr.GetClient(),
		api,
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("FLB"),
		store,
	).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "FLB")
		os.Exit(1)
	}
}
