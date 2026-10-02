package identity_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// repoState is the in-memory identity backend for handler tests.
// Username matching is case-insensitive like the legacy collation.
type repoState struct {
	mu        sync.Mutex
	users     map[string]*identity.User
	passwords map[string]string
	sessions  map[string]*identity.Session
	byName    map[string]string
	byEmail   map[string]string
	next      int
}

// newRepoState builds an empty repoState.
func newRepoState() *repoState {
	return &repoState{
		users:     map[string]*identity.User{},
		passwords: map[string]string{},
		sessions:  map[string]*identity.Session{},
		byName:    map[string]string{},
		byEmail:   map[string]string{},
	}
}

// id returns the next deterministic 24-hex id.
func (r *repoState) id() string {
	r.next++

	return fmt.Sprintf("665f1e2b9d1a2c3b4d5e%04x", r.next)
}

// CreateUserAndSession implements identity.Repository by inserting the
// user and initial session in one simulated write.
func (r *repoState) CreateUserAndSession(
	_ context.Context,
	user identity.Credentials,
	session identity.Session,
) (string, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byName[strings.ToLower(user.Username)]; ok {
		return "", "", identity.ErrUsernameTaken
	}
	if _, ok := r.byEmail[user.Email]; ok {
		return "", "", identity.ErrEmailTaken
	}

	id := r.id()
	r.users[id] = &identity.User{
		ID:       id,
		Username: user.Username,
		Email:    user.Email,
		Role:     identity.RoleUser,
	}
	r.byName[strings.ToLower(user.Username)] = id
	r.byEmail[user.Email] = id
	r.passwords[id] = user.PasswordHash

	sessionID := r.id()
	r.sessions[sessionID] = &identity.Session{
		ID:          sessionID,
		UserID:      id,
		RefreshHash: session.RefreshHash,
		ExpiresAt:   session.ExpiresAt,
	}

	return id, sessionID, nil
}

// FindUserByID implements identity.Repository.
func (r *repoState) FindUserByID(
	_ context.Context,
	id string,
) (*identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return copyUser(r.users[id]), nil
}

// FindAuthRecordByUsername implements identity.Repository with the
// legacy case-insensitive username match, carrying the stored password
// hash.
func (r *repoState) FindAuthRecordByUsername(
	_ context.Context,
	username string,
) (*identity.AuthRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, _ := r.authRecordLocked(
		r.users[r.byName[strings.ToLower(username)]],
	)

	return record, nil
}

// FindAuthRecordByEmail implements identity.Repository, carrying the
// stored password hash.
func (r *repoState) FindAuthRecordByEmail(
	_ context.Context,
	email string,
) (*identity.AuthRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, _ := r.authRecordLocked(r.users[r.byEmail[email]])

	return record, nil
}

// authRecordLocked copies the user together with its stored password
// hash. Callers must hold mu. The error result keeps the repository
// call shape consistent.
func (r *repoState) authRecordLocked(
	user *identity.User,
) (*identity.AuthRecord, error) {
	if user == nil {
		return nil, nil
	}

	matched := *user

	return &identity.AuthRecord{
		User:         &matched,
		PasswordHash: r.passwords[user.ID],
	}, nil
}

// CreateSession implements identity.Repository.
func (r *repoState) CreateSession(
	_ context.Context,
	session identity.Session,
) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.id()
	r.sessions[id] = &identity.Session{
		ID:          id,
		UserID:      session.UserID,
		RefreshHash: session.RefreshHash,
		ExpiresAt:   session.ExpiresAt,
	}

	return id, nil
}

// LoadSession implements identity.Repository.
func (r *repoState) LoadSession(
	_ context.Context,
	id string,
) (*identity.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session := r.sessions[id]; session != nil {
		matched := *session

		return &matched, nil
	}

	return nil, nil
}

// RotateSession implements identity.Repository with a compare-and-set.
func (r *repoState) RotateSession(
	_ context.Context,
	id, currentHash, nextHash string,
	expiresAt time.Time,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[id]
	if session == nil || session.RevokedAt != nil ||
		!session.ExpiresAt.After(time.Now()) ||
		session.RefreshHash != currentHash {
		return false, nil
	}

	session.RefreshHash = nextHash
	session.ExpiresAt = expiresAt

	return true, nil
}

// RevokeSession implements identity.Repository with a compare-and-set.
func (r *repoState) RevokeSession(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[id]
	if session == nil || session.RevokedAt != nil {
		return false, nil
	}

	now := time.Now()
	session.RevokedAt = &now

	return true, nil
}

// SeedUser stores a user with a hashed password directly.
func (r *repoState) SeedUser(
	username, email, password, role string,
) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.id()
	r.users[id] = &identity.User{
		ID:       id,
		Username: username,
		Email:    email,
		Role:     identity.Role(role),
	}
	r.byName[strings.ToLower(username)] = id
	r.byEmail[email] = id
	r.passwords[id] = string(hash)

	return id
}

// copyUser drains a copied user for safe reads.
func copyUser(user *identity.User) *identity.User {
	if user == nil {
		return nil
	}

	matched := *user

	return &matched
}
