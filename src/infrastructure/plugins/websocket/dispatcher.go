package plugin_websocket

import (
	"context"
	"encoding/json"
)

type WebSocketMessageHandler func(
    ctx context.Context,
    msg *WebsocketMessage,
)


func HandleWebSocketLoop(
	ws *ThreadSafeWriter,
	ctx context.Context,
	handlers map[string]WebSocketMessageHandler,
) {
	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			return
		}

		var msg WebsocketMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		handler, ok := handlers[msg.Event]
		if !ok {
			continue
		}

		handler(ctx, &msg)
	}
}