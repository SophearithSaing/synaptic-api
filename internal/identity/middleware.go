package identity

import (
	"context"
	"errors"
	"net/http"
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

// RequireCSRF enforces the double-submit token pair on every unsafe
// method: the csrf_token cookie value must equal the X-CSRF-Token
// header value.
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

// Authenticator resolves the bearer-or-cookie access token to the
// current user, reloading the role from persistence on every request.
// It depends only on the user lookup it actually performs.
type Authenticator struct {
	issuer *TokenIssuer
	users  UserResolver
}

// NewAuthenticator builds Authenticator middleware from the token
// issuer and a user resolver.
func NewAuthenticator(issuer *TokenIssuer, users UserResolver) *Authenticator {
	return &Authenticator{issuer: issuer, users: users}
}

// Middleware authenticates the request or fails with 401. Repository
// failures surface as 500s; only missing tokens or identities are 401s.
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

// authenticate extracts the token header-first, verifies it, and
// re-resolves the user. Unauthenticated requests report ErrUnauthorized
// while lookup failures keep their error.
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

	user, err := a.users.FindUserByID(r.Context(), verified.Sub)
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

// CurrentUser returns the authenticated user attached to the request.
// It reports nil outside of authenticated handlers.
func CurrentUser(r *http.Request) *User {
	user, _ := r.Context().Value(userContextKey{}).(*User)

	return user
}

// RequireRole fails with 403 when the authenticated user is missing an
// allowed role. Use after the Authenticator middleware.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := CurrentUser(r)
			if user == nil {
				web.WriteError(w, r, unauthorized)
				return
			}

			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			web.WriteError(w, r, web.NewError(
				http.StatusForbidden, "Access denied",
			))
		})
	}
}
