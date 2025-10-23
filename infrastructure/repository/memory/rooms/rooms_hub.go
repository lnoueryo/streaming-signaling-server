package rooms_hub

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	"streaming-server.com/infrastructure/logger"
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
	return &Hub{rooms: make(map[int]*Room)}
}

func (h *Hub) RoomExists(roomID int) bool {
	_, ok := h.rooms[roomID]
	return ok
}

func (h *Hub) getOrCreate(roomID int) *Room {
	rt := h.rooms[roomID]
	if rt == nil {
		rt = &Room{
			clients: make(map[int]*RtcClient),
			tracks:  make(map[string]*webrtc.TrackLocalStaticRTP),
		}
		h.rooms[roomID] = rt
	}
	return rt
}

func (h *Hub) getRoom(roomID int) (*Room, error) {
	room, ok := h.rooms[roomID];if !ok {
		return nil, errors.New("room not found")
	}
	return room, nil
}

// func (h *Hub) AddPeerConnection(roomID, userID int, pc *webrtc.PeerConnection) error {
// 	client, err := h.getClient(roomID, userID); if err != nil {
// 		return err
// 	}
// 	client.PeerConn = pc
// 	return nil
// }

func (h *Hub) getClient(roomID, userID int) (*RtcClient, error) {
	room, ok := h.rooms[roomID]; if !ok {
		return nil, errors.New("client not found")
	}
	client, err := room.getClient(userID);if err != nil {
		return nil, err
	}
	return client, nil
}

func (h *Hub) DeleteRoom(roomID int) {
	delete(h.rooms, roomID)
}

// func (h *Hub) AddICECandidate(roomID, userID int, candidate webrtc.ICECandidateInit) error {
// 	client, err := h.getClient(roomID, userID); if err != nil {
// 		return err
// 	}
// 	if client.PeerConn == nil {
// 		return errors.New("no peer conn")
// 	}
// 	if err := client.PeerConn.AddICECandidate(candidate); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (h *Hub) SetRemoteDescription(roomID, userID int, sdp string) error {
// 	client, err := h.getClient(roomID, userID); if err != nil {
// 		return err
// 	}
// 	if err := client.PeerConn.SetRemoteDescription(webrtc.SessionDescription{
// 		Type: webrtc.SDPTypeAnswer,
// 		SDP:  sdp,
// 	}); err != nil {
// 		return err
// 	}
// 	return nil
// }

var (
	listLock = sync.RWMutex{}
	peerConnections = map[int]*peerConnectionState{}
	trackLocals = map[string]*webrtc.TrackLocalStaticRTP{}
)


type peerConnectionState struct {
	peerConnection *webrtc.PeerConnection
	websocket      *live_video_hub.ThreadSafeWriter
}

func (h *Hub) AddPeerConnection(userId int, peerConnection *webrtc.PeerConnection, c *live_video_hub.ThreadSafeWriter) {
	listLock.Lock()
	peerConnections[userId] = &peerConnectionState{peerConnection, c}
	listLock.Unlock()
	log.Debug("AddPeerConnection: %v", peerConnections[userId].peerConnection.ConnectionState())
}

func (h *Hub) AddICECandidate(roomID, userID int, candidate webrtc.ICECandidateInit) error {
	log.Debug("add ice")
	if err := peerConnections[userID].peerConnection.AddICECandidate(candidate); err != nil {
		return err
	}
	return nil
}

func (h *Hub) SetRemoteDescription(roomID, userID int, sdp string) error {
	log.Debug("AddPeerConnection: %v", peerConnections)
	peer, ok := peerConnections[userID];if !ok {
		return errors.New("no peer")
	}
	if err := peer.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}); err != nil {
		return err
	}
	return nil
}

func (h *Hub) SetViewerEvent(peerConnection *webrtc.PeerConnection, c *live_video_hub.ThreadSafeWriter) {
	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		if writeErr := c.WriteJSON(&live_video_hub.WebsocketMessage{
			Type: "candidate",
			Data:  i.ToJSON(),
		}); writeErr != nil {
			log.Error("Failed to write JSON: %v", writeErr)
		}
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		log.Info("Connection state change: %s", p)

		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Error("Failed to close PeerConnection: %v", err)
			}
		case webrtc.PeerConnectionStateClosed:
			h.SignalPeerConnections()
		default:
		}
	})

	peerConnection.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Info("ICE connection state changed: %s", is)
	})
}

func (h *Hub) SetBroadcasterEvent(peerConnection *webrtc.PeerConnection, c *live_video_hub.ThreadSafeWriter) {
	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Error("Failed to marshal candidate to json: %v", err)

			return
		}

		log.Info("Send candidate to client: %s", candidateString)

		if writeErr := c.WriteJSON(&live_video_hub.WebsocketMessage{
			Type: "candidate",
			Data:  i.ToJSON(),
		}); writeErr != nil {
			log.Error("Failed to write JSON: %v", writeErr)
		}
	})

	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		log.Info("Connection state change: %s", p)

		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Error("Failed to close PeerConnection: %v", err)
			}
		case webrtc.PeerConnectionStateClosed:
			h.SignalPeerConnections()
		default:
		}
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Info("Got remote track: Kind=%s, ID=%s, PayloadType=%d", t.Kind(), t.ID(), t.PayloadType())

		trackLocal := h.AddTrack(t)
		defer h.RemoveTrack(trackLocal)

		buf := make([]byte, 1500)
		rtpPkt := &rtp.Packet{}

		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if err = rtpPkt.Unmarshal(buf[:i]); err != nil {
				log.Error("Failed to unmarshal incoming RTP packet: %v", err)

				return
			}

			rtpPkt.Extension = false
			rtpPkt.Extensions = nil

			if err = trackLocal.WriteRTP(rtpPkt); err != nil {
				return
			}
		}
	})

	peerConnection.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Info("ICE connection state changed: %s", is)
	})
}

// Add to list of tracks and fire renegotation for all PeerConnections.
func (h *Hub) AddTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		h.SignalPeerConnections()
	}()

	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	trackLocals[t.ID()] = trackLocal

	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections.
func (h *Hub) RemoveTrack(t *webrtc.TrackLocalStaticRTP) {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		h.SignalPeerConnections()
	}()

	delete(trackLocals, t.ID())
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks.
func (h *Hub) SignalPeerConnections() {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		h.dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range peerConnections {
			if peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				peerConnections[i].peerConnection.Close()
				log.Error("delete peer: %v", peerConnections[i].peerConnection.ConnectionState())
				delete(peerConnections, i)
				return true
			}

			existingSenders := map[string]bool{}

			for _, sender := range peerConnections[i].peerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := trackLocals[sender.Track().ID()]; !ok {
					if err := peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := peerConnections[i].peerConnection.AddTrack(trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := peerConnections[i].peerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			if err = peerConnections[i].websocket.WriteJSON(&live_video_hub.WebsocketMessage{
				Type: "offer",
				Data:  offer,
			}); err != nil {
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
				h.SignalPeerConnections()
			}()

			return
		}

		if !attemptSync() {
			break
		}
	}
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call.
func (h *Hub) dispatchKeyFrame() {
	listLock.Lock()
	defer listLock.Unlock()

	for i := range peerConnections {
		for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}
