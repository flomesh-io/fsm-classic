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
    gw "github.com/flomesh-io/fsm/pkg/gateway"
	"github.com/google/go-cmp/cmp"
	"k8s.io/kubernetes/pkg/util/async"
	"time"
)

type SyncFunc func()

type EventHandlerConfig struct {
	MinSyncPeriod time.Duration
	SyncPeriod    time.Duration
	BurstSyncs    int
	Cache         gw.Cache
	SyncFunc      SyncFunc
}

type FsmEventHandler struct {
	cache      gw.Cache
	syncRunner *async.BoundedFrequencyRunner
}

func NewEventHandler(config EventHandlerConfig) EventHandler {
	if config.SyncFunc == nil {
		panic("SyncFunc is required")
	}

	handler := &FsmEventHandler{
		cache: config.Cache,
	}
	handler.syncRunner = async.NewBoundedFrequencyRunner("gateway-sync-runner", config.SyncFunc, config.MinSyncPeriod, config.SyncPeriod, config.BurstSyncs)

	return handler
}

func (e *FsmEventHandler) OnAdd(obj interface{}) {
	if e.onChange(nil, obj) {
		e.Sync()
	}
}

func (e *FsmEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if e.onChange(oldObj, newObj) {
		e.Sync()
	}
}

func (e *FsmEventHandler) OnDelete(obj interface{}) {
	if e.onChange(obj, nil) {
		e.Sync()
	}
}

func (e *FsmEventHandler) onChange(oldObj, newObj interface{}) bool {
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

func (e *FsmEventHandler) Sync() {
	e.syncRunner.Run()
}

func (e *FsmEventHandler) Start(ctx context.Context) error {
    e.syncRunner.Loop(stopCh)

    return nil
}

//func (e *FsmEventHandler) Start(stopCh <-chan struct{}) {
//	e.syncRunner.Loop(stopCh)
//}
