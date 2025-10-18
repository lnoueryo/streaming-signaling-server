package create_live_video_usecase

import (
	"streaming-server.com/application/usecases/shared"
	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type CreateLiveVideoUsecase struct {
	// roomRepository live_video_hub.Interface
}

func NewCreateLiveVideo() *CreateLiveVideoUsecase {
	return &CreateLiveVideoUsecase{
		// roomRepo,
	}
}

func (u *CreateLiveVideoUsecase) Do(
	param *CreateLiveVideoInput,
) *shared.UsecaseResult[bool] {
	// DBから取得
	// room := u.repository.getRoom()
	// if room == nil {
	// 	return
	// }
	if param.RoomID != 1 {
		return shared.NewErrorResult[bool]("not-found", "Not Found")
	}
	return shared.NewSuccessResult(true)
}
