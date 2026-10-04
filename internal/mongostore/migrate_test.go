package mongostore_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
)

// TestRunMigrations verifies reference renames, ObjectID normalization,
// idempotency, and rollback on invalid identifiers.
func TestRunMigrations(t *testing.T) {
	ctx := context.Background()
	client, err := mongostore.Connect(ctx, startMongo(t, ctx))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Disconnect(context.Background()); err != nil {
			t.Logf("disconnect mongo: %v", err)
		}
	})

	t.Run("migrates and reruns", func(t *testing.T) {
		database := client.Database("migration_success")
		categoryID := objectIDOf(t, "5eed00000000000000000011")
		topicID := objectIDOf(t, "5eed00000000000000000021")

		if _, err := database.Collection("topics").InsertMany(ctx, []any{
			bson.D{
				{Key: "_id", Value: topicID},
				{Key: "category", Value: categoryID},
			},
			bson.D{
				{Key: "_id", Value: objectIDOf(
					t, "5eed00000000000000000022",
				)},
				{Key: "category", Value: categoryID.Hex()},
			},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Collection("questionSets").InsertMany(ctx, []any{
			bson.D{
				{Key: "_id", Value: objectIDOf(
					t, "5eed00000000000000000031",
				)},
				{Key: "topic", Value: topicID},
			},
			bson.D{
				{Key: "_id", Value: objectIDOf(
					t, "5eed00000000000000000032",
				)},
				{Key: "topic", Value: topicID.Hex()},
			},
		}); err != nil {
			t.Fatal(err)
		}

		if err := mongostore.RunMigrations(ctx, database); err != nil {
			t.Fatal(err)
		}
		if err := mongostore.RunMigrations(ctx, database); err != nil {
			t.Fatalf("rerun migration: %v", err)
		}

		assertMigratedReference(
			t, ctx, database.Collection("topics"),
			"category", "categoryId", categoryID,
		)
		assertMigratedReference(
			t, ctx, database.Collection("questionSets"),
			"topic", "topicId", topicID,
		)
	})

	t.Run("rolls back invalid identifiers", func(t *testing.T) {
		database := client.Database("migration_failure")
		categoryID := objectIDOf(t, "5eed00000000000000000011")
		if _, err := database.Collection("topics").InsertOne(ctx, bson.D{
			{Key: "category", Value: categoryID},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Collection("questionSets").InsertOne(ctx, bson.D{
			{Key: "topic", Value: "not-an-object-id"},
		}); err != nil {
			t.Fatal(err)
		}

		if err := mongostore.RunMigrations(ctx, database); err == nil {
			t.Fatal("migration succeeded with an invalid identifier")
		}

		raw, err := database.Collection("topics").FindOne(ctx, bson.D{}).Raw()
		if err != nil {
			t.Fatal(err)
		}
		if raw.Lookup("category").Type != bson.TypeObjectID {
			t.Fatal("category was not restored by transaction rollback")
		}
		if raw.Lookup("categoryId").Type != 0 {
			t.Fatal("categoryId survived transaction rollback")
		}
	})
}

func assertMigratedReference(
	t *testing.T,
	ctx context.Context,
	collection *mongo.Collection,
	oldField string,
	newField string,
	want bson.ObjectID,
) {
	t.Helper()

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		t.Fatal(err)
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		count++
		if cursor.Current.Lookup(oldField).Type != 0 {
			t.Errorf("document %d still has %s", count, oldField)
		}
		reference := cursor.Current.Lookup(newField)
		if reference.Type != bson.TypeObjectID {
			t.Errorf(
				"document %d %s type = %s, want objectId",
				count, newField, reference.Type,
			)
			continue
		}
		if reference.ObjectID() != want {
			t.Errorf(
				"document %d %s = %s, want %s",
				count, newField, reference.ObjectID(), want,
			)
		}
	}
	if err := cursor.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("migrated %d documents, want 2", count)
	}
}
