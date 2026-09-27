package web_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	web.WriteJSON(recorder, http.StatusCreated, map[string]string{"id": "1"})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("got status %d, want 201", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("got content type %q", got)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["id"] != "1" {
		t.Fatalf("got body %v", body)
	}
}

func TestWriteErrorRendersAppError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	web.WriteError(recorder, request, web.NewError(http.StatusConflict, "Taken"))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("got status %d, want 409", recorder.Code)
	}

	var body errorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.StatusCode != 409 || body.Message != "Taken" ||
		body.Error != "Conflict" {
		t.Fatalf("got body %+v", body)
	}
}

func TestWriteErrorHidesUnhandledErrors(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	web.WriteError(recorder, request, errors.New("database exploded"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want 500", recorder.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["message"] != "Internal server error" {
		t.Fatalf("got body %v", body)
	}
	if _, leaked := body["error"]; leaked {
		t.Fatalf("unexpected error label in body %v", body)
	}
	if body["message"] == "database exploded" {
		t.Fatal("internal error detail leaked to the client")
	}
}
