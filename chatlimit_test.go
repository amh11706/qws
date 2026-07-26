package qws

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestChatLimiterAllowsBurstThenThrottles(t *testing.T) {
	l := &chatLimiter{}
	now := time.Now()

	for i := 0; i < chatBurst; i++ {
		if !l.allow(now) {
			t.Fatalf("message %d of the burst was throttled", i+1)
		}
	}
	if l.allow(now) {
		t.Fatal("burst was not capped")
	}

	// The bucket refills at chatPerSecond, so a second buys chatPerSecond more.
	now = now.Add(time.Second)
	for i := 0; i < int(chatPerSecond); i++ {
		if !l.allow(now) {
			t.Fatalf("refill did not grant message %d", i+1)
		}
	}
	if l.allow(now) {
		t.Fatal("refill granted more than the sustained rate")
	}
}

func TestChatLimiterDoesNotAccumulateBeyondBurst(t *testing.T) {
	l := &chatLimiter{}
	now := time.Now()
	l.allow(now)

	// A long idle period must not bank an unlimited number of messages.
	now = now.Add(time.Hour)
	for i := 0; i < chatBurst; i++ {
		if !l.allow(now) {
			t.Fatalf("message %d after idling was throttled", i+1)
		}
	}
	if l.allow(now) {
		t.Fatal("idling banked more than chatBurst messages")
	}
}

func TestChatTooLong(t *testing.T) {
	if ChatTooLong(strings.Repeat("a", MaxChatRunes)) {
		t.Fatal("a message exactly at the limit was rejected")
	}
	if !ChatTooLong(strings.Repeat("a", MaxChatRunes+1)) {
		t.Fatal("a message over the limit was accepted")
	}

	// The client counts UTF-16 units, so it would already have stopped the user
	// well before MaxChatRunes astral emoji. Counting runes here must not be
	// stricter than that, or we reject text the client was happy to send.
	if ChatTooLong(strings.Repeat("\U0001F986", MaxChatRunes)) {
		t.Fatal("multibyte characters were counted as more than one rune")
	}
}

func TestAllowChatSharedAcrossConnectionsOfOneUser(t *testing.T) {
	// Two UserConns of the same account share one *User, so they must share
	// one budget rather than getting chatBurst each.
	u := &User{Name: "Somebody"}
	a, b := &UserConn{user: u}, &UserConn{user: u}

	used := 0
	for i := 0; i < chatBurst*2; i++ {
		c := a
		if i%2 == 1 {
			c = b
		}
		if c.User().AllowChat() {
			used++
		}
	}
	if used > chatBurst {
		t.Fatalf("two connections got %d messages, want at most %d", used, chatBurst)
	}
}

// TestChatLimiterConcurrentGrantsAreExact stands in for the race detector,
// which needs cgo and a C compiler that is not available here. Each round uses
// a fresh limiter and releases every goroutine at once, so all of them contend
// on the same handful of tokens. Unsynchronised access shows up as over
// granting, either from a lost decrement or from several goroutines each
// seeing a zero last and resetting the bucket to full.
func TestChatLimiterConcurrentGrantsAreExact(t *testing.T) {
	const rounds, senders = 300, 64

	for round := 0; round < rounds; round++ {
		l := &chatLimiter{}
		now := time.Now()
		var granted atomic.Int64
		var wg sync.WaitGroup
		start := make(chan struct{})

		wg.Add(senders)
		for i := 0; i < senders; i++ {
			go func() {
				defer wg.Done()
				<-start
				if l.allow(now) {
					granted.Add(1)
				}
			}()
		}
		close(start)
		wg.Wait()

		if got := granted.Load(); got != chatBurst {
			t.Fatalf("round %d: %d simultaneous senders were granted %d messages, want exactly %d",
				round, senders, got, chatBurst)
		}
	}
}

// TestAllowChatLazyInitIsSynchronised drives the User.chat lazy creation from
// many goroutines at once on a fresh User. The limiter itself is covered above,
// but that test calls allow directly, so without this one nothing concurrently
// exercises the nil check that creates it. Run under -race.
func TestAllowChatLazyInitIsSynchronised(t *testing.T) {
	const rounds, senders = 200, 64

	for round := 0; round < rounds; round++ {
		u := &User{Name: "Somebody"}
		var granted atomic.Int64
		var wg sync.WaitGroup
		start := make(chan struct{})

		wg.Add(senders)
		for i := 0; i < senders; i++ {
			go func() {
				defer wg.Done()
				<-start
				if u.AllowChat() {
					granted.Add(1)
				}
			}()
		}
		close(start)
		wg.Wait()

		// A torn lazy init would hand out more than one limiter, so the
		// senders would collectively get more than a single bucket's worth.
		if got := granted.Load(); got != chatBurst {
			t.Fatalf("round %d: %d senders sharing one user were granted %d messages, want exactly %d",
				round, senders, got, chatBurst)
		}
	}
}
