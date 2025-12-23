package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type ThreadSafeWriter struct {
	UserID string
	*websocket.Conn
	sync.Mutex
}

type WebsocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}



func (t *ThreadSafeWriter) Send(event string, data string) error {
	t.Lock()
	defer t.Unlock()
	return t.Conn.WriteJSON(WebsocketMessage{event, data})
}