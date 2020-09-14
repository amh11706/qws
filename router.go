package qws

import (
	"errors"
	"log"
	"sync"
)

type Handler interface {
	ServeWS(*UserConn, *RawMessage)
}

type HandlerFunc func(*UserConn, *RawMessage)

func (f HandlerFunc) ServeWS(c *UserConn, m *RawMessage) {
	f(c, m)
}

type Router struct {
	routes map[string]Handler
	lock   sync.Mutex
}

func (r *Router) ServeWS(c *UserConn, m *RawMessage) {
	if r.routes == nil {
		log.Println("No assigned handlers for user", c.User.Id)
		return
	}
	var bestMatch Handler
	var bestLen = 0

	r.lock.Lock()
	for command, handler := range r.routes {
		if command == m.Cmd {
			bestMatch = handler
			break
		}
		if len(command) < len(m.Cmd) && m.Cmd[:len(command)] == command {
			if bestMatch == nil || bestLen < len(command) {
				bestLen, bestMatch = len(command), handler
			}
		}
	}
	r.lock.Unlock()

	if bestMatch != nil {
		if bestLen > 0 && len(m.Cmd) > bestLen {
			m.Cmd = m.Cmd[bestLen:]
		}
		bestMatch.ServeWS(c, m)
	} else {
		log.Println("No matching handlers for user", c.User.Id, "and cmd", m.Cmd)
	}
}

func (r *Router) HandleFunc(command string, h func(c *UserConn, m *RawMessage)) error {
	return r.Handle(command, HandlerFunc(h))
}

func (r *Router) Handle(command string, h Handler) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.routes == nil {
		r.routes = make(map[string]Handler)
	} else if _, set := r.routes[command]; set {
		return errors.New("AddCommand: command already registered: " + command)
	}
	r.routes[command] = h
	return nil
}

func (r *Router) RemoveCommand(command string) {
	r.lock.Lock()
	delete(r.routes, command)
	r.lock.Unlock()
}

func NewRouter() *Router {
	return &Router{}
}
