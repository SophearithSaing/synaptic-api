package web_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

func newRouter() http.Handler {
	return web.NewRouter("http://localhost:4200", nil)
}

func TestRootReturnsHelloWorld(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "Hello World!" {
		t.Fatalf("got body %q", recorder.Body.String())
	}
}

func TestHealthLive(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(
		recorder, httptest.NewRequest(http.MethodGet, "/health/live", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", recorder.Code)
	}
}

func TestHealthReady(t *testing.T) {
	t.Parallel()

	t.Run("available", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		newRouter().ServeHTTP(
			recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil),
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", recorder.Code)
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		t.Parallel()

		router := web.NewRouter(
			"http://localhost:4200",
			func(context.Context) error { return errors.New("mongo down") },
		)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(
			recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil),
		)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("got status %d, want 503", recorder.Code)
		}
	})
}

// errorBody mirrors the legacy NestJS error response shape.
type errorBody struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

func TestUnknownRouteMatchesLegacyShape(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(
		recorder, httptest.NewRequest(http.MethodGet, "/nope", nil),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want 404", recorder.Code)
	}

	var body errorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.StatusCode != 404 || body.Error != "Not Found" {
		t.Fatalf("got body %+v", body)
	}
	if body.Message != "Cannot GET /nope" {
		t.Fatalf("got message %q", body.Message)
	}
}

func TestUnsupportedMethodMatchesLegacyShape(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(
		recorder, httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want 404", recorder.Code)
	}

	var body errorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Message != "Cannot POST /" {
		t.Fatalf("got message %q", body.Message)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)

	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got !=
		"http://localhost:4200" {
		t.Fatalf("got allow-origin %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got !=
		"true" {
		t.Fatalf("got allow-credentials %q", got)
	}
}

func TestCORSRejectsOtherOrigins(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)

	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("got unexpected allow-origin %q", got)
	}
}
