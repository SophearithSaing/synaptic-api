package testutil

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
)

// EnsureIdentityIndexes delegates to the production index definitions
// owned by mongostore so tests and the startup bootstrap stay one
// source of truth.
func EnsureIdentityIndexes(ctx context.Context, db *mongo.Database) error {
	return mongostore.EnsureIdentityIndexes(ctx, db)
}
