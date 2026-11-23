package web

import (
	"encoding/json"
	"io"
	"net/http"
)

// maxBodyBytes is the global request body limit, matching the legacy 5 MB
// body parser configuration.
const maxBodyBytes = 5 << 20

// DecodeJSON reads a single JSON value from the request body into value.
// Unknown fields, trailing data, and oversized bodies are rejected.
func DecodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return NewError(http.StatusBadRequest, "Invalid request body")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return NewError(
			http.StatusBadRequest,
			"Request body must contain a single JSON value",
		)
	}

	return nil
}
