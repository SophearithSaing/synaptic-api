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
	claims := jwt.MapClaims{
		"sub":      userID,
		"email":    email,
		"username": username,
		"iss":      t.issuer,
		"aud":      t.audience,
		"iat":      jwt.NewNumericDate(now),
		"exp":      jwt.NewNumericDate(now.Add(t.accessTTL)),
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
	parsed, err := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithAudience(t.audience),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(time.Now),
	).Parse(token, func(*jwt.Token) (any, error) {
		return t.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errTokenMissingClaims
	}

	sub, err := claims.GetSubject()
	if err != nil || sub == "" {
		return nil, errTokenMissingClaims
	}

	email, username := claimString(claims, "email"), claimString(claims, "username")
	if email == "" || username == "" {
		return nil, errTokenMissingClaims
	}

	return &VerifiedToken{Sub: sub, Email: email, Username: username}, nil
}

// claimString reads a string claim.
func claimString(claims jwt.MapClaims, name string) string {
	value, _ := claims[name].(string)

	return value
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
