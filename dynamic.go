package qws

import (
	"encoding/json"
	"log"
	"reflect"
	"runtime"
)

type DynamicHandler struct {
	elType reflect.Type
	f      reflect.Value
}

func (h *DynamicHandler) ServeWS(c *UserConn, m *RawMessage) {
	var out []reflect.Value
	if h.elType == nil {
		out = h.f.Call([]reflect.Value{reflect.ValueOf(c)})
	} else {
		i := reflect.New(h.elType).Interface()
		err := json.Unmarshal(m.Data, i)
		if err != nil {
			f := runtime.FuncForPC(h.f.Pointer())
			file, line := runtime.FuncForPC(h.f.Pointer()).FileLine(f.Entry())
			log.Printf(
				"\x1b[36m%s(%d)\x1b[0m invalid ws parameter for func at %s:%d %v\n",
				c.User.Name, c.Copy, file, line, string(m.Data),
			)
			return
		}
		out = h.f.Call([]reflect.Value{reflect.ValueOf(c), reflect.ValueOf(i).Elem()})
	}

	if m.Id > 0 && len(out) > 0 {
		c.Conn.mutex.Lock()
		_ = c.Conn.WriteJSON(Message{Id: m.Id, Data: out[0].Interface()})
		c.Conn.mutex.Unlock()
	} else if m.Id > 0 {
		c.Conn.mutex.Lock()
		_ = c.Conn.WriteJSON(Message{Id: m.Id})
		c.Conn.mutex.Unlock()
	}
}

func NewDynamicHandler(f interface{}) *DynamicHandler {
	h := &DynamicHandler{f: reflect.ValueOf(f)}
	t := h.f.Type()
	if t.NumIn() > 1 {
		h.elType = t.In(1)
	}
	return h
}