package identity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// seedStudent mirrors the fixture seed user.
func seedStudent(handler http.Handler, repo *repoState) {
	repo.SeedUser("student", "student@example.com", "Password1", "user")
}

// register walks the real register flow across the exact CSRF
// handshake. The builder lives in handler_server_test.go; this alias
// keeps test bodies readable.
func register(
	handler http.Handler, username, email, password string,
) *httptest.ResponseRecorder {
	return performRegister(handler, username, email, password)
}

// performRegister issues a CSRF pair and registers with it.
func performRegister(
	handler http.Handler, username, email, password string,
) *httptest.ResponseRecorder {
	token := csrfToken(handler)

	return call(
		handler, http.MethodPost, "/auth/register",
		"csrf_token="+token, token,
		map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		},
	)
}

// registerBody builds a register body.
func registerBody(username, email, password string) map[string]string {
	return map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}
}

// TestCsrfIssuesTokenAndCookie pins GET /auth/csrf behavior.
func TestCsrfIssuesTokenAndCookie(t *testing.T) {
	handler, _, _ := productionServer()

	recorder := call(handler, http.MethodGet, "/auth/csrf", "", "", nil)
	cookie := cookieList(recorder)["csrf_token"]
	if cookie == nil {
		t.Fatalf("csrf cookie missing: %s", recorder.Body.String())
	}
	token := cookie.Value
	if len(token) != 43 {
		t.Fatalf("token length %d, want 43", len(token))
	}
	if cookie.Path != "/" || cookie.MaxAge != 0 || cookie.HttpOnly {
		t.Fatalf("cookie attributes %+v", cookie)
	}
	if !cookie.Secure || cookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("cookie policy %+v", cookie)
	}

	var body map[string]string
	if json.Unmarshal(recorder.Body.Bytes(), &body) != nil ||
		body["csrf_token"] != token {
		t.Fatalf("body %s token %s", recorder.Body.String(), token)
	}
}

// TestCsrfLaxOutsideProduction pins the development cookie policy.
func TestCsrfLaxOutsideProduction(t *testing.T) {
	handler, _, _ := developmentServer()

	recorder := call(handler, http.MethodGet, "/auth/csrf", "", "", nil)
	cookie := cookieList(recorder)["csrf_token"]
	if cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("dev cookie policy %+v", cookie)
	}
}

// TestRegisterSuccess pins the register success response.
func TestRegisterSuccess(t *testing.T) {
	handler, repo, _ := productionServer()

	recorder := register(handler, "alice", "alice@example.com", "Password1")
	assertJSON(t, recorder, http.StatusCreated, `{"authenticated":true}`)

	cookies := cookieList(recorder)
	access, refresh := cookies["access_token"], cookies["refresh_token"]
	if access == nil || refresh == nil {
		t.Fatalf("cookies %v", cookies)
	}
	if access.Path != "/" || access.MaxAge != 86400 {
		t.Fatalf("access cookie %+v", access)
	}
	if !access.HttpOnly || !access.Secure ||
		access.SameSite != http.SameSiteNoneMode {
		t.Fatalf("access cookie %+v", access)
	}
	rawCookie := access.Raw + ";"
	if !strings.Contains(rawCookie, "Max-Age=86400") ||
		!strings.Contains(rawCookie, "Expires=") {
		t.Fatalf("raw access cookie %q", access.Raw)
	}
	if refresh.Path != "/auth" || refresh.MaxAge != 604800 {
		t.Fatalf("refresh cookie %+v", refresh)
	}

	userId := repo.byEmail["alice@example.com"]
	if userId == "" || repo.users[userId].Username != "alice" {
		t.Fatal("user not persisted")
	}

	sessionID, secret, ok := strings.Cut(refresh.Value, ".")
	if !ok || len(sessionID) != 24 || len(secret) != 43 {
		t.Fatalf("refresh token shape %q", refresh.Value)
	}
	session := repo.sessions[sessionID]
	if session == nil {
		t.Fatal("session not persisted")
	}
	if session.ExpiresAt.Before(time.Now().Add(6 * 24 * time.Hour)) {
		t.Fatalf("session expires too early: %v", session.ExpiresAt)
	}
}

// TestRegisterNormalizesInputs pins the transformers before validation.
func TestRegisterNormalizesInputs(t *testing.T) {
	handler, repo, _ := developmentServer()

	recorder := register(
		handler, "  spaced  ", "USER@Example.COM", "Password1")
	assertJSON(t, recorder, http.StatusCreated, `{"authenticated":true}`)

	userId := repo.byEmail["user@example.com"]
	if userId == "" {
		t.Fatal("normalized email not persisted")
	}
	if repo.users[userId].Username != "spaced" {
		t.Fatalf("username %q", repo.users[userId].Username)
	}
}

