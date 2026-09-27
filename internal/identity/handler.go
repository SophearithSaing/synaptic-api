package identity

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// Handler owns the authentication routes.
type Handler struct {
	service       *Service
	authenticator *Authenticator
	options       Options
}

// clearedCookieInstant is the fixed past Expires instant of cleared
// auth cookies, matching Express clearCookie (new Date(1)).
var clearedCookieInstant = time.Unix(1, 0)

// registerOrder and loginOrder are the schema declaration orders used
// for validation message ordering.
var (
	registerOrder = []string{"username", "email", "password"}
	loginOrder    = []string{"identifier", "password"}
)

// authenticatedTrue is the pinned auth status body.
var authenticatedTrue = map[string]bool{"authenticated": true}

// trimLower is the email input transformer.
func trimLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// trim is the username and identifier input transformer.
func trim(value string) string {
	return strings.TrimSpace(value)
}

// authTransforms maps each property to its input transformer.
var authTransforms = map[string]func(string) string{
	"username":   trim,
	"email":      trimLower,
	"identifier": trim,
}

// NewHandler builds the authentication routes handler.
func NewHandler(
	service *Service,
	authenticator *Authenticator,
	options Options,
) *Handler {
	return &Handler{
		service:       service,
		authenticator: authenticator,
		options:       options,
	}
}

// Mount registers the auth routes on the request mux.
func (h *Handler) Mount(mux *http.ServeMux) {
	mux.Handle("POST /auth/register", web.Chain(
		http.HandlerFunc(h.register), RequireCSRF,
	))
	mux.Handle("POST /auth/login", web.Chain(
		http.HandlerFunc(h.login), RequireCSRF,
	))
	mux.Handle("POST /auth/refresh", web.Chain(
		http.HandlerFunc(h.refresh), RequireCSRF,
	))
	mux.Handle("POST /auth/logout", web.Chain(
		http.HandlerFunc(h.logout), RequireCSRF,
	))
	mux.Handle("GET /auth/csrf", http.HandlerFunc(h.csrf))
	mux.Handle("GET /auth/me", h.authenticator.Middleware(
		http.HandlerFunc(h.me),
	))
}

// register validates the body, creates the account, and sets auth
// cookies.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	fields, ok := h.decodeBody(w, r, registerSchema, registerOrder)
	if !ok {
		return
	}

	passwordHash, err := HashPassword(fields["password"])
	if err != nil {
		web.WriteError(w, r, err)
		return
	}

	tokens, err := h.service.Register(r.Context(), Credentials{
		Username:     fields["username"],
		Email:        fields["email"],
		PasswordHash: passwordHash,
	})
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	web.WriteJSON(w, http.StatusCreated, authenticatedTrue)
}

// login validates the body and authenticates the identifier.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	fields, ok := h.decodeBody(w, r, loginSchema, loginOrder)
	if !ok {
		return
	}

	tokens, err := h.service.Login(
		r.Context(), fields["identifier"], fields["password"],
	)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	web.WriteJSON(w, http.StatusCreated, authenticatedTrue)
}

// refresh rotates the session and re-sets both auth cookies.
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(refreshTokenCookieName)

	tokens, err := h.service.Refresh(r.Context(), cookieValue(cookie))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	web.WriteJSON(w, http.StatusCreated, authenticatedTrue)
}

// logout revokes the session and clears both auth cookies with a bare
// 201 and empty body: the response carries only the Set-Cookie
// headers, no Content-Type.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(refreshTokenCookieName)
	_ = h.service.Logout(r.Context(), cookieValue(cookie))

	http.SetCookie(w, h.clearCookie(accessTokenCookieName, "/"))
	http.SetCookie(w, h.clearCookie(refreshTokenCookieName, "/auth"))
	w.WriteHeader(http.StatusCreated)
}

// csrf issues a fresh CSRF token as a JS-readable session cookie.
func (h *Handler) csrf(w http.ResponseWriter, r *http.Request) {
	token, err := RandomSecret()
	if err != nil {
		web.WriteError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.options.SecureCookies,
		SameSite: h.sameSite(),
	})
	web.WriteJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

// me returns the authenticated user reloaded from persistence.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	if user == nil {
		web.WriteError(w, r, unauthorized)
		return
	}

	web.WriteJSON(w, http.StatusOK, meResponse{
		Email:    user.Email,
		Username: user.Username,
		Role:     string(user.Role),
		UserID:   user.ID,
	})
}

// meResponse is the ordered GET /auth/me body shape.
type meResponse struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	UserID   string `json:"userId"`
}

// sameSite picks the environment-specific SameSite policy.
func (h *Handler) sameSite() http.SameSite {
	if h.options.SecureCookies {
		return http.SameSiteNoneMode
	}

	return http.SameSiteLaxMode
}

// authCookie builds an auth cookie with the environment policy.
// Positive maxAge cookies carry both Max-Age and Expires.
func (h *Handler) authCookie(
	name, value, path string,
	maxAge time.Duration,
) *http.Cookie {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   h.options.SecureCookies,
		SameSite: h.sameSite(),
		MaxAge:   int(maxAge / time.Second),
	}
	if maxAge > 0 {
		cookie.Expires = time.Now().Add(maxAge)
	}

	return cookie
}

// clearCookie expires an auth cookie in the far past with no Max-Age,
// matching the pinned Express clearCookie output.
func (h *Handler) clearCookie(name, path string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Path:     path,
		HttpOnly: true,
		Secure:   h.options.SecureCookies,
		SameSite: h.sameSite(),
		Expires:  clearedCookieInstant,
	}
}

// setAuthCookies writes the access and refresh auth cookies.
func (h *Handler) setAuthCookies(
	w http.ResponseWriter,
	tokens SessionTokens,
) {
	http.SetCookie(w, h.authCookie(
		accessTokenCookieName, tokens.AccessToken, "/", h.options.AccessTTL,
	))
	http.SetCookie(w, h.authCookie(
		refreshTokenCookieName,
		tokens.RefreshToken,
		"/auth",
		h.options.RefreshTTL,
	))
}

// decodeBody reads the JSON object body, validates it against the
// schema, and returns the transformed string values. It writes the
// pinned 400 responses when the body is unusable.
func (h *Handler) decodeBody(
	w http.ResponseWriter,
	r *http.Request,
	schema map[string]fieldRules,
	order []string,
) (map[string]string, bool) {
	var fields map[string]json.RawMessage
	if err := web.DecodeJSON(w, r, &fields); err != nil {
		web.WriteError(w, r, err)
		return nil, false
	}

	if messages := validateFields(fields, schema, order, authTransforms); len(messages) > 0 {
		web.WriteJSON(w, http.StatusBadRequest, &web.Error{
			StatusCode: http.StatusBadRequest,
			Message:    messages,
			ErrorName:  http.StatusText(http.StatusBadRequest),
		})
		return nil, false
	}

	values := make(map[string]string)
	for _, name := range order {
		value := decodeField(fields[name], authTransforms[name])
		if value == nil {
			continue
		}
		values[name] = *value
	}

	return values, true
}

// writeServiceError maps identity errors to pinned HTTP bodies.
func (h *Handler) writeServiceError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		web.WriteError(w, r, unauthorized)
	case errors.Is(err, ErrUsernameTaken), errors.Is(err, ErrEmailTaken):
		web.WriteError(w, r, web.NewError(http.StatusConflict, err.Error()))
	default:
		web.WriteError(w, r, err)
	}
}

// cookieValue returns a cookie's value, tolerating nil.
func cookieValue(cookie *http.Cookie) string {
	if cookie == nil {
		return ""
	}

	return cookie.Value
}
