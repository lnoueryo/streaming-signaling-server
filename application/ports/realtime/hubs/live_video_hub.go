package live_video_hub

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"streaming-server.com/infrastructure/webrtc/broadcast"
)

type RtcClient struct {
    UserID   int
    Conn     *websocket.Conn
    PeerConn *webrtc.PeerConnection
}

type Interface interface {
    DeleteRoom(roomID int)
    SetRemoteDescription(roomID int, userID int, sdp string) error
    AddICECandidate(roomID, userID int, cand webrtc.ICECandidateInit) error
    SignalPeerConnections(roomId int) error
    AddPeerConnection(roomID, userId int, peerConnection *broadcast.PeerClient) error
    ClosePeerConnection(roomID, userId int) error
    AddTrack(roomId int, t *webrtc.TrackLocalStaticRTP) error
    RemoveTrack(roomId int, t *webrtc.TrackLocalStaticRTP) error
}