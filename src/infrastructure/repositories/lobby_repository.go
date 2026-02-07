package repositories

import (
	"sync"

	"streaming-signaling.jounetsism.biz/src/domain/entities"
)

type LobbyRepository struct {
	lock sync.Mutex
	item map[string]*entities.Lobby
}

func NewLobbyRepository() *LobbyRepository {
	return &LobbyRepository{
		lock: sync.Mutex{},
		item: make(map[string]*entities.Lobby),
	}
}

func (lr *LobbyRepository) GetOrCreate(id string) *entities.Lobby {
    lr.lock.Lock()
    defer lr.lock.Unlock()

    lobby, ok := lr.item[id];if !ok {
        return lobby
    }

    lobby = entities.NewLobby()
    lr.item[id] = lobby

    return lobby
}

func (lr *LobbyRepository) DeleteIfEmptity(id string) {
    lobby, ok := lr.item[id];if !ok {
        return
    }
	if len(lobby.WSClients) == 0 {
		delete(lr.item, id)
	}
}

func (lr *LobbyRepository) Lock() {
    lr.lock.Lock()
}

func (lr *LobbyRepository) Unlock() {
    lr.lock.Unlock()
}