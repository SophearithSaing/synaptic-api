package identity

import (
	"errors"
	"net/http"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

var unauthorized = &web.Error{
	StatusCode: http.StatusUnauthorized,
	Message:    "Unauthorized",
}

var (
	ErrUnauthorized  = errors.New("Unauthorized")
	ErrUsernameTaken = errors.New("Username already exists")
	ErrEmailTaken    = errors.New("Email already exists")
)
