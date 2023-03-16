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
    "github.com/flomesh-io/fsm/controllers"
    clusterv1alpha1 "github.com/flomesh-io/fsm/controllers/cluster/v1alpha1"
    gatewayv1beta1 "github.com/flomesh-io/fsm/controllers/gateway/v1beta1"
    nsigv1alpha1 "github.com/flomesh-io/fsm/controllers/namespacedingress/v1alpha1"
    proxyprofilev1alpha1 "github.com/flomesh-io/fsm/controllers/proxyprofile/v1alpha1"
    svcexpv1alpha1 "github.com/flomesh-io/fsm/controllers/serviceexport/v1alpha1"
    svcimpv1alpha1 "github.com/flomesh-io/fsm/controllers/serviceimport/v1alpha1"
    svclb "github.com/flomesh-io/fsm/controllers/servicelb"
    "github.com/flomesh-io/fsm/pkg/version"
    "k8s.io/klog/v2"
)

func (c *ManagerConfig) RegisterReconcilers() error {
	mc := c.configStore.MeshConfig.GetConfig()
	rc := &controllers.ReconcilerConfig{
		Manager:            c.manager,
		ConfigStore:        c.configStore,
		K8sAPI:             c.k8sAPI,
		CertificateManager: c.certificateManager,
		RepoClient:         c.repoClient,
		Broker:             c.broker,
		Scheme:             c.manager.GetScheme(),
		Client:             c.manager.GetClient(),
	}
	reconcilers := make(map[string]controllers.Reconciler)

	reconcilers["ProxyProfile"] = proxyprofilev1alpha1.NewReconciler(rc)
	reconcilers["Cluster"] = clusterv1alpha1.NewReconciler(rc)
	reconcilers["ServiceExport"] = svcexpv1alpha1.NewReconciler(rc)

    if version.IsEndpointSliceEnabled(c.k8sAPI) {
        reconcilers["ServiceImport"] = svcimpv1alpha1.NewReconciler(rc)
    }

	if mc.IsGatewayApiEnabled() {
		reconcilers["GatewayClass"] = gatewayv1beta1.NewGatewayClassReconciler(rc)
		reconcilers["Gateway"] = gatewayv1beta1.NewGatewayReconciler(rc)
		reconcilers["HTTPRoute"] = gatewayv1beta1.NewHTTPRouteReconciler(rc)
	}

	if mc.IsNamespacedIngressEnabled() {
		reconcilers["NamespacedIngress"] = nsigv1alpha1.NewReconciler(rc)
	}

	if mc.IsServiceLBEnabled() {
		reconcilers["ServiceLB(service)"] = svclb.NewServiceReconciler(rc)
		reconcilers["ServiceLB(node)"] = svclb.NewNodeReconciler(rc)
	}

	for name, r := range reconcilers {
		if err := r.SetupWithManager(c.manager); err != nil {
			klog.Errorf("Failed to setup reconciler %s: %s", name, err)
			return err
		}
	}

	return nil
}
