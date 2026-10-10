package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

type payload struct {
	Name string `json:"name"`
}

// decodePOST decodes the given body as a POST request.
func decodePOST(t *testing.T, body string) error {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost, "/", strings.NewReader(body),
	)
	recorder := httptest.NewRecorder()

	var value payload
	return web.DecodeJSON(recorder, request, &value)
}

// requireBadRequest asserts the error is a 400 web.Error.
func requireBadRequest(t *testing.T, err error) {
	t.Helper()

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("got error %v, want a *web.Error", err)
	}
	if webErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", webErr.StatusCode)
	}
}

func TestDecodeJSONAcceptsValidBody(t *testing.T) {
	t.Parallel()

	if err := decodePOST(t, `{"name":"synaptic"}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	requireBadRequest(t, decodePOST(t, `{"name":"a","extra":true}`))
}

func TestDecodeJSONRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	requireBadRequest(t, decodePOST(t, `{"name":`))
}

func TestDecodeJSONRejectsTrailingValues(t *testing.T) {
	t.Parallel()

	requireBadRequest(t, decodePOST(t, `{"name":"a"} {"name":"b"}`))
}

func TestDecodeJSONRejectsTrailingClosingDelimiter(t *testing.T) {
	t.Parallel()

	requireBadRequest(t, decodePOST(t, `{"name":"a"} }`))
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	body := `{"name":"` + strings.Repeat("a", 5<<20) + `"}`
	requireBadRequest(t, decodePOST(t, body))
}
