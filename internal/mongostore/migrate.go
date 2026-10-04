package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RunMigrations applies the database migrations required by the current API.
func RunMigrations(ctx context.Context, database *mongo.Database) error {
	session, err := database.Client().StartSession()
	if err != nil {
		return fmt.Errorf("start migration session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		if err := migrateReference(
			ctx,
			database.Collection("topics"),
			"category",
			"categoryId",
		); err != nil {
			return nil, err
		}
		if err := migrateReference(
			ctx,
			database.Collection("questionSets"),
			"topic",
			"topicId",
		); err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("migrate catalog references: %w", err)
	}

	return nil
}

// migrateReference renames and normalizes one collection reference field.
func migrateReference(
	ctx context.Context,
	collection *mongo.Collection,
	oldField string,
	newField string,
) error {
	conflicts, err := collection.CountDocuments(ctx, bson.D{
		{Key: oldField, Value: bson.D{{Key: "$exists", Value: true}}},
		{Key: newField, Value: bson.D{{Key: "$exists", Value: true}}},
	})
	if err != nil {
		return fmt.Errorf("count %s conflicts: %w", collection.Name(), err)
	}
	if conflicts != 0 {
		return fmt.Errorf(
			"%s has %d documents with both %s and %s",
			collection.Name(), conflicts, oldField, newField,
		)
	}

	if _, err := collection.UpdateMany(
		ctx,
		bson.D{{Key: oldField, Value: bson.D{{Key: "$exists", Value: true}}}},
		mongo.Pipeline{
			setObjectID(newField, oldField),
			bson.D{{Key: "$unset", Value: oldField}},
		},
	); err != nil {
		return fmt.Errorf(
			"rename %s.%s to %s: %w",
			collection.Name(), oldField, newField, err,
		)
	}

	if _, err := collection.UpdateMany(
		ctx,
		bson.D{{Key: newField, Value: bson.D{{Key: "$type", Value: "string"}}}},
		mongo.Pipeline{setObjectID(newField, newField)},
	); err != nil {
		return fmt.Errorf(
			"normalize %s.%s: %w", collection.Name(), newField, err,
		)
	}

	total, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("count %s documents: %w", collection.Name(), err)
	}
	valid, err := collection.CountDocuments(ctx, bson.D{{
		Key: newField,
		Value: bson.D{{
			Key:   "$type",
			Value: "objectId",
		}},
	}})
	if err != nil {
		return fmt.Errorf("validate %s.%s: %w", collection.Name(), newField, err)
	}
	if valid != total {
		return fmt.Errorf(
			"%s has %d documents without an ObjectID %s",
			collection.Name(), total-valid, newField,
		)
	}

	return nil
}

// setObjectID builds an update stage that converts a field to an ObjectID.
func setObjectID(target string, source string) bson.D {
	return bson.D{{Key: "$set", Value: bson.D{{
		Key: target,
		Value: bson.D{{
			Key:   "$toObjectId",
			Value: "$" + source,
		}},
	}}}}
}