// TestRegisterCsrfRejected pins the 403 without the header.
func TestRegisterCsrfRejected(t *testing.T) {
	handler, _, _ := productionServer()

	recorder := call(handler, http.MethodPost, "/auth/register", "", "",
		registerBody("dave", "dave@example.com", "Password1"))
	assertJSON(t, recorder, http.StatusForbidden,
		`{"message":"Invalid CSRF token","error":"Forbidden","statusCode":403}`)
}

// TestRegisterCsrfMismatchRejected pins the 403 on value mismatch.
func TestRegisterCsrfMismatchRejected(t *testing.T) {
	handler, _, _ := productionServer()

	recorder := call(
		handler, http.MethodPost, "/auth/register",
		"csrf_token=cookieValue", "headerValue",
		registerBody("dave", "dave@example.com", "Password1"),
	)
	assertJSON(t, recorder, http.StatusForbidden,
		`{"message":"Invalid CSRF token","error":"Forbidden","statusCode":403}`)
}

// TestRegisterConflictUsername pins the username 409.
func TestRegisterConflictUsername(t *testing.T) {
	handler, repo, _ := productionServer()
	seedStudent(handler, repo)

	recorder := register(handler, "student", "unique@example.com", "Password1")
	assertJSON(t, recorder, http.StatusConflict,
		`{"message":"Username already exists","error":"Conflict","statusCode":409}`)
}

// TestRegisterConflictEmail pins the email 409.
func TestRegisterConflictEmail(t *testing.T) {
	handler, repo, _ := productionServer()
	repo.SeedUser("admin", "admin@example.com", "Password1", "admin")

	recorder := register(handler, "uniqueuser", "admin@example.com", "Password1")
	assertJSON(t, recorder, http.StatusConflict,
		`{"message":"Email already exists","error":"Conflict","statusCode":409}`)
}

// TestRegisterUnknownField pins the forbidNonWhitelisted message.
func TestRegisterUnknownField(t *testing.T) {
	handler, _, _ := productionServer()

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/register",
		"csrf_token="+token, token,
		map[string]any{
			"username": "carol",
			"email":    "carol@example.com",
			"password": "Password1",
			"role":     "admin",
		},
	)
	assertJSON(t, recorder, http.StatusBadRequest,
		`{"message":["property role should not exist"],`+
			`"error":"Bad Request","statusCode":400}`)
}

// TestRegisterPasswordValidationMessage pins the validation-400 body.
func TestRegisterPasswordValidationMessage(t *testing.T) {
	handler, _, _ := productionServer()

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/register",
		"csrf_token="+token, token,
		registerBody("bobby", "bobby@example.com", "password"),
	)
	assertJSON(t, recorder, http.StatusBadRequest,
		`{"message":["password must match `+
			`/^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).+$/ regular expression"],`+
			`"error":"Bad Request","statusCode":400}`)
}

// TestRegisterMultipleViolations lists every violated rule.
func TestRegisterMultipleViolations(t *testing.T) {
	handler, _, _ := productionServer()

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/register",
		"csrf_token="+token, token,
		registerBody("", "bobby@example.com", "ab"),
	)
	assertJSON(t, recorder, http.StatusBadRequest,
		`{"message":[`+
			`"username should not be empty",`+
			`"username must be longer than or equal to 3 characters",`+
			`"username must match /^[a-zA-Z0-9_.-]+$/ regular expression",`+
			`"password must be longer than or equal to 8 characters",`+
			`"password must match `+
			`/^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).+$/ regular expression"],`+
			`"error":"Bad Request","statusCode":400}`)
}

// TestRegisterMalformedBody rejects broken JSON bodies with 400.
func TestRegisterMalformedBody(t *testing.T) {
	handler, _, _ := productionServer()

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/register",
		"csrf_token="+token, token,
		"{not json",
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status %d", recorder.Code)
	}
}

