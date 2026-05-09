package errors

import (
	"fmt"
)

// ErrorCode represents a specific type of wisdom error.
type ErrorCode string

const (
	CodeInvalidParams     ErrorCode = "INVALID_PARAMS"
	CodeNotFound          ErrorCode = "NOT_FOUND"
	CodeInternal          ErrorCode = "INTERNAL_ERROR"
	CodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	CodeUnavailable       ErrorCode = "UNAVAILABLE"
	CodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
)

// WisdomError is a structured error for the Wisdom engine.
type WisdomError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *WisdomError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code ErrorCode, msg string) *WisdomError {
	return &WisdomError{Code: code, Message: msg}
}

func Wrap(code ErrorCode, msg string, err error) *WisdomError {
	return &WisdomError{Code: code, Message: msg, Err: err}
}
