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
	clusterv1alpha1 "github.com/flomesh-io/fsm/controllers/cluster/v1alpha1"
	gatewayv1beta1 "github.com/flomesh-io/fsm/controllers/gateway/v1beta1"
	nsigv1alpha1 "github.com/flomesh-io/fsm/controllers/namespacedingress/v1alpha1"
	proxyprofilev1alpha1 "github.com/flomesh-io/fsm/controllers/proxyprofile/v1alpha1"
	svcexpv1alpha1 "github.com/flomesh-io/fsm/controllers/serviceexport/v1alpha1"
	svcimpv1alpha1 "github.com/flomesh-io/fsm/controllers/serviceimport/v1alpha1"
	svclb "github.com/flomesh-io/fsm/controllers/servicelb"
	"github.com/flomesh-io/fsm/pkg/util"
	"k8s.io/klog/v2"
	"os"
)

func (c *ManagerConfig) RegisterReconcilers() {
	c.registerProxyProfile()
	c.registerCluster()
	c.registerServiceExport()
	c.registerServiceImport()

	mc := c.configStore.MeshConfig.GetConfig()
	if mc.IsGatewayApiEnabled() {
		c.registerGatewayAPIs()
	}

	if mc.IsNamespacedIngressEnabled() {
		c.registerNamespacedIngress()
	}

	if mc.ServiceLB.Enabled {
		c.registerServiceLB()
	}
}

func (c *ManagerConfig) registerProxyProfile() {
	mgr := c.manager
	if err := (&proxyprofilev1alpha1.ProxyProfileReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ProxyProfile"),
		K8sApi:                  c.k8sAPI,
		ControlPlaneConfigStore: c.configStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ProxyProfile")
		os.Exit(1)
	}
}

func (c *ManagerConfig) registerCluster() {
	mgr := c.manager
	if err := (clusterv1alpha1.New(
		mgr.GetClient(),
		c.k8sAPI,
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("Cluster"),
		c.configStore,
		c.broker,
		c.certificateManager,
		util.RegisterOSExitHandlers(),
	)).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
}

func (c *ManagerConfig) registerServiceExport() {
	mgr := c.manager
	if err := (&svcexpv1alpha1.ServiceExportReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  c.k8sAPI,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceExport"),
		ControlPlaneConfigStore: c.configStore,
		Broker:                  c.broker,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceExport")
		os.Exit(1)
	}
}

func (c *ManagerConfig) registerServiceImport() {
	mgr := c.manager
	if err := (&svcimpv1alpha1.ServiceImportReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  c.k8sAPI,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceImport"),
		ControlPlaneConfigStore: c.configStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceImport")
		os.Exit(1)
	}
}

func (c *ManagerConfig) registerNamespacedIngress() {
	mgr := c.manager
	if err := (&nsigv1alpha1.NamespacedIngressReconciler{
		Client:                  mgr.GetClient(),
		K8sAPI:                  c.k8sAPI,
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("NamespacedIngress"),
		ControlPlaneConfigStore: c.configStore,
		CertMgr:                 c.certificateManager,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "NamespacedIngress")
		os.Exit(1)
	}
}

func (c *ManagerConfig) registerGatewayAPIs() {
	mgr := c.manager
	if err := (&gatewayv1beta1.GatewayReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("Gateway"),
		K8sAPI:   c.k8sAPI,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "Gateway")
		os.Exit(1)
	}

	if err := (&gatewayv1beta1.GatewayClassReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("GatewayClass"),
		K8sAPI:   c.k8sAPI,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "GatewayClass")
		os.Exit(1)
	}

	if err := (&gatewayv1beta1.HTTPRouteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("HTTPRoute"),
		K8sAPI:   c.k8sAPI,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "HTTPRoute")
		os.Exit(1)
	}
}

func (c *ManagerConfig) registerServiceLB() {
	mgr := c.manager
	if err := (&svclb.ServiceReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceLB"),
		K8sAPI:                  c.k8sAPI,
		ControlPlaneConfigStore: c.configStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceLB(Service)")
		os.Exit(1)
	}
	if err := (&svclb.NodeReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Recorder:                mgr.GetEventRecorderFor("ServiceLB"),
		K8sAPI:                  c.k8sAPI,
		ControlPlaneConfigStore: c.configStore,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatal(err, "unable to create controller", "controller", "ServiceLB(Node)")
		os.Exit(1)
	}
}
