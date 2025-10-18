package response

import (
	"encoding/json"
	http_controllers "streaming-server.com/interface/controllers/http"
)

type CreateLiveVideoResponse struct {
	
}

func NewCreateLiveVideoResponse() http_controllers.Response {
	return &CreateLiveVideoResponse{}
}

func(c *CreateLiveVideoResponse) ToJson() []byte {
	raw, _ := json.Marshal(c)
	return raw
}