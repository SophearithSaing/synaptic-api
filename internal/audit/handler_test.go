package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	page  int64
	limit int64
}

func (repository *pageRepository) Create(context.Context, Record) (string, error)         { return "", nil }
func (repository *pageRepository) LinkLiveQuestion(context.Context, string, string) error { return nil }
func (repository *pageRepository) List(_ context.Context, page int64, limit int64) (Page, error) {
	repository.calls++
	repository.page, repository.limit = page, limit
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
	t.Run("cookie and bounds", func(t *testing.T) {
		response := requestHandler(handler, "?page=2&limit=100", "cookie")
		if response.Code != http.StatusOK {
			t.Fatalf("cookie status=%d", response.Code)
		}
		if repository.page != 2 || repository.limit != 100 {
			t.Fatalf("args=%d,%d", repository.page, repository.limit)
		}
		response = requestHandler(handler, "?limit=101", "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("bound status=%d", response.Code)
		}
	})
	t.Run("whitelist and accumulated errors", func(t *testing.T) {
		response := requestHandler(handler, "?page=&limit=0&extra=x", "")
		want := `{"message":["property extra should not exist","page must not be less than 1","limit must not be less than 1"],"error":"Bad Request","statusCode":400}` + "\n"
		if response.Code != http.StatusBadRequest || response.Body.String() != want {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})
	t.Run("numeric transforms repeated overflow and role reload", func(t *testing.T) {
		response := requestHandler(handler, "?page=1.0&limit=%201e1%20", "")
		if response.Code != http.StatusOK || repository.page != 1 || repository.limit != 10 {
			t.Fatalf("transform=%d args=%d,%d", response.Code, repository.page, repository.limit)
		}
		response = requestHandler(handler, "?page=1&page=2", "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("repeated=%d", response.Code)
		}
		response = requestHandler(handler, "?page=9223372036854775807&limit=100", "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("overflow=%d", response.Code)
		}
		admin.Role = identity.RoleUser
		response = requestHandler(handler, "", "")
		admin.Role = identity.RoleAdmin
		if response.Code != http.StatusForbidden {
			t.Fatalf("reloaded role=%d", response.Code)
		}
	})
	t.Run("populated and null response shapes", func(t *testing.T) {
		repository.Page = Page{Items: []Record{{ID: "a", Operation: OperationQuestionGeneration, Model: "m", Prompt: "p", Output: "o", LiveQuestion: nil}, {ID: "b", Operation: OperationWrittenGrading, Model: "m", Prompt: "p", Output: "o", LiveQuestion: &LiveQuestion{ID: "q", Status: "pending"}}}, Total: 2, Page: 1, Limit: 20}
		response := requestHandler(handler, "", "")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"liveQuestion":null`) || !strings.Contains(response.Body.String(), `"operation":"written-grading"`) {
			t.Fatalf("body=%s", response.Body.String())
		}
	})
	t.Run("repository error", func(t *testing.T) {
		repository.err = context.DeadlineExceeded
		response := requestHandler(handler, "", "")
		repository.err = nil
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", response.Code)
		}
	})
}

func TestHandlerValidationMatchesContractFixture(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "contract", "fixtures", "get-ai-logs", "validation-400.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Response struct {
			Body any `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(fixture, &contract); err != nil {
		t.Fatal(err)
	}
	repository := &pageRepository{}
	admin := &identity.User{ID: "user", Email: "admin@example.com", Username: "admin", Role: identity.RoleAdmin}
	response := requestHandler(newTestHandler(t, repository, admin), "?limit=0", "")
	var actual any
	if err := json.Unmarshal(response.Body.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusBadRequest || !reflect.DeepEqual(actual, contract.Response.Body) {
		t.Fatalf("actual=%#v contract=%#v", actual, contract.Response.Body)
	}
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
		if r.Header.Get("X-Test-Auth") == "cookie" {
			r.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		} else {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		mux.ServeHTTP(w, r)
	})
}

func requestHandler(handler http.Handler, query string, authentication string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/ai/logs"+query, nil)
	if authentication != "" {
		request.Header.Set("X-Test-Auth", authentication)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
