package catalog_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

const validQuestionSetJSON = `{"topic":"665f1e2b9d1a2c3b4d5e0004",` +
	`"setType":"regular","level":0,"questions":[{"id":"q1",` +
	`"type":"mcq","prompt":"Prompt","options":[{"id":"a",` +
	`"text":"Answer"}],"correctOptionId":"a",` +
	`"targetConcepts":["concept"],"feedback":{"correct":"Yes",` +
	`"incorrect":"No"},"rubrics":{"keyPoints":["point"],` +
	`"misconceptions":["mistake"]}}]}`

// buildAdminCatalog builds catalog routes with the real authoring service.
func buildAdminCatalog(t *testing.T) (http.Handler, string) {
	return buildAdminCatalogWith(t, &authoringRepository{})
}

// buildAdminCatalogWith builds admin routes over the supplied authoring stub.
func buildAdminCatalogWith(
	t *testing.T, authoring *authoringRepository,
) (http.Handler, string) {
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
		catalog.NewService(authoring, validator),
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

// TestBulkQuestionSetDecoderKeepsStrictBodyFailures verifies malformed,
// unknown, nested-invalid, trailing, and oversized bodies stay bad requests.
func TestBulkQuestionSetDecoderKeepsStrictBodyFailures(t *testing.T) {
	handler, token := buildAdminCatalog(t)
	bodies := []string{
		"[", `[{"unknown":true}]`, `[{"questions":{}}]`, "[] ]",
		"[" + strings.Repeat(" ", (5<<20)+1) + "]",
	}
	for _, body := range bodies {
		response := catalogAdminRequest(t, handler, token,
			http.MethodPost, "/questions/create", body)
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("strict body status %d", response.StatusCode)
		}
	}
}

// TestAuthoringRoutesSucceedThroughService verifies every write route uses
// the service result and its pinned successful status.
func TestAuthoringRoutesSucceedThroughService(t *testing.T) {
	set := &catalog.QuestionSet{ID: "665f1e2b9d1a2c3b4d5e0006",
		TopicID: "665f1e2b9d1a2c3b4d5e0004", SetType: "regular",
		Questions: []catalog.Question{}}
	stub := &authoringRepository{
		category: &catalog.Category{ID: "1", Title: "Category"},
		topic:    &catalog.Topic{ID: "2", Title: "Topic"}, question: set,
	}
	handler, token := buildAdminCatalogWith(t, stub)
	cases := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPost, "/categories/category/create",
			`{"title":"Category","slug":"category","description":"d","icon":"i"}`,
			http.StatusCreated},
		{http.MethodPost, "/topics/create",
			`{"title":"Topic","slug":"topic","description":"d","icon":"i",` +
				`"tags":["tag"],"category":"665f1e2b9d1a2c3b4d5e0003"}`,
			http.StatusCreated},
		{http.MethodPost, "/questions/create", "[" + validQuestionSetJSON + "]",
			http.StatusCreated},
		{http.MethodPatch, "/questions/update",
			`[{"id":"665f1e2b9d1a2c3b4d5e0006","level":0}]`, http.StatusOK},
		{http.MethodPatch, "/questions/665f1e2b9d1a2c3b4d5e0006",
			`{"level":0}`, http.StatusOK},
		{http.MethodDelete, "/categories/665f1e2b9d1a2c3b4d5e0003", "",
			http.StatusNoContent},
		{http.MethodDelete, "/topics/665f1e2b9d1a2c3b4d5e0004", "",
			http.StatusNoContent},
		{http.MethodDelete, "/questions/665f1e2b9d1a2c3b4d5e0006", "",
			http.StatusNoContent},
	}
	for _, test := range cases {
		response := catalogAdminRequest(t, handler, token,
			test.method, test.path, test.body)
		if response.StatusCode != test.status {
			t.Fatalf("%s %s: status %d, want %d", test.method, test.path,
				response.StatusCode, test.status)
		}
	}
}

// TestAuthoringGuardsPinPrecedenceAndRoleMessage covers bearer and cookie CSRF
// guard precedence before request decoding.
func TestAuthoringGuardsPinPrecedenceAndRoleMessage(t *testing.T) {
	admin, token := buildAdminCatalog(t)
	missingCSRF := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/questions/create",
		strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer "+token)
	admin.ServeHTTP(missingCSRF, request)
	assertBody(t, missingCSRF.Result(), http.StatusForbidden,
		`{"message":"Invalid CSRF token","error":"Forbidden","statusCode":403}`)

	missingAuth := catalogAdminRequest(t, admin, "", http.MethodPost,
		"/questions/create", "{}")
	assertBody(t, missingAuth, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)

	student, studentToken := buildCatalog(newRepoState())
	forbidden := catalogAdminRequest(t, student, studentToken, http.MethodPost,
		"/questions/create", "{}")
	assertBody(t, forbidden, http.StatusForbidden,
		`{"message":"Forbidden resource","error":"Forbidden","statusCode":403}`)
}

// TestAuthoringRoutesMapServiceErrors verifies reference and storage failures.
func TestAuthoringRoutesMapServiceErrors(t *testing.T) {
	handler, token := buildAdminCatalogWith(t, &authoringRepository{
		err: catalog.ErrTopicReferenced,
	})
	referenced := catalogAdminRequest(t, handler, token, http.MethodDelete,
		"/topics/665f1e2b9d1a2c3b4d5e0004", "")
	if referenced.StatusCode != http.StatusConflict {
		t.Fatalf("referenced status %d", referenced.StatusCode)
	}

	handler, token = buildAdminCatalogWith(t, &authoringRepository{
		err: errors.New("duplicate key"),
	})
	duplicate := catalogAdminRequest(t, handler, token, http.MethodPost,
		"/categories/category/create",
		`{"title":"Category","slug":"category","description":"d","icon":"i"}`)
	if duplicate.StatusCode != http.StatusInternalServerError {
		t.Fatalf("duplicate status %d", duplicate.StatusCode)
	}
}
