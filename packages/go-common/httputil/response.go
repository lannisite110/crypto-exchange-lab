package httputil

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
)

// Envelope is the standard JSON response wrapper for REST APIs.
type Envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error *ErrorBody  `json:"error,omitempty"`
}

// ErrorBody mirrors AppError for JSON responses.
type ErrorBody struct {
	Code    apperrors.Code `json:"code"`
	Message string         `json:"message"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// OK writes a 200 success envelope.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Envelope{OK: true, Data: data})
}

// Fail writes an error envelope with an appropriate HTTP status.
func Fail(w http.ResponseWriter, err *apperrors.AppError) {
	status := http.StatusInternalServerError
	switch err.Code {
	case apperrors.CodeInvalidArgument:
		status = http.StatusBadRequest
	case apperrors.CodeNotFound:
		status = http.StatusNotFound
	case apperrors.CodeConflict:
		status = http.StatusConflict
	case apperrors.CodeInsufficient:
		status = http.StatusUnprocessableEntity
	}
	JSON(w, status, Envelope{
		OK: false,
		Error: &ErrorBody{
			Code:    err.Code,
			Message: err.Message,
		},
	})
}
