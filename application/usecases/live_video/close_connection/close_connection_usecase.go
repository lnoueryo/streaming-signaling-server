package close_connection_usecase

import (
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/ws"
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
	conn *ws.ThreadSafeWriter,
) error {
	log.Debug("🧩 RemoveClient called: room=%d user=%d", params.RoomID, params.UserID)
	u.roomRepository.ClosePeerConnection(params.RoomID, params.UserID)
	u.roomRepository.SignalPeerConnections(params.RoomID)
	return nil
}
