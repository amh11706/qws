package qws

import (
	"context"
	"encoding/json"
	"log"

	"github.com/amh11706/qws/outcmds"
	"github.com/gorilla/websocket"
)

type UserList map[int64]*UserConn

// Broadcast sends the provided message to every user in the list.
func (l UserList) Broadcast(ctx context.Context, cmd outcmds.Cmd, data interface{}) error {
	m, err := getPrepared(cmd, data)
	if err != nil {
		return err
	}
	for _, u := range l {
		err = u.SendPrepared(ctx, m)
		if err != nil {
			log.Println(err)
			u.Close()
		}
	}
	return nil
}

// BroadcastExcept sends the provided message to every user in the list except
// the provided user.
func (l UserList) BroadcastExcept(ctx context.Context, cmd outcmds.Cmd, data interface{}, e *UserConn) error {
	return l.BroadcastFilter(ctx, cmd, data, func(u *UserConn) bool {
		return u.SId != e.SId
	})
}

// BroadcastFilter sends the provided message to every user in the list for which
// the provided filter func returns true.
func (l UserList) BroadcastFilter(ctx context.Context, cmd outcmds.Cmd, data interface{}, filter func(*UserConn) bool) error {
	m, err := getPrepared(cmd, data)
	if err != nil {
		return err
	}
	for _, u := range l {
		if !filter(u) {
			continue
		}
		err = u.SendPrepared(ctx, m)
		if err != nil {
			log.Println(err)
			u.Close()
		}
	}
	return nil
}

func getPrepared(cmd outcmds.Cmd, data interface{}) (*websocket.PreparedMessage, error) {
	b, err := json.Marshal(Message{Cmd: cmd, Data: data})
	if err != nil {
		return nil, err
	}
	return websocket.NewPreparedMessage(websocket.TextMessage, b)
}
