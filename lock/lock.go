package lock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/safe"
)

const lockReleaseGrace = 200 * time.Millisecond

type Lock struct {
	mu    sync.Mutex
	lock  chan struct{}
	ctx   context.Context
	depth byte
	owner string
}

func (l *Lock) LockWithLabel(ctx context.Context, label string) error {
	if l == nil || l.lock == nil {
		return ErrorNilLock
	}

	l.mu.Lock()
	if l.ctx == ctx {
		l.depth++
		l.mu.Unlock()
		return nil
	}
	l.mu.Unlock()

	select {
	case <-ctx.Done():
		return ErrorCtxCancelled
	case <-l.lock:
		l.mu.Lock()
		defer l.mu.Unlock()
		safe.Go(func() { l.check(ctx) }, nil)
		l.ctx = ctx
		l.owner = label
		return nil
	}
}

func NewLock() *Lock {
	l := &Lock{lock: make(chan struct{}, 1)}
	l.lock <- struct{}{}
	return l
}

var (
	ErrorNilLock      = errors.New("Cannot lock nil lock")
	ErrorCtxCancelled = errors.New("Failed to get lock: ctx cancelled")
)

func (l *Lock) Lock(ctx context.Context) error {
	return l.LockWithLabel(ctx, "")
}

func (l *Lock) check(ctx context.Context) {
	if ctx == nil {
		return
	}
	done := ctx.Done()
	if done == nil {
		logger.CheckStack(fmt.Errorf("lock %p acquired with non-cancelable context", l))
		return
	}

	<-done
	time.Sleep(lockReleaseGrace)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ctx == ctx {
		l.releaseLocked()
		logger.CheckStack(fmt.Errorf("released lock %p after grace period: owner=%q ctx=%T canceled=%v depth=%d", l, l.owner, ctx, ctx.Err(), l.depth))
	}
}

func (l *Lock) MustLock(ctx context.Context) {
	err := l.Lock(ctx)
	if err != nil {
		panic(err)
	}
}

func (l *Lock) MustLockWithLabel(ctx context.Context, label string) {
	err := l.LockWithLabel(ctx, label)
	if err != nil {
		panic(err)
	}
}

func (l *Lock) Unlock() {
	if l == nil || l.lock == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.depth > 0 {
		l.depth--
		return
	}
	if l.ctx == nil {
		return
	}
	l.releaseLocked()
}

func (l *Lock) releaseLocked() {
	l.ctx = nil
	l.owner = ""
	select {
	case l.lock <- struct{}{}:
	default:
	}
}
