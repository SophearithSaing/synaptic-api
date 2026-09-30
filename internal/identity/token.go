package identity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims carries the identity claims of an access token on top of the
// registered claim set.
type Claims struct {
	// Email is the user's normalized login email address.
	Email string `json:"email"`
	// Username is the user's public display name.
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// TokenIssuer signs and verifies HS256 access tokens with pinned
// issuer, audience, and methods.
type TokenIssuer struct {
	secret    []byte
	issuer    string
	audience  string
	accessTTL time.Duration
}

// NewTokenIssuer builds a TokenIssuer from configuration secrets.
func NewTokenIssuer(
	secret, issuer, audience string,
	accessTTL time.Duration,
) *TokenIssuer {
	return &TokenIssuer{
		secret:    []byte(secret),
		issuer:    issuer,
		audience:  audience,
		accessTTL: accessTTL,
	}
}

// Issue signs an access token carrying the user identity claims.
func (t *TokenIssuer) Issue(
	userID, email, username string,
	now time.Time,
) (string, error) {
	claims := Claims{
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    t.issuer,
			Audience:  jwt.ClaimStrings{t.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(t.secret)
}

// VerifiedToken holds the verified access token claims.
type VerifiedToken struct {
	Sub      string
	Email    string
	Username string
}

var errTokenMissingClaims = errors.New("token is missing required claims")

// Verify checks the token signature, issuer, audience, and expiry, and
// extracts the identity claims.
func (t *TokenIssuer) Verify(token string) (*VerifiedToken, error) {
	claims := &Claims{}
	if _, err := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithAudience(t.audience),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(time.Now),
	).ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}); err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	if claims.Subject == "" || claims.Email == "" || claims.Username == "" {
		return nil, errTokenMissingClaims
	}

	return &VerifiedToken{
		Sub:      claims.Subject,
		Email:    claims.Email,
		Username: claims.Username,
	}, nil
}

// RandomSecret returns a 32-byte base64url-encoded secret.
func RandomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random secret: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// SerializeRefreshToken joins a session id and refresh secret.
func SerializeRefreshToken(sessionID, secret string) string {
	return sessionID + "." + secret
}

// ParseRefreshToken splits a refresh token into its session id and
// secret, rejecting malformed values.
func ParseRefreshToken(refreshToken string) (sessionID, secret string, err error) {
	sessionID, secret, found := strings.Cut(refreshToken, ".")
	if !found || sessionID == "" || secret == "" {
		return "", "", ErrUnauthorized
	}

	return sessionID, secret, nil
}
