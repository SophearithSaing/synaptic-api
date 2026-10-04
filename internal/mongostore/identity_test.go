package mongostore_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
)

// wiring is the identity stack over a real replica-set database.
type wiring struct {
	mux *http.ServeMux
	db  *mongo.Database
}

// newWiring starts the Mongo container and wires the identity stack
// with production cookie options.
func newWiring(t *testing.T) *wiring {
	t.Helper()

	ctx := context.Background()
	uri := startMongo(t, ctx)

	client, err := mongostore.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_ = client.Disconnect(cleanupCtx)
	})

	database := client.Database("identitytest")
	if err := mongostore.EnsureIdentityIndexes(ctx, database); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	store := mongostore.NewIdentityStore(database)
	issuer := identity.NewTokenIssuer(
		"secret", "synaptic", "synaptic-client", time.Hour,
	)
	options := identity.Options{
		AccessTTL:     time.Hour,
		RefreshTTL:    7 * 24 * time.Hour,
		SecureCookies: true,
	}
	service := identity.NewService(store, issuer, options)
	handler := identity.NewHandler(
		service, identity.NewAuthenticator(issuer, store), options,
	)
	mux := http.NewServeMux()
	handler.Mount(mux)

	return &wiring{mux: mux, db: database}
}

// register posts a registration body and asserts the pinned 201.
func register(t *testing.T, w *wiring, username, email string) *http.Response {
	t.Helper()

	token := csrfToken(w)

	request := httptest.NewRequest(http.MethodPost, "/auth/register",
		strings.NewReader(`{"username":"`+username+
			`","email":"`+email+`","password":"Password1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", "csrf_token="+token)
	request.Header.Set("X-CSRF-Token", token)
	recorder := httptest.NewRecorder()
	w.mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("register around %s: status %d body %s",
			username, recorder.Code, recorder.Body.String())
	}

	return recorder.Result()
}

// csrfToken returns a fresh CSRF token from the wiring.
func csrfToken(w *wiring) string {
	recorder := httptest.NewRecorder()
	w.mux.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/auth/csrf", nil),
	)
	var body map[string]string
	if json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
		return ""
	}

	return body["csrf_token"]
}

// cookieValue reads one named cookie out of a response.
func cookieValue(response *http.Response, name string) (string, bool) {
	for _, cookie := range response.Cookies() {
		if cookie.Name == name {
			return cookie.Value, true
		}
	}

	return "", false
}

// refreshPost calls POST /auth/refresh with a refresh cookie.
func refreshPost(t *testing.T, w *wiring, refreshToken string) *http.Response {
	t.Helper()

	token := csrfToken(w)
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	request.Header.Set("Cookie", "csrf_token="+token+
		"; refresh_token="+refreshToken)
	request.Header.Set("X-CSRF-Token", token)

	served := httptest.NewRecorder()
	w.mux.ServeHTTP(served, request)

	return served.Result()
}

// loginPost calls POST /auth/login with an identifier.
func loginPost(
	t *testing.T, w *wiring, identifier, password string,
) *http.Response {
	t.Helper()

	token := csrfToken(w)
	request := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"identifier":"`+identifier+
			`","password":"`+password+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", "csrf_token="+token)
	request.Header.Set("X-CSRF-Token", token)

	served := httptest.NewRecorder()
	w.mux.ServeHTTP(served, request)

	return served.Result()
}

// logoutPost calls POST /auth/logout with an optional refresh cookie.
func logoutPost(
	t *testing.T, w *wiring, refreshToken string,
) *http.Response {
	t.Helper()

	token := csrfToken(w)
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	if refreshToken != "" {
		request.Header.Set("Cookie", "csrf_token="+token+
			"; refresh_token="+refreshToken)
	} else {
		request.Header.Set("Cookie", "csrf_token="+token)
	}
	request.Header.Set("X-CSRF-Token", token)

	served := httptest.NewRecorder()
	w.mux.ServeHTTP(served, request)

	return served.Result()
}

// TestRealRegisterPersistsExactBSON verifies the users/authSessions
// document shapes.
func TestRealRegisterPersistsExactBSON(t *testing.T) {
	w := newWiring(t)
	response := register(t, w, "newuser", "newuser@example.com")

	_, ok := cookieValue(response, "access_token")
	if !ok {
		t.Fatal("access_token cookie missing")
	}
	refresh, ok := cookieValue(response, "refresh_token")
	if !ok {
		t.Fatal("refresh_token cookie missing")
	}

	var user map[string]any
	if err := w.db.Collection("users").FindOne(context.Background(),
		bson.M{"username": "newuser"}).Decode(&user); err != nil {
		t.Fatalf("load user: %v", err)
	}
	for _, key := range []string{
		"username", "email", "password", "role", "createdAt", "updatedAt",
		"__v",
	} {
		if _, present := user[key]; !present {
			t.Errorf("users document missing %s", key)
		}
	}
	if user["role"] != "user" || int(user["__v"].(int32)) != 0 {
		t.Errorf("user role/version %v/%v", user["role"], user["__v"])
	}

	var session map[string]any
	err := w.db.Collection("authSessions").FindOne(
		context.Background(),
		bson.M{"_id": mustObjectID(t, sessionIDOf(refresh))},
	).Decode(&session)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	for _, key := range []string{
		"userId", "refreshTokenHash", "expiresAt", "createdAt",
		"updatedAt", "__v",
	} {
		if _, present := session[key]; !present {
			t.Errorf("authSessions document missing %s", key)
		}
	}
	if _, revoked := session["revokedAt"]; revoked {
		t.Error("new authSession must omit revokedAt")
	}
	expiresAt, ok := session["expiresAt"].(bson.DateTime)
	if !ok {
		t.Fatalf("expiresAt type %T", session["expiresAt"])
	}
	expiry := time.UnixMilli(int64(expiresAt))
	if !expiry.After(time.Now().Add(6 * 24 * time.Hour)) {
		t.Errorf("expiresAt %v", expiry)
	}
}

// mustObjectID parses a hex id.
func mustObjectID(t *testing.T, hex string) bson.ObjectID {
	t.Helper()

	objectID, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		t.Fatalf("object id %q: %v", hex, err)
	}

	return objectID
}

// sessionIDOf splits the session id from a refresh token.
func sessionIDOf(refreshToken string) string {
	sessionID, _, _ := strings.Cut(refreshToken, ".")

	return sessionID
}

// TestRealRegisterDuplicates pins the 409 bodies with real indexes.
func TestRealRegisterDuplicates(t *testing.T) {
	w := newWiring(t)
	register(t, w, "student", "student@example.com")

	token := csrfToken(w)
	duplicate := func(username, email string) *http.Response {
		request := httptest.NewRequest(http.MethodPost, "/auth/register",
			strings.NewReader(`{"username":"`+username+
				`","email":"`+email+`","password":"Password1"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Cookie", "csrf_token="+token)
		request.Header.Set("X-CSRF-Token", token)
		served := httptest.NewRecorder()
		w.mux.ServeHTTP(served, request)

		return served.Result()
	}

	caseDuplicate := duplicate("STUDENT", "other@example.com")
	if caseDuplicate.StatusCode != http.StatusConflict {
		t.Fatalf("username duplicate status %d", caseDuplicate.StatusCode)
	}
	if got := errorMessage(caseDuplicate); got != "Username already exists" {
		t.Fatalf("username duplicate message %q", got)
	}

	emailDuplicate := duplicate("othername", "STUDENT@example.com")
	if emailDuplicate.StatusCode != http.StatusConflict {
		t.Fatalf("email duplicate status %d", emailDuplicate.StatusCode)
	}
	if got := errorMessage(emailDuplicate); got != "Email already exists" {
		t.Fatalf("email duplicate message %q", got)
	}
}

// errorMessage extracts the message field of an error body.
func errorMessage(response *http.Response) string {
	var body map[string]any
	if json.NewDecoder(response.Body).Decode(&body) != nil {
		return ""
	}
	message, _ := body["message"].(string)

	return message
}

// TestRealLoginPaths covers username, case-insensitive, and email
// lookups with real data.
func TestRealLoginPaths(t *testing.T) {
	w := newWiring(t)
	register(t, w, "pathuser", "paths@example.com")

	for _, identifier := range []string{
		"pathuser", "PathUser", "paths@example.com",
	} {
		if got := loginPost(t, w, identifier, "Password1"); got.StatusCode !=
			http.StatusCreated {
			t.Fatalf("login %q status %d", identifier, got.StatusCode)
		}
	}

	wrong := loginPost(t, w, "pathuser", "WrongPass1")
	if wrong.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password status %d", wrong.StatusCode)
	}
}

// TestRealConcurrentRefreshRotation pins single-winner rotation.
func TestRealConcurrentRefreshRotation(t *testing.T) {
	w := newWiring(t)
	response := register(t, w, "rotator", "rotate@example.com")
	refresh, ok := cookieValue(response, "refresh_token")
	if !ok {
		t.Fatal("refresh_token cookie missing")
	}

	token := csrfToken(w)
	attempts := 8
	results := make([]int, attempts)

	var waiter sync.WaitGroup
	for attempt := range attempts {
		waiter.Add(1)
		go func(slot int) {
			defer waiter.Done()

			request := httptest.NewRequest(
				http.MethodPost, "/auth/refresh", nil,
			)
			request.Header.Set("Cookie", "csrf_token="+token+
				"; refresh_token="+refresh)
			request.Header.Set("X-CSRF-Token", token)
			served := httptest.NewRecorder()
			w.mux.ServeHTTP(served, request)
			results[slot] = served.Code
		}(attempt)
	}
	waiter.Wait()

	succeeded := 0
	for _, code := range results {
		if code == http.StatusCreated {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("rotations succeeded %d, want 1 (%v)", succeeded, results)
	}
}

// TestRealLogoutVerification pins logout secret verification.
func TestRealLogoutVerification(t *testing.T) {
	w := newWiring(t)
	response := register(t, w, "logoutuser", "logout@example.com")
	refresh, ok := cookieValue(response, "refresh_token")
	if !ok {
		t.Fatal("refresh_token cookie missing")
	}

	logout := logoutPost(t, w, refresh)
	if logout.StatusCode != http.StatusCreated {
		t.Fatalf("logout status %d", logout.StatusCode)
	}

	after := refreshPost(t, w, refresh)
	if after.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout refresh status %d", after.StatusCode)
	}

	// The session document is revoked.
	var session map[string]any
	if err := w.db.Collection("authSessions").FindOne(context.Background(),
		bson.M{"_id": mustObjectID(t, sessionIDOf(refresh))}).
		Decode(&session); err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, revoked := session["revokedAt"]; !revoked {
		t.Fatal("session missing revokedAt after logout")
	}
}

// TestRealLogoutWrongSecretKeepsSession pins the accepted change: a
// wrong secret logs out with 201 and empty body but never revokes.
func TestRealLogoutWrongSecretKeepsSession(t *testing.T) {
	w := newWiring(t)
	response := register(t, w, "wrongsecret", "wrongsecret@example.com")
	refresh, ok := cookieValue(response, "refresh_token")
	if !ok {
		t.Fatal("refresh_token cookie missing")
	}

	sessionID, _, _ := strings.Cut(refresh, ".")
	wrong := refresh[:len(sessionID)+1] + "wrongwrongwrongwrongwrongwron"

	forged := logoutPost(t, w, wrong)
	if forged.StatusCode != http.StatusCreated {
		t.Fatalf("forged logout status %d", forged.StatusCode)
	}

	var session map[string]any
	if err := w.db.Collection("authSessions").FindOne(context.Background(),
		bson.M{"_id": mustObjectID(t, sessionID)}).Decode(&session); err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, revoked := session["revokedAt"]; revoked {
		t.Fatal("wrong secret must not revoke the session")
	}
}

// TestRealLogoutWithoutCookieStillSucceeds pins the no-cookie fixture.
func TestRealLogoutWithoutCookieStillSucceeds(t *testing.T) {
	w := newWiring(t)

	logout := logoutPost(t, w, "")
	if logout.StatusCode != http.StatusCreated {
		t.Fatalf("logout status %d", logout.StatusCode)
	}

	by, err := io.ReadAll(logout.Body)
	if err != nil {
		t.Fatalf("read logout body: %v", err)
	}
	if len(by) != 0 {
		t.Fatalf("logout body %q", by)
	}
}
