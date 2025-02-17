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
	"github.com/amh11706/qws/safe"
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

const connectionTimeout = 10 * time.Second

type Conn struct {
	conn                *websocket.Conn
	sendChan            chan *websocket.PreparedMessage
	closed              bool
	ip                  string
	pingTimer           *time.Ticker
	lastMessageReceived time.Time
}

func NewConn(conn *websocket.Conn, ip string) *Conn {
	c := &Conn{
		conn:                conn,
		sendChan:            make(chan *websocket.PreparedMessage, 50),
		ip:                  ip,
		lastMessageReceived: time.Now(),
		pingTimer:           time.NewTicker(connectionTimeout / 2),
	}
	return c
}

func (c *Conn) ListenWrite(ctx context.Context) {
	go c.listenWrite(ctx)
}

type UserInfoer interface {
	Id() int64
	User() *User
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
	user       *User
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
		user:       user,
		Conn:       NewConn(conn, ip),
		router:     &Router{},
		cmdRouter:  &CmdRouter{},
		closeHooks: make([]CloseHandler, 0, 4),
	}
	return uConn
}

func NewBot(id int64, user *User) *UserConn {
	return &UserConn{SId: id, user: user}
}

func (c *UserConn) User() *User {
	return c.user
}

func (c *UserConn) IsGuest() bool {
	return c.user.IsGuest()
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
	if c == nil || c.user == nil {
		return fallbackLock
	}
	return c.user.Lock
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
	return c.user.AdminLvl
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
	return int64(u.user.Id)
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
	if u == nil || u.user == nil {
		return ""
	}
	return string(u.user.Name)
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
	if err := c.user.Lock.Lock(ctx); err != nil {
		return err
	}
	defer c.user.Lock.Unlock()
	if c.closed {
		return errors.New("connection closed")
	}
	c.closeHooks = append(c.closeHooks, ch)
	return nil
}

func (c *UserConn) RemoveCloseHook(ctx context.Context, ch CloseHandler) error {
	if ch == nil {
		return nil
	}
	if err := c.user.Lock.Lock(ctx); err != nil {
		return err
	}
	defer c.user.Lock.Unlock()
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
	if c == nil || c.user == nil {
		return "Missing User"
	}
	if c.Copy > 1 || c.user.Name == "Guest" {
		return fmt.Sprintf("%s(%d)", c.user.Name, c.Copy)
	}
	return string(c.user.Name)
}

// UserName used for chat messages
func (c *UserConn) UserName() UserName {
	return UserName{From: c.Name(), Copy: c.Copy, Admin: c.AdminLevel(), Decoration: string(c.user.Decoration)}
}

func (c *UserConn) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	c.user.Lock.MustLock(ctx)
	defer c.user.Lock.Unlock()
	chs := c.closeHooks
	c.closeHooks = nil
	c.closed = true
	for i := len(chs) - 1; i >= 0; i-- {
		safe.GoWithValue(func(h CloseHandler) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			(*h)(ctx, c)
		}, chs[i], nil)
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

const MaxJsonSize = 100000

func PrepareJsonMessage(cmd outcmds.Cmd, data interface{}) (*websocket.PreparedMessage, error) {
	m := &Message{Cmd: cmd, Data: data}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxJsonSize {
		id := AddMessage(b)
		return websocket.NewPreparedMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"httpid":"%s"}`, id.String())))
	}
	return websocket.NewPreparedMessage(websocket.TextMessage, b)
}

func (c *Conn) SendRaw(ctx context.Context, data *Message) {
	if c == nil || c.closed {
		return
	}
	b, err := json.Marshal(data)
	if logger.Check(err) {
		return
	}
	if len(b) > MaxJsonSize {
		id := AddMessage(b)
		b = []byte(fmt.Sprintf(`{"httpid":"%s"}`, id.String()))
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
		logger.CheckStack(errors.New("sendChan is full"))
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
	if len(m) == 0 {
		return
	}
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
		uConn.pingTimer.Reset(connectionTimeout / 2)
		uConn.lastMessageReceived = time.Now()
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

var pingMessage, _ = websocket.NewPreparedMessage(websocket.PingMessage, []byte("keepalive"))

func (c *Conn) listenWrite(ctx context.Context) {
	c.conn.SetPongHandler(func(msg string) error {
		c.lastMessageReceived = time.Now()
		return nil
	})

	done := ctx.Done()
	for {
		if time.Since(c.lastMessageReceived) > connectionTimeout {
			log.Println("Connection closed for missed pong")
			c.Close()
			return
		}
		select {
		case <-c.pingTimer.C:
			err := c.conn.WritePreparedMessage(pingMessage)
			if err != nil {
				c.Close()
				if err.Error() != "websocket: close sent" {
					log.Println("Connection closed for ping error:", err)
				}
				return
			}
		case m := <-c.sendChan:
			if !c.closed && logger.Check(c.conn.WritePreparedMessage(m)) {
				c.Close()
			}
			continue
		case <-done:
			return
		}
	}
}
