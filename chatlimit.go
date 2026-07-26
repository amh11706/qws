package qws

import (
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

// Chat is rate limited separately from everything else on the socket. Gameplay
// traffic (moves, shots) is bursty by nature and a limit loose enough for it
// would not slow a spammer down, so only the handful of paths that broadcast
// user written text call AllowChat.
const (
	// chatBurst messages can be sent back to back before the rate applies.
	chatBurst = 10
	// chatPerSecond is the sustained rate the bucket refills at.
	chatPerSecond = 2.0
)

// ChatRateMessage is shown to a user who has run their chat budget out.
const ChatRateMessage = "You are sending messages too quickly. Please slow down."

// MaxChatRunes matches the maxlength on the client's chat input, so a normal
// client can never trip it. Runes rather than bytes: the browser counts UTF-16
// units, where an astral emoji costs 2 and here it costs 1, which keeps the
// server from rejecting anything the client was willing to send.
const MaxChatRunes = 200

// ChatTooLongMessage is shown to a user whose message exceeds MaxChatRunes.
var ChatTooLongMessage = fmt.Sprintf("That message is too long. Chat is limited to %d characters.", MaxChatRunes)

// ChatTooLong reports whether m is over the chat length limit.
func ChatTooLong(m string) bool {
	return utf8.RuneCountInString(m) > MaxChatRunes
}

// chatLimiter is a token bucket. It is only reachable through User.AllowChat,
// which creates it on first use.
type chatLimiter struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
}

func (l *chatLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.last.IsZero() {
		l.tokens = chatBurst
	} else {
		l.tokens += now.Sub(l.last).Seconds() * chatPerSecond
		if l.tokens > chatBurst {
			l.tokens = chatBurst
		}
	}
	l.last = now

	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

// chatLimiterLock guards lazy creation of User.chat. A User can be built as a
// bare literal in a dozen places, so the limiter cannot be set up in a
// constructor. Contention here is irrelevant: it is only taken on chat.
var chatLimiterLock sync.Mutex

// AllowChat reports whether this user may send another chat message now, and
// consumes a token if so. The limiter lives on User rather than on UserConn so
// that every connection of an account shares one budget.
func (u *User) AllowChat() bool {
	if u == nil {
		return true
	}
	chatLimiterLock.Lock()
	if u.chat == nil {
		u.chat = &chatLimiter{}
	}
	l := u.chat
	chatLimiterLock.Unlock()

	return l.allow(time.Now())
}
