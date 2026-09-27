package identity

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service implements registration, login, refresh rotation, logout, and
// user lookup over a Repository.
type Service struct {
	repo    Repository
	issuer  *TokenIssuer
	options Options
}

// NewService builds a Service.
func NewService(
	repo Repository,
	issuer *TokenIssuer,
	options Options,
) *Service {
	return &Service{repo: repo, issuer: issuer, options: options}
}

// Register creates an account and a refresh session.
func (s *Service) Register(
	ctx context.Context,
	credentials Credentials,
) (SessionTokens, error) {
	id, err := s.repo.CreateUser(ctx, credentials)
	if err != nil {
		return SessionTokens{}, err
	}

	user := User{
		ID:       id,
		Username: credentials.Username,
		Email:    credentials.Email,
		Role:     RoleUser,
	}

	access, err := s.issuer.Issue(
		user.ID, user.Email, user.Username, time.Now(),
	)
	if err != nil {
		return SessionTokens{}, err
	}

	token, err := s.createSessionToken(ctx, user.ID)
	if err != nil {
		return SessionTokens{}, err
	}

	return SessionTokens{AccessToken: access, RefreshToken: token}, nil
}

// Login authenticates an identifier (username or email) and starts a
// refresh session.
func (s *Service) Login(
	ctx context.Context,
	identifier, password string,
) (SessionTokens, error) {
	var (
		user *User
		err  error
	)
	if strings.Contains(identifier, "@") {
		user, err = s.repo.FindUserByEmail(ctx, identifier)
	} else {
		user, err = s.repo.FindUserByUsername(ctx, identifier)
	}
	if err != nil || user == nil {
		return SessionTokens{}, ErrUnauthorized
	}

	hash, err := s.repo.PasswordHash(ctx, user.ID)
	if err != nil || bcrypt.CompareHashAndPassword(
		[]byte(hash), []byte(password),
	) != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	access, err := s.issuer.Issue(
		user.ID, user.Email, user.Username, time.Now(),
	)
	if err != nil {
		return SessionTokens{}, err
	}

	token, err := s.createSessionToken(ctx, user.ID)
	if err != nil {
		return SessionTokens{}, err
	}

	return SessionTokens{AccessToken: access, RefreshToken: token}, nil
}

// Refresh validates a refresh token and atomically rotates its secret.
func (s *Service) Refresh(
	ctx context.Context,
	refreshToken string,
) (SessionTokens, error) {
	sessionID, secret, err := ParseRefreshToken(refreshToken)
	if err != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	session, err := s.repo.LoadSession(ctx, sessionID)
	if err != nil || session == nil {
		return SessionTokens{}, ErrUnauthorized
	}
	if session.RevokedAt != nil ||
		!session.ExpiresAt.After(time.Now()) ||
		bcrypt.CompareHashAndPassword(
			[]byte(session.RefreshHash), []byte(secret),
		) != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil || user == nil {
		return SessionTokens{}, ErrUnauthorized
	}

	nextSecret, err := RandomSecret()
	if err != nil {
		return SessionTokens{}, err
	}

	nextHash, err := HashSecret(nextSecret)
	if err != nil {
		return SessionTokens{}, err
	}

	rotated, err := s.repo.RotateSession(
		ctx,
		session.ID,
		session.RefreshHash,
		nextHash,
		time.Now().Add(s.options.RefreshTTL),
	)
	if err != nil {
		return SessionTokens{}, err
	}
	if !rotated {
		return SessionTokens{}, ErrUnauthorized
	}

	access, err := s.issuer.Issue(
		user.ID, user.Email, user.Username, time.Now(),
	)
	if err != nil {
		return SessionTokens{}, err
	}

	return SessionTokens{
		AccessToken:  access,
		RefreshToken: SerializeRefreshToken(session.ID, nextSecret),
	}, nil
}

// Logout revokes the session referenced by the refresh token. Missing,
// malformed, or unknown tokens and secret mismatches never fail: the
// caller clears the cookies regardless.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	sessionID, secret, err := ParseRefreshToken(refreshToken)
	if err != nil {
		return nil
	}

	session, err := s.repo.LoadSession(ctx, sessionID)
	if err != nil || session == nil {
		return nil
	}
	if bcrypt.CompareHashAndPassword(
		[]byte(session.RefreshHash), []byte(secret),
	) != nil {
		return nil
	}

	_, err = s.repo.RevokeSession(ctx, session.ID)

	return err
}

// UserByID re-resolves a user by id for role checks.
func (s *Service) UserByID(
	ctx context.Context,
	id string,
) (*User, error) {
	return s.repo.FindUserByID(ctx, id)
}

// createSessionToken persists a new refresh session and serializes its
// token.
func (s *Service) createSessionToken(
	ctx context.Context,
	userID string,
) (string, error) {
	secret, err := RandomSecret()
	if err != nil {
		return "", err
	}

	hash, err := HashSecret(secret)
	if err != nil {
		return "", err
	}

	id, err := s.repo.CreateSession(ctx, Session{
		UserID:      userID,
		RefreshHash: hash,
		ExpiresAt:   time.Now().Add(s.options.RefreshTTL),
	})
	if err != nil {
		return "", err
	}

	return SerializeRefreshToken(id, secret), nil
}

// HashSecret bcrypt-hashes a refresh secret for persistence.
func HashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(secret), bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// HashPassword bcrypt-hashes a plaintext password for persistence.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password), bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
