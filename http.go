package qws

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// large websocket messages can cause issues for some clients,
// so we convert larger messages to an http request

var messageMap = make(map[uuid.UUID][]byte)
var messageLock = make(chan struct{}, 1)

func AddMessage(msg []byte) uuid.UUID {
	messageLock <- struct{}{}
	defer func() { <-messageLock }()
	id := uuid.New()
	messageMap[id] = msg
	time.AfterFunc(5*time.Second, func() {
		messageLock <- struct{}{}
		defer func() { <-messageLock }()
		delete(messageMap, id)
	})
	return id
}

func GetMessage(id uuid.UUID) []byte {
	messageLock <- struct{}{}
	defer func() { <-messageLock }()
	msg, exists := messageMap[id]
	if exists {
		delete(messageMap, id)
	}
	return msg
}

func HandleHttpMessage(w http.ResponseWriter, r *http.Request) {
	lastSlash := strings.LastIndex(r.URL.Path, "/")
	if lastSlash == -1 || lastSlash == len(r.URL.Path)-1 {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(r.URL.Path[lastSlash+1:])
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}
	msg := GetMessage(id)
	if msg == nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	w.Write(msg)
}
