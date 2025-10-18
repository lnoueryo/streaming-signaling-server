package http_controllers

import (
	"encoding/json"
	"log"
	"net/http"
)

type Request struct {
	Params map[string]string
	Query  map[string]string
	Body   map[string]interface{}
	User   *UserContext
}

type UserContext struct {
	ID    int
	Email string
	Role  string
}

func NewRequest(req *http.Request) *Request {
	params := extractParams(req)
	query := extractQuery(req)
	body := parseBody(req)
	user := extractUser(req)
log.Print(params)
	return &Request{
		Params: params,
		Query:  query,
		Body:   body,
		User:   user,
	}
}

func extractParams(req *http.Request) map[string]string {
	if params, ok := req.Context().Value("params").(map[string]string); ok {
		return params
	}
	return map[string]string{}
}

func extractQuery(req *http.Request) map[string]string {
	query := map[string]string{}
	for k, v := range req.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	return query
}

func parseBody(req *http.Request) map[string]interface{} {
	body := map[string]interface{}{}
	if req.Body == nil {
		return body
	}
	defer req.Body.Close()
	_ = json.NewDecoder(req.Body).Decode(&body)
	return body
}


// TODO 認証導入時に調整
func extractUser(req *http.Request) *UserContext {
	if user, ok := req.Context().Value("user").(*UserContext); ok {
		return user
	}
	return nil
}