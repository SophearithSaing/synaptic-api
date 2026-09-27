package identity_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// buildServer wires the identity routes over a fake repository. mode is
// "production" for Secure None cookies or anything else for the dev
// policy.
func buildServer(
	mode string,
) (http.Handler, *repoState, *identity.TokenIssuer) {
	repo := newRepoState()
	issuer := identity.NewTokenIssuer(
		"secret", "synaptic", "synaptic-client", 24*time.Hour)
	options := identity.Options{
		AccessTTL:     24 * time.Hour,
		RefreshTTL:    7 * 24 * time.Hour,
		SecureCookies: mode == "production",
	}
	service := identity.NewService(repo, issuer, options)
	handler := identity.NewHandler(
		service, identity.NewAuthenticator(issuer, repo), options,
	)
	mux := http.NewServeMux()
	handler.Mount(mux)

	return mux, repo, issuer
}

// productionServer and developmentServer are buildServer presets.
var (
	productionServer = func() (http.Handler, *repoState, *identity.TokenIssuer) {
		handler, repo, issuer := buildServer("production")

		return handler, repo, issuer
	}
	developmentServer = func() (http.Handler, *repoState, *identity.TokenIssuer) {
		handler, repo, issuer := buildServer("development")

		return handler, repo, issuer
	}
)

// call performs a request and returns the recorder.
func call(
	handler http.Handler,
	method, path, cookieHeader, csrfHeader string,
	body any,
) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		buildRequest(method, path, cookieHeader, csrfHeader, body),
	)

	return recorder
}

// buildRequest prepares a request like an HTTP client would.
func buildRequest(
	method, path, cookieHeader, csrfHeader string,
	body any,
) *http.Request {
	var reader io.Reader
	if body != nil {
		if raw, ok := body.(string); ok {
			reader = strings.NewReader(raw)
		} else {
			reader = strings.NewReader(mustJSON(body))
		}
	}

	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookieHeader != "" {
		request.Header.Set("Cookie", cookieHeader)
	}
	if csrfHeader != "" {
		request.Header.Set("X-CSRF-Token", csrfHeader)
	}

	return request
}

// mustJSON encodes a body.
func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// csrfToken issues a fresh CSRF token through GET /auth/csrf.
func csrfToken(handler http.Handler) string {
	recorder := call(handler, http.MethodGet, "/auth/csrf", "", "", nil)
	var body map[string]string
	if json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
		panic(recorder.Body.String())
	}

	return body["csrf_token"]
}

// cookieList separates a response's cookies by name.
func cookieList(recorder *httptest.ResponseRecorder) map[string]*http.Cookie {
	cookies := map[string]*http.Cookie{}
	for _, cookie := range recorder.Result().Cookies() {
		cookies[cookie.Name] = cookie
	}

	return cookies
}

// cookieHeader serializes selected cookies for a follow-up request.
func cookieHeader(cookies map[string]*http.Cookie, names ...string) string {
	pairs := make([]string, 0, len(names))
	for _, name := range names {
		if cookies[name] != nil {
			pairs = append(pairs, name+"="+cookies[name].Value)
		}
	}

	return strings.Join(pairs, "; ")
}

// loginBody is the standard login request body.
func loginBody(identifier, password string) map[string]string {
	return map[string]string{"identifier": identifier, "password": password}
}

// assertJSON compares a JSON response exactly, ignoring a trailing
// newline.
func assertJSON(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	status int,
	want string,
) {
	t.Helper()

	if recorder.Code != status {
		t.Fatalf("status %d body %s", recorder.Code, recorder.Body.String())
	}
	if strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("body %s, want %s", recorder.Body.String(), want)
	}
}

// csrfPair merges the csrf_token cookie into a cookie header.
func csrfPair(extra, token string) string {
	if extra == "" {
		return "csrf_token=" + token
	}

	return "csrf_token=" + token + "; " + extra
}

// loginWith issues a CSRF pair and logs in.
func loginWith(
	handler http.Handler, identifier, password string,
) *httptest.ResponseRecorder {
	token := csrfToken(handler)

	return call(
		handler, http.MethodPost, "/auth/login",
		"csrf_token="+token, token, loginBody(identifier, password),
	)
}
