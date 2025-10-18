package shared

type CommonErrorType string

const (
	ValidateError CommonErrorType = "validate"
	NotFoundError CommonErrorType = "not-found"
	InternalError CommonErrorType = "internal"
)

type UsecaseError struct {
	Type    CommonErrorType `json:"type"`
	Message string           `json:"message"`
}

type UsecaseResult[T any] struct {
	success *T
	err     *UsecaseError
}

func NewSuccessResult[T any](data T) *UsecaseResult[T] {
	return &UsecaseResult[T]{success: &data}
}

func NewErrorResult[T any](typ CommonErrorType, msg string) *UsecaseResult[T] {
	return &UsecaseResult[T]{err: &UsecaseError{Type: typ, Message: msg}}
}

func (r *UsecaseResult[T]) IsSuccess() bool {
	return r.err == nil
}

func (r *UsecaseResult[T]) IsError() bool {
	return r.err != nil
}

func (r *UsecaseResult[T]) Success() *T {
	return r.success
}

func (r *UsecaseResult[T]) Error() *UsecaseError {
	return r.err
}