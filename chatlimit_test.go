package qws

import (
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