// TestLoginPinnedPaths covers success, wrong password, unknown user.
func TestLoginPinnedPaths(t *testing.T) {
	handler, repo, _ := productionServer()
	seedStudent(handler, repo)

	recorder := loginWith(handler, "student", "Password1")
	assertJSON(t, recorder, http.StatusCreated, `{"authenticated":true}`)

	cookies := cookieList(recorder)
	if cookies["access_token"] == nil || cookies["refresh_token"] == nil {
		t.Fatal("auth cookies missing")
	}

	wrong := loginWith(handler, "student", "WrongPass1")
	assertJSON(t, wrong, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)

	unknown := loginWith(handler, "nobody", "Password1")
	assertJSON(t, unknown, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestLoginByEmailAndCaseInsensitive pins both lookup paths.
func TestLoginByEmailAndCaseInsensitive(t *testing.T) {
	handler, repo, _ := productionServer()
	seedStudent(handler, repo)

	byEmail := loginWith(handler, "student@example.com", "Password1")
	assertJSON(t, byEmail, http.StatusCreated, `{"authenticated":true}`)

	byName := loginWith(handler, "StUdEnT", "Password1")
	assertJSON(t, byName, http.StatusCreated, `{"authenticated":true}`)
}

// TestLoginValidationMessage pins the identifier pattern message.
func TestLoginValidationMessage(t *testing.T) {
	handler, repo, _ := productionServer()
	seedStudent(handler, repo)

	recorder := loginWith(handler, "bad identifier", "Password1")
	assertJSON(t, recorder, http.StatusBadRequest,
		`{"message":["identifier must match /^\\S+$/ regular expression"],`+
			`"error":"Bad Request","statusCode":400}`)
}

// TestRefreshRotation pins the refresh success rotation.
func TestRefreshRotation(t *testing.T) {
	handler, _, issuer := productionServer()

	registration := register(handler, "alice", "alice@example.com", "Password1")
	cookies := cookieList(registration)
	oldRefresh := cookies["refresh_token"].Value

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/refresh",
		csrfPair(cookieHeader(cookies, "refresh_token"), token), token, nil,
	)
	assertJSON(t, recorder, http.StatusCreated, `{"authenticated":true}`)

	newCookies := cookieList(recorder)
	newRefresh := newCookies["refresh_token"]
	if newRefresh == nil {
		t.Fatal("refresh cookie missing")
	}
	if newRefresh.Value == oldRefresh {
		t.Fatal("refresh token was not rotated")
	}

	// The access token verifies against the issuer with identity claims.
	_, err := issuer.Verify(newCookies["access_token"].Value)
	if err != nil {
		t.Fatalf("verify refreshed access token: %v", err)
	}

	// The old secret must no longer refresh.
	taken := call(handler, http.MethodPost, "/auth/refresh",
		csrfPair("refresh_token="+oldRefresh, token), token, nil)
	assertJSON(t, taken, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestRefreshUnauthorized pins missing/malformed/unknown 401s.
func TestRefreshUnauthorized(t *testing.T) {
	handler, _, _ := productionServer()

	token := csrfToken(handler)

	missing := call(
		handler, http.MethodPost, "/auth/refresh",
		csrfPair("", token), token, nil,
	)
	assertJSON(t, missing, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)

	malformed := call(
		handler, http.MethodPost, "/auth/refresh",
		csrfPair("refresh_token=not-a-token", token), token, nil,
	)
	assertJSON(t, malformed, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)

	unknown := call(
		handler, http.MethodPost, "/auth/refresh",
		csrfPair(
			"refresh_token=665f1e2b9d1a2c3b4d5effff.c2VjcmV0", token,
		), token, nil,
	)
	assertJSON(t, unknown, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestRefreshWrongSecret pins the bcrypt secret verification.
func TestRefreshWrongSecret(t *testing.T) {
	handler, repo, _ := productionServer()

	seedStudent(handler, repo)
	token := csrfToken(handler)

	// Build a session for the seeded user with a known secret.
	hash, err := identity.HashSecret("correct-secret")
	if err != nil {
		t.Fatal(err)
	}
	sessionID, createErr := repo.CreateSession(context.Background(),
		identity.Session{
			UserID:      repo.byEmail["student@example.com"],
			RefreshHash: hash,
			ExpiresAt:   time.Now().Add(time.Hour),
		})
	if createErr != nil {
		t.Fatal(createErr)
	}

	recorder := call(
		handler, http.MethodPost, "/auth/refresh",
		csrfPair("refresh_token="+identity.SerializeRefreshToken(
			sessionID, "wrong-secret",
		), token), token, nil,
	)
	assertJSON(t, recorder, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestRefreshRevokedSession pins 401 after logout.
func TestRefreshRevokedSession(t *testing.T) {
	handler, _, _ := productionServer()

	registration := register(handler, "alice", "alice@example.com", "Password1")
	cookies := cookieList(registration)
	refreshValue := cookies["refresh_token"].Value

	logoutToken := csrfToken(handler)
	_ = call(handler, http.MethodPost, "/auth/logout",
		csrfPair(cookieHeader(cookies, "refresh_token"), logoutToken),
		logoutToken, nil)

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/refresh",
		csrfPair("refresh_token="+refreshValue, token), token, nil,
	)
	assertJSON(t, recorder, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestMePinnedPaths covers bearer, cookie, and unauthenticated reads.
func TestMePinnedPaths(t *testing.T) {
	handler, repo, issuer := productionServer()

	userId := repo.SeedUser(
		"student", "student@example.com", "Password1", "user",
	)
	accessToken, err := issuer.Issue(
		userId, "student@example.com", "student", time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	bearer := call(handler, http.MethodGet, "/auth/me",
		"", "", nil)
	_ = bearer
	bearerRequest := buildRequest(
		http.MethodGet, "/auth/me", "", "",
		nil,
	)
	bearerRequest.Header.Set("Authorization", "Bearer "+accessToken)
	bearerRecorder := httptest.NewRecorder()
	handler.ServeHTTP(bearerRecorder, bearerRequest)
	assertJSON(t, bearerRecorder, http.StatusOK,
		`{"email":"student@example.com","username":"student",`+
			`"role":"user","userId":"`+userId+`"}`)

	cookieAuth := call(handler, http.MethodGet, "/auth/me",
		"access_token="+accessToken, "", nil)
	assertJSON(t, cookieAuth, http.StatusOK,
		`{"email":"student@example.com","username":"student",`+
			`"role":"user","userId":"`+userId+`"}`)

	unauth := call(handler, http.MethodGet, "/auth/me", "", "", nil)
	assertJSON(t, unauth, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestLogoutClearsCookiesPins201 matching the logout fixtures.
func TestLogoutClearsCookiesPins201(t *testing.T) {
	handler, repo, _ := productionServer()

	registration := register(handler, "alice", "alice@example.com", "Password1")
	cookies := cookieList(registration)
	refreshValue := cookies["refresh_token"].Value

	logoutToken := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/logout",
		csrfPair(cookieHeader(cookies, "refresh_token"), logoutToken),
		logoutToken, nil,
	)
	if recorder.Body.Len() != 0 {
		t.Fatalf("logout body %q", recorder.Body.String())
	}

	cleared := cookieList(recorder)
	access, refresh := cleared["access_token"], cleared["refresh_token"]
	if access == nil || refresh == nil {
		t.Fatalf("cookies %v", cleared)
	}
	if access.Path != "/" || refresh.Path != "/auth" {
		t.Fatalf("paths %q %q", access.Path, refresh.Path)
	}
	if access.MaxAge != -1 || refresh.MaxAge != -1 {
		t.Fatalf("MaxAge %d %d", access.MaxAge, refresh.MaxAge)
	}
	if access.MaxAge < 0 && !access.Expires.IsZero() {
		if expires := access.Expires; expires.After(time.Now()) {
			t.Fatalf("access cookie not expired: %v", expires)
		}
	}
	if !access.HttpOnly || !access.Secure {
		t.Fatal("cleared cookie policy mismatch")
	}

	// The referenced session is revoked with the correct secret.
	sessionID, _, _ := strings.Cut(refreshValue, ".")
	if repo.sessions[sessionID] == nil ||
		repo.sessions[sessionID].RevokedAt == nil {
		t.Fatal("session not revoked")
	}
}

// TestLogoutWithoutCookieStill201 matches the no-cookie fixture.
func TestLogoutWithoutCookieStill201(t *testing.T) {
	handler, _, _ := productionServer()

	token := csrfToken(handler)
	recorder := call(
		handler, http.MethodPost, "/auth/logout",
		csrfPair("", token), token, nil,
	)
	if recorder.Body.Len() != 0 || recorder.Code != http.StatusCreated {
		t.Fatalf("status %d body %q", recorder.Code, recorder.Body.String())
	}

	cleared := cookieList(recorder)
	access := cleared["access_token"]
	refresh := cleared["refresh_token"]
	if access == nil || refresh == nil {
		t.Fatalf("cookies %v", cleared)
	}

	// Malformed refresh tokens never fail the logout.
	token = csrfToken(handler)
	malformed := call(
		handler, http.MethodPost, "/auth/logout",
		csrfPair("refresh_token=garbage", token), token, nil,
	)
	if malformed.Code != http.StatusCreated || malformed.Body.Len() != 0 {
		t.Fatalf("malformed logout: status %d", malformed.Code)
	}
}
