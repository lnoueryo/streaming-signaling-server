package live_video_request

import (
	"context"
	"encoding/json"
	"errors"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	get_offer_usecase "streaming-server.com/application/usecases/live_video/get_offer"
)

type GetOfferRawMessage struct {
	SDP string `json:"sdp"`
}

func GetOfferRequest(ctx context.Context, msg interface{}) (*live_video_dto.Params, *get_offer_usecase.Message, error) {
	var rawMessage GetOfferRawMessage
	var params *live_video_dto.Params
	var rawParams = &RawParams{}
	var message *get_offer_usecase.Message
    raw, err := json.Marshal(rawMessage)
    if err != nil {
        return params, message, err
    }

	if err := json.Unmarshal(raw, &msg); err != nil {
		return params, message, err
	}
	params, _ = rawParams.parse(ctx)
	message, _ = rawMessage.parse()
	return params, message, nil
}

func (raw *GetOfferRawMessage) parse() (*get_offer_usecase.Message, error) {
	var message get_offer_usecase.Message
	sdp, err := raw.getSdp();if err != nil {
		return &message, err
	}
	message.SDP = sdp
	return &message, nil
}

func (raw *GetOfferRawMessage) getSdp() (string, error) {
	var sdp string = raw.SDP
	if raw.SDP == "" {
		return sdp, errors.New("sdp required")
	}
	return sdp, nil
}
