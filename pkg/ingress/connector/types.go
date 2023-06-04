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

package connector

import (
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/ingress/cache"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	mcsevent "github.com/flomesh-io/fsm-classic/pkg/mcs/event"
	"time"
)

type Connector struct {
	k8sAPI     *kube.K8sAPI
	cache      *cache.Cache
	clusterCfg *config.Store
	broker     *mcsevent.Broker
}

func NewConnector(k8sAPI *kube.K8sAPI, broker *mcsevent.Broker, certMgr certificate.Manager, clusterCfg *config.Store, resyncPeriod time.Duration) *Connector {
	return &Connector{
		k8sAPI:     k8sAPI,
		cache:      cache.NewCache(k8sAPI, clusterCfg, broker, certMgr, resyncPeriod),
		clusterCfg: clusterCfg,
		broker:     broker,
	}
}
