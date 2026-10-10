package catalog_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// buildAdminCatalog builds catalog routes with the real authoring service.
func buildAdminCatalog(t *testing.T) (http.Handler, string) {
	t.Helper()

	user := &identity.User{
		ID: "665f1e2b9d1a2c3b4d5e6f70", Username: "admin",
		Email: "admin@example.com", Role: identity.RoleAdmin,
	}
	issuer := identity.NewTokenIssuer(
		"secret", "synaptic", "synaptic-client", time.Hour,
	)
	token, err := issuer.Issue(user.ID, user.Email, user.Username, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	validator, err := catalog.NewAuthoringValidator()
	if err != nil {
		t.Fatal(err)
	}
	handler := catalog.NewHandler(newRepoState(),
		catalog.NewService(&authoringRepository{}, validator),
		identity.NewAuthenticator(issuer, &authRepo{user: user}))
	mux := http.NewServeMux()
	handler.Mount(mux)
	return mux, token
}

// catalogAdminRequest sends an authenticated, CSRF-protected admin request.
func catalogAdminRequest(
	t *testing.T, handler http.Handler, token, method, path, body string,
) *http.Response {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-CSRF-Token", "csrf")
	request.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf"})
	handler.ServeHTTP(recorder, request)
	return recorder.Result()
}

// TestBulkQuestionSetDecoderRequiresArray pins all JSON non-array responses.
func TestBulkQuestionSetDecoderRequiresArray(t *testing.T) {
	handler, token := buildAdminCatalog(t)
	for _, path := range []string{"/questions/create", "/questions/update"} {
		for _, body := range []string{"{}", `"value"`, "0", "null"} {
			method := http.MethodPost
			if path == "/questions/update" {
				method = http.MethodPatch
			}
			response := catalogAdminRequest(t, handler, token,
				method, path, body)
			assertBody(t, response, http.StatusBadRequest,
				`{"message":"Validation failed (parsable array expected)",`+
					`"error":"Bad Request","statusCode":400}`)
		}
	}
}

// TestBulkQuestionSetDecoderAcceptsEmptyArray verifies an empty array reaches
// the real authoring service instead of being rejected by decoding.
func TestBulkQuestionSetDecoderAcceptsEmptyArray(t *testing.T) {
	handler, token := buildAdminCatalog(t)

	created := catalogAdminRequest(t, handler, token,
		http.MethodPost, "/questions/create", "[]")
	assertBody(t, created, http.StatusCreated, "[]")

	updated := catalogAdminRequest(t, handler, token,
		http.MethodPatch, "/questions/update", "[]")
	assertBody(t, updated, http.StatusOK, "[]")
}
