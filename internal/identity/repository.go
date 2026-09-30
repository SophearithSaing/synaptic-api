package identity

import (
	"context"
	"time"
)

// Repository stores users and auth sessions. Implementations map the
// exact users/authSessions BSON representations, translate duplicate
// key errors into the sentinel errors above, resolve each
// authentication lookup in one query, and keep user creation,
// initial session creation consistent.
type Repository interface {
	// CreateUserAndSession persists a new account and its initial
	// refresh session transactionally, returning the account and
	// session hex identities. Duplicate keys map to the sentinels.
	CreateUserAndSession(
		ctx context.Context,
		user Credentials,
		session Session,
	) (userID, sessionID string, err error)
	// FindUserByID resolves a user by hex ObjectId.
	FindUserByID(ctx context.Context, id string) (*User, error)
	// FindAuthRecordByUsername resolves the login record of an
	// account by case-insensitive username, or nil when no account
	// matches. The record carries the user and the stored password
	// hash from one query.
	FindAuthRecordByUsername(
		ctx context.Context,
		username string,
	) (*AuthRecord, error)
	// FindAuthRecordByEmail resolves the login record of an account
	// by exact email address, or nil when no account matches. The
	// record carries the user and the stored password hash from one
	// query.
	FindAuthRecordByEmail(ctx context.Context, email string) (*AuthRecord, error)
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

// UserResolver is the smallest identity lookup the authenticator
// needs: it re-resolves an access-token subject on every request.
type UserResolver interface {
	// FindUserByID resolves a user by hex ObjectId.
	FindUserByID(ctx context.Context, id string) (*User, error)
}
