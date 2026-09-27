package testutil

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIdentityIndexes creates the legacy identity indexes: the
// case-insensitive unique username, the unique email, and the session
// userId/TTL indexes. Integration tests and future migrations share
// these definitions.
func EnsureIdentityIndexes(ctx context.Context, db *mongo.Database) error {
	users := []mongo.IndexModel{
		{
			Keys: bson.M{"username": 1},
			Options: options.Index().SetUnique(true).
				SetCollation(&options.Collation{
					Locale: "en", Strength: 2,
				}),
		},
		{Keys: bson.M{"email": 1}, Options: options.Index().SetUnique(true)},
	}
	for _, model := range users {
		if _, err := db.Collection("users").Indexes().CreateOne(
			ctx, model,
		); err != nil {
			return err
		}
	}

	sessions := []mongo.IndexModel{
		{Keys: bson.M{"userId": 1}, Options: options.Index()},
		{
			Keys:    bson.M{"expiresAt": 1},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	}
	for _, model := range sessions {
		if _, err := db.Collection("authSessions").Indexes().CreateOne(
			ctx, model,
		); err != nil {
			return err
		}
	}

	return nil
}
