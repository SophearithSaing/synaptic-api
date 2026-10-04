// Package identity implements authentication and session management.
package identity

import (
	"time"
)

// Role is a user privilege level.
type Role string

// Supported roles.
const (
	// RoleUser is the default role assigned at registration.
	RoleUser Role = "user"
	// RoleAdmin grants administrator access.
	RoleAdmin Role = "admin"
)

// User is the authenticated identity of an account.
type User struct {
	ID       string
	Username string
	Email    string
	Role     Role
}

// Credentials is the hashed membership data persisted for a new user.
type Credentials struct {
	Username     string
	Email        string
	PasswordHash string
}

// Session is a stored refresh session.
type Session struct {
	ID          string
	UserID      string
	RefreshHash string
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}

// SessionTokens is the result of a successful authentication.
type SessionTokens struct {
	AccessToken  string
	RefreshToken string
}

// AuthRecord contains the account data needed for login.
type AuthRecord struct {
	User         *User
	PasswordHash string
}

// Options configures token lifetimes and cookie policy.
type Options struct {
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	SecureCookies bool
}
