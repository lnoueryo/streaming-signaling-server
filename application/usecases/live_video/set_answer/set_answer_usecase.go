package set_answer_usecase

import (
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type SetAnswerUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewSetAnswer(roomRepo live_video_hub.Interface) *SetAnswerUsecase {
	return &SetAnswerUsecase{
		roomRepo,
	}
}

func (u *SetAnswerUsecase) Do(
	params *live_video_dto.Params,
	message *Message,
	conn *live_video_hub.ThreadSafeWriter,
) error {
	err := u.roomRepository.SetRemoteDescription(params.RoomID, params.UserID, message.SDP);if err != nil {
		log.Error("%v", err)
	}
	// TODO どこかで共通化
	msg := struct {
		Type string `json:"type"`
		Data struct {
			RoomID      int    `json:"roomId"`
			PublisherID string `json:"publisherId"`
		} `json:"data"`
	}{
		Type: "answered",
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
	log.Debug("send answered")
	return nil
}
