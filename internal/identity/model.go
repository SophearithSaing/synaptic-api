// Package identity implements the authentication core: user
// registration, login, JWT issuing, refresh-session rotation, logout,
// CSRF protection, and the authenticated-user middleware reused by
// protected routes.
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
	// ID is the user's MongoDB ObjectId as a 24-character hex string.
	ID string
	// Username is the public display name.
	Username string
	// Email is the normalized login email address.
	Email string
	// Role is the account privilege level.
	Role Role
}

// Credentials is the hashed membership data persisted for a new user.
type Credentials struct {
	// Username is the trimmed display name.
	Username string
	// Email is the trimmed, lowercased login email.
	Email string
	// PasswordHash is the bcrypt-encoded password.
	PasswordHash string
}

// Session is a stored refresh session.
type Session struct {
	// ID is the auth-session MongoDB ObjectId as a hex string.
	ID string
	// UserID is the owning user's ObjectId as a hex string.
	UserID string
	// RefreshHash is the bcrypt hash of the current refresh secret.
	RefreshHash string
	// ExpiresAt is the session expiry instant.
	ExpiresAt time.Time
	// RevokedAt is set when the session is revoked.
	RevokedAt *time.Time
}

// SessionTokens is the result of a successful authentication.
type SessionTokens struct {
	// AccessToken is the signed JWT access token.
	AccessToken string
	// RefreshToken is the "<sessionId>.<secret>" refresh token.
	RefreshToken string
}

// AuthRecord is the login view of an account: the resolved user with
// its stored password hash, produced by one authentication query.
type AuthRecord struct {
	// User is the resolved account.
	User *User
	// PasswordHash is the stored bcrypt-encoded password.
	PasswordHash string
}

// Options configures token lifetimes and cookie policy.
type Options struct {
	// AccessTTL is the access token and cookie lifetime.
	AccessTTL time.Duration
	// RefreshTTL is the refresh session and cookie lifetime.
	RefreshTTL time.Duration
	// SecureCookies enables the production cookie policy (Secure with
	// SameSite=None); otherwise cookies use SameSite=Lax without
	// Secure.
	SecureCookies bool
}
