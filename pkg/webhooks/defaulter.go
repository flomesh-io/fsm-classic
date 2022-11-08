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

package webhooks

import (
	"context"
	"encoding/json"
	clusterv1alpha1 "github.com/flomesh-io/fsm/apis/cluster/v1alpha1"
	nsigv1alpha1 "github.com/flomesh-io/fsm/apis/namespacedingress/v1alpha1"
	pfv1alpha1 "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1"
	svcexpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceexport/v1alpha1"
	svcimpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceimport/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

type mutatingHandler struct {
	defaulter Defaulter
	decoder   *admission.Decoder
}

var _ admission.DecoderInjector = &mutatingHandler{}

// InjectDecoder injects the decoder into a mutatingHandler.
func (h *mutatingHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

// Handle handles admission requests.
func (h *mutatingHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	if h.defaulter == nil {
		panic("defaulter should never be nil")
	}

	obj := h.getObject()
	if obj == nil {
		return admission.Allowed("Not supported Kind")
	}

	if err := h.decoder.Decode(req, obj); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Default the object
	h.defaulter.SetDefaults(obj)
	marshalled, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Create the patch
	return admission.PatchResponseFromRaw(req.Object.Raw, marshalled)
}

func (h *mutatingHandler) getObject() runtime.Object {
	switch strings.ToLower(h.defaulter.Kind()) {
	case "configmap":
		return &corev1.ConfigMap{}
	case "proxyprofile":
		return &pfv1alpha1.ProxyProfile{}
	case "cluster":
		return &clusterv1alpha1.Cluster{}
	case "namespacedingress":
		return &nsigv1alpha1.NamespacedIngress{}
	case "serviceimport":
		return &svcimpv1alpha1.ServiceImport{}
	case "serviceexport":
		return &svcexpv1alpha1.ServiceExport{}
	}

	return nil
}

func DefaultingWebhookFor(defaulter Defaulter) *admission.Webhook {
	return &admission.Webhook{
		Handler: &mutatingHandler{defaulter: defaulter},
	}
}
