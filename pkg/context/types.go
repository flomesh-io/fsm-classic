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

package context

import (
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/event/handler"
	"github.com/flomesh-io/fsm-classic/pkg/ingress/connector"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	mcsevent "github.com/flomesh-io/fsm-classic/pkg/mcs/event"
	"github.com/flomesh-io/fsm-classic/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type FsmContext struct {
	client.Client
	Manager            manager.Manager
	Scheme             *runtime.Scheme
	ConfigStore        *config.Store
	K8sAPI             *kube.K8sAPI
	CertificateManager certificate.Manager
	RepoClient         *repo.PipyRepoClient
	Broker             *mcsevent.Broker
	EventHandler       handler.EventHandler
	Connector          *connector.Connector
	StopCh             <-chan struct{}
}