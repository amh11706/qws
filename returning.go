package qws

import "context"

type ReturningFunc func(ctx context.Context, c *UserConn, m *RawMessage) interface{}

func (f ReturningFunc) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	if m.Id == 0 {
		f(ctx, c, m)
		return
	}
	r := f(ctx, c, m)
	c.mutex.MustLock(ctx)
	_ = c.WriteJSON(Message{Id: m.Id, Data: r})
	m.Id = 0
	c.mutex.Unlock()
}
