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
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/webhooks"
	"github.com/flomesh-io/fsm/pkg/webhooks/cluster"
	"github.com/flomesh-io/fsm/pkg/webhooks/cm"
	"github.com/flomesh-io/fsm/pkg/webhooks/gateway"
	"github.com/flomesh-io/fsm/pkg/webhooks/gatewayclass"
	"github.com/flomesh-io/fsm/pkg/webhooks/globaltrafficpolicy"
	"github.com/flomesh-io/fsm/pkg/webhooks/httproute"
	"github.com/flomesh-io/fsm/pkg/webhooks/ingress"
	"github.com/flomesh-io/fsm/pkg/webhooks/namespacedingress"
	"github.com/flomesh-io/fsm/pkg/webhooks/proxyprofile"
	"github.com/flomesh-io/fsm/pkg/webhooks/serviceexport"
	"github.com/flomesh-io/fsm/pkg/webhooks/serviceimport"
	"io/ioutil"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"os"
)

func (c *ManagerConfig) RegisterWebHooks() error {
	registers, err := c.webhookRegisters()

	if err != nil {
		return err
	}

	if err := c.createWebhookConfigurations(registers); err != nil {
		return err
	}

	c.registerWebhookHandlers(registers)

	return nil
}

func (c *ManagerConfig) webhookRegisters() ([]webhooks.Register, error) {
	mc := c.configStore.MeshConfig.GetConfig()

	cert, err := issueCertForWebhook(c.certificateManager, mc)
	if err != nil {
		return nil, err
	}

	cfg := c.registerConfig(mc, cert)

	return c.registers(cfg), nil
}

func (c *ManagerConfig) createWebhookConfigurations(registers []webhooks.Register) error {
	mutatingWebhooks, validatingWebhooks := c.webhooks(registers)

	// Mutating
	if mwc := flomeshadmission.NewMutatingWebhookConfiguration(mutatingWebhooks); mwc != nil {
		mutating := c.k8sAPI.Client.
			AdmissionregistrationV1().
			MutatingWebhookConfigurations()
		if _, err := mutating.Create(context.Background(), mwc, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				existingMwc, err := mutating.Get(context.Background(), mwc.Name, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("Unable to get MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
					return err
				}

				existingMwc.Webhooks = mwc.Webhooks
				_, err = mutating.Update(context.Background(), existingMwc, metav1.UpdateOptions{})
				if err != nil {
					// Should be not conflict for a leader-election manager, error is error
					klog.Errorf("Unable to update MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
					return err
				}
			} else {
				klog.Errorf("Unable to create MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
				return err
			}
		}
	}

	// Validating
	if vwc := flomeshadmission.NewValidatingWebhookConfiguration(validatingWebhooks); vwc != nil {
		validating := c.k8sAPI.Client.
			AdmissionregistrationV1().
			ValidatingWebhookConfigurations()
		if _, err := validating.Create(context.Background(), vwc, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				existingVmc, err := validating.Get(context.Background(), vwc.Name, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("Unable to get ValidatingWebhookConfigurations %q, %s", vwc.Name, err.Error())
					return err
				}

				existingVmc.Webhooks = vwc.Webhooks
				_, err = validating.Update(context.Background(), existingVmc, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf("Unable to update ValidatingWebhookConfigurations %q, %s", vwc.Name, err.Error())
					return err
				}
			} else {
				klog.Errorf("Unable to create ValidatingWebhookConfigurations %q, %s", vwc.Name, err.Error())
				return err
			}
		}
	}

	return nil
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

func (c *ManagerConfig) webhooks(registers []webhooks.Register) (mutating []admissionregv1.MutatingWebhook, validating []admissionregv1.ValidatingWebhook) {
	for _, r := range registers {
		m, v := r.GetWebhooks()

		if len(m) > 0 {
			mutating = append(mutating, m...)
		}

		if len(v) > 0 {
			validating = append(validating, v...)
		}
	}

	return mutating, validating
}

func (c *ManagerConfig) registerWebhookHandlers(registers []webhooks.Register) {
	hookServer := c.manager.GetWebhookServer()

	for _, r := range registers {
		for path, handler := range r.GetHandlers() {
			hookServer.Register(path, handler)
		}
	}
}

func (c *ManagerConfig) registers(cfg *webhooks.RegisterConfig) []webhooks.Register {
	mc := c.configStore.MeshConfig.GetConfig()
	registers := make([]webhooks.Register, 0)

	registers = append(registers, injector.NewRegister(cfg))

	registers = append(registers, cluster.NewRegister(cfg))
	registers = append(registers, cm.NewRegister(cfg))
	registers = append(registers, proxyprofile.NewRegister(cfg))
	registers = append(registers, serviceexport.NewRegister(cfg))
	registers = append(registers, serviceimport.NewRegister(cfg))
	registers = append(registers, globaltrafficpolicy.NewRegister(cfg))

	if mc.IsIngressEnabled() {
		registers = append(registers, ingress.NewRegister(cfg))
		if mc.IsNamespacedIngressEnabled() {
			registers = append(registers, namespacedingress.NewRegister(cfg))
		}
	}

	if mc.IsGatewayApiEnabled() {
		registers = append(registers, gateway.NewRegister(cfg))
		registers = append(registers, gatewayclass.NewRegister(cfg))
		registers = append(registers, httproute.NewRegister(cfg))
	}

	return registers
}

func (c *ManagerConfig) registerConfig(mc *config.MeshConfig, cert *certificate.Certificate) *webhooks.RegisterConfig {
	return &webhooks.RegisterConfig{
		Manager:            c.manager,
		ConfigStore:        c.configStore,
		K8sAPI:             c.k8sAPI,
		CertificateManager: c.certificateManager,
		RepoClient:         c.repoClient,
		Broker:             c.broker,
		WebhookSvcNs:       config.GetFsmNamespace(),
		WebhookSvcName:     mc.Webhook.ServiceName,
		CaBundle:           cert.CA,
	}
}
