package close_connection_usecase

import (
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type CloseConnectionUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewCloseConnection(
	roomRepo live_video_hub.Interface,
) *CloseConnectionUsecase {
	return &CloseConnectionUsecase{
		roomRepo,
	}
}

func (u *CloseConnectionUsecase) Do(
	params *live_video_dto.Params,
	conn *live_video_hub.ThreadSafeWriter,
) error {
	log.Debug("🧩 RemoveClient called: room=%d user=%d", params.RoomID, params.UserID)
	// u.roomRepository.RemoveClient(params.RoomID, params.UserID)
	return nil
}
