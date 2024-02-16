package qws

import (
	"context"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/outcmds"
)

type UserName struct {
	From  string `json:"from"`
	Copy  int64  `json:"copy"`
	Admin int64  `json:"admin"`
}

type UserList map[int64]*UserConn

// Broadcast sends the provided message to every user in the list.
func (l UserList) Broadcast(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
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
func (l UserList) BroadcastExcept(ctx context.Context, cmd outcmds.Cmd, data interface{}, e *UserConn) {
	l.BroadcastFilter(ctx, cmd, data, func(u *UserConn) bool {
		return u.SId != e.SId
	})
}

// BroadcastFilter sends the provided message to every user in the list for which
// the provided filter func returns true.
func (l UserList) BroadcastFilter(ctx context.Context, cmd outcmds.Cmd, data interface{}, filter func(*UserConn) bool) {
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
