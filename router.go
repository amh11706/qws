package qws

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/incmds"
	"nhooyr.io/websocket/wsjson"
)

type Handler interface {
	ServeWS(context.Context, *UserConn, *RawMessage)
}

type HandlerFunc func(context.Context, *UserConn, *RawMessage)

func (f HandlerFunc) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	f(ctx, c, m)
}

type Router struct {
	routes map[incmds.Cmd]Handler
	lock   sync.Mutex
}

func (r *Router) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	if r.routes == nil {
		log.Println("No assigned handlers for user", c.User.Id)
		return
	}

	cmd := m.Cmd
	r.lock.Lock()
	if cmd > incmds.LobbyCmds && r.routes[incmds.LobbyCmds] != nil {
		cmd = incmds.LobbyCmds
	}
	if handler := r.routes[cmd]; handler != nil {
		r.lock.Unlock()
		handler.ServeWS(ctx, c, m)
		if m.Id > 0 {
			logger.Error("Sent missed return id for message:", m)
			_ = wsjson.Write(ctx, c.Conn.conn, Message{Id: m.Id})
			m.Id = 0
		}
	} else {
		log.Println("No matching handlers for user", c.User.Id, "and cmd", m.Cmd)
		fmt.Println(r.routes)
		r.lock.Unlock()
	}
}

func (r *Router) HandleFunc(command incmds.Cmd, h HandlerFunc) error {
	return r.Handle(command, HandlerFunc(h))
}

func (r *Router) Handle(command incmds.Cmd, h Handler) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.routes == nil {
		r.routes = make(map[incmds.Cmd]Handler)
	} else if _, set := r.routes[command]; set {
		return fmt.Errorf("AddCommand: command already registered: %d", command)
	}
	r.routes[command] = h
	return nil
}

func (r *Router) HandleDynamic(command incmds.Cmd, h interface{}) error {
	return r.Handle(command, NewDynamicHandler(h))
}

func (r *Router) HandleReturning(command incmds.Cmd, h func(ctx context.Context, c *UserConn, m *RawMessage) interface{}) error {
	return r.Handle(command, ReturningFunc(h))
}

func (r *Router) RemoveCommand(command incmds.Cmd) {
	if r == nil {
		return
	}
	r.lock.Lock()
	delete(r.routes, command)
	r.lock.Unlock()
}

func NewRouter() *Router {
	return &Router{}
}
