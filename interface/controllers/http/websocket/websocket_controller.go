package websocket_controller

import (
	"net/http"

	create_live_video_usecase "streaming-server.com/application/usecases/live_video/create_live_video"
	"streaming-server.com/infrastructure/logger"
	http_controllers "streaming-server.com/interface/controllers/http"
	"streaming-server.com/interface/controllers/http/websocket/request"
)

type Controller struct {
	CreateLiveVideoUsecase *create_live_video_usecase.CreateLiveVideoUsecase
}

func NewController(
	createLiveVideoUsecase *create_live_video_usecase.CreateLiveVideoUsecase,
) *Controller {
	c := &Controller{}
	c.CreateLiveVideoUsecase = createLiveVideoUsecase
	return c
}

var log = logger.Log

func (c *Controller) CreateLiveVideo(w http.ResponseWriter, r *http.Request) {
	// TODO 認証
	params, err := request.NewCreateLiveVideoRequest(r); if err != nil {
		log.Error(err)
		http_controllers.HttpError.Response(w, err)
		return
	}
	result := c.CreateLiveVideoUsecase.Do(params.ToInput())
	if result.IsError() {
		http_controllers.HttpError.Response(w, result.Error())
		return
	}
	// return response.NewCreateLiveVideoResponse(result.Success()).ToJson()
}
