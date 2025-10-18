package request

import (
	"errors"
	"net/http"
	"strconv"
	create_live_video_usecase "streaming-server.com/application/usecases/live_video/create_live_video"
	"streaming-server.com/application/usecases/shared"
	http_controllers "streaming-server.com/interface/controllers/http"
)


type CreateLiveVideoParams struct {
	RoomID int
	UserID int
}

type Query struct {
	RoomID int
	UserID int
}

type CreateLiveVideoRequest struct {
	Params *CreateLiveVideoParams
}

func NewCreateLiveVideoRequest(r *http.Request) (*CreateLiveVideoRequest, *shared.UsecaseError) {
	usecaseError := &shared.UsecaseError{
		Type: "validation",
	}
	val := r.Context().Value("request")
	if val == nil {
		usecaseError.Message = "invalid request"
		return nil, usecaseError
	}
	request, ok := val.(*http_controllers.Request)
	if !ok {
		usecaseError.Message = "invalid request"
		return nil, usecaseError
	}
	createLiveVideoRequest := &CreateLiveVideoRequest{}
	err := createLiveVideoRequest.parseParams(request);if err != nil {
		usecaseError.Message = err.Error()
		return nil, usecaseError
	}

	return createLiveVideoRequest, nil
}

func(c *CreateLiveVideoRequest) parseParams(r *http_controllers.Request) error {
	createLiveVideoParams := &CreateLiveVideoParams{}
	err := createLiveVideoParams.getRoomId(r.Params);if err != nil {
		return err
	}
	err = createLiveVideoParams.getUserId(r.Params);if err != nil {
		return err
	}
	c.Params = createLiveVideoParams
	return nil
}

func(c *CreateLiveVideoRequest) ToInput() *create_live_video_usecase.CreateLiveVideoInput {
	return &create_live_video_usecase.CreateLiveVideoInput{
		RoomID: c.Params.RoomID,
		UserID: c.Params.UserID,
	}
}

func(c *CreateLiveVideoParams) getRoomId(params map[string]string) error {
	roomId, ok := params["roomId"];if !ok {
		return errors.New("no roomId")
	}
	id, err := strconv.Atoi(roomId)
	if err != nil {
		return errors.New("invalid roomId format")
	}
	if id <= 0 {
		return errors.New("invalid roomId value")
	}
	c.RoomID = id
	return nil
}

func(c *CreateLiveVideoParams) getUserId(params map[string]string) error {
	userId, ok := params["userId"];if !ok {
		return errors.New("no userId")
	}
	id, err := strconv.Atoi(userId)
	if err != nil {
		return errors.New("invalid userId format")
	}
	if id <= 0 {
		return errors.New("invalid userId value")
	}
	c.UserID = id
	return nil
}