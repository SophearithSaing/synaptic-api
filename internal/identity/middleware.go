package identity

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// csrfCookieName and csrfHeaderName are the double-submit token
// locations.
const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

// accessTokenCookieName, refreshTokenCookieName are the auth cookie
// names.
const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
)

// userContextKey is the context key for the authenticated user.
type userContextKey struct{}

// RequireCSRF validates double-submit tokens on unsafe methods.
func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(csrfCookieName)
		header := r.Header.Get(csrfHeaderName)
		if err != nil || header == "" || cookie.Value != header {
			web.WriteError(w, r, web.NewError(
				http.StatusForbidden, "Invalid CSRF token",
			))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isSafeMethod reports whether the method carries no mutation.
func isSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}

	return false
}

// Authenticator resolves access tokens to current users.
type Authenticator struct {
	issuer *TokenIssuer
	users  UserResolver
}

// NewAuthenticator builds authentication middleware.
func NewAuthenticator(issuer *TokenIssuer, users UserResolver) *Authenticator {
	return &Authenticator{issuer: issuer, users: users}
}

// Middleware authenticates a request.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.authenticate(r)

		switch {
		case errors.Is(err, ErrUnauthorized):
			web.WriteError(w, r, unauthorized)
			return
		case err != nil:
			web.WriteError(w, r, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(
			context.WithValue(r.Context(), userContextKey{}, user),
		))
	})
}

// authenticate verifies a request token and resolves its user.
func (a *Authenticator) authenticate(r *http.Request) (*User, error) {
	token := bearerToken(r)
	if token == "" {
		if cookie, err := r.Cookie(accessTokenCookieName); err == nil {
			token = cookie.Value
		}
	}
	if token == "" {
		return nil, ErrUnauthorized
	}

	verified, err := a.issuer.Verify(token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := a.users.GetUserByID(r.Context(), verified.Sub)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

// bearerToken reads the Authorization header of a Bearer request.
func bearerToken(r *http.Request) string {
	const scheme = "Bearer "

	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, scheme) {
		return ""
	}

	return strings.TrimSpace(value[len(scheme):])
}

// CurrentUser returns the request's authenticated user.
func CurrentUser(r *http.Request) *User {
	user, _ := r.Context().Value(userContextKey{}).(*User)

	return user
}

// RequireRole restricts a handler to the allowed roles.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := CurrentUser(r)
			if user == nil {
				web.WriteError(w, r, unauthorized)
				return
			}

			if slices.Contains(roles, user.Role) {
				next.ServeHTTP(w, r)
				return
			}

			web.WriteError(w, r, web.NewError(
				http.StatusForbidden, "Forbidden resource",
			))
		})
	}
}
