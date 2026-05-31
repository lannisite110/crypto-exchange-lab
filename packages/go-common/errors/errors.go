package errors

import "fmt"

// Code is a stable API error identifier.
type Code string

const (
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeInsufficient    Code = "INSUFFICIENT_BALANCE"
	CodeInternal        Code = "INTERNAL"
)

// AppError is the standard error shape returned by HTTP APIs.
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New creates an AppError with the given code and message.
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Is reports whether err is an AppError with the given code.
func Is(err error, code Code) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*AppError); ok {
		return ae.Code == code
	}
	return false
}
