package set_candidate_usecase

import (
	"github.com/pion/webrtc/v4"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/ws"
)

var log = logger.Log

type SetCandidateUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewSetCandidate(roomRepo live_video_hub.Interface) *SetCandidateUsecase {
	return &SetCandidateUsecase{
		roomRepo,
	}
}

func (u *SetCandidateUsecase) Do(
	params *live_video_dto.Params,
	message *Message,
	conn *ws.ThreadSafeWriter,
) error {
	cand := webrtc.ICECandidateInit{
		Candidate:     message.Candidate,
		SDPMid:        message.SDPMid,
		SDPMLineIndex: message.SDPMLineIndex,
	}

	err := u.roomRepository.AddICECandidate(params.RoomID, params.UserID, cand);if err != nil {
		log.Error("%v", err)
	}
	log.Info("👌 Send: Set Candidate")
	return nil
}
