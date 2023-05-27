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
	"fmt"
	flomeshadmission "github.com/flomesh-io/fsm-classic/pkg/admission"
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/injector"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm-classic/pkg/webhooks"
	clusterwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/cluster"
	cmwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/cm"
	flbsecretwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/flb/secret"
	flbsvcwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/flb/service"
	gatewaywh "github.com/flomesh-io/fsm-classic/pkg/webhooks/gateway"
	gatewayclasswh "github.com/flomesh-io/fsm-classic/pkg/webhooks/gatewayclass"
	gtpwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/globaltrafficpolicy"
	httproutewh "github.com/flomesh-io/fsm-classic/pkg/webhooks/httproute"
	ingwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/ingress"
	idwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/namespacedingress"
	pfwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/proxyprofile"
	referencepolicywh "github.com/flomesh-io/fsm-classic/pkg/webhooks/referencepolicy"
	svcexpwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/serviceexport"
	svcimpwh "github.com/flomesh-io/fsm-classic/pkg/webhooks/serviceimport"
	tcproutewh "github.com/flomesh-io/fsm-classic/pkg/webhooks/tcproute"
	tlsroutewh "github.com/flomesh-io/fsm-classic/pkg/webhooks/tlsroute"
	udproutewh "github.com/flomesh-io/fsm-classic/pkg/webhooks/udproute"
	"io/ioutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func createWebhookConfigurations(k8sApi *kube.K8sAPI, configStore *config.Store, certMgr certificate.Manager) {
	mc := configStore.MeshConfig.GetConfig()
	cert, err := issueCertForWebhook(certMgr, mc)
	if err != nil {
		os.Exit(1)
	}

	ns := config.GetFsmNamespace()
	svcName := mc.Webhook.ServiceName
	caBundle := cert.CA
	webhooks.RegisterWebhooks(ns, svcName, caBundle)
	if mc.GatewayApi.Enabled {
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

func issueCertForWebhook(certMgr certificate.Manager, mc *config.MeshConfig) (*certificate.Certificate, error) {
	// TODO: refactoring it later, configurable CN and dns names
	cert, err := certMgr.IssueCertificate(
		mc.Webhook.ServiceName,
		commons.DefaultCAValidityPeriod,
		[]string{
			mc.Webhook.ServiceName,
			fmt.Sprintf("%s.%s.svc", mc.Webhook.ServiceName, config.GetFsmNamespace()),
			fmt.Sprintf("%s.%s.svc.cluster.local", mc.Webhook.ServiceName, config.GetFsmNamespace()),
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

func registerToWebhookServer(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	hookServer := mgr.GetWebhookServer()
	mc := controlPlaneConfigStore.MeshConfig.GetConfig()

	// Proxy Injector
	klog.Infof("Parameters: proxy-image=%s, proxy-init-image=%s", mc.PipyImage(), mc.ProxyInitImage())
	hookServer.Register(commons.ProxyInjectorWebhookPath,
		&webhook.Admission{
			Handler: &injector.ProxyInjector{
				Client:      mgr.GetClient(),
				Recorder:    mgr.GetEventRecorderFor("ProxyInjector"),
				ConfigStore: controlPlaneConfigStore,
				K8sAPI:      api,
			},
		},
	)

	// Cluster
	hookServer.Register(commons.ClusterMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(clusterwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.ClusterValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(clusterwh.NewValidator(api)),
	)

	// ProxyProfile
	hookServer.Register(commons.ProxyProfileMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(pfwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.ProxyProfileValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(pfwh.NewValidator(api)),
	)

	// NamespacedIngress
	hookServer.Register(commons.NamespacedIngressMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(idwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.NamespacedIngressValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(idwh.NewValidator(api)),
	)

	// ServiceExport
	hookServer.Register(commons.ServiceExportMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(svcexpwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.ServiceExportValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(svcexpwh.NewValidator(api)),
	)

	// ServiceImport
	hookServer.Register(commons.ServiceImportMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(svcimpwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.ServiceImportValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(svcimpwh.NewValidator(api)),
	)

	// GlobalTrafficPolicy
	hookServer.Register(commons.GlobalTrafficPolicyMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(gtpwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.GlobalTrafficPolicyValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(gtpwh.NewValidator(api)),
	)

	// core ConfigMap
	hookServer.Register(commons.ConfigMapMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(cmwh.NewDefaulter(api)),
	)
	hookServer.Register(commons.ConfigMapValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(cmwh.NewValidator(api)),
	)

	// networking v1 Ingress
	hookServer.Register(commons.IngressMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(ingwh.NewDefaulter(api)),
	)
	hookServer.Register(commons.IngressValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(ingwh.NewValidator(api)),
	)

	// FLB Service
	hookServer.Register(commons.FLBServiceMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(flbsvcwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.FLBServiceValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(flbsvcwh.NewValidator(api, controlPlaneConfigStore)),
	)

	// FLB Secret
	hookServer.Register(commons.FLBSecretMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(flbsecretwh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.FLBSecretValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(flbsecretwh.NewValidator(api, controlPlaneConfigStore)),
	)

	// Gateway API
	if mc.GatewayApi.Enabled {
		registerGatewayApiToWebhookServer(mgr, api, controlPlaneConfigStore)
	}
}

func registerGatewayApiToWebhookServer(mgr manager.Manager, api *kube.K8sAPI, controlPlaneConfigStore *config.Store) {
	hookServer := mgr.GetWebhookServer()

	// Gateway
	hookServer.Register(commons.GatewayMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(gatewaywh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.GatewayValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(gatewaywh.NewValidator(api)),
	)

	// GatewayClass
	hookServer.Register(commons.GatewayClassMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(gatewayclasswh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.GatewayClassValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(gatewayclasswh.NewValidator(api)),
	)

	// ReferencePolicy
	hookServer.Register(commons.ReferencePolicyMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(referencepolicywh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.ReferencePolicyValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(referencepolicywh.NewValidator(api)),
	)

	// HTTPRoute
	hookServer.Register(commons.HTTPRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(httproutewh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.HTTPRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(httproutewh.NewValidator(api)),
	)

	// TCPRoute
	hookServer.Register(commons.TCPRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(tcproutewh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.TCPRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(tcproutewh.NewValidator(api)),
	)

	// TLSRoute
	hookServer.Register(commons.TLSRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(tlsroutewh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.TLSRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(tlsroutewh.NewValidator(api)),
	)

	// UDPRoute
	hookServer.Register(commons.UDPRouteMutatingWebhookPath,
		webhooks.DefaultingWebhookFor(udproutewh.NewDefaulter(api, controlPlaneConfigStore)),
	)
	hookServer.Register(commons.UDPRouteValidatingWebhookPath,
		webhooks.ValidatingWebhookFor(udproutewh.NewValidator(api)),
	)
}
