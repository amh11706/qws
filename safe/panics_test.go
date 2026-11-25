package safe

import (
	"testing"
	"time"
)

func TestGoCountsRecoveredPanics(t *testing.T) {
	before := Panics()[PanicBackground]
	done := make(chan struct{})
	Go(func() { panic("boom") }, func(interface{}) { close(done) })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("panic was not recovered")
	}
	// onError runs after RecordPanic, so the count is in by now
	if got := Panics()[PanicBackground]; got != before+1 {
		t.Errorf("background panics %d, want %d", got, before+1)
	}
}
