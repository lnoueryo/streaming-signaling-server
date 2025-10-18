package join_room_usecase

import (
	"github.com/gorilla/websocket"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type JoinRoomUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewJoinRoom(
	roomRepo live_video_hub.Interface,
) *JoinRoomUsecase {
	return &JoinRoomUsecase{
		roomRepo,
	}
}

func (u *JoinRoomUsecase) Do(
	params *live_video_dto.Params,
	conn *websocket.Conn,
) error {
	// room の認証が必要な場合ここ
	u.roomRepository.Join(params.RoomID, params.UserID, conn)
	conn.SetCloseHandler(func(code int, text string) error {
		log.Info("Request: Close Connection")
		u.roomRepository.RemoveClient(params.RoomID, params.UserID)
		return nil
	})
	// TODO どこかで共通化
	msg := struct {
		Type string `json:"type"`
		Data struct {
			RoomID      int    `json:"roomId"`
			PublisherID string `json:"publisherId"`
		} `json:"data"`
	}{
		Type: "join",
		Data: struct {
			RoomID      int    `json:"roomId"`
			PublisherID string `json:"publisherId"`
		}{
			RoomID:      params.RoomID,
			PublisherID: "publisherID",
		},
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Error("WriteJSON error: ", err)
		return err
	}
	log.Info("👌 Send: Join")
	return nil
}
