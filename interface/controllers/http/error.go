package http_controllers

import (
	"net/http"

	"streaming-server.com/application/usecases/shared"
)

const (
	ValidationError shared.CommonErrorType = "validation"
	NotFoundError   shared.CommonErrorType = "not-found"
	InternalError   shared.CommonErrorType = "internal"
)

// エラータイプ → HTTPステータスコードマップ
var (
	ErrorTypeMapper = map[shared.CommonErrorType]int{
		ValidationError: http.StatusBadRequest,
		NotFoundError:   http.StatusNotFound,
		InternalError:   http.StatusInternalServerError,
	}
	HttpError = &HttpErrorResponse{}
)

// エラー構造体
type HttpErrorResponse struct {}

// ステータスコードを取得する便利メソッド
func (e *HttpErrorResponse) Response(w http.ResponseWriter, usecaseError *shared.UsecaseError) {
	if code, ok := ErrorTypeMapper[usecaseError.Type]; ok {
		http.Error(w, usecaseError.Message, code)
		return
	}
	http.Error(w, usecaseError.Message, http.StatusInternalServerError)
}