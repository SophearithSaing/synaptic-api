package identity

import (
	"context"
	"time"
)

// Repository stores users and auth sessions. Implementations map the
// exact users/authSessions BSON representations and translate duplicate
// key errors into the sentinel errors above.
type Repository interface {
	// CreateUser persists a new account and returns its hex identity.
	CreateUser(ctx context.Context, user Credentials) (string, error)
	// FindUserByID resolves a user by hex ObjectId.
	FindUserByID(ctx context.Context, id string) (*User, error)
	// FindUserByUsername resolves a user by case-insensitive username.
	FindUserByUsername(ctx context.Context, username string) (*User, error)
	// FindUserByEmail resolves a user by exact email address.
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	// PasswordHash resolves the stored bcrypt hash of a user's
	// password.
	PasswordHash(ctx context.Context, id string) (string, error)
	// CreateSession persists a refresh session and returns its hex
	// identity.
	CreateSession(ctx context.Context, session Session) (string, error)
	// LoadSession resolves an auth session by hex ObjectId.
	LoadSession(ctx context.Context, id string) (*Session, error)
	// RotateSession atomically replaces the refresh hash and expiry
	// when the session is unrevoked, unexpired, and still holds
	// currentHash. No match leaves the session unchanged and reports
	// false.
	RotateSession(
		ctx context.Context,
		id, currentHash, nextHash string,
		expiresAt time.Time,
	) (bool, error)
	// RevokeSession sets revokedAt when the session is not already
	// revoked.
	RevokeSession(ctx context.Context, id string) (bool, error)
}
