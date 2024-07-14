package qws

import (
	"context"
)

type ReturningFunc func(ctx context.Context, c *UserConn, m *RawMessage) interface{}

func (f ReturningFunc) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	if m.Id == 0 {
		r := f(ctx, c, m)
		if v, ok := r.(string); ok && v != "" {
			c.SendInfo(ctx, v)
		}
		return
	}
	r := f(ctx, c, m)
	c.SendRaw(ctx, &Message{Id: m.Id, Data: r})
	m.Id = 0
}
