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
    gwcache "github.com/flomesh-io/fsm/pkg/gateway/cache"
    "github.com/google/go-cmp/cmp"
    "k8s.io/kubernetes/pkg/util/async"
    "time"
)

type EventHandlerConfig struct {
    MinSyncPeriod time.Duration
    SyncPeriod time.Duration
    BurstSyncs int
    Cache *gwcache.GatewayCache
}

type EventHandler struct {
    cache *gwcache.GatewayCache
    syncRunner *async.BoundedFrequencyRunner
}

func NewEventHandler(config EventHandlerConfig) *EventHandler {
    handler := &EventHandler{
        cache:      config.Cache,
    }
    handler.syncRunner = async.NewBoundedFrequencyRunner("gateway-sync-runner", handler.buildConfigs, config.MinSyncPeriod, config.SyncPeriod, config.BurstSyncs)

    return handler
}

func (e *EventHandler) OnAdd(obj interface{}) {
    if e.onChange(nil, obj) {
        e.Sync()
    }
}

func (e *EventHandler) OnUpdate(oldObj, newObj interface{}) {
    if e.onChange(oldObj, newObj) {
        e.Sync()
    }
}

func (e *EventHandler) OnDelete(obj interface{}) {
   if e.onChange(obj, nil) {
        e.Sync()
    }
}

func (e *EventHandler) onChange(oldObj, newObj interface{}) bool {
    if newObj == nil {
        return e.cache.Delete(oldObj)
    } else {
        if oldObj == nil {
            return e.cache.Insert(newObj)
        } else {
            if cmp.Equal(oldObj, newObj) {
                return false
            }

            del := e.cache.Delete(oldObj)
            ins := e.cache.Insert(newObj)

            return del || ins
        }
    }
}

func (e *EventHandler) Sync() {
    e.syncRunner.Run()
}

func (e *EventHandler) Start(stopCh <-chan struct{}) {
    e.syncRunner.Loop(stopCh)
}

func (e *EventHandler) buildConfigs() {

}




