package qws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/incmds"
	"github.com/amh11706/qws/lock"
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
	*websocket.Conn
	mutex *lock.Lock
}

func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{conn, lock.NewLock()}
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

func (c *Conn) Send(ctx context.Context, cmd outcmds.Cmd, data interface{}) error {
	if c == nil {
		return nil
	}
	if err := c.mutex.Lock(ctx); err != nil {
		return err
	}
	defer c.mutex.Unlock()
	err := c.WriteJSON(Message{Cmd: cmd, Data: data})
	if err != nil {
		c.Close()
	}
	return err
}

func NewInfo(m string) *Info {
	return &Info{Message: m}
}

func (c *Conn) SendInfo(ctx context.Context, m string) {
	logger.Check(c.Send(ctx, outcmds.ChatMessage, &Info{Message: m}))
}

func (c *Conn) WriteMessage(ctx context.Context, mType int, data []byte) error {
	if c == nil {
		return nil
	}
	if err := c.mutex.Lock(ctx); err != nil {
		return err
	}
	defer c.mutex.Unlock()
	return c.Conn.WriteMessage(mType, data)
}

func (c *Conn) SendPrepared(ctx context.Context, m *websocket.PreparedMessage) error {
	if c == nil {
		return nil
	}
	if err := c.mutex.Lock(ctx); err != nil {
		return err
	}
	defer c.mutex.Unlock()
	return c.WritePreparedMessage(m)
}
