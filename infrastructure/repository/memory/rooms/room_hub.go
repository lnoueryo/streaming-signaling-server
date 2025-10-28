package rooms_hub

import (
	"context"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"streaming-server.com/infrastructure/webrtc/broadcast"
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
	listLock sync.RWMutex
	clients map[int]*broadcast.PeerClient
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	cancelFunc context.CancelFunc
}

func NewRoom() *Room {
	return &Room{
		sync.RWMutex{},
		make(map[int]*broadcast.PeerClient),
		make(map[string]*webrtc.TrackLocalStaticRTP),
		nil,
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


func (r *Room) getClient(userID int) (*broadcast.PeerClient, error) {
	client, ok := r.clients[userID];if !ok {
		return nil, errors.New("client not found")
	}
	return client, nil
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call.
func (room *Room) dispatchKeyFrame() {
	room.listLock.Lock()
	defer room.listLock.Unlock()

	for i := range room.clients {
		for _, receiver := range room.clients[i].Peer.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = room.clients[i].Peer.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}