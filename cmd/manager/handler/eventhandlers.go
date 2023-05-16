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

package handler

import (
	"context"
	"fmt"
	svcimpv1alpha1 "github.com/flomesh-io/fsm/apis/serviceimport/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/config/listener"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/event/handler"
	gwcache "github.com/flomesh-io/fsm/pkg/gateway/cache"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	rtcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"time"
)

func GetResourceEventHandler(ctx *fctx.FsmContext) handler.EventHandler {
	gatewayCache := gwcache.NewGatewayCache(gwcache.GatewayCacheConfig{
		Client: ctx.Manager.GetClient(),
		Cache:  ctx.Manager.GetCache(),
	})

	return handler.NewEventHandler(handler.EventHandlerConfig{
		MinSyncPeriod: 5 * time.Second,
		SyncPeriod:    30 * time.Second,
		BurstSyncs:    5,
		Cache:         gatewayCache,
		SyncFunc:      gatewayCache.BuildConfigs,
		StopCh:        ctx.StopCh,
	})
}

func RegisterEventHandlers(ctx *fctx.FsmContext) error {
	klog.Infof("[MGR] Registering Event Handlers ...")

	// FIXME: make it configurable
	resyncPeriod := 15 * time.Minute
	mc := ctx.ConfigStore.MeshConfig.GetConfig()

	configHandler := config.NewConfigurationHandler(
		config.NewFlomeshConfigurationHandler(configChangeListeners(ctx, mc)),
	)

	if err := informOnResource(&corev1.ConfigMap{}, configHandler, ctx.Manager.GetCache(), resyncPeriod); err != nil {
		klog.Errorf("failed to create informer for configmaps: %s", err)
		return err
	}

	if mc.IsGatewayApiEnabled() {
		if ctx.EventHandler == nil {
			return fmt.Errorf("GatewayAPI is enabled, but no valid EventHanlder is provided")
		}

		for name, r := range map[string]client.Object{
			"namespaces":     &corev1.Namespace{},
			"services":       &corev1.Service{},
			"endpoints":      &corev1.Endpoints{},
			"serviceimports": &svcimpv1alpha1.ServiceImport{},
			"endpointslices": &discoveryv1.EndpointSlice{},
			"gatewayclasses": &gwv1beta1.GatewayClass{},
			"gateways":       &gwv1beta1.Gateway{},
			"httproutes":     &gwv1beta1.HTTPRoute{},
		} {
			if err := informOnResource(r, ctx.EventHandler, ctx.Manager.GetCache(), resyncPeriod); err != nil {
				klog.Errorf("failed to create informer for %s: %s", name, err)
				return err
			}
		}
	}

	return nil
}

func configChangeListeners(ctx *fctx.FsmContext, mc *config.MeshConfig) []config.MeshConfigChangeListener {
	listeners := []config.MeshConfigChangeListener{
		listener.NewBasicConfigListener(ctx),
		listener.NewProxyProfileConfigListener(ctx),
		listener.NewLoggingConfigListener(ctx),
	}

	if mc.IsIngressEnabled() {
		listeners = append(listeners, listener.NewIngressConfigListener(ctx))
	}

	return listeners
}

func informOnResource(obj client.Object, handler cache.ResourceEventHandler, cache rtcache.Cache, resyncPeriod time.Duration) error {
	informer, err := cache.GetInformer(context.TODO(), obj)
	if err != nil {
		return err
	}

	if handler != nil {
		_, err = informer.AddEventHandlerWithResyncPeriod(handler, resyncPeriod)
		return err
	}

	return nil
}
