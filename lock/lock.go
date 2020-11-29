package lock

import (
	"context"
	"errors"
)

type Lock chan struct{}

func NewLock() Lock {
	l := make(Lock, 1)
	l.Unlock()
	return l
}

var (
	NilLock      = errors.New("Cannot lock nil lock")
	CtxCancelled = errors.New("Failed to get lock: ctx cancelled")
)

func (l Lock) Lock(ctx context.Context) error {
	if l == nil {
		return NilLock
	}
	select {
	case <-ctx.Done():
		return CtxCancelled
	case <-l:
		return nil
	}
}

func (l Lock) MustLock(ctx context.Context) {
	if l == nil {
		panic(NilLock)
	}
	select {
	case <-ctx.Done():
		panic(CtxCancelled)
	case <-l:
		return
	}
}

func (l Lock) Unlock() {
	if len(l) != 0 {
		panic("Unlock on already unlocked lock")
	}
	l <- struct{}{}
}
