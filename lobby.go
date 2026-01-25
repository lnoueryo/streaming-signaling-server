package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type TrackParticipant struct{
    UserInfo
    StreamID string `json:"streamId"`
    TrackID  string `json:"trackId"`
}

type Lobby struct {
	ID           string
	listLock     sync.Mutex
	wsConnections map[*websocket.Conn]*ThreadSafeWriter
}

func (r *Lobby) BroadcastLobby(event string, data string) {
    for _, conn := range r.wsConnections {
        conn.Send(event, data)
    }
}

type Lobbies struct {
	item map[string]*Lobby
	lock sync.RWMutex
}

func (r *Lobbies) getLobby(id string) (*Lobby, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	room, ok := r.item[id]
	return room, ok
}

func (r *Lobbies) getOrCreate(id string) *Lobby {
    r.lock.Lock()
    defer r.lock.Unlock()

    room := r.item[id]
    if room != nil {
        return room
    }

    room = &Lobby{
        ID:            id,
        wsConnections: make(map[*websocket.Conn]*ThreadSafeWriter),
    }
    r.item[id] = room

    return room
}

func (r *Lobbies) deleteLobby(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.item, id)
}

func (r *Lobbies) cleanupEmptyLobby(id string) {
	room, ok := r.getLobby(id)
	if !ok {
		return
	}

	room.listLock.Lock()
	defer room.listLock.Unlock()

	if len(room.wsConnections) == 0 {
		r.deleteLobby(id)
	}
}
