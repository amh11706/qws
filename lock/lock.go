package lock

import "context"

type Lock chan struct{}

func NewLock() Lock {
	l := make(Lock, 1)
	l.Unlock()
	return l
}

func (l Lock) Lock(ctx context.Context) {
	if l == nil {
		panic("Cannot lock nil lock")
	}
	select {
	case <-ctx.Done():
		panic("Failed to get lock: ctx cancelled")
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
