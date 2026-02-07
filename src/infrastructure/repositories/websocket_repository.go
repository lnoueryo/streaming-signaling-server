package repositories

import (
	"sync"

	"streaming-signaling.jounetsism.biz/src/domain/entities"
	plugin_websocket "streaming-signaling.jounetsism.biz/src/infrastructure/plugins/websocket"
)

type WebsocketRepository struct {
	lock sync.Mutex
	lobbies map[string]map[string]*WSClient
}

type WSClient struct {
	*entities.WSClient
	*plugin_websocket.ThreadSafeWriter
}

func NewWebsocketRepository() *WebsocketRepository {
	return &WebsocketRepository{
		lock: sync.Mutex{},
		lobbies: make(map[string]map[string]*WSClient),
	}
}

func (wr *WebsocketRepository) Create(wsclient *entities.WSClient, ws *plugin_websocket.ThreadSafeWriter) *entities.WSClient {
    wr.lock.Lock()
    defer wr.lock.Unlock()
    lobby, ok := wr.lobbies[wsclient.LobbyID];if !ok {
		lobby = make(map[string]*WSClient)
		wr.lobbies[wsclient.LobbyID] = lobby
	}

	lobby[wsclient.ConnID] = &WSClient{
		wsclient,
		ws,
	}
    return wsclient
}

func (wr *WebsocketRepository) Delete(lotteryId string, connId string) {
    lobby, ok := wr.lobbies[lotteryId];if !ok {
        return
    }
	delete(lobby, connId)
	if len(lobby) == 0 {
		delete(wr.lobbies, lotteryId)
	}
}

func (lr *WebsocketRepository) Lock() {
    lr.lock.Lock()
}

func (lr *WebsocketRepository) Unlock() {
    lr.lock.Unlock()
}