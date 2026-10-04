package mongostore

import (
	"context"
	"errors"
	"fmt"
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

// CreateUserAndSession creates a user and session transactionally.
func (s *IdentityStore) CreateUserAndSession(
	ctx context.Context,
	user identity.Credentials,
	session identity.Session,
) (string, string, error) {
	client := s.users.Database().Client()

	databaseSession, err := client.StartSession()
	if err != nil {
		return "", "", fmt.Errorf("start session: %w", err)
	}
	defer databaseSession.EndSession(ctx)

	created, err := databaseSession.WithTransaction(
		ctx,
		func(ctx context.Context) (any, error) {
			objectID, err := s.insertUser(ctx, newInsertedUser(user))
			if err != nil {
				return nil, err
			}

			sessionDocument := newSessionDocument(objectID, session)
			if _, err := s.sessions.InsertOne(
				ctx, sessionDocument,
			); err != nil {
				return nil, err
			}

			return registeredIDs{
				UserID:    objectID,
				SessionID: sessionDocument.ID,
			}, nil
		},
	)
	if err != nil {
		return "", "", mapDuplicateKey(err)
	}

	ids, ok := created.(registeredIDs)
	if !ok {
		return "", "", fmt.Errorf(
			"registration returned %T, want registeredIDs", created,
		)
	}

	return ids.UserID.Hex(), ids.SessionID.Hex(), nil
}

// registeredIDs carries the identities the registration transaction
// created.
type registeredIDs struct {
	UserID    bson.ObjectID
	SessionID bson.ObjectID
}

// newInsertedUser builds the users document for a new account.
func newInsertedUser(user identity.Credentials) UserDocument {
	now := time.Now()

	return UserDocument{
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.PasswordHash,
		Role:      string(identity.RoleUser),
		CreatedAt: now,
		UpdatedAt: now,
		Version:   0,
	}
}

// insertUser inserts the user document with a pre-assigned identity,
// translating duplicate key conflicts into the identity sentinels.
func (s *IdentityStore) insertUser(
	ctx context.Context,
	document UserDocument,
) (bson.ObjectID, error) {
	document.ID = bson.NewObjectID()

	if _, err := s.users.InsertOne(ctx, document); err != nil {
		return bson.NilObjectID, mapDuplicateKey(err)
	}

	return document.ID, nil
}

// newSessionDocument builds an authSessions document with a fresh
// identity.
func newSessionDocument(
	userID bson.ObjectID,
	session identity.Session,
) AuthSessionDocument {
	now := time.Now()

	return AuthSessionDocument{
		ID:               bson.NewObjectID(),
		UserID:           userID,
		RefreshTokenHash: session.RefreshHash,
		ExpiresAt:        session.ExpiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
		Version:          0,
	}
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

	document := newSessionDocument(userID, session)
	if _, err := s.sessions.InsertOne(ctx, document); err != nil {
		return "", err
	}

	return document.ID.Hex(), nil
}

// GetUserByID gets a user by identifier.
func (s *IdentityStore) GetUserByID(
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

// GetAuthRecordByUsername gets login data by username.
func (s *IdentityStore) GetAuthRecordByUsername(
	ctx context.Context,
	username string,
) (*identity.AuthRecord, error) {
	document := s.users.FindOne(ctx, bson.M{"username": username},
		options.FindOne().SetCollation(&options.Collation{
			Locale:   "en",
			Strength: 2,
		}))

	return decodeAuthRecord(document)
}

// GetAuthRecordByEmail gets login data by email.
func (s *IdentityStore) GetAuthRecordByEmail(
	ctx context.Context,
	email string,
) (*identity.AuthRecord, error) {
	document := s.users.FindOne(ctx, bson.M{"email": email})

	return decodeAuthRecord(document)
}

// GetSessionByID gets a refresh session by identifier.
func (s *IdentityStore) GetSessionByID(
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

// RotateSession replaces valid refresh credentials atomically.
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

// RevokeSession revokes an active session.
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

// decodeUser maps a FindOne result to the domain user, mapping a
// missing document to a nil result and tolerating missing roles.
func decodeUser(document *mongo.SingleResult) (*identity.User, error) {
	user, err := decodeUserDocument(document)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// decodeAuthRecord maps a FindOne result to the login record,
// mapping a missing document to a nil record.
func decodeAuthRecord(document *mongo.SingleResult) (*identity.AuthRecord, error) {
	user, err := decodeUserDocument(document)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	var raw UserDocument
	if err := document.Decode(&raw); err != nil {
		return nil, err
	}

	return &identity.AuthRecord{
		User:         user,
		PasswordHash: raw.Password,
	}, nil
}

// decodeUserDocument decodes a users FindOne result. A missing
// document reports a nil user without an error.
func decodeUserDocument(document *mongo.SingleResult) (*identity.User, error) {
	if err := document.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	var raw UserDocument
	if err := document.Decode(&raw); err != nil {
		return nil, err
	}

	user := &identity.User{
		ID:       raw.ID.Hex(),
		Username: raw.Username,
		Email:    raw.Email,
		Role:     identity.Role(raw.Role),
	}
	if user.Role != identity.RoleAdmin {
		user.Role = identity.RoleUser
	}

	return user, nil
}

// mapDuplicateKey translates a Mongo duplicate-key error into the
// identity sentinels by inspecting the raw server key pattern through
// the write exception.
func mapDuplicateKey(err error) error {
	if !mongo.IsDuplicateKeyError(err) {
		return err
	}

	writeException := mongo.WriteException{}
	if !errors.As(err, &writeException) {
		return err
	}

	for _, writeError := range writeException.WriteErrors {
		switch collidedKey(writeError.Raw) {
		case "username":
			return identity.ErrUsernameTaken
		case "email":
			return identity.ErrEmailTaken
		}
	}

	return err
}

// collidedKey reads the colliding index key field from the raw server
// document of a duplicate-key error. Modern servers carry keyPattern
// on the error document; some mongos positions only write it inside
// errInfo.
func collidedKey(raw bson.Raw) string {
	for _, location := range []string{"keyPattern", "errInfo.keyPattern"} {
		key := rawKeyValue(raw, location)
		if key.Type != bson.TypeEmbeddedDocument {
			continue
		}

		keys, err := key.Document().Elements()
		if err != nil || len(keys) != 1 {
			continue
		}

		return keys[0].Key()
	}

	return ""
}

// rawKeyValue reads a dotted lookup path as a raw value.
func rawKeyValue(raw bson.Raw, path string) bson.RawValue {
	head, tail, found := strings.Cut(path, ".")
	value := raw.Lookup(head)
	if !found {
		return value
	}
	if value.Type != bson.TypeEmbeddedDocument {
		return bson.RawValue{}
	}

	return rawKeyValue(value.Document(), tail)
}
