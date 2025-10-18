package http_controllers

type Response interface {
	ToJson() []byte
}