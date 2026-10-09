package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

type userResolver struct{ user *identity.User }

func (resolver userResolver) GetUserByID(context.Context, string) (*identity.User, error) {
	return resolver.user, nil
}

type pageRepository struct {
	Page
	err   error
	calls int
}

func (repository *pageRepository) Create(context.Context, Record) (string, error)         { return "", nil }
func (repository *pageRepository) LinkLiveQuestion(context.Context, string, string) error { return nil }
func (repository *pageRepository) List(context.Context, int64, int64) (Page, error) {
	repository.calls++
	return repository.Page, repository.err
}

func TestHandlerAuthenticationPaginationAndResponse(t *testing.T) {
	repository := &pageRepository{Page: Page{Items: []Record{}, Total: 0, Page: 1, Limit: 20}}
	admin := &identity.User{ID: "user", Email: "admin@example.com", Username: "admin", Role: identity.RoleAdmin}
	handler := newTestHandler(t, repository, admin)
	t.Run("unauthorized", func(t *testing.T) {
		issuer := identity.NewTokenIssuer("secret", "issuer", "audience", time.Hour)
		authenticator := identity.NewAuthenticator(issuer, userResolver{user: admin})
		mux := http.NewServeMux()
		NewHandler(repository, authenticator).Mount(mux)
		response := requestHandler(mux, "", "")
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", response.Code)
		}
	})
	t.Run("forbidden", func(t *testing.T) {
		user := &identity.User{ID: "user", Email: "user@example.com", Username: "user", Role: identity.RoleUser}
		response := requestHandler(newTestHandler(t, repository, user), "", "")
		if response.Code != http.StatusForbidden {
			t.Fatalf("status=%d", response.Code)
		}
	})
	t.Run("validation", func(t *testing.T) {
		response := requestHandler(handler, "?limit=0", "")
		if response.Code != http.StatusBadRequest || response.Body.String() != `{"message":["limit must not be less than 1"],"error":"Bad Request","statusCode":400}`+"\n" {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})
	t.Run("defaults", func(t *testing.T) {
		response := requestHandler(handler, "", "")
		if response.Code != http.StatusOK || response.Body.String() != `{"items":[],"total":0,"page":1,"limit":20}`+"\n" {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})
}

func newTestHandler(t *testing.T, repository Repository, user *identity.User) http.Handler {
	t.Helper()
	issuer := identity.NewTokenIssuer("secret", "issuer", "audience", time.Hour)
	authenticator := identity.NewAuthenticator(issuer, userResolver{user: user})
	handler := NewHandler(repository, authenticator)
	mux := http.NewServeMux()
	handler.Mount(mux)
	token, err := issuer.Issue(user.ID, user.Email, user.Username, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(w, r)
	})
}

func requestHandler(handler http.Handler, query string, _ string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/ai/logs"+query, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
