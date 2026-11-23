package mongostore

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// TestTopicMappersPreserveMetadata verifies populated-topic metadata survives
// both stored and in-memory authoring mappings.
func TestTopicMappersPreserveMetadata(t *testing.T) {
	timestamp := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	document := TopicDocument{
		ID: bson.NewObjectID(), CategoryID: bson.NewObjectID(), Title: "Topic",
		Slug: "topic", Description: "Description", Icon: "icon",
		Tags: []string{"tag"}, CreatedAt: timestamp, UpdatedAt: timestamp,
		Version: 4,
	}
	raw, err := bson.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	store := &CatalogStore{}
	decoded, err := store.decodeTopic(context.Background(), raw,
		func(context.Context, bson.ObjectID) (*catalog.Category, error) {
			return nil, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	assertTopicMetadata(t, decoded, timestamp, document.Version)
	assertTopicMetadata(t, topicFromDocument(document, nil), timestamp,
		document.Version)
}

// assertTopicMetadata verifies metadata used by populated topic responses.
func assertTopicMetadata(
	t *testing.T, topic *catalog.Topic, timestamp time.Time, version int,
) {
	t.Helper()
	if !topic.CreatedAt.Equal(timestamp) || !topic.UpdatedAt.Equal(timestamp) {
		t.Fatalf("timestamps %#v %#v", topic.CreatedAt, topic.UpdatedAt)
	}
	if topic.Version != version {
		t.Fatalf("version %d, want %d", topic.Version, version)
	}
}
