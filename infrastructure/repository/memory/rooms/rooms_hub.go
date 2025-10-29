package rooms_hub

import (
	"context"
	"errors"
	"sync"
	"time"
	"github.com/pion/webrtc/v4"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/webrtc/broadcast"
)

type Hub struct {
	rooms map[int]*Room // roomID -> runtime
	mu    sync.RWMutex
}

var (
	// _ live_video_hub.Interface = (*Hub)(nil)
	log = *logger.Log
)

func New() *Hub {
	rooms := make(map[int]*Room)
	room := NewRoom()
	rooms[1] = room
	return &Hub{ rooms, sync.RWMutex{} }
}

func (r *Hub) getOrCreate(roomID int) *Room {
	room := r.rooms[roomID]
	if room == nil {
		room = &Room{
			sync.RWMutex{},
			make(map[int]*broadcast.PeerClient),
			make(map[string]*webrtc.TrackLocalStaticRTP),
			nil,
		}
		r.rooms[roomID] = room
		ctx, cancel := context.WithCancel(context.Background())
		room.cancelFunc = cancel // Roomにキャンセル関数を保存

		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					room.dispatchKeyFrame()
				}
			}
		}()
	}
	return room
}

func (r *Hub) getRoom(roomID int) (*Room, error) {
	room, ok := r.rooms[roomID];if !ok {
		return nil, errors.New("room not found")
	}
	// request a keyframe every 3 seconds
	go func() {
		for range time.NewTicker(time.Second * 3).C {
			room.dispatchKeyFrame()
		}
	}()

	return room, nil
}

func (h *Hub) DeleteRoom(roomID int) {
	delete(h.rooms, roomID)
}

func (h *Hub) AddPeerConnection(roomId, userId int, peerClient *broadcast.PeerClient) error {
	room := h.getOrCreate(roomId)
	room.listLock.Lock()
	room.clients[userId] = peerClient
	room.listLock.Unlock()
	return nil
}

func (h *Hub) ClosePeerConnection(roomId, userId int) error {
	room, err := h.getRoom(roomId);if err != nil {
		return err
	}
	room.listLock.Lock()
	room.clients[userId].Peer.Close()
	room.listLock.Unlock()
	return nil
}

func (h *Hub) AddICECandidate(roomId, userId int, candidate webrtc.ICECandidateInit) error {
	room, err := h.getRoom(roomId);if err != nil {
		return err
	}
	room.listLock.Lock()
	if err := room.clients[userId].Peer.AddICECandidate(candidate); err != nil {
		return err
	}
	room.listLock.Unlock()
	return nil
}

func (h *Hub) SetRemoteDescription(roomId, userId int, sdp string) error {
	room, err := h.getRoom(roomId);if err != nil {
		return err
	}
	log.Debug("AddPeerConnection: %v", room.clients)
	client, ok := room.clients[userId];if !ok {
		return errors.New("no client")
	}
	if err := client.Peer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}); err != nil {
		return err
	}
	return nil
}

// Add to list of tracks and fire renegotation for all PeerConnections.
func (h *Hub) AddTrack(roomId int, t *webrtc.TrackLocalStaticRTP) error {
	room, err := h.getRoom(roomId);if err != nil {
		return err
	}
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		h.SignalPeerConnections(roomId)
	}()
	room.trackLocals[t.ID()] = t
	return nil
}

// Remove from list of tracks and fire renegotation for all PeerConnections.
func (h *Hub) RemoveTrack(roomId int, t *webrtc.TrackLocalStaticRTP) error {
	room, err := h.getRoom(roomId);if err != nil {
		return err
	}
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		h.SignalPeerConnections(roomId)
	}()
	delete(room.trackLocals, t.ID())
	return nil
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks.
func (h *Hub) SignalPeerConnections(roomId int) error {
	log.Debug("SignalPeerConnections")
	room, err := h.getRoom(roomId);if err != nil {
		return err
	}
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		room.dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range room.clients {
			if room.clients[i].Peer.ConnectionState() == webrtc.PeerConnectionStateClosed {
				log.Debug("delete peer: %v", room.clients[i].Peer.ConnectionState())
				room.clients[i].Peer.Close()
				room.clients[i].Peer = nil
				room.clients[i].WS.Close()
				room.clients[i].WS = nil
				delete(room.clients, i)
				if len(room.clients) == 0 {
					delete(h.rooms, roomId)
				}
				return true
			}

			existingSenders := map[string]bool{}

			for _, sender := range room.clients[i].Peer.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := room.trackLocals[sender.Track().ID()]; !ok {
					if err := room.clients[i].Peer.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range room.clients[i].Peer.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range room.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := room.clients[i].Peer.AddTrack(room.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := room.clients[i].Peer.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = room.clients[i].Peer.SetLocalDescription(offer); err != nil {
				return true
			}

			if err = room.clients[i].WS.Send("offer", offer); err != nil {
				return true
			}
		}

		return tryAgain
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				h.SignalPeerConnections(roomId)
			}()

			return nil
		}

		if !attemptSync() {
			break
		}
	}
	return nil
}
