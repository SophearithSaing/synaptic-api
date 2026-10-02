package identity

import (
	"errors"
	"net/http"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// unauthorized is the pinned 401 body without an error key.
var unauthorized = &web.Error{
	StatusCode: http.StatusUnauthorized,
	Message:    "Unauthorized",
}

// Errors surfaced to the HTTP layer with pinned response bodies.
var (
	// ErrUnauthorized describes missing or invalid credentials.
	ErrUnauthorized = errors.New("Unauthorized")
	// ErrUsernameTaken is returned when a duplicate username index fires.
	ErrUsernameTaken = errors.New("Username already exists")
	// ErrEmailTaken is returned when a duplicate email index fires.
	ErrEmailTaken = errors.New("Email already exists")
)
