package set_candidate_usecase

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
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
	conn *websocket.Conn,
) error {
	cand := webrtc.ICECandidateInit{
		Candidate:     message.Candidate,
		SDPMid:        message.SDPMid,
		SDPMLineIndex: message.SDPMLineIndex,
	}

	err := u.roomRepository.AddICECandidate(params.RoomID, params.UserID, cand);if err != nil {
		log.Error("%v", err)
	}
	// TODO どこかで共通化
	msg := struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: "set_candidate",
		Data: struct {
			RoomID      int    `json:"roomId"`
			PublisherID string `json:"publisherId"`
		}{
			RoomID:      params.RoomID,
			PublisherID: "publisherID",
		},
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Error("WriteJSON error:", err)
		return err
	}
	log.Info("👌 Send: Set Candidate")
	return nil
}
