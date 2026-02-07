package websocket_controllers

import (
	"context"

	plugin_websocket "streaming-signaling.jounetsism.biz/src/infrastructure/plugins/websocket"
)

type MediaController struct {
}

func NewMediaController() *MediaController {
	return &MediaController{}
}

func(mc *MediaController) CreatePeer(ctx context.Context, msg *plugin_websocket.WebsocketMessage) {
	// grpc_clientのメソッドをここに記述
}

func(mc *MediaController) AddCandidate(ctx context.Context, msg *plugin_websocket.WebsocketMessage) {

}

func(mc *MediaController) SetAnswer(ctx context.Context, msg *plugin_websocket.WebsocketMessage) {

}
