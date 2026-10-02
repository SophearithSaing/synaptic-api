package web

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Error is an application failure rendered as a stable JSON body
// matching the legacy NestJS error shape. Field order matches the
// legacy encoder output.
type Error struct {
	// Message describes the failure for the client. It may be a string
	// or a list of validation messages.
	Message any `json:"message"`
	// ErrorName is the conventional HTTP error label, when applicable.
	ErrorName string `json:"error,omitempty"`
	// StatusCode is the HTTP status code.
	StatusCode int `json:"statusCode"`
}

// Error returns the message when it is a string, or the status text.
func (e *Error) Error() string {
	if message, ok := e.Message.(string); ok {
		return message
	}

	return http.StatusText(e.StatusCode)
}

// NewError builds an Error with the conventional label for the status.
func NewError(status int, message any) *Error {
	return &Error{
		StatusCode: status,
		Message:    message,
		ErrorName:  http.StatusText(status),
	}
}

// WriteJSON writes the value as a JSON response with the given status.
// The content type matches the pinned legacy header exactly.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// WriteError maps an error to a stable JSON response. Errors that are not
// *Error are logged and rendered as a generic 500 body.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		slog.ErrorContext(r.Context(), "unhandled error", "error", err)
		appErr = &Error{
			StatusCode: http.StatusInternalServerError,
			Message:    "Internal server error",
		}
	}

	WriteJSON(w, appErr.StatusCode, appErr)
}
