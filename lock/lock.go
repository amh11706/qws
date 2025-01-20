package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/safe"
)

type Lock struct {
	lock  chan struct{}
	ctx   context.Context
	depth byte
}

func NewLock() *Lock {
	l := &Lock{lock: make(chan struct{}, 1)}
	l.Unlock()
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
	if l.ctx == ctx {
		l.depth++
		return nil
	}
	select {
	case <-ctx.Done():
		return ErrorCtxCancelled
	case <-l.lock:
		safe.Go(func() { l.check(ctx) }, nil)
		l.ctx = ctx
		return nil
	}
}

func (l *Lock) check(ctx context.Context) {
	doneChan := ctx.Done()
	if doneChan == nil {
		time.Sleep(5 * time.Second)
	} else {
		<-doneChan
	}
	if l.ctx == ctx {
		l.Unlock()
		logger.CheckStack(fmt.Errorf("Released expired lock!"))
	}
}

func (l *Lock) MustLock(ctx context.Context) {
	err := l.Lock(ctx)
	if err != nil {
		panic(err)
	}
}

func (l *Lock) Unlock() {
	if l.depth > 0 {
		l.depth--
		return
	}
	if len(l.lock) != 0 {
		panic("Unlock on already unlocked lock")
	}
	l.ctx = nil
	l.lock <- struct{}{}
}
