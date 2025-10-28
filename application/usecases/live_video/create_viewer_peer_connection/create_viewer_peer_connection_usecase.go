package create_viewer_peer_connection_usecase

import (
	"github.com/pion/webrtc/v4"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/webrtc/broadcast"
	"streaming-server.com/infrastructure/ws"
)

var (
	log = logger.Log
)

type CreateViewerPeerConnectionUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewCreateViewerPeerConnection(roomRepo live_video_hub.Interface) *CreateViewerPeerConnectionUsecase {
	return &CreateViewerPeerConnectionUsecase{
		roomRepo,
	}
}

func (u *CreateViewerPeerConnectionUsecase) Do(
	params *live_video_dto.Params,
	conn *ws.ThreadSafeWriter,
) error {
	pcs := broadcast.NewPeerConnection(conn)
	// defer pcs.Peer.Close()
	u.roomRepository.AddPeerConnection(params.RoomID, params.UserID, pcs)

	pcs.Peer.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		if writeErr := conn.Send("candidate", i.ToJSON()); writeErr != nil {
			log.Error("Failed to write JSON: %v", writeErr)
		}
	})

	pcs.Peer.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		log.Info("Connection state change: %s", p)

		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := pcs.Peer.Close(); err != nil {
				log.Error("Failed to close PeerConnection: %v", err)
			}
		case webrtc.PeerConnectionStateClosed:
			u.roomRepository.SignalPeerConnections(params.RoomID)
		default:
		}
	})

	pcs.Peer.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Info("ICE connection state changed: %s", is)
	})
	u.roomRepository.SignalPeerConnections(params.RoomID)
	return nil
}
