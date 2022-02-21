package qws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/incmds"
	"github.com/amh11706/qws/outcmds"
	"github.com/gorilla/websocket"
)

type RawMessage struct {
	Cmd  incmds.Cmd      `json:"cmd,omitempty"`
	Id   uint32          `json:"id,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Message struct {
	Cmd  outcmds.Cmd `json:"cmd,omitempty"`
	Id   uint32      `json:"id,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type Info struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}

type Conn struct {
	conn     *websocket.Conn
	sendChan chan *Message
	closed   bool
}

func NewConn(ctx context.Context, conn *websocket.Conn) *Conn {
	c := &Conn{conn: conn, sendChan: make(chan *Message, 50)}
	go c.listen(ctx, 10*time.Second)
	return c
}

type Setting struct {
	Group string
	Name  string
}

type UserConn struct {
	*Conn
	User           *User
	Router         *Router
	CmdRouter      *CmdRouter
	Settings       map[string]byte
	SId            int64
	Copy           int64
	InLobby        int64
	OpenContainers map[uint32]struct{}
}

func (c *UserConn) PrintName() string {
	if c.User == nil {
		return ""
	}
	if c.Copy > 1 {
		return fmt.Sprintf("%s(%d)", c.User.Name, c.Copy)
	}
	return string(c.User.Name)
}

func (c *UserConn) UserName() UserName {
	return UserName{From: string(c.User.Name), Copy: c.Copy, Admin: int64(c.User.AdminLvl)}
}

func (c *Conn) Close() {
	c.conn.Close()
	c.closed = true
}

func (c *Conn) Send(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
	c.SendMessage(ctx, &Message{Cmd: cmd, Data: data})
}

func (c *Conn) SendSync(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
	c.SendMessageSync(ctx, &Message{Cmd: cmd, Data: data})
}

func (c *Conn) SendMessage(ctx context.Context, m *Message) {
	if c == nil || c.closed {
		return
	}
	if len(c.sendChan) == cap(c.sendChan) {
		logger.CheckStack(errors.New("sendChan is full."))
		c.Close()
	} else {
		c.sendChan <- m
	}
}

func (c *Conn) SendMessageSync(ctx context.Context, m *Message) {
	if c == nil || c.closed {
		return
	}
	if !c.closed && logger.Check(c.conn.WriteJSON(m)) {
		c.Close()
	}
}

func NewInfo(m string) *Info {
	return &Info{Message: m}
}

func (c *Conn) SendInfo(ctx context.Context, m string) {
	c.Send(ctx, outcmds.ChatMessage, &Info{Message: m})
}

func (c *Conn) listen(ctx context.Context, timeout time.Duration) {
	lastResponse := time.Now()
	c.conn.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	done := ctx.Done()
	timer := time.After(timeout / 2)
	for {
		if time.Since(lastResponse) > timeout {
			log.Println("Connection closed for missed pong")
			c.Close()
			return
		}
		select {
		case <-timer:
			break
		case m := <-c.sendChan:
			if !c.closed && logger.Check(c.conn.WriteJSON(m)) {
				c.Close()
			}
			continue
		case <-done:
			return
		}

		err := c.conn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
		if err != nil {
			c.Close()
			if err.Error() != "websocket: close sent" {
				log.Println("Connection closed for ping error:", err)
			}
			return
		}
		timer = time.After(timeout / 2)
	}
}
