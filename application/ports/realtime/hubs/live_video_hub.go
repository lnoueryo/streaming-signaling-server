package live_video_hub

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type RtcClient struct {
    UserID   int
    Conn     *websocket.Conn
    PeerConn *webrtc.PeerConnection
}

type ThreadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

type WebsocketMessage struct {
	Type string `json:"type"`
	Data  interface{} `json:"data"`
}

func (t *ThreadSafeWriter) WriteJSON(v any) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}

type Interface interface {
    // RemoveClient(roomID, userID int)
    // Join(roomID, userID int, conn *websocket.Conn)
    // RoomExists(roomID int) bool
    // AddPeerConnection(roomID, userID int, pc *webrtc.PeerConnection)
    // SetVideoTrack(roomID, userID int, localTrack *webrtc.TrackLocalStaticRTP)
    // SetAudioTrack(roomID, userID int, localTrack *webrtc.TrackLocalStaticRTP)
    // AddPublisherTracks(roomID, userID int, pc *webrtc.PeerConnection)
    // SetRemoteDescription(roomID, userID int, sdp string)

    // AddPeerConnection(roomID int, userID int, pc *webrtc.PeerConnection) error
    DeleteRoom(roomID int)
    RoomExists(roomID int) bool
    SetRemoteDescription(roomID int, userID int, sdp string) error
    AddICECandidate(roomID, userID int, cand webrtc.ICECandidateInit) error
    SignalPeerConnections()
    // AddTrack(roomId, userId int, t *webrtc.TrackRemote)
    AddPeerConnection(userId int, peerConnection *webrtc.PeerConnection, c *ThreadSafeWriter)
    SetViewerEvent(peerConnection *webrtc.PeerConnection, c *ThreadSafeWriter)
    SetBroadcasterEvent(peerConnection *webrtc.PeerConnection, c *ThreadSafeWriter)
    AddTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP
    RemoveTrack(t *webrtc.TrackLocalStaticRTP)
}