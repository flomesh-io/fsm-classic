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

package reconciler

import (
	"github.com/flomesh-io/fsm/controllers"
	clusterv1alpha1 "github.com/flomesh-io/fsm/controllers/cluster/v1alpha1"
	"github.com/flomesh-io/fsm/controllers/flb"
	gatewayv1beta1 "github.com/flomesh-io/fsm/controllers/gateway/v1beta1"
	mcsv1alpha1 "github.com/flomesh-io/fsm/controllers/mcs/v1alpha1"
	nsigv1alpha1 "github.com/flomesh-io/fsm/controllers/namespacedingress/v1alpha1"
	proxyprofilev1alpha1 "github.com/flomesh-io/fsm/controllers/proxyprofile/v1alpha1"
	svclb "github.com/flomesh-io/fsm/controllers/servicelb"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/version"
	"k8s.io/klog/v2"
)

func RegisterReconcilers(ctx *fctx.FsmContext) error {
	mc := ctx.ConfigStore.MeshConfig.GetConfig()

	reconcilers := make(map[string]controllers.Reconciler)

	reconcilers["ProxyProfile"] = proxyprofilev1alpha1.NewReconciler(ctx)
	reconcilers["MCS(Cluster)"] = clusterv1alpha1.NewReconciler(ctx)
	reconcilers["MCS(ServiceExport)"] = mcsv1alpha1.NewServiceExportReconciler(ctx)

	if mc.ShouldCreateServiceAndEndpointSlicesForMCS() && version.IsEndpointSliceEnabled(ctx.K8sAPI) {
		reconcilers["MCS(ServiceImport)"] = mcsv1alpha1.NewServiceImportReconciler(ctx)
		reconcilers["MCS(Service)"] = mcsv1alpha1.NewServiceReconciler(ctx)
		reconcilers["MCS(EndpointSlice)"] = mcsv1alpha1.NewEndpointSliceReconciler(ctx)
	}

	if mc.IsGatewayApiEnabled() {
		reconcilers["GatewayAPI(GatewayClass)"] = gatewayv1beta1.NewGatewayClassReconciler(ctx)
		reconcilers["GatewayAPI(Gateway)"] = gatewayv1beta1.NewGatewayReconciler(ctx)
		reconcilers["GatewayAPI(HTTPRoute)"] = gatewayv1beta1.NewHTTPRouteReconciler(ctx)
	}

	if mc.IsNamespacedIngressEnabled() {
		reconcilers["NamespacedIngress"] = nsigv1alpha1.NewReconciler(ctx)
	}

	if mc.IsServiceLBEnabled() {
		reconcilers["ServiceLB(Service)"] = svclb.NewServiceReconciler(ctx)
		reconcilers["ServiceLB(Node)"] = svclb.NewNodeReconciler(ctx)
	}

	if mc.IsFLBEnabled() {
		reconcilers["FLB"] = flb.NewReconciler(ctx)
	}

	for name, r := range reconcilers {
		if err := r.SetupWithManager(ctx.Manager); err != nil {
			klog.Errorf("Failed to setup reconciler %s: %s", name, err)
			return err
		}
	}

	return nil
}
