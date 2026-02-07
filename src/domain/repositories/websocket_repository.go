package repository_interfaces

import (
	"streaming-signaling.jounetsism.biz/src/domain/entities"
	plugin_websocket "streaming-signaling.jounetsism.biz/src/infrastructure/plugins/websocket"
)

type IWebsocketRepository interface{
	Create(wsclient *entities.WSClient, conn *plugin_websocket.ThreadSafeWriter) *entities.WSClient
	Delete(lobbyId string, connId string)
}