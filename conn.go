package qws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
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
	sendChan chan *websocket.PreparedMessage
	closed   bool
}

func NewConn(ctx context.Context, conn *websocket.Conn) *Conn {
	c := &Conn{conn: conn, sendChan: make(chan *websocket.PreparedMessage, 50)}
	return c
}

func (c *Conn) ListenWrite(ctx context.Context) {
	go c.listenWrite(ctx, 10*time.Second)
}

type Setting struct {
	Group string
	Name  string
}

type UserConn struct {
	*Conn
	User       *User
	Router     *Router
	CmdRouter  *CmdRouter
	Settings   map[string]byte
	SId        int64
	Copy       int64
	InLobby    int64
	closeHooks []CloseHandler
}

func NewUserConn(ctx context.Context, user *User, conn *websocket.Conn) *UserConn {
	uConn := &UserConn{
		User:       user,
		Conn:       NewConn(ctx, conn),
		Settings:   make(map[string]byte),
		Router:     &Router{},
		CmdRouter:  &CmdRouter{},
		closeHooks: make([]CloseHandler, 0, 4),
	}
	return uConn
}

type Player struct {
	UserName
	SId int64 `json:"sId"`
}

func (c *UserConn) MarshalJSON() ([]byte, error) {
	return json.Marshal(Player{UserName: c.UserName(), SId: c.SId})
}

func (c *UserConn) AddCloseHook(ctx context.Context, ch CloseHandler) error {
	if ch == nil {
		return nil
	}
	if err := c.User.Lock.Lock(ctx); err != nil {
		return err
	}
	if c.closed {
		return errors.New("Connection closed")
	}
	c.closeHooks = append(c.closeHooks, ch)
	c.User.Lock.Unlock()
	return nil
}

func (c *UserConn) RemoveCloseHook(ctx context.Context, ch CloseHandler) error {
	if ch == nil {
		return nil
	}
	if err := c.User.Lock.Lock(ctx); err != nil {
		return err
	}
	for i, h := range c.closeHooks {
		if ch == h {
			last := len(c.closeHooks) - 1
			c.closeHooks[last], c.closeHooks[i] = c.closeHooks[i], c.closeHooks[last]
			c.closeHooks = c.closeHooks[:last]
			break
		}
	}
	c.User.Lock.Unlock()
	return nil
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

func (c *UserConn) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.User.Lock.MustLock(ctx)
	chs := c.closeHooks
	c.closeHooks = nil
	c.closed = true
	c.User.Lock.Unlock()
	for i := len(chs) - 1; i >= 0; i-- {
		(*chs[i])(ctx, c)
	}
	c.conn.Close()
}

func (c *Conn) Close() {
	c.conn.Close()
	c.closed = true
}

func (c *Conn) Send(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
	if c == nil || c.closed {
		return
	}
	m, err := PrepareJsonMessage(cmd, data)
	if logger.Check(err) {
		return
	}
	c.SendMessage(ctx, m)
}

func (c *Conn) SendSync(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
	c.SendMessageSync(ctx, &Message{Cmd: cmd, Data: data})
}

func PrepareJsonMessage(cmd outcmds.Cmd, data interface{}) (*websocket.PreparedMessage, error) {
	m := &Message{Cmd: cmd, Data: data}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return websocket.NewPreparedMessage(websocket.TextMessage, b)
}

func (c *Conn) SendRaw(ctx context.Context, data interface{}) {
	if c == nil || c.closed {
		return
	}
	b, err := json.Marshal(data)
	if logger.Check(err) {
		return
	}
	m, err := websocket.NewPreparedMessage(websocket.TextMessage, b)
	if logger.Check(err) {
		return
	}
	c.SendMessage(ctx, m)
}

func (c *Conn) SendMessage(ctx context.Context, m *websocket.PreparedMessage) {
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

func (uConn *UserConn) ListenRead(ctx context.Context) {
	for {
		m := &RawMessage{}
		err := uConn.conn.ReadJSON(m)
		wsErr, ok := err.(*websocket.CloseError)
		if ok && wsErr.Code == websocket.CloseGoingAway {
			return
		}
		if logger.Check(err) {
			return
		}

		go uConn.handleMessage(ctx, m)
	}
}

func (uConn *UserConn) handleMessage(ctx context.Context, m *RawMessage) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("WS Panic serving", uConn.PrintName()+":", r)
			debug.PrintStack()
			uConn.SendInfo(ctx, "Something went wrong...")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	uConn.Router.ServeWS(ctx, uConn, m)
	cancel()
}

func (c *Conn) listenWrite(ctx context.Context, timeout time.Duration) {
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
			if !c.closed && logger.Check(c.conn.WritePreparedMessage(m)) {
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
