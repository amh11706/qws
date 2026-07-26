package qws

import (
	"strings"
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
