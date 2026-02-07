package entities

import (
	"sync"

)

type Lobby struct {
	ID           string
	listLock     sync.Mutex
	WSClients map[string]*WSClient
}

func NewLobby() *Lobby {
	return &Lobby{
		WSClients: make(map[string]*WSClient),
	}
}

func (l *Lobby) Lock() {
    l.listLock.Lock()
}

func (l *Lobby) Unlock() {
    l.listLock.Unlock()
}