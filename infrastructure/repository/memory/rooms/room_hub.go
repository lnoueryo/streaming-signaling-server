package rooms_hub

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"streaming-server.com/infrastructure/logger"
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
	tracks  map[int]*Tracks         // publisherUserID -> tracks
	mu      sync.RWMutex
}

func NewRoom() *Room {
	return &Room{
		make(map[int]*RtcClient),
		make(map[int]*Tracks),
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

func (r *Room) RemoveTracks(userID int) error {

    client, err := r.getClient(userID);if err != nil {
        return err
    }

    track, isPublisher := r.tracks[userID];if !isPublisher {
        return nil
    }

    for _, viewer := range r.clients {
        if viewer == client {
            continue
        }

        for _, sender := range viewer.PeerConn.GetSenders() {
			senderTrack := sender.Track()
			if senderTrack == nil {
				continue
			}
			if senderTrack == track.Video || senderTrack == track.Audio {
				_ = sender.ReplaceTrack(nil)
				_ = viewer.PeerConn.RemoveTrack(sender)
			}
        }
    }
    r.removeTrack(userID)
	return nil
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
	client, err := r.getClient(userID);if err != nil {
		return err
	}
	client.Conn.Close()
	client.PeerConn.Close()
	delete(r.clients, userID)
	return nil
}

func (r *Room) removeTrack(userID int) {
	track := r.tracks[userID]
	track.Video = nil
	track.Audio = nil
	delete(r.tracks, userID)
	logger.Log.Debug("delete track")
}

func (r *Room) GetTrack(userID int) (*Tracks, error) {
	track, ok := r.tracks[userID]; if !ok {
		return nil, errors.New("track not found")
	}
	return track, nil
}

func (r *Room) CreatePublisherTrack(userID int) (*Tracks, error) {
	track, ok := r.tracks[userID]; if !ok {
		return nil, errors.New("track not found")
	}
	return track, nil
}

func (r *Room) SetVideoTrack(userID int, localTrack *webrtc.TrackLocalStaticRTP) {
	track := r.tracks[userID]
	track.Video = localTrack
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