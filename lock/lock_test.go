package lock

import (
	"context"
	"testing"
	"time"
)

func TestLockWaitsBeforeReleasingCanceledContext(t *testing.T) {
	l := NewLock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := l.Lock(ctx); err != nil {
		t.Fatalf("first lock acquire failed: %v", err)
	}

	cancel()

	acquired := make(chan struct{})
	go func() {
		if err := l.Lock(context.Background()); err != nil {
			t.Errorf("second lock acquire returned error: %v", err)
			return
		}
		close(acquired)
		l.Unlock()
	}()

	select {
	case <-acquired:
		t.Fatal("lock was released immediately after the context was canceled")
	case <-time.After(100 * time.Millisecond):
	}

	l.Unlock()

	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("lock was not released after the owner unlocked it")
	}
}
