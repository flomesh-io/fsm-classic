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

package utils

import (
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/tidwall/sjson"
	"k8s.io/klog/v2"
)

func UpdateIngressTLSConfig(basepath string, repoClient *repo.PipyRepoClient, mc *config.MeshConfig) error {
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return err
	}

	for path, value := range map[string]interface{}{
		"tls.enabled": mc.Ingress.TLS.Enabled,
		"tls.listen":  mc.Ingress.TLS.Listen,
		"tls.mTLS":    mc.Ingress.TLS.MTLS,
	} {
		json, err = sjson.Set(json, path, value)
		if err != nil {
			klog.Errorf("Failed to update TLS config: %s", err)
			return err
		}
	}

	return updateMainJson(basepath, repoClient, json)
}

func IssueCertForIngress(basepath string, repoClient *repo.PipyRepoClient, certMgr certificate.Manager, mc *config.MeshConfig) error {
	// 1. issue cert
	cert, err := certMgr.IssueCertificate("ingress-pipy", commons.DefaultCAValidityPeriod, []string{})
	if err != nil {
		klog.Errorf("Issue certificate for ingress-pipy error: %s", err)
		return err
	}

	// 2. get main.json
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return err
	}

	newJson, err := sjson.Set(json, "tls", map[string]interface{}{
		"enabled": mc.Ingress.TLS.Enabled,
		"listen":  mc.Ingress.TLS.Listen,
		"mTLS":    mc.Ingress.TLS.MTLS,
		"certificate": map[string]interface{}{
			"cert": string(cert.CrtPEM),
			"key":  string(cert.KeyPEM),
			"ca":   string(cert.CA),
		},
	})
	if err != nil {
		klog.Errorf("Failed to update TLS config: %s", err)
		return err
	}

	// 6. update main.json
	return updateMainJson(basepath, repoClient, newJson)
}

func UpdateSSLPassthrough(basepath string, repoClient *repo.PipyRepoClient, enabled bool, upstreamPort int32) error {
	klog.V(5).Infof("SSL passthrough is enabled, updating repo config ...")
	// 1. get main.json
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return err
	}

	// 2. update ssl passthrough config
	klog.V(5).Infof("SSLPassthrough enabled=%t", enabled)
	klog.V(5).Infof("SSLPassthrough upstreamPort=%d", upstreamPort)
	newJson, err := sjson.Set(json, "sslPassthrough", map[string]interface{}{
		"enabled":      enabled,
		"upstreamPort": upstreamPort,
	})
	if err != nil {
		klog.Errorf("Failed to update sslPassthrough: %s", err)
		return err
	}

	// 3. update main.json
	return updateMainJson(basepath, repoClient, newJson)
}
