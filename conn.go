package qws

import (
	"encoding/json"
	"fmt"
	"sync"

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
	*websocket.Conn
	mutex sync.Mutex
}

type Setting struct {
	Group string
	Name  string
}

type UserConn struct {
	Conn           *Conn
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
	return fmt.Sprintf("%s(%d)", c.User.Name, c.Copy)
}

func (c *Conn) Send(cmd outcmds.Cmd, data interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	err := c.Conn.WriteJSON(Message{Cmd: cmd, Data: data})
	if err != nil {
		c.Close()
	}
	return err
}

func NewInfo(m string) *Info {
	return &Info{Message: m}
}

func (c *Conn) SendInfo(m string) {
	logger.Check(c.Send(outcmds.ChatMessage, &Info{Message: m}))
}

func (c *Conn) WriteMessage(mType int, data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.Conn.WriteMessage(mType, data)
}

func (c *Conn) SendPrepared(m *websocket.PreparedMessage) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.Conn.WritePreparedMessage(m)
}
