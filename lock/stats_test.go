package lock

import (
	"context"
	"testing"
	"time"
)

// statsFor reads one label's totals; tests use labels of their own, as the counters are
// shared by the whole process.
func statsFor(label string) LabelStats {
	return Snapshot().Labels[label]
}

// since is what a label recorded after before was read, so a test sees only its own
// work even when -count runs it again in the same process.
func since(label string, before LabelStats) LabelStats {
	now := statsFor(label)
	return LabelStats{
		Acquisitions:   now.Acquisitions - before.Acquisitions,
		Timeouts:       now.Timeouts - before.Timeouts,
		CloseCalls:     now.CloseCalls - before.CloseCalls,
		ForcedReleases: now.ForcedReleases - before.ForcedReleases,
		WaitSeconds:    now.WaitSeconds - before.WaitSeconds,
		HeldSeconds:    now.HeldSeconds - before.HeldSeconds,
	}
}

func TestStatsCountUncontendedAcquireAndHold(t *testing.T) {
	before, beforeReentry := statsFor("test.uncontended"), statsFor("test.uncontended.reentry")
	l := NewLock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := l.LockWithLabel(ctx, "test.uncontended"); err != nil {
		t.Fatal(err)
	}
	if err := l.LockWithLabel(ctx, "test.uncontended.reentry"); err != nil { // same ctx: reentrant
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	l.Unlock()
	l.Unlock()

	s := since("test.uncontended", before)
	if s.Acquisitions != 1 || s.Timeouts != 0 || s.CloseCalls != 0 || s.ForcedReleases != 0 {
		t.Errorf("counts: %+v", s)
	}
	if s.HeldSeconds < 0.015 {
		t.Errorf("held %.3fs, want about 0.02s", s.HeldSeconds)
	}
	if r := since("test.uncontended.reentry", beforeReentry); r.Acquisitions != 0 {
		t.Errorf("reentrant acquisition was counted: %+v", r)
	}
}

func TestStatsCountTimeout(t *testing.T) {
	before := statsFor("test.timeout.waiter")
	l := NewLock()
	if err := l.LockWithLabel(t.Context(), "test.timeout.holder"); err != nil {
		t.Fatal(err)
	}
	defer l.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := l.LockWithLabel(ctx, "test.timeout.waiter"); err != ErrorCtxCancelled {
		t.Fatalf("got %v, want ErrorCtxCancelled", err)
	}
	s := since("test.timeout.waiter", before)
	if s.Timeouts != 1 || s.Acquisitions != 0 {
		t.Errorf("counts: %+v", s)
	}
	if s.WaitSeconds < 0.025 {
		t.Errorf("waited %.3fs, want about the 0.03s deadline", s.WaitSeconds)
	}
}

func TestStatsCountCloseCallOnlyPastHalfTheDeadline(t *testing.T) {
	for _, c := range []struct {
		label     string
		releaseIn time.Duration
		deadline  time.Duration
		closeCall bool
	}{
		{"test.closecall.yes", 120 * time.Millisecond, 200 * time.Millisecond, true},
		{"test.closecall.no", 10 * time.Millisecond, 2 * time.Second, false},
	} {
		before := statsFor(c.label)
		l := NewLock()
		if err := l.LockWithLabel(t.Context(), c.label+".holder"); err != nil {
			t.Fatal(err)
		}
		time.AfterFunc(c.releaseIn, l.Unlock)

		ctx, cancel := context.WithTimeout(context.Background(), c.deadline)
		if err := l.LockWithLabel(ctx, c.label); err != nil {
			t.Fatalf("%s: %v", c.label, err)
		}
		l.Unlock()
		cancel()

		s := since(c.label, before)
		if s.Acquisitions != 1 || (s.CloseCalls == 1) != c.closeCall {
			t.Errorf("%s: %+v, want close call %v", c.label, s, c.closeCall)
		}
	}
}

func TestStatsCountForcedRelease(t *testing.T) {
	before := statsFor("test.forced")
	l := NewLock()
	ctx, cancel := context.WithCancel(context.Background())
	if err := l.LockWithLabel(ctx, "test.forced"); err != nil {
		t.Fatal(err)
	}
	cancel() // the holder's context ends without it ever unlocking

	wait, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	if err := l.LockWithLabel(wait, "test.forced.next"); err != nil {
		t.Fatalf("lock was not force-released: %v", err)
	}
	l.Unlock()
	if s := since("test.forced", before); s.ForcedReleases != 1 || s.HeldSeconds < lockReleaseGrace.Seconds() {
		t.Errorf("holder: %+v, want one forced release held past the grace period", s)
	}
}

func TestHistogramIsCumulative(t *testing.T) {
	h := newHistogram()
	for _, d := range []time.Duration{0, 5 * time.Millisecond, 300 * time.Millisecond, 10 * time.Second} {
		h.observe(d)
	}
	s := h.snapshot()
	// Buckets: 1ms 10ms 50ms 100ms 250ms 500ms 1s 2s +Inf
	want := []uint64{1, 2, 2, 2, 2, 3, 3, 3, 4}
	for i := range want {
		if s.Cumulative[i] != want[i] {
			t.Fatalf("cumulative %v, want %v", s.Cumulative, want)
		}
	}
	if s.Count != 4 || s.SumSeconds < 10.3 || s.SumSeconds > 10.31 {
		t.Errorf("count %d sum %.4f, want 4 and 10.305", s.Count, s.SumSeconds)
	}
}

func TestUnlabeledAcquisitionsAreNamed(t *testing.T) {
	before := statsFor(unlabeled).Acquisitions
	l := NewLock()
	if err := l.Lock(t.Context()); err != nil {
		t.Fatal(err)
	}
	l.Unlock()
	if after := statsFor(unlabeled).Acquisitions; after != before+1 {
		t.Errorf("unlabeled acquisitions %d -> %d", before, after)
	}
}
