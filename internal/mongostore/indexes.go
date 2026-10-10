package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIdentityIndexes creates the legacy identity indexes: the
// case-insensitive unique username, the unique email, and the session
// userId and TTL indexes. The application runs it as an explicit
// startup bootstrap so a fresh production or local database matches
// the integration-test contract before serving traffic.
func EnsureIdentityIndexes(ctx context.Context, database *mongo.Database) error {
	users := []mongo.IndexModel{
		{
			Keys: bson.M{"username": 1},
			Options: options.Index().SetUnique(true).
				SetCollation(&options.Collation{
					Locale:   "en",
					Strength: 2,
				}),
		},
		{Keys: bson.M{"email": 1}, Options: options.Index().SetUnique(true)},
	}
	for _, model := range users {
		if _, err := database.Collection("users").Indexes().CreateOne(
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
		if _, err := database.Collection("authSessions").Indexes().CreateOne(
			ctx, model,
		); err != nil {
			return err
		}
	}

	return nil
}

// EnsureThrottleIndexes creates the throttle TTL index, reclaiming
// rate-limit windows once their purge marker — the later of the
// counting horizon and an active block — passes. The application runs
// it as an explicit startup bootstrap alongside the identity indexes.
func EnsureThrottleIndexes(ctx context.Context, database *mongo.Database) error {
	_, err := database.Collection("throttles").Indexes().CreateOne(ctx,
		mongo.IndexModel{
			Keys:    bson.M{"purgeAt": 1},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	)

	return err
}

// EnsureCatalogIndexes creates the audited unique slug indexes. Question sets
// intentionally have no unique compound key because duplicate groups are part
// of the catalog contract.
func EnsureCatalogIndexes(ctx context.Context, database *mongo.Database) error {
	for _, collection := range []string{"categories", "topics"} {
		if _, err := database.Collection(collection).Indexes().CreateOne(ctx,
			mongo.IndexModel{
				Keys:    bson.M{"slug": 1},
				Options: options.Index().SetUnique(true),
			},
		); err != nil {
			return err
		}
	}

	return nil
}
