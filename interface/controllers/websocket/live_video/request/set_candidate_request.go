package live_video_request

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	set_candidate_usecase "streaming-server.com/application/usecases/live_video/set_candidate"
)

type SetCandidateRawMessage struct {
	Candidate     string  `json:"candidate"`
	SDPMid        string `json:"sdpMid"`
	SDPMLineIndex interface{} `json:"sdpMLineIndex"`
}

func SetCandidateRequest(ctx context.Context, msg interface{}) (*live_video_dto.Params, *set_candidate_usecase.Message, error) {
	var rawMessage SetCandidateRawMessage
	var params *live_video_dto.Params
	var rawParams = &RawParams{}
	var message *set_candidate_usecase.Message
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

func (raw *SetCandidateRawMessage) parse() (*set_candidate_usecase.Message, error) {
	var message set_candidate_usecase.Message
	candidate, err := raw.getCandidate();if err != nil {
		return &message, err
	}
	sdpMid, err := raw.getSdpMid();if err != nil {
		return &message, err
	}
	sdpMLineIndex, err := raw.getSdpMLineIndex();if err != nil {
		return &message, err
	}
	message.Candidate = candidate
	message.SDPMid = &sdpMid
	message.SDPMLineIndex = &sdpMLineIndex
	return &message, nil
}

func (raw *SetCandidateRawMessage) getCandidate() (string, error) {
	var candidate string = raw.Candidate
	if candidate == "" {
		return candidate, errors.New("candidate required")
	}
	return candidate, nil
}

func (raw *SetCandidateRawMessage) getSdpMid() (string, error) {
	var sdpMid string = raw.SDPMid
	if sdpMid == "" {
		return sdpMid, errors.New("sdp required")
	}
	return sdpMid, nil
}

func (raw *SetCandidateRawMessage) getSdpMLineIndex() (uint16, error) {
	var sdpMLineIndex uint16

	switch v := raw.SDPMLineIndex.(type) {
	case float64:
		sdpMLineIndex = uint16(v)
	case int:
		sdpMLineIndex = uint16(v)
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, errors.New("invalid sdpMLineIndex format")
		}
		sdpMLineIndex = uint16(i)
	case nil:
		return 0, errors.New("sdpMLineIndex is missing")
	default:
		return 0, errors.New("sdpMLineIndex must be a number or numeric string")
	}

	return sdpMLineIndex, nil
}