package set_answer_usecase

import (
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/ws"
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
	conn *ws.ThreadSafeWriter,
) error {
	err := u.roomRepository.SetRemoteDescription(params.RoomID, params.UserID, message.SDP);if err != nil {
		log.Error("%v", err)
	}
	log.Debug("send answered")
	return nil
}
