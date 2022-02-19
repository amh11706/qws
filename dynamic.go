package qws

import (
	"context"
	"encoding/json"
	"log"
	"reflect"
	"runtime"

	"github.com/amh11706/logger"
	"nhooyr.io/websocket/wsjson"
)

type DynamicHandler struct {
	elType reflect.Type
	f      reflect.Value
}

func (h *DynamicHandler) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	var out []reflect.Value
	if h.elType == nil {
		out = h.f.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(c)})
	} else {
		i := reflect.New(h.elType).Interface()
		err := json.Unmarshal(m.Data, i)
		if logger.Check(err) {
			f := runtime.FuncForPC(h.f.Pointer())
			file, line := runtime.FuncForPC(h.f.Pointer()).FileLine(f.Entry())
			log.Printf(
				"\x1b[36m%s(%d)\x1b[0m invalid ws parameter for func at %s:%d %v\n",
				c.User.Name, c.Copy, file, line, string(m.Data),
			)
			return
		}
		out = h.f.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(c), reflect.ValueOf(i).Elem()})
	}

	if m.Id > 0 && len(out) > 0 {
		_ = wsjson.Write(ctx, c.Conn.conn, Message{Id: m.Id, Data: out[0].Interface()})
		m.Id = 0
	} else if m.Id > 0 {
		_ = wsjson.Write(ctx, c.Conn.conn, Message{Id: m.Id})
		m.Id = 0
	}
}

func NewDynamicHandler(f interface{}) *DynamicHandler {
	h := &DynamicHandler{f: reflect.ValueOf(f)}
	t := h.f.Type()
	if t.NumIn() > 2 {
		h.elType = t.In(2)
	}
	return h
}
