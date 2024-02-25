package qws

import (
	"context"

	"github.com/amh11706/logger"
	"github.com/amh11706/qws/outcmds"
	"github.com/amh11706/qws/slice"
	"github.com/gorilla/websocket"
)

type UserName struct {
	From  string     `json:"from"`
	Copy  int64      `json:"copy"`
	Admin AdminLevel `json:"admin"`
}

type MessageSender interface {
	SendMessage(ctx context.Context, m *websocket.PreparedMessage)
	Send(ctx context.Context, cmd outcmds.Cmd, data interface{})
	SendInfo(ctx context.Context, data string)
	SendRaw(ctx context.Context, data interface{})
	Close()
	Router() *Router
	CmdRouter() *CmdRouter
	AddCloseHook(context.Context, CloseHandler) error
	RemoveCloseHook(context.Context, CloseHandler) error
}

type UserConner interface {
	SetInLobby(int64)
	MessageSender
	UserInfoer
}

type UserList[T UserConner] map[int64]T

func (l UserList[T]) MarshalJSON() ([]byte, error) {
	return slice.NewVisibleCheckerMap(l).MarshalJSON()
}

type FilteredUserList[T UserInfoer] struct {
	slice.DefaultVisibleCheckerMap[int64, T]
	adminLevel AdminLevel
}

func (l FilteredUserList[T]) MarshalJSON() ([]byte, error) {
	return slice.MarshalMapAsSliceJSON(l, 64)
}

func (l FilteredUserList[T]) IsVisible(u T) bool {
	return !u.IsGhosted() || l.adminLevel >= u.AdminLevel()
}

func NewFilteredUserList[T UserInfoer](m map[int64]T, u AdminLevel) FilteredUserList[T] {
	return FilteredUserList[T]{slice.NewVisibleCheckerMap(m), u}
}

func (l UserList[T]) FilterForAdminLevel(al AdminLevel) FilteredUserList[T] {
	return NewFilteredUserList[T](l, al)
}

func (l UserList[T]) GroupByAdminLevel() map[AdminLevel]UserList[T] {
	m := make(map[AdminLevel]UserList[T])
	for _, u := range l {
		if _, ok := m[u.AdminLevel()]; !ok {
			m[u.AdminLevel()] = make(UserList[T])
		}
		m[u.AdminLevel()][u.Id()] = u
	}
	return m
}

func (l UserList[T]) BroadcastByAdminLevel(ctx context.Context) {
	for al, ul := range l.GroupByAdminLevel() {
		ul.Broadcast(ctx, outcmds.PlayerList, l.FilterForAdminLevel(al))
	}
}

// Broadcast sends the provided message to every user in the list.
func (l UserList[T]) Broadcast(ctx context.Context, cmd outcmds.Cmd, data interface{}) {
	m, err := PrepareJsonMessage(cmd, data)
	if logger.Check(err) {
		return
	}
	for _, u := range l {
		u.SendMessage(ctx, m)
	}
}

// BroadcastExcept sends the provided message to every user in the list except
// the provided user.
func (l UserList[T]) BroadcastExcept(ctx context.Context, cmd outcmds.Cmd, data interface{}, e UserConner) {
	l.BroadcastFilter(ctx, cmd, data, func(u UserConner) bool {
		return u.Id() != e.Id()
	})
}

// BroadcastFilter sends the provided message to every user in the list for which
// the provided filter func returns true.
func (l UserList[T]) BroadcastFilter(ctx context.Context, cmd outcmds.Cmd, data interface{}, filter func(UserConner) bool) {
	m, err := PrepareJsonMessage(cmd, data)
	if logger.Check(err) {
		return
	}
	for _, u := range l {
		if filter(u) {
			u.SendMessage(ctx, m)
		}
	}
}
