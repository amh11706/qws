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
	conn     *websocket.Conn
	sendChan chan *websocket.PreparedMessage
	closed   bool
	ip       string
}

func NewConn(conn *websocket.Conn, ip string) *Conn {
	c := &Conn{conn: conn, sendChan: make(chan *websocket.PreparedMessage, 50), ip: ip}
	return c
}

func (c *Conn) ListenWrite(ctx context.Context) {
	go c.listenWrite(ctx, 10*time.Second)
}

type UserInfoer interface {
	Id() int64
	UserId() int64
	AdminLevel() AdminLevel
	// Name is the user's raw name, not formatted for display
	Name() string
	// FilterName is used as a unique id when the actual id is not known
	FilterName() string
	// UserName used for chat messages
	UserName() UserName
	// PrintName used for boat names and server logs
	PrintName() string
	InLobby() int64
	IsBot() bool
	IsGhosted() bool
	IsIgnored() bool
	IsGuest() bool
	Lock() *lock.Lock
}

type UserConn struct {
	*Conn
	*User
	router     *Router
	cmdRouter  *CmdRouter
	SId        int64
	Copy       int64
	inLobby    int64
	Ghosted    bool
	closeHooks []CloseHandler
}

func NewUserConn(user *User, conn *websocket.Conn, ip string) *UserConn {
	uConn := &UserConn{
		User:       user,
		Conn:       NewConn(conn, ip),
		router:     &Router{},
		cmdRouter:  &CmdRouter{},
		closeHooks: make([]CloseHandler, 0, 4),
	}
	return uConn
}

func (c *UserConn) IsIgnored() bool {
	return false
}

func (c *UserConn) Ip() string {
	return c.ip
}

func (c *UserConn) IsBot() bool {
	return c.Conn == nil
}

func (c *UserConn) IsGhosted() bool {
	return c.Ghosted
}

// hopefully never needed, but this is better than crashing
var fallbackLock = &lock.Lock{}

func (c *UserConn) Lock() *lock.Lock {
	if c == nil || c.User == nil {
		return fallbackLock
	}
	return c.User.Lock
}

func (c *UserConn) Id() int64 {
	if c == nil {
		return 0
	}
	return c.SId
}

func (c *UserConn) Router() *Router {
	if c == nil {
		return nil
	}
	return c.router
}

func (c *UserConn) CmdRouter() *CmdRouter {
	return c.cmdRouter
}

func (c *UserConn) AdminLevel() AdminLevel {
	if c == nil {
		return 0
	}
	return c.User.AdminLvl
}

func (c *UserConn) InLobby() int64 {
	return c.inLobby
}

func (c *UserConn) SetInLobby(id int64) {
	c.inLobby = id
}

func (u *UserConn) UserId() int64 {
	if u == nil {
		return 0
	}
	return int64(u.User.Id)
}

// FilterName is used as a unique id when the actual id is not known
func (u *UserConn) FilterName() string {
	name := u.Name()
	if name == "Guest" {
		// Guest requires the copy number to identify a unique user since many guests can have the same name
		return u.PrintName()
	}
	return name
}

// Name is the user's raw name, not formatted for display
func (u *UserConn) Name() string {
	if u == nil || u.User == nil {
		return ""
	}
	return string(u.User.Name)
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
	defer c.User.Lock.Unlock()
	if c.closed {
		return errors.New("Connection closed")
	}
	c.closeHooks = append(c.closeHooks, ch)
	return nil
}

func (c *UserConn) RemoveCloseHook(ctx context.Context, ch CloseHandler) error {
	if ch == nil {
		return nil
	}
	if err := c.User.Lock.Lock(ctx); err != nil {
		return err
	}
	defer c.User.Lock.Unlock()
	for i, h := range c.closeHooks {
		if ch == h {
			last := len(c.closeHooks) - 1
			c.closeHooks[last], c.closeHooks[i] = c.closeHooks[i], c.closeHooks[last]
			c.closeHooks = c.closeHooks[:last]
			break
		}
	}
	return nil
}

// PrintName used for boat names and server logs
func (c *UserConn) PrintName() string {
	if c == nil || c.User == nil {
		return "Missing User"
	}
	if c.Copy > 1 || c.User.Name == "Guest" {
		return fmt.Sprintf("%s(%d)", c.User.Name, c.Copy)
	}
	return string(c.User.Name)
}

// UserName used for chat messages
func (c *UserConn) UserName() UserName {
	return UserName{From: c.Name(), Copy: c.Copy, Admin: c.AdminLevel(), Decoration: string(c.User.Decoration)}
}

func (c *UserConn) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c.User.Lock.MustLock(ctx)
	defer c.User.Lock.Unlock()
	chs := c.closeHooks
	c.closeHooks = nil
	c.closed = true
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
			message := fmt.Sprintf("Panic serving %s: %v", uConn.PrintName(), r)
			logger.CheckStack(errors.New(message))
			uConn.SendInfo(ctx, "Something went wrong...")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	uConn.router.ServeWS(ctx, uConn, m)
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
