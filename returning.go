package qws

import (
	"context"

	"nhooyr.io/websocket/wsjson"
)

type ReturningFunc func(ctx context.Context, c *UserConn, m *RawMessage) interface{}

func (f ReturningFunc) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	if m.Id == 0 {
		f(ctx, c, m)
		return
	}
	r := f(ctx, c, m)
	_ = wsjson.Write(ctx, c.Conn.conn, Message{Id: m.Id, Data: r})
	m.Id = 0
}
