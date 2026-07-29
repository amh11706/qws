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

const lockReleaseGrace = time.Second

type Lock struct {
	mu    sync.Mutex
	lock  chan struct{}
	ctx   context.Context
	depth byte
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
		return nil
	}
}

func (l *Lock) check(ctx context.Context) {
	if ctx == nil {
		return
	}
	time.Sleep(lockReleaseGrace)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ctx == ctx {
		l.releaseLocked()
		logger.CheckStack(fmt.Errorf("Released expired lock after grace period"))
	}
}

func (l *Lock) MustLock(ctx context.Context) {
	err := l.Lock(ctx)
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
	select {
	case l.lock <- struct{}{}:
	default:
	}
}
