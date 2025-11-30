package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Ginエンジンのインスタンスを作成
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	InitFirebase()
	r.Use(FirebaseAuthMiddleware())
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})
	r.GET("/ws/live/:roomId", websocketBroadcastHandler)
	r.GET("/ws/live/:roomId/user", websocketBroadcastHandler)
	r.GET("/ws/live/:roomId/user/delete", websocketBroadcastHandler)
	r.Run(":8080")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = &Rooms{
	map[string]*Room{},
	sync.RWMutex{},
}

type RTCSession struct {
	WS *ThreadSafeWriter
    Peer      *webrtc.PeerConnection
    sigMu       sync.Mutex
    makingOffer bool
    needRenego  bool
}

func websocketBroadcastHandler(c *gin.Context) {
	user := getUser(c)
	roomId := c.Param("roomId")
	userId := user.ID
	room, ok := rooms.getRoom(roomId);if ok {
		_, ok := room.clients[userId];if ok {
			errMsg := "他の端末で参加しています。"
			log.Warn(errMsg)
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		
	}

	unsafeConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Failed to upgrade HTTP to Websocket: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ws := NewThreadSafeWriter(unsafeConn)
	defer ws.Close() //nolint
	pc := NewPeerConnection()
	defer pc.Close() //nolint

	// Add our new PeerConnection to global list
	room = rooms.getOrCreate(roomId)
	room.listLock.Lock()
	client, ok := room.clients[userId];if ok {
		client.WS.Close()
		client.Peer.Close()
	}
	room.clients[userId] = &RTCSession{ws, pc, sync.Mutex{}, false, false}
	room.listLock.Unlock()

	// Trickle ICE. Emit server candidate to client
	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		if writeErr := ws.Send("candidate", i.ToJSON()); writeErr != nil {
			log.Error("Failed to write JSON: %v", writeErr)
		}
	})

	// If PeerConnection is closed remove it from global list
	pc.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		log.Info("Connection state change: %s", p)

		switch p {
		case webrtc.PeerConnectionStateFailed:
			_ = pc.Close()
		case webrtc.PeerConnectionStateDisconnected:
			// 猶予を与える
			go func() {
				time.Sleep(20 * time.Second)
				if pc.ConnectionState() == webrtc.PeerConnectionStateDisconnected {
					_ = pc.Close()
				}
			}()
		case webrtc.PeerConnectionStateClosed:
			// クライアント即削除（部屋ロック下で）
			room := rooms.getOrCreate(roomId)
			room.listLock.Lock()
			delete(room.clients, userId)
			room.listLock.Unlock()
			signalPeerConnections(roomId)
		}
	})

	pc.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Info("Got remote track: Kind=%s, ID=%s, PayloadType=%d", t.Kind(), t.ID(), t.PayloadType())

		trackLocal := addTrack(roomId, t)
		defer removeTrack(roomId, trackLocal)

		buf := make([]byte, 1500)
		rtpPkt := &rtp.Packet{}

		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if err = rtpPkt.Unmarshal(buf[:i]); err != nil {
				log.Errorf("Failed to unmarshal incoming RTP packet: %v", err)

				return
			}

			rtpPkt.Extension = false
			rtpPkt.Extensions = nil

			if err = trackLocal.WriteRTP(rtpPkt); err != nil {
				return
			}
		}
	})

	pc.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Infof("ICE connection state changed: %s", is)
	})

	// Signal for the new PeerConnection
	signalPeerConnections(roomId)

	message := &WebsocketMessage{}
	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			log.Errorf("Failed to read message: %v", err)
			return
		}

		if err := json.Unmarshal(raw, &message); err != nil {
			log.Error("Failed to unmarshal json to message: %v", err)
			return
		}
		log.Debug("Got message: %s", message.Event)
		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			raw, err := json.Marshal(message.Data);if err != nil {
				log.Errorf("Failed to unmarshal json to candidate: %v", err)

				return
			}
			if err := json.Unmarshal([]byte(raw), &candidate); err != nil {
				log.Errorf("Failed to unmarshal json to candidate: %v", err)

				return
			}

			log.Infof("Got candidate: %v", candidate)

			if err := pc.AddICECandidate(candidate); err != nil {
				log.Errorf("Failed to add ICE candidate: %v", err)

				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			raw, err := json.Marshal(message.Data);if err != nil {
				log.Errorf("Failed to unmarshal json to candidate: %v", err)

				return
			}
			if err := json.Unmarshal([]byte(raw), &answer); err != nil {
				log.Errorf("Failed to unmarshal json to answer: %v", err)

				return
			}

			if err := pc.SetRemoteDescription(answer); err != nil {
				log.Errorf("Failed to set remote description: %v", err)
				return
			}

			// オファー直列化の解除と再実行
			room, _ := rooms.getRoom(roomId)
			room.listLock.RLock()
			cli := room.clients[userId]
			room.listLock.RUnlock()
			if cli != nil {
				cli.sigMu.Lock()
				cli.makingOffer = false
				doAgain := cli.needRenego
				cli.needRenego = false
				cli.sigMu.Unlock()

				if doAgain {
					// 再度差分反映（今度は makingOffer=false なので実行される）
					go signalPeerConnections(roomId)
				}
			}
		default:
			log.Errorf("unknown message: %+v", message)
		}
	}
}

func deleteWebsocketClient(c *gin.Context) {
	user := getUser(c)
	roomId := c.Param("roomId")
	userId := user.ID
	room, ok := rooms.getRoom(roomId); if !ok {
		errMsg := "既にトークルームが存在していません"
		log.Warn(errMsg)
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	client, ok := room.clients[userId]; if !ok {
		errMsg := "トークルームにユーザーは参加していません"
		log.Warn(errMsg)
		c.JSON(http.StatusNoContent, gin.H{})
		return
	}
	
	client.Peer.Close()
	client.WS.Close()
	// var data interface{}
	// client.WS.Send("close", data)
	c.JSON(http.StatusNoContent, gin.H{})
}

func getUser(c *gin.Context) UserInfo {
	userVal, exists := c.Get("user")
	if !exists {
		return UserInfo{}
	}
	return userVal.(UserInfo)
}