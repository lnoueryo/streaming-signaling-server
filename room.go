// ==============================
// rooms.go
// ==============================

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

type Room struct {
	ID           string
	listLock     sync.Mutex
	wsConnections map[*websocket.Conn]*ThreadSafeWriter
}

func (r *Room) BroadcastLobby(event string, data string) {
    for _, conn := range r.wsConnections {
        conn.Send(event, data)
    }
}

type Rooms struct {
	item map[string]*Room
	lock sync.RWMutex
}

func (r *Rooms) getRoom(id string) (*Room, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	room, ok := r.item[id]
	return room, ok
}

func (r *Rooms) getOrCreate(id string) *Room {
    r.lock.Lock()
    defer r.lock.Unlock()

    room := r.item[id]
    if room != nil {
        return room
    }

    room = &Room{
        ID:            id,
        wsConnections: make(map[*websocket.Conn]*ThreadSafeWriter),
    }
    r.item[id] = room

    return room
}

func (r *Rooms) deleteRoom(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.item, id)
}

func (r *Rooms) cleanupEmptyRoom(id string) {
	room, ok := r.getRoom(id)
	if !ok {
		return
	}

	room.listLock.Lock()
	defer room.listLock.Unlock()

	if len(room.wsConnections) == 0 {
		r.deleteRoom(id)
	}
}
