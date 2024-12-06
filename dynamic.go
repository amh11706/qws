package qws

import (
	"context"
	"encoding/json"
	"log"
	"reflect"

	"github.com/amh11706/logger"
)

type DynamicFunc[In any, Out any] func(context.Context, *UserConn, In) Out

type DynamicHandler[T any, R any] struct {
	f DynamicFunc[T, R]
}

func (h *DynamicHandler[T, R]) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	in := new(T)
	if len(m.Data) > 0 {
		err := json.Unmarshal(m.Data, in)
		if logger.Check(err) {
			log.Printf(
				"\x1b[36m%s\x1b[0m invalid ws parameter for cmd %d: %v\n",
				c.PrintName(), m.Cmd, string(m.Data),
			)
			return
		}
	}
	out := h.f(ctx, c, *in)

	outValue := reflect.ValueOf(out)
	nilOutput := outValue.IsZero()
	if m.Id > 0 && !nilOutput {
		c.SendRaw(ctx, &Message{Id: m.Id, Data: out})
		m.Id = 0
	} else if m.Id > 0 {
		c.SendRaw(ctx, &Message{Id: m.Id})
		m.Id = 0
	} else if !nilOutput {
		if outValue.Kind() == reflect.String {
			c.SendInfo(ctx, outValue.String())
		}
	}
}

func NewDynamicHandler[T any, R any](f DynamicFunc[T, R]) *DynamicHandler[T, R] {
	h := &DynamicHandler[T, R]{f: f}
	return h
}
