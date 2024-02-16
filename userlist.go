package qws

import (
	"context"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/outcmds"
	"github.com/gorilla/websocket"
)

type UserName struct {
	From  string `json:"from"`
	Copy  int64  `json:"copy"`
	Admin int64  `json:"admin"`
}

type MessageSender interface {
	SendMessage(ctx context.Context, m *websocket.PreparedMessage)
	Id() int64
}

type UserList[T MessageSender] map[int64]T

// Broadcast sends the provided message to every user in the list.
func (l UserList[T]) Broadcast(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
	m, err := PrepareJsonMessage(cmd, data)
	if logger.Check(err) {
		return
	}
	for _, u := range l {
		u.SendMessage(ctx, m)
	}
}

// BroadcastExcept sends the provided message to every user in the list except
// the provided user.
func (l UserList[T]) BroadcastExcept(ctx context.Context, cmd outcmds.Cmd, data interface{}, e T) {
	l.BroadcastFilter(ctx, cmd, data, func(u T) bool {
		return u.Id() != e.Id()
	})
}

// BroadcastFilter sends the provided message to every user in the list for which
// the provided filter func returns true.
func (l UserList[T]) BroadcastFilter(ctx context.Context, cmd outcmds.Cmd, data interface{}, filter func(T) bool) {
	m, err := PrepareJsonMessage(cmd, data)
	if logger.Check(err) {
		return
	}
	for _, u := range l {
		if filter(u) {
			u.SendMessage(ctx, m)
		}
	}
}
