package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

func main() {
	// Ginエンジンのインスタンスを作成
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	InitFirebase()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})
	wsAuth := r.Group("/ws")
	wsAuth.Use(FirebaseWebsocketAuth())
	wsAuth.GET("/live/:roomId", websocketBroadcastHandler)
	wsAuth.GET("/live/:roomId/viewer", websocketViewerHandler)

	httpAuth := r.Group("/")
	httpAuth.Use(FirebaseHttpAuth())
	httpAuth.GET("/room/:roomId/user", checkIfCanJoin)
	httpAuth.GET("/room/:roomId/user/delete", deleteRtcClient)
	r.Run(":8080")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = &Rooms{
	map[string]*Room{},
	sync.RWMutex{},
}

type RTCClient struct {
	ID string
	WS *ThreadSafeWriter
    Peer      *webrtc.PeerConnection
    sigMu       sync.Mutex
    makingOffer bool
    needRenego  bool
}


