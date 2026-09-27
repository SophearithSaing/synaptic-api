package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// throttleBody is the pinned 429 JSON body.
const throttleBody = `{"statusCode":429,"message":` +
	`"ThrottlerException: Too Many Requests"}` + "\n"

func TestThrottlerAllowsWithinGlobalLimit(t *testing.T) {
	throttler := web.NewThrottler(
		web.ThrottleConfig{Limit: 2, TTL: time.Minute, Block: time.Minute},
		map[string]web.ThrottleConfig{},
	)
	handler := throttler.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for range 2 {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/anything", nil)
		request.RemoteAddr = "10.0.0.1:1234"
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", recorder.Code)
		}
	}
}

func TestThrottlerBlocksWithRetryAfter(t *testing.T) {
	throttler := web.NewThrottler(
		web.ThrottleConfig{Limit: 1, TTL: time.Minute, Block: 5 * time.Minute},
		map[string]web.ThrottleConfig{},
	)
	handler := throttler.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	first := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	request.RemoteAddr = "10.0.0.2:1234"
	handler.ServeHTTP(first, request)
	if first.Code != http.StatusOK {
		t.Fatalf("first request got status %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d, want 429", second.Code)
	}
	if got := second.Body.String(); got != throttleBody {
		t.Fatalf("body %q, want %q", got, throttleBody)
	}
	if got := second.Header().Get("Retry-After"); got != "300" {
		t.Fatalf("Retry-After %q, want 300", got)
	}
}

func TestThrottlerRouteOverride(t *testing.T) {
	throttler := web.NewThrottler(
		web.ThrottleConfig{Limit: 100, TTL: time.Minute, Block: time.Minute},
		map[string]web.ThrottleConfig{
			"POST /auth/register": {
				Limit: 3, TTL: time.Minute, Block: 5 * time.Minute,
			},
		},
	)
	handler := throttler.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for range 3 {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
		request.RemoteAddr = "10.0.0.3:1234"
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("request got status %d, want 200", recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	request.RemoteAddr = "10.0.0.3:1234"
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d, want 429", recorder.Code)
	}
	if got := recorder.Header().Get("Retry-After"); got != "300" {
		t.Fatalf("Retry-After %q, want 300", got)
	}

	// A request to another route stays allowed even while the register
	// override is blocked.
	other := httptest.NewRecorder()
	otherRequest := httptest.NewRequest(http.MethodGet, "/anything", nil)
	otherRequest.RemoteAddr = "10.0.0.3:1234"
	handler.ServeHTTP(other, otherRequest)
	if other.Code != http.StatusOK {
		t.Fatalf("other route got status %d, want 200", other.Code)
	}
}
