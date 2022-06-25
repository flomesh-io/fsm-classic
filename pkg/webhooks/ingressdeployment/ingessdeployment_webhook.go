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

package ingressdeployment

import (
	"context"
	ingdpv1alpha1 "github.com/flomesh-io/fsm/apis/ingressdeployment/v1alpha1"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/pkg/errors"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	kind      = "IngressDeployment"
	groups    = "flomesh.io"
	resources = "ingressdeployments"
	versions  = "v1alpha1"

	mwPath = commons.IngressDeploymentMutatingWebhookPath
	mwName = "mingressdeployment.kb.flomesh.io"
	vwPath = commons.IngressDeploymentValidatingWebhookPath
	vwName = "vingressdeployment.kb.flomesh.io"
)

func RegisterWebhooks(webhookSvcNs, webhookSvcName string, caBundle []byte) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{groups},
		[]string{versions},
		[]string{resources},
	)

	mutatingWebhook := flomeshadmission.NewMutatingWebhook(
		mwName,
		webhookSvcNs,
		webhookSvcName,
		mwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	validatingWebhook := flomeshadmission.NewValidatingWebhook(
		vwName,
		webhookSvcNs,
		webhookSvcName,
		vwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type IngressDeploymentDefaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func NewDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *IngressDeploymentDefaulter {
	return &IngressDeploymentDefaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *IngressDeploymentDefaulter) Kind() string {
	return kind
}

func (w *IngressDeploymentDefaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*ingdpv1alpha1.IngressDeployment)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", c.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", c.Spec)

	meshConfig := w.configStore.MeshConfig.GetConfig()

	if meshConfig == nil {
		return
	}

	klog.V(4).Infof("After setting default values, spec=%#v", c.Spec)
}

type IngressDeploymentValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *IngressDeploymentValidator) Kind() string {
	return kind
}

func (w *IngressDeploymentValidator) ValidateCreate(obj interface{}) error {
	ingressdeployment, ok := obj.(*ingdpv1alpha1.IngressDeployment)
	if !ok {
		return nil
	}

	list, err := w.k8sAPI.FlomeshClient.
		IngressdeploymentV1alpha1().
		IngressDeployments(ingressdeployment.Namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return err
	}

	// There's already an IngressDeployment in this namespace, return error
	if len(list.Items) > 0 {
		return errors.Errorf(
			"There's already %d IngressDeploymnent(s) in namespace %q. Each namespace can have ONLY ONE IngressDeployment.",
			len(list.Items),
			ingressdeployment.Namespace,
		)
	}

	return doValidation(ingressdeployment)
}

func (w *IngressDeploymentValidator) ValidateUpdate(oldObj, obj interface{}) error {
	//oldIngressDeployment, ok := oldObj.(*ingdpv1alpha1.IngressDeployment)
	//if !ok {
	//	return nil
	//}
	//
	//ingressdeployment, ok := obj.(*ingdpv1alpha1.IngressDeployment)
	//if !ok {
	//	return nil
	//}
	//
	//if oldIngressDeployment.Namespace != ingressdeployment.Namespace {
	//    return errors.Errorf("")
	//}

	return doValidation(obj)
}

func (w *IngressDeploymentValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI) *IngressDeploymentValidator {
	return &IngressDeploymentValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	return nil
}
