package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type Room struct {
	ID string
	listLock sync.RWMutex
	clients map[int]*RTCSession
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	cancelFunc context.CancelFunc
}

func (r *Room)addTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.signalPeerConnections()
	}()
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}
	r.trackLocals[t.ID()] = trackLocal
	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections.
func (r *Room)removeTrack(t *webrtc.TrackLocalStaticRTP) {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.signalPeerConnections()
	}()

	delete(r.trackLocals, t.ID())
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call.
func (r *Room) dispatchKeyFrame() {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	for i := range r.clients {
		for _, receiver := range r.clients[i].Peer.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = r.clients[i].Peer.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks.
func (r *Room) signalPeerConnections() {
	room, _ := rooms.getRoom(r.ID)
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		room.dispatchKeyFrame()
	}()
	attemptSync := func() (tryAgain bool) {
		for i := range room.clients {
			if room.clients[i].Peer.ConnectionState() == webrtc.PeerConnectionStateClosed {
				delete(room.clients, i)
				if len(room.clients) == 0 {
					delete(rooms.item, r.ID)
				}
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
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
				if r == nil {
					return
				}
				time.Sleep(time.Second * 3)
				r.signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

type Rooms struct {
	item map[string]*Room
	lock sync.RWMutex
}

func (r *Rooms) getOrCreate(roomID string) *Room {
	room := r.item[roomID]
	if room == nil {
		room = &Room{
			roomID,
			sync.RWMutex{},
			make(map[int]*RTCSession),
			make(map[string]*webrtc.TrackLocalStaticRTP),
			nil,
		}
		r.item[roomID] = room
	}
	ctx, cancel := context.WithCancel(context.Background())
	room.cancelFunc = cancel // Roomにキャンセル関数を保存

	go func() {
		ticker := time.NewTicker(3 * time.Second)
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
	return room
}

func (r *Rooms) getRoom(roomID string) (*Room, error) {
	room, ok := r.item[roomID];if !ok {
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

func (r *Rooms) deleteRoom(roomID string) {
	if room, ok := r.item[roomID]; ok {
		room.cancelFunc() // goroutine停止
		delete(r.item, roomID)
	}
}