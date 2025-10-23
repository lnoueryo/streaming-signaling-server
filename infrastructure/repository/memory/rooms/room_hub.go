package rooms_hub

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type RtcClient struct {
	UserID   int
	Conn     *websocket.Conn
	PeerConn *webrtc.PeerConnection
}

type Tracks struct {
	Video *webrtc.TrackLocalStaticRTP
	Audio *webrtc.TrackLocalStaticRTP
}

type Room struct {
	clients map[int]*RtcClient     // userID -> client
	tracks  map[string]*webrtc.TrackLocalStaticRTP         // publisherUserID -> tracks
	mu      sync.RWMutex
}

func NewRoom() *Room {
	return &Room{
		make(map[int]*RtcClient),
		make(map[string]*webrtc.TrackLocalStaticRTP),
		sync.RWMutex{},
	}
}

func NewClient(
	userID int,
	conn *websocket.Conn,
	peerConn *webrtc.PeerConnection,
) *RtcClient {
	return &RtcClient{
		userID,
		conn,
		peerConn,
	}
}


func (r *Room) getClient(userID int) (*RtcClient, error) {
	client, ok := r.clients[userID];if !ok {
		return nil, errors.New("client not found")
	}
	return client, nil
}

func (r *Room) addClient(userID int, conn *websocket.Conn) {
	client := NewClient(userID, conn, nil)
	r.clients[userID] = client
}

func (r *Room) removeClient(userID int) error {
	client, err := r.getClient(userID)
	if err != nil {
		return err
	}

	if client.Conn != nil {
		_ = client.Conn.Close()
		client.Conn = nil
	}
	if client.PeerConn != nil {
		_ = client.PeerConn.Close()
		client.PeerConn = nil
	}

	delete(r.clients, userID)
	log.Info("🧹 Removed client: %d", userID)
	return nil
}


func (r *Room) HasClient() bool {
	return 0 < len(r.clients)
}

func (c *RtcClient) HasPeerConnection() bool {
	return c.PeerConn != nil
}

func (c *RtcClient) ClosePeerConnection() {
	c.PeerConn.Close()
}