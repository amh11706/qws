package qws

import (
	"fmt"
	"log"
	"sync"

	"github.com/amh11706/qws/incmds"
)

type Handler interface {
	ServeWS(*UserConn, *RawMessage)
}

type HandlerFunc func(*UserConn, *RawMessage)

func (f HandlerFunc) ServeWS(c *UserConn, m *RawMessage) {
	f(c, m)
}

type Router struct {
	routes map[incmds.Cmd]Handler
	lock   sync.Mutex
}

func (r *Router) ServeWS(c *UserConn, m *RawMessage) {
	if r.routes == nil {
		log.Println("No assigned handlers for user", c.User.Id)
		return
	}

	cmd := m.Cmd
	if cmd > incmds.LobbyCmds && r.routes[incmds.LobbyCmds] != nil {
		cmd = incmds.LobbyCmds
	}
	if handler := r.routes[cmd]; handler != nil {
		handler.ServeWS(c, m)
	} else {
		log.Println("No matching handlers for user", c.User.Id, "and cmd", m.Cmd)
		fmt.Println(r.routes)
	}
}

func (r *Router) HandleFunc(command incmds.Cmd, h func(c *UserConn, m *RawMessage)) error {
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

func (r *Router) RemoveCommand(command incmds.Cmd) {
	r.lock.Lock()
	delete(r.routes, command)
	r.lock.Unlock()
}

func NewRouter() *Router {
	return &Router{}
}
