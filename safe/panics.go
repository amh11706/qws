package safe

import "sync"

// Where a panic was recovered, for the places in this module.
const (
	PanicBackground = "background" // a goroutine started with Go or GoWithValue
	PanicWSMessage  = "ws_message" // a websocket message handler
)

var (
	panicsMu sync.Mutex
	panics   = map[string]uint64{}
)

// RecordPanic counts a recovered panic, for metrics.
//
// @param where a short fixed name for the recovering code, such as PanicWSMessage
func RecordPanic(where string) {
	panicsMu.Lock()
	panics[where]++
	panicsMu.Unlock()
}

// Panics returns how many panics have been recovered since the process started.
//
// @return a copy of the counts, keyed by where they were recovered
func Panics() map[string]uint64 {
	panicsMu.Lock()
	defer panicsMu.Unlock()
	out := make(map[string]uint64, len(panics))
	for k, v := range panics {
		out[k] = v
	}
	return out
}
