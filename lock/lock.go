package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/amh11706/logger"
)

type Lock struct {
	lock chan struct{}
	ctx  context.Context
}

func NewLock() *Lock {
	l := &Lock{lock: make(chan struct{}, 1)}
	l.Unlock()
	return l
}

var (
	NilLock      = errors.New("Cannot lock nil lock")
	CtxCancelled = errors.New("Failed to get lock: ctx cancelled")
)

func (l *Lock) Lock(ctx context.Context) error {
	if l == nil {
		return NilLock
	}
	select {
	case <-ctx.Done():
		return CtxCancelled
	case <-l.lock:
		time.AfterFunc(5*time.Second, l.check(ctx))
		return nil
	}
}

func (l *Lock) check(ctx context.Context) func() {
	return func() {
		if l.ctx == ctx {
			l.Unlock()
			logger.CheckStack(fmt.Errorf("Released dead lock!"))
		}
	}
}

func (l *Lock) MustLock(ctx context.Context) {
	if l == nil || l.lock == nil {
		panic(NilLock)
	}
	select {
	case <-ctx.Done():
		panic(CtxCancelled)
	case <-l.lock:
		time.AfterFunc(5*time.Second, l.check(ctx))
		return
	}
}

func (l *Lock) Unlock() {
	if len(l.lock) != 0 {
		panic("Unlock on already unlocked lock")
	}
	l.ctx = nil
	l.lock <- struct{}{}
}
