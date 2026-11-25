package lock

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Buckets are the upper bounds, in seconds, of the wait and hold histograms.
var Buckets = []float64{0.001, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2}

// closeCallShare makes an acquisition a close call when its wait used more than this share
// of the time its context had left: contention that nearly became a timeout.
const closeCallShare = 0.5

// unlabeled names acquisitions made through Lock and MustLock, which pass no label.
const unlabeled = "unlabeled"

// counters are one label's totals. Every field is atomic, so recording never takes a lock.
type counters struct {
	acquisitions   atomic.Uint64
	timeouts       atomic.Uint64
	closeCalls     atomic.Uint64
	forcedReleases atomic.Uint64
	waitNanos      atomic.Int64
	heldNanos      atomic.Int64
}

// histogram counts durations per bucket; buckets[i] counts those at most Buckets[i], and
// the last counts the rest.
type histogram struct {
	buckets  []atomic.Uint64
	sumNanos atomic.Int64
}

func newHistogram() *histogram {
	return &histogram{buckets: make([]atomic.Uint64, len(Buckets)+1)}
}

func (h *histogram) observe(d time.Duration) {
	seconds := d.Seconds()
	i := sort.SearchFloat64s(Buckets, seconds)
	h.buckets[i].Add(1)
	h.sumNanos.Add(int64(d))
}

var (
	byLabel sync.Map // label -> *counters
	waits   = newHistogram()
	holds   = newHistogram()
)

func labelCounters(label string) *counters {
	if label == "" {
		label = unlabeled
	}
	if c, ok := byLabel.Load(label); ok {
		return c.(*counters)
	}
	c, _ := byLabel.LoadOrStore(label, &counters{})
	return c.(*counters)
}

// budget is how long ctx allowed from start, or 0 when it has no deadline.
func budget(ctx context.Context, start time.Time) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline.Sub(start)
	}
	return 0
}

func recordAcquired(label string, wait, allowed time.Duration) {
	c := labelCounters(label)
	c.acquisitions.Add(1)
	c.waitNanos.Add(int64(wait))
	if allowed > 0 && float64(wait) > closeCallShare*float64(allowed) {
		c.closeCalls.Add(1)
	}
	waits.observe(wait)
}

func recordTimeout(label string, wait time.Duration) {
	c := labelCounters(label)
	c.timeouts.Add(1)
	c.waitNanos.Add(int64(wait))
	waits.observe(wait)
}

func recordReleased(owner string, held time.Duration, forced bool) {
	c := labelCounters(owner)
	c.heldNanos.Add(int64(held))
	if forced {
		c.forcedReleases.Add(1)
	}
	holds.observe(held)
}

// LabelStats are the totals for one lock label since the process started.
type LabelStats struct {
	// Acquisitions counts waits that got the lock; reentrant acquisitions are not counted.
	Acquisitions uint64
	// Timeouts counts waits whose context ended first.
	Timeouts uint64
	// CloseCalls counts acquisitions that used more than half the time their context had.
	CloseCalls uint64
	// ForcedReleases counts holds whose context ended while still holding, which check
	// released after its grace period.
	ForcedReleases uint64
	// WaitSeconds is the time spent waiting, timeouts included.
	WaitSeconds float64
	// HeldSeconds is the time spent holding, by holders with this label.
	HeldSeconds float64
}

// HistogramStats is a histogram in Prometheus form.
type HistogramStats struct {
	// Cumulative[i] counts observations at most Buckets[i]; the last entry counts all.
	Cumulative []uint64
	Count      uint64
	SumSeconds float64
}

// Stats are every lock statistic since the process started.
type Stats struct {
	Labels map[string]LabelStats
	Wait   HistogramStats
	Held   HistogramStats
}

// Snapshot reads every lock statistic. Concurrent recording may land between two fields,
// so totals can disagree by an acquisition or two.
//
// @return the statistics, by label and as wait and hold histograms
func Snapshot() Stats {
	s := Stats{Labels: map[string]LabelStats{}, Wait: waits.snapshot(), Held: holds.snapshot()}
	byLabel.Range(func(k, v any) bool {
		c := v.(*counters)
		s.Labels[k.(string)] = LabelStats{
			Acquisitions:   c.acquisitions.Load(),
			Timeouts:       c.timeouts.Load(),
			CloseCalls:     c.closeCalls.Load(),
			ForcedReleases: c.forcedReleases.Load(),
			WaitSeconds:    time.Duration(c.waitNanos.Load()).Seconds(),
			HeldSeconds:    time.Duration(c.heldNanos.Load()).Seconds(),
		}
		return true
	})
	return s
}

func (h *histogram) snapshot() HistogramStats {
	s := HistogramStats{Cumulative: make([]uint64, len(h.buckets))}
	var total uint64
	for i := range h.buckets {
		total += h.buckets[i].Load()
		s.Cumulative[i] = total
	}
	s.Count = total
	s.SumSeconds = time.Duration(h.sumNanos.Load()).Seconds()
	return s
}
