// Package httpapi exposes the parent control-plane HTTP surface.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		writeJSON(w, http.StatusInternalServerError, errorBody{
			Code:    string(apperr.CodeFailed),
			Message: "internal error",
		})
		return
	}
	status := statusForCode(ae.Code)
	writeJSON(w, status, errorBody{
		Code:    string(ae.Code),
		Message: ae.Message,
		Fields:  ae.Fields,
	})
}

func statusForCode(code apperr.Code) int {
	switch code {
	case apperr.CodeInvalid:
		return http.StatusBadRequest
	case apperr.CodeUnauthorized:
		return http.StatusUnauthorized
	case apperr.CodeForbidden:
		return http.StatusForbidden
	case apperr.CodeNotFound:
		return http.StatusNotFound
	case apperr.CodeConflict:
		return http.StatusConflict
	case apperr.CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
