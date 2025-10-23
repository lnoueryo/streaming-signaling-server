package get_offer_usecase

import (
	"github.com/pion/webrtc/v4"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type GetOfferUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewGetOffer(roomRepo live_video_hub.Interface,) *GetOfferUsecase {
	return &GetOfferUsecase{
		roomRepo,
	}
}

func (u *GetOfferUsecase) Do(
	params *live_video_dto.Params,
	message *Message,
	conn *live_video_hub.ThreadSafeWriter,
) error {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Error("Failed to creates a PeerConnection: %v", err)
		return err
	}

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Error("Failed to add transceiver: %v", err)
			return err
		}
	}

	// Add our new PeerConnection to global list
	u.roomRepository.AddPeerConnection(params.UserID, peerConnection, conn)
	u.roomRepository.SetBroadcasterEvent(peerConnection, conn)


	// Signal for the new PeerConnection
	u.roomRepository.SignalPeerConnections()

	return nil
}
