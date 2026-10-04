package identity

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service implements authentication workflows.
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

// Register creates an account and its initial session.
func (s *Service) Register(
	ctx context.Context,
	credentials Credentials,
) (SessionTokens, error) {
	secret, err := RandomSecret()
	if err != nil {
		return SessionTokens{}, err
	}

	hash, err := HashSecret(secret)
	if err != nil {
		return SessionTokens{}, err
	}

	expiresAt := time.Now().Add(s.options.RefreshTTL)

	userID, sessionID, err := s.repo.CreateUserAndSession(
		ctx, credentials, Session{RefreshHash: hash, ExpiresAt: expiresAt},
	)
	if err != nil {
		return SessionTokens{}, err
	}

	access, err := s.issuer.Issue(
		userID, credentials.Email, credentials.Username, time.Now(),
	)
	if err != nil {
		return SessionTokens{}, err
	}

	return SessionTokens{
		AccessToken:  access,
		RefreshToken: SerializeRefreshToken(sessionID, secret),
	}, nil
}

// Login authenticates an identifier and starts a session.
func (s *Service) Login(
	ctx context.Context,
	identifier, password string,
) (SessionTokens, error) {
	record, err := s.authRecord(ctx, identifier)
	if err != nil {
		return SessionTokens{}, err
	}
	if record == nil || record.User == nil ||
		bcrypt.CompareHashAndPassword(
			[]byte(record.PasswordHash), []byte(password),
		) != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	return s.startSession(ctx, record.User)
}

// authRecord gets login data by email or username.
func (s *Service) authRecord(
	ctx context.Context,
	identifier string,
) (*AuthRecord, error) {
	if strings.Contains(identifier, "@") {
		return s.repo.GetAuthRecordByEmail(ctx, identifier)
	}

	return s.repo.GetAuthRecordByUsername(ctx, identifier)
}

// Refresh validates and rotates a refresh token.
func (s *Service) Refresh(
	ctx context.Context,
	refreshToken string,
) (SessionTokens, error) {
	sessionID, secret, err := ParseRefreshToken(refreshToken)
	if err != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return SessionTokens{}, err
	}
	if session == nil || session.RevokedAt != nil ||
		!session.ExpiresAt.After(time.Now()) ||
		bcrypt.CompareHashAndPassword(
			[]byte(session.RefreshHash), []byte(secret),
		) != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return SessionTokens{}, err
	}
	if user == nil {
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

// Logout revokes the session referenced by a refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	sessionID, secret, err := ParseRefreshToken(refreshToken)
	if err != nil {
		return nil
	}

	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
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

// startSession creates access and refresh tokens for a user.
func (s *Service) startSession(
	ctx context.Context,
	user *User,
) (SessionTokens, error) {
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

// createSessionToken creates and serializes a refresh session.
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
