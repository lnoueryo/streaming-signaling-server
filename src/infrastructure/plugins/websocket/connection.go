package plugin_websocket

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	application_shared "streaming-signaling.jounetsism.biz/src/application/shared"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(c *http.Request) bool { return true },
}

type ThreadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

type WebsocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func NewThreadSafeWriter(c *gin.Context) *ThreadSafeWriter {
	unsafeConn, err := Upgrader.Upgrade(c.Writer, c.Request, nil);if err != nil {
        return nil
    }
	return &ThreadSafeWriter{
		Conn:   unsafeConn,
		Mutex:  sync.Mutex{},
	}
}

func (t *ThreadSafeWriter) Send(event string, data string) error {
	t.Lock()
	defer t.Unlock()
	return t.Conn.WriteJSON(WebsocketMessage{event, data})
}

type WSClient struct {
    application_shared.AuthUser        // Domain のフィールドを埋め込む
    *ThreadSafeWriter
}

func NewWSClient(
	// ID string,
	// Name string,
	// Email string,
	// Image string,
	User application_shared.AuthUser,
	conn *ThreadSafeWriter,
) *WSClient {
	return &WSClient{
		User,
		conn,
	}
}