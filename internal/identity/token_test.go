package identity_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// verifiedIssue cycles an issued token back through Verify.
func verifiedIssue(t *testing.T, issuer *identity.TokenIssuer) *identity.VerifiedToken {
	t.Helper()

	token, err := issuer.Issue(
		"665f1e2b9d1a2c3b4d5e6f70", "a@example.com", "alice", time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	verified, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	return verified
}

// TestTokenIssueAndVerifyPinsClaims checks the claim round-trip.
func TestTokenIssueAndVerifyPinsClaims(t *testing.T) {
	issuer := identity.NewTokenIssuer("secret", "synaptic", "client", time.Hour)

	verified := verifiedIssue(t, issuer)
	if verified.Sub != "665f1e2b9d1a2c3b4d5e6f70" ||
		verified.Email != "a@example.com" || verified.Username != "alice" {
		t.Fatalf("claims %+v", verified)
	}
}

// TestTokenRejectsWrongSecret enforces the HS256 secret.
func TestTokenRejectsWrongSecret(t *testing.T) {
	issuer := identity.NewTokenIssuer("secret", "synaptic", "client", time.Hour)
	other := identity.NewTokenIssuer("different", "synaptic", "client", time.Hour)

	token, err := issuer.Issue(
		"665f1e2b9d1a2c3b4d5e6f70", "a@example.com", "alice", time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Verify(token); err == nil {
		t.Fatal("wrong secret must fail")
	}
}

// TestTokenRejectsWrongIssuerOrAudience checks the pinned checks.
func TestTokenRejectsWrongIssuerOrAudience(t *testing.T) {
	issuer := identity.NewTokenIssuer("secret", "synaptic", "client", time.Hour)

	token, err := issuer.Issue(
		"665f1e2b9d1a2c3b4d5e6f70", "a@example.com", "alice", time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	wrongIssuer := identity.NewTokenIssuer("secret", "other", "client", time.Hour)
	if _, err := wrongIssuer.Verify(token); err == nil {
		t.Fatal("wrong issuer must fail")
	}

	wrongAudience := identity.NewTokenIssuer("secret", "synaptic", "other", time.Hour)
	if _, err := wrongAudience.Verify(token); err == nil {
		t.Fatal("wrong audience must fail")
	}
}

// TestTokenExpired pins expiry enforcement.
func TestTokenExpired(t *testing.T) {
	issuer := identity.NewTokenIssuer("secret", "synaptic", "client", time.Hour)

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      "665f1e2b9d1a2c3b4d5e6f70",
		"email":    "a@example.com",
		"username": "alice",
		"iss":      "synaptic",
		"aud":      "client",
		"iat":      time.Now().Add(-2 * time.Hour).Unix(),
		"exp":      time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := expired.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Verify(signed); err == nil {
		t.Fatal("expired token must fail")
	}
}

// TestTokenRejectsWrongAlgorithm pins the method restriction.
func TestTokenRejectsWrongAlgorithm(t *testing.T) {
	issuer := identity.NewTokenIssuer("secret", "synaptic", "client", time.Hour)

	foreign := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub":      "665f1e2b9d1a2c3b4d5e6f70",
		"email":    "a@example.com",
		"username": "alice",
		"iss":      "synaptic",
		"aud":      "client",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	signed, err := foreign.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Verify(signed); err == nil {
		t.Fatal("HS512 token must fail")
	}
}
