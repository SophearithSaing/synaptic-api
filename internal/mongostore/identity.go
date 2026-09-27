package mongostore

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
)

// IdentityStore implements identity.Repository against the legacy users
// and authSessions collections, preserving their BSON shapes.
type IdentityStore struct {
	users    *mongo.Collection
	sessions *mongo.Collection
}

// NewIdentityStore builds an IdentityStore over a database.
func NewIdentityStore(database *mongo.Database) *IdentityStore {
	return &IdentityStore{
		users:    database.Collection("users"),
		sessions: database.Collection("authSessions"),
	}
}

// CreateUser inserts a user document and maps duplicate key errors to
// the identity sentinels by index name.
func (s *IdentityStore) CreateUser(
	ctx context.Context,
	user identity.Credentials,
) (string, error) {
	now := time.Now()

	insert := UserDocument{
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.PasswordHash,
		Role:      string(identity.RoleUser),
		CreatedAt: now,
		UpdatedAt: now,
		Version:   0,
	}
	result, err := s.users.InsertOne(ctx, insert)
	if err != nil {
		return "", mapDuplicateKey(err)
	}

	return objectIDHex(result.InsertedID), nil
}

// FindUserByID resolves a user by hex ObjectId.
func (s *IdentityStore) FindUserByID(
	ctx context.Context,
	id string,
) (*identity.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil
	}

	document := s.users.FindOne(ctx, bson.M{"_id": objectID})

	return decodeUser(document)
}

// FindUserByUsername resolves a user with the legacy case-insensitive
// English collation.
func (s *IdentityStore) FindUserByUsername(
	ctx context.Context,
	username string,
) (*identity.User, error) {
	document := s.users.FindOne(ctx, bson.M{"username": username},
		options.FindOne().SetCollation(&options.Collation{
			Locale:   "en",
			Strength: 2,
		}))

	return decodeUser(document)
}

// FindUserByEmail resolves a user by exact normalized email.
func (s *IdentityStore) FindUserByEmail(
	ctx context.Context,
	email string,
) (*identity.User, error) {
	document := s.users.FindOne(ctx, bson.M{"email": email})

	return decodeUser(document)
}

// PasswordHash resolves the stored bcrypt password hash.
func (s *IdentityStore) PasswordHash(
	ctx context.Context,
	id string,
) (string, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return "", nil
	}

	var document UserDocument
	err = s.users.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", nil
		}
		return "", err
	}

	return document.Password, nil
}

// CreateSession inserts an auth session and returns its hex identity.
func (s *IdentityStore) CreateSession(
	ctx context.Context,
	session identity.Session,
) (string, error) {
	userID, err := bson.ObjectIDFromHex(session.UserID)
	if err != nil {
		return "", err
	}

	now := time.Now()
	insert := AuthSessionDocument{
		UserID:           userID,
		RefreshTokenHash: session.RefreshHash,
		ExpiresAt:        session.ExpiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
		Version:          0,
	}
	result, err := s.sessions.InsertOne(ctx, insert)
	if err != nil {
		return "", err
	}

	return objectIDHex(result.InsertedID), nil
}

// LoadSession resolves an auth session by hex ObjectId.
func (s *IdentityStore) LoadSession(
	ctx context.Context,
	id string,
) (*identity.Session, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil
	}

	var document AuthSessionDocument
	err = s.sessions.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	service := &identity.Session{
		ID:          document.ID.Hex(),
		UserID:      document.UserID.Hex(),
		RefreshHash: document.RefreshTokenHash,
		ExpiresAt:   document.ExpiresAt,
		RevokedAt:   document.RevokedAt,
	}

	return service, nil
}

// RotateSession compares-and-set the refresh hash and expiry on an
// unrevoked, unexpired session that still holds currentHash.
func (s *IdentityStore) RotateSession(
	ctx context.Context,
	id, currentHash, nextHash string,
	expiresAt time.Time,
) (bool, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, nil
	}

	now := time.Now()
	result, err := s.sessions.UpdateOne(ctx, bson.M{
		"_id":              objectID,
		"revokedAt":        bson.M{"$exists": false},
		"expiresAt":        bson.M{"$gt": now},
		"refreshTokenHash": currentHash,
	}, bson.M{"$set": bson.M{
		"refreshTokenHash": nextHash,
		"expiresAt":        expiresAt,
		"updatedAt":        now,
	}})
	if err != nil {
		return false, err
	}

	return result.MatchedCount == 1, nil
}

// RevokeSession sets revokedAt when the session is not already
// revoked.
func (s *IdentityStore) RevokeSession(
	ctx context.Context,
	id string,
) (bool, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, nil
	}

	now := time.Now()
	result, err := s.sessions.UpdateOne(ctx, bson.M{
		"_id":       objectID,
		"revokedAt": bson.M{"$exists": false},
	}, bson.M{"$set": bson.M{"revokedAt": now, "updatedAt": now}})
	if err != nil {
		return false, err
	}

	return result.MatchedCount == 1, nil
}

// decodeUser maps a FindOne result to the domain user, tolerating
// missing role defaults.
func decodeUser(document *mongo.SingleResult) (*identity.User, error) {
	var raw UserDocument
	if err := document.Decode(&raw); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return mapUser(&raw), nil
}

// mapUser converts the BSON user document into the domain user. Legacy
// documents may omit role; the default is the plain user role.
func mapUser(document *UserDocument) *identity.User {
	role := identity.Role(document.Role)
	if role != identity.RoleAdmin {
		role = identity.RoleUser
	}

	return &identity.User{
		ID:       document.ID.Hex(),
		Username: document.Username,
		Email:    document.Email,
		Role:     role,
	}
}

// mapDuplicateKey translates a Mongo duplicate-key error into the
// identity sentinels by colliding index name.
func mapDuplicateKey(err error) error {
	if !mongo.IsDuplicateKeyError(err) {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "username_1"):
		return identity.ErrUsernameTaken
	case strings.Contains(message, "email_1"):
		return identity.ErrEmailTaken
	}

	return err
}

// objectIDHex converts an inserted _id value into a hex string.
func objectIDHex(value any) string {
	if objectID, ok := value.(bson.ObjectID); ok {
		return objectID.Hex()
	}

	return ""
}
