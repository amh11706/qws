package qws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/incmds"
	"github.com/amh11706/qws/outcmds"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
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
}

func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{conn}
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

func (c *Conn) Send(ctx context.Context, cmd outcmds.Cmd, data interface{}) error {
	if c == nil {
		return nil
	}
	err := wsjson.Write(ctx, c.Conn, Message{Cmd: cmd, Data: data})
	if err != nil {
		c.Close(websocket.StatusAbnormalClosure, "Failed to write message.")
	}
	return err
}

func (c *Conn) SendMessage(ctx context.Context, m *Message) error {
	if c == nil {
		return nil
	}
	err := wsjson.Write(ctx, c.Conn, m)
	if err != nil {
		c.Close(websocket.StatusAbnormalClosure, "Failed to write message.")
	}
	return err
}

func NewInfo(m string) *Info {
	return &Info{Message: m}
}

func (c *Conn) SendInfo(ctx context.Context, m string) {
	logger.Check(c.Send(ctx, outcmds.ChatMessage, &Info{Message: m}))
}
