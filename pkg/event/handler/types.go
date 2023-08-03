package handler

import (
	"context"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type EventHandler interface {
	cache.ResourceEventHandler
	manager.Runnable
}

type EventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
	DeleteFunc func(obj interface{})
	StartFunc  func(ctx context.Context) error
}

// OnAdd calls AddFunc if it's not nil.
func (r EventHandlerFuncs) OnAdd(obj interface{}) {
	if r.AddFunc != nil {
		r.AddFunc(obj)
	}
}

// OnUpdate calls UpdateFunc if it's not nil.
func (r EventHandlerFuncs) OnUpdate(oldObj, newObj interface{}) {
	if r.UpdateFunc != nil {
		r.UpdateFunc(oldObj, newObj)
	}
}

// OnDelete calls DeleteFunc if it's not nil.
func (r EventHandlerFuncs) OnDelete(obj interface{}) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

func (r EventHandlerFuncs) Start(ctx context.Context) error {
	if r.StartFunc != nil {
		return r.StartFunc(ctx)
	}

	return nil
}
