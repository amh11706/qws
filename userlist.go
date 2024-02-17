package qws

import (
	"context"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/outcmds"
	"github.com/gorilla/websocket"
)

type UserName struct {
	From  string     `json:"from"`
	Copy  int64      `json:"copy"`
	Admin AdminLevel `json:"admin"`
}

type UserConner interface {
	UserConn() *UserConn
}

type MessageSender interface {
	SendMessage(ctx context.Context, m *websocket.PreparedMessage)
	Send(ctx context.Context, cmd outcmds.Cmd, data interface{})
	SendInfo(ctx context.Context, data string)
	SendRaw(ctx context.Context, data interface{})
	Id() int64
	UserId() int64
	Name() string
	Close()
	Router() *Router
	CmdRouter() *CmdRouter
	AddCloseHook(context.Context, CloseHandler) error
	RemoveCloseHook(context.Context, CloseHandler) error
	AdminLevel() AdminLevel
	UserName() UserName
	PrintName() string
	InLobby() int64
	SetInLobby(int64)
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
func (l UserList[T]) BroadcastExcept(ctx context.Context, cmd outcmds.Cmd, data interface{}, e MessageSender) {
	l.BroadcastFilter(ctx, cmd, data, func(u MessageSender) bool {
		return u.Id() != e.Id()
	})
}

// BroadcastFilter sends the provided message to every user in the list for which
// the provided filter func returns true.
func (l UserList[T]) BroadcastFilter(ctx context.Context, cmd outcmds.Cmd, data interface{}, filter func(MessageSender) bool) {
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
