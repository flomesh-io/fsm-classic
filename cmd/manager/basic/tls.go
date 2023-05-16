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

package basic

import (
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config/utils"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"k8s.io/klog/v2"
)

func SetupTLS(ctx *fctx.FsmContext) error {
	klog.Infof("[MGR] Setting up TLS ...")

	mc := ctx.ConfigStore.MeshConfig.GetConfig()
	klog.V(5).Infof("mc.Ingress.TLS=%v", mc.Ingress.TLS)

	if mc.Ingress.TLS.Enabled {
		if mc.Ingress.TLS.SSLPassthrough.Enabled {
			// SSL Passthrough
			if err := utils.UpdateSSLPassthrough(
				commons.DefaultIngressBasePath,
				ctx.RepoClient,
				mc.Ingress.TLS.SSLPassthrough.Enabled,
				mc.Ingress.TLS.SSLPassthrough.UpstreamPort,
			); err != nil {
				return err
			}
		} else {
			// TLS Offload
			if err := utils.IssueCertForIngress(commons.DefaultIngressBasePath, ctx.RepoClient, ctx.CertificateManager, mc); err != nil {
				return err
			}
		}
	}

	return nil
}
