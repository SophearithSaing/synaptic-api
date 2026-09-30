package catalog_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// authRepo is the minimal identity backend for catalog handler tests.
// It implements only identity.UserResolver, the smallest interface the
// authenticator depends on: catalog routes only re-resolve users
// by id.
type authRepo struct {
	user *identity.User
}

// FindUserByID implements identity.UserResolver.
func (r *authRepo) FindUserByID(
	_ context.Context, _ string,
) (*identity.User, error) {
	if r.user != nil {
		matched := *r.user

		return &matched, nil
	}

	return nil, nil
}

// buildCatalog wires the catalog routes over a stub repository with a
// seeded authenticated student user.
func buildCatalog(repo catalog.Repository) (http.Handler, string) {
	auth := &authRepo{user: &identity.User{
		ID:       "665f1e2b9d1a2c3b4d5e6f70",
		Username: "student",
		Email:    "student@example.com",
		Role:     identity.RoleUser,
	}}
	issuer := identity.NewTokenIssuer(
		"secret", "synaptic", "synaptic-client", time.Hour,
	)

	token, err := issuer.Issue(
		auth.user.ID, auth.user.Email, auth.user.Username, time.Now(),
	)
	if err != nil {
		panic(err)
	}

	handler := catalog.NewHandler(repo, identity.NewAuthenticator(issuer, auth))
	mux := http.NewServeMux()
	handler.Mount(mux)

	return mux, token
}

// catalogGet calls GET with a bearer token.
func catalogGet(
	t *testing.T,
	handler http.Handler,
	token, path string,
) *http.Response {
	t.Helper()

	served := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(served, request)

	return served.Result()
}
