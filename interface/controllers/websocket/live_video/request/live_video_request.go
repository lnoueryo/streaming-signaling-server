package live_video_request

import (
	"context"
	"errors"
	"strconv"

	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
)

type RawParams struct {
	RoomID string
	UserID string
}

func (raw *RawParams) parse(ctx context.Context) (*live_video_dto.Params, error) {
	params := &live_video_dto.Params{}
	paramsMap, ok := ctx.Value("params").(map[string]string)
	if !ok {
		return nil, errors.New("invalid params type in context")
	}

	rawRoomId, ok := paramsMap["roomId"];if !ok {
		return params, errors.New("room id required")
	}
	raw.RoomID = rawRoomId
	rawUserId, ok := paramsMap["userId"];if !ok {
		return params, errors.New("user id required")
	}
	raw.UserID = rawUserId

	roomID, err := raw.getRoomId();if err != nil {
		return params, err
	}
	userID, err := raw.getUserId();if err != nil {
		return params, err
	}

	params.RoomID = roomID
	params.UserID = userID
	return params, nil
}

func (raw *RawParams) getRoomId() (int, error) {
	id, err := strconv.Atoi(raw.RoomID)
	if err != nil {
		return id, errors.New("invalid roomId format")
	}
	if id <= 0 {
		return id, errors.New("invalid roomId value")
	}
	return id, nil
}

func (raw *RawParams) getUserId() (int, error) {
	id, err := strconv.Atoi(raw.UserID)
	if err != nil {
		return id, errors.New("invalid userId format")
	}

	if id <= 0 {
		return id, errors.New("invalid userId value")
	}
	return id, nil
}