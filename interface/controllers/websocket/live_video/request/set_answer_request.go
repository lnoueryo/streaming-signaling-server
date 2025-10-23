package live_video_request

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	set_answer_usecase "streaming-server.com/application/usecases/live_video/set_answer"
)

type SetAnswerRawMessage struct {
	SDP string `json:"sdp"`
}

func SetAnswerRequest(ctx context.Context, msg interface{}) (*live_video_dto.Params, *set_answer_usecase.Message, error) {
	var rawMessage SetAnswerRawMessage
	var params *live_video_dto.Params
	var rawParams = &RawParams{}
	var message *set_answer_usecase.Message
    raw, err := json.Marshal(msg)
    if err != nil {
		return params, message, err
    }
	
	if err := json.Unmarshal(raw, &rawMessage); err != nil {
		log.Print("rawMessage %v", rawMessage)
		return params, message, err
	}
	params, _ = rawParams.parse(ctx)
	message, _ = rawMessage.parse()
	return params, message, nil
}

func (raw *SetAnswerRawMessage) parse() (*set_answer_usecase.Message, error) {
	var message set_answer_usecase.Message
	sdp, err := raw.getSdp();if err != nil {
		return &message, err
	}
	message.SDP = sdp
	return &message, nil
}

func (raw *SetAnswerRawMessage) getSdp() (string, error) {
	var sdp string = raw.SDP
	if raw.SDP == "" {
		return sdp, errors.New("sdp required")
	}
	return sdp, nil
}