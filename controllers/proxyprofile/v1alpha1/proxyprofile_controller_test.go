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
package v1alpha1

import (
	//"context"
	//"reflect"
	//"time"

	. "github.com/onsi/ginkgo"
	//. "github.com/onsi/gomega"
	//batchv1 "k8s.io/api/batch/v1"
	//batchv1beta1 "k8s.io/api/batch/v1beta1"
	//v1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/types"
	//
	//v1alpha1 "github.com/flomesh-io/traffic-guru/apis/proxyprofile/v1alpha1"
)

var _ = Describe("ProxyProfile controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		ProxyProfileName = "test-proxy"
		ProxyNamespace   = "default"
	)

	Context("When updating Proxy", func() {
		It("Should update Proxy Status. Increase active replica count.", func() {
			By("By creating a new Proxy")
			//ctx := context.Background()
			// TODO: load tempalte and unmarshal to runtime object
			//Expect(k8sClient.Create(ctx, cronJob)).Should(Succeed())
		})
	})
})
