package controllers

import (
	"github.com/gin-gonic/gin"
	"streaming-signaling.jounetsism.biz/src/domain/entities"
	repository_interfaces "streaming-signaling.jounetsism.biz/src/domain/repositories"
	plugin_websocket "streaming-signaling.jounetsism.biz/src/infrastructure/plugins/websocket"
	"streaming-signaling.jounetsism.biz/src/presentation/controllers/http/shared"
	websocket_controllers "streaming-signaling.jounetsism.biz/src/presentation/controllers/websocket/media"
)

type LobbyController struct{
    websocketRepository repository_interfaces.IWebsocketRepository
}

func NewLobbyController(
    websocketRepository repository_interfaces.IWebsocketRepository,
) *LobbyController {
    return &LobbyController{
        websocketRepository: websocketRepository,
    }
}

func (lc *LobbyController)ConnectWebsocket(c *gin.Context) {
    user := shared.GetUser(c)
    lobbyId := c.Param("lobbyId")
    ws := plugin_websocket.NewThreadSafeWriter(c)

    wsclient := entities.NewWSClient(user.ID, user.Email, user.Name, user.Image, lobbyId)
    lc.websocketRepository.Create(wsclient, ws)

    defer func() {
        lc.websocketRepository.Delete(lobbyId, wsclient.ID)
        ws.Close()
    }()

    wc := websocket_controllers.NewMediaController()

    handlers := map[string]plugin_websocket.WebSocketMessageHandler{
        "offer":     wc.CreatePeer,
        "candidate": wc.AddCandidate,
        "answer":    wc.SetAnswer,
    }
    plugin_websocket.HandleWebSocketLoop(ws, c, handlers)

    // msg := &WebsocketMessage{}

    // for {
    //     // ---------- read WS message ----------
    //     _, raw, err := ws.ReadMessage()
    //     if err != nil {
    //         log.Errorf("WS read error: %v", err)
    //         return
    //     }

    //     if err := json.Unmarshal(raw, msg); err != nil {
    //         log.Errorf("json unmarshal error: %v", err)
    //         return
    //     }

    //     switch msg.Event {
    //     case "offer":
    //         if err := CreatePeer(lobbyId, user); err != nil {
    //             log.Errorf("create peer error: %v", err)
    //             return
    //         }
    //     case "candidate":
    //         if err := AddCandidate(lobbyId, user, msg.Data); err != nil {
    //             log.Errorf("add candidate error: %v", err)
    //             return
    //         }
    //     case "answer":
    //         if err := SetAnswer(lobbyId, user, msg.Data); err != nil {
    //             log.Errorf("set answer error: %v", err)
    //             return
    //         }
    //     }
    // }
}