package qws

type ReturningFunc func(c *UserConn, m *RawMessage) interface{}

func (f ReturningFunc) ServeWS(c *UserConn, m *RawMessage) {
	if m.Id == 0 {
		f(c, m)
		return
	}
	r := f(c, m)
	c.mutex.Lock()
	_ = c.WriteJSON(Message{Id: m.Id, Data: r})
	c.mutex.Unlock()
}
