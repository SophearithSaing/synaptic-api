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

// Register creates an account and its initial refresh session. The
// repository transaction makes the pair all-or-nothing, so a failed
// session cannot leave an account without credentials.
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

// Login authenticates an identifier (username or email) and starts a
// refresh session. The authentication record — user and stored
// password hash — resolves in one repository query so the two reads
// cannot observe different account states. Repository failures
// propagate; only username/email resolution and credential checks
// produce ErrUnauthorized.
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

// authRecord resolves the authentication record of an identifier:
// exact email lookups for "@"-containing identifiers, otherwise the
// legacy case-insensitive username match.
func (s *Service) authRecord(
	ctx context.Context,
	identifier string,
) (*AuthRecord, error) {
	if strings.Contains(identifier, "@") {
		return s.repo.FindAuthRecordByEmail(ctx, identifier)
	}

	return s.repo.FindAuthRecordByUsername(ctx, identifier)
}

// Refresh validates a refresh token and atomically rotates its secret.
// Repository failures propagate; only missing or mismatched identity
// data reports ErrUnauthorized.
func (s *Service) Refresh(
	ctx context.Context,
	refreshToken string,
) (SessionTokens, error) {
	sessionID, secret, err := ParseRefreshToken(refreshToken)
	if err != nil {
		return SessionTokens{}, ErrUnauthorized
	}

	session, err := s.repo.LoadSession(ctx, sessionID)
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

	user, err := s.repo.FindUserByID(ctx, session.UserID)
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

// Logout revokes the session referenced by the refresh token. Missing,
// malformed, unknown tokens, and secret mismatches are not failures:
// revocation errors are.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	sessionID, secret, err := ParseRefreshToken(refreshToken)
	if err != nil {
		return nil
	}

	session, err := s.repo.LoadSession(ctx, sessionID)
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

// startSession issues the access token and creates the refresh session
// for an authenticated user.
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
