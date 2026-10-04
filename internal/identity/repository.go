package identity

import (
	"context"
	"time"
)

// Repository stores users and authentication sessions.
type Repository interface {
	// CreateUserAndSession creates an account and its initial session.
	CreateUserAndSession(
		ctx context.Context,
		user Credentials,
		session Session,
	) (userID, sessionID string, err error)
	// GetUserByID gets a user by identifier.
	GetUserByID(ctx context.Context, id string) (*User, error)
	// GetAuthRecordByUsername gets login data by username.
	GetAuthRecordByUsername(
		ctx context.Context,
		username string,
	) (*AuthRecord, error)
	// GetAuthRecordByEmail gets login data by email.
	GetAuthRecordByEmail(ctx context.Context, email string) (*AuthRecord, error)
	// CreateSession creates a refresh session.
	CreateSession(ctx context.Context, session Session) (string, error)
	// GetSessionByID gets a refresh session by identifier.
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	// RotateSession replaces a valid session's refresh credentials.
	RotateSession(
		ctx context.Context,
		id, currentHash, nextHash string,
		expiresAt time.Time,
	) (bool, error)
	// RevokeSession revokes an active session.
	RevokeSession(ctx context.Context, id string) (bool, error)
}

// UserResolver resolves authenticated users.
type UserResolver interface {
	// GetUserByID gets a user by identifier.
	GetUserByID(ctx context.Context, id string) (*User, error)
}
