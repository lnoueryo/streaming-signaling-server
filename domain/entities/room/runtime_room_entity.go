package room_entity

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	user_entity "streaming-server.com/domain/entities/user"
)

type RuntimeRoom struct {
	ID string
	listLock sync.RWMutex
	Users map[int]*user_entity.RuntimeUser
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	cancelFunc context.CancelFunc
	FfmpegCmd *exec.Cmd
    VideoIn int
    AudioIn int
    OutPort int
    SdpPath string
}


func NewRuntimeRoom(id string) *RuntimeRoom {
	return &RuntimeRoom{
		id,
		sync.RWMutex{},
		make(map[int]*user_entity.RuntimeUser),
		make(map[string]*webrtc.TrackLocalStaticRTP),
		nil,
		nil,
		0,
		0,
		0,
		"",
	}
}

func (r *RuntimeRoom) GetClient(userId int) (*user_entity.RuntimeUser, error) {
	client, ok := r.Users[userId];if !ok {
		return nil, errors.New("client not found")
	}
	return client, nil
}

func (r *RuntimeRoom) AddCancelFunc(cancelFunc context.CancelFunc) {
	r.cancelFunc = cancelFunc
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call.
func (r *RuntimeRoom) DispatchKeyFrame() {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	for i := range r.Users {
		for _, receiver := range r.Users[i].Peer.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = r.Users[i].Peer.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (r *RuntimeRoom) AddUser(user *user_entity.RuntimeUser) {
	r.listLock.Lock()
	r.Users[user.ID] = user
	r.listLock.Unlock()
}

func (r *RuntimeRoom) AddICECandidate(userId int, candidate webrtc.ICECandidateInit) error {
	r.listLock.Lock()
	user, _ := r.GetClient(userId)
	if err := user.Peer.AddICECandidate(candidate); err != nil {
		return err
	}
	r.listLock.Unlock()
	return nil
}

func (r *RuntimeRoom) SetRemoteDescription(userId int, sdp string) error {
	r.listLock.Lock()
	user, _ := r.GetClient(userId)
	if err := user.Peer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}); err != nil {
		return err
	}
	r.listLock.Unlock()
	return nil
}

func (r *RuntimeRoom) AddTrack(t *webrtc.TrackLocalStaticRTP) error {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.SignalPeerConnections()
	}()
	r.trackLocals[t.ID()] = t
	return nil
}

func (r *RuntimeRoom) RemoveTrack(t *webrtc.TrackLocalStaticRTP) error {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.SignalPeerConnections()
	}()
	delete(r.trackLocals, t.ID())
	return nil
}

func (r *RuntimeRoom) SignalPeerConnections() error {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.DispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range r.Users {
			if r.Users[i].Peer.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.Users[i].Peer.Close()
				r.Users[i].Peer = nil
				r.Users[i].WS.Close()
				r.Users[i].WS = nil
				delete(r.Users, i)
				return true
			}

			existingSenders := map[string]bool{}

			for _, sender := range r.Users[i].Peer.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				if _, ok := r.trackLocals[sender.Track().ID()]; !ok {
					if err := r.Users[i].Peer.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			for _, receiver := range r.Users[i].Peer.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			for trackID := range r.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := r.Users[i].Peer.AddTrack(r.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := r.Users[i].Peer.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = r.Users[i].Peer.SetLocalDescription(offer); err != nil {
				return true
			}

			if err = r.Users[i].WS.Send("offer", offer); err != nil {
				return true
			}
		}
		return tryAgain
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			go func() {
				time.Sleep(time.Second * 3)
				r.SignalPeerConnections()
			}()
			return nil
		}
		if !attemptSync() {
			break
		}
	}
	return nil
}