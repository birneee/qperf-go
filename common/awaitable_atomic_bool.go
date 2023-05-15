package common

import (
	"context"
	"sync"
	"sync/atomic"
)

type AwaitableAtomicBool struct {
	mutex            sync.Mutex
	inner            atomic.Bool
	currentCtx       context.Context
	currentCtxCancel context.CancelFunc
}

func NewAwaitableAtomicBool(initialValue bool) *AwaitableAtomicBool {
	a := &AwaitableAtomicBool{}
	a.inner.Store(initialValue)
	a.currentCtx, a.currentCtxCancel = context.WithCancel(context.Background())
	return a
}

func (b *AwaitableAtomicBool) Load() bool {
	return b.inner.Load()
}

func (b *AwaitableAtomicBool) Store(value bool) {
	b.mutex.Lock()
	b.inner.Store(value)
	b.currentCtxCancel()
	b.currentCtx, b.currentCtxCancel = context.WithCancel(context.Background())
	b.mutex.Unlock()
}

func (b *AwaitableAtomicBool) AwaitTrue() {
	// fast path
	if b.inner.Load() {
		return
	}
	// slow path
	b.mutex.Lock()
	if b.inner.Load() {
		b.mutex.Unlock()
		return
	}
	changeChan := b.currentCtx.Done()
	b.mutex.Unlock()
	<-changeChan
}
