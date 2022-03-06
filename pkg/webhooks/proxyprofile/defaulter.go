/*
 * The NEU License
 *
 * Copyright (c) 2022.  flomesh.io
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is furnished to do
 * so, subject to the following conditions:
 *
 * (1)The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * (2)If the software or part of the code will be directly used or used as a
 * component for commercial purposes, including but not limited to: public cloud
 *  services, hosting services, and/or commercial software, the logo as following
 *  shall be displayed in the eye-catching position of the introduction materials
 * of the relevant commercial services or products (such as website, product
 * publicity print), and the logo shall be linked or text marked with the
 * following URL.
 *
 * LOGO : http://flomesh.cn/assets/flomesh-logo.png
 * URL : https://github.com/flomesh-io
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package proxyprofile

import (
	"context"
	"encoding/json"
	"github.com/flomesh-io/fsm/api/v1alpha1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type mutatingHandler struct {
	defaulter *ProxProfileDefaulter
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

	pf := &v1alpha1.ProxyProfile{}
	if err := h.decoder.Decode(req, pf); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Default the object
	h.defaulter.SetDefaults(pf)
	marshalled, err := json.Marshal(pf)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Create the patch
	return admission.PatchResponseFromRaw(req.Object.Raw, marshalled)
}
