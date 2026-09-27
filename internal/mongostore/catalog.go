package mongostore

import (
	"context"
	"encoding/json"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// CatalogStore implements catalog.Repository with the exact stored
// shapes, reading loose BSON without repairing legacy documents.
type CatalogStore struct {
	categories   *mongo.Collection
	topics       *mongo.Collection
	questionSets *mongo.Collection
}

// NewCatalogStore builds a CatalogStore over a database.
func NewCatalogStore(database *mongo.Database) *CatalogStore {
	return &CatalogStore{
		categories:   database.Collection("categories"),
		topics:       database.Collection("topics"),
		questionSets: database.Collection("questionSets"),
	}
}

// Categories lists every category sorted by title.
func (s *CatalogStore) Categories(
	ctx context.Context,
) ([]catalog.Category, error) {
	cursor, err := s.categories.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "title", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []catalog.Category
	for cursor.Next(ctx) {
		category, err := decodeCategory(cursor.Current)
		if err != nil {
			return nil, err
		}
		results = append(results, *category)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// CategoryByID resolves one category by hex ObjectId.
func (s *CatalogStore) CategoryByID(
	ctx context.Context,
	id string,
) (*catalog.Category, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, catalog.ErrCategoryNotFound
	}

	raw, err := s.findRaw(ctx, s.categories, objectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, catalog.ErrCategoryNotFound
	}

	return decodeCategory(raw)
}

// Topics lists every topic sorted by title with the nested category.
func (s *CatalogStore) Topics(ctx context.Context) ([]catalog.Topic, error) {
	cursor, err := s.topics.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "title", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []catalog.Topic
	for cursor.Next(ctx) {
		topic, err := s.decodeTopic(ctx, cursor.Current)
		if err != nil {
			return nil, err
		}
		results = append(results, *topic)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// TopicByID resolves one topic by hex ObjectId with the nested
// category.
func (s *CatalogStore) TopicByID(
	ctx context.Context,
	id string,
) (*catalog.Topic, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, catalog.ErrTopicNotFound
	}

	raw, err := s.findRaw(ctx, s.topics, objectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, catalog.ErrTopicNotFound
	}

	return s.decodeTopic(ctx, raw)
}

// QuestionSetByID resolves one question set by hex ObjectId.
func (s *CatalogStore) QuestionSetByID(
	ctx context.Context,
	id string,
	populate bool,
) (*catalog.QuestionSet, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, catalog.ErrQuestionSetNotFound
	}

	raw, err := s.findRaw(ctx, s.questionSets, objectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, catalog.ErrQuestionSetNotFound
	}

	return s.decodeQuestionSet(ctx, raw, populate)
}

// QuestionSetsByTopicSlug lists the question sets for a topic slug by
// stored (natural) order.
func (s *CatalogStore) QuestionSetsByTopicSlug(
	ctx context.Context,
	slug string,
	populate bool,
) ([]catalog.QuestionSet, error) {
	topicRaw, err := s.topics.FindOne(ctx, bson.M{"slug": slug}).Raw()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, catalog.ErrTopicNotFound
		}
		return nil, err
	}

	cursor, err := s.questionSets.Find(ctx, bson.M{
		"topic": topicRaw.Lookup("_id").ObjectID(),
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []catalog.QuestionSet
	for cursor.Next(ctx) {
		questionSet, err := s.decodeQuestionSet(ctx, cursor.Current, populate)
		if err != nil {
			return nil, err
		}
		results = append(results, *questionSet)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// findRaw loads one document untouched.
func (s *CatalogStore) findRaw(
	ctx context.Context,
	collection *mongo.Collection,
	objectID bson.ObjectID,
) (bson.Raw, error) {
	raw, err := collection.FindOne(ctx, bson.M{"_id": objectID}).Raw()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return raw, nil
}

// decodeCategory maps the stored category document.
func decodeCategory(raw bson.Raw) (*catalog.Category, error) {
	var document CategoryDocument
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, err
	}

	return &catalog.Category{
		ID:          document.ID.Hex(),
		Title:       document.Title,
		Slug:        document.Slug,
		Description: document.Description,
		Icon:        document.Icon,
	}, nil
}

// decodeTopic maps the stored topic document, resolving a stored
// ObjectId category against the categories collection. A missing or
// legacy string-typed category keeps the nested response nil.
func (s *CatalogStore) decodeTopic(
	ctx context.Context,
	raw bson.Raw,
) (*catalog.Topic, error) {
	var document TopicDocument
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, err
	}

	var nested *catalog.Category
	category := raw.Lookup("category")
	if category.Type == bson.TypeObjectID {
		resolved, err := s.categoryByObjectID(ctx, category.ObjectID())
		if err != nil {
			return nil, err
		}
		nested = resolved
	}

	return &catalog.Topic{
		ID:          document.ID.Hex(),
		Title:       document.Title,
		Slug:        document.Slug,
		Description: document.Description,
		Icon:        document.Icon,
		Tags:        document.Tags,
		Category:    nested,
	}, nil
}

// categoryByObjectID resolves one category document; a dangling
// reference leaves the nested category nil.
func (s *CatalogStore) categoryByObjectID(
	ctx context.Context,
	objectID bson.ObjectID,
) (*catalog.Category, error) {
	raw, err := s.findRaw(ctx, s.categories, objectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	return decodeCategory(raw)
}

// decodeQuestionSet maps the stored question set, optionally embedding
// the raw stored topic document.
func (s *CatalogStore) decodeQuestionSet(
	ctx context.Context,
	raw bson.Raw,
	populate bool,
) (*catalog.QuestionSet, error) {
	var document QuestionSetDocument
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, err
	}

	topic, err := s.questionSetTopic(ctx, raw, populate)
	if err != nil {
		return nil, err
	}

	questions, err := questionPassthrough(raw)
	if err != nil {
		return nil, err
	}

	return &catalog.QuestionSet{
		ID:        document.ID.Hex(),
		Topic:     topic,
		SetType:   document.SetType,
		Level:     document.Level,
		Questions: questions,
		CreatedAt: catalog.ISO8601(document.CreatedAt),
		UpdatedAt: catalog.ISO8601(document.UpdatedAt),
	}, nil
}

// questionSetTopic resolves the response topic value. Populated shapes
// carry the stored topic document as raw JSON; otherwise the stored
// reference passes through with ObjectIds rendered as hex strings.
func (s *CatalogStore) questionSetTopic(
	ctx context.Context,
	raw bson.Raw,
	populate bool,
) (any, error) {
	reference := raw.Lookup("topic")

	if populate {
		populated, err := s.topicReference(ctx, reference)
		if err == nil {
			return populated, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
	}

	return topicHexOrState(reference), nil
}

// topicReference loads the stored topic document as raw JSON for the
// populated question-set shape. A dangling reference renders the
// stored reference value unchanged.
func (s *CatalogStore) topicReference(
	ctx context.Context,
	reference bson.RawValue,
) (json.RawMessage, error) {
	if reference.Type != bson.TypeObjectID {
		return ToRawJSON(reference)
	}

	raw, err := s.findRaw(ctx, s.topics, reference.ObjectID())
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return ToRawJSON(reference)
	}

	return ToRawJSON(bson.RawValue{
		Type:  bson.TypeEmbeddedDocument,
		Value: raw,
	})
}

// topicHexOrState maps the unpopulated topic reference: ObjectIds to
// hex strings, stored hex strings pass through, other shapes render
// as JSON.
func topicHexOrState(reference bson.RawValue) any {
	switch reference.Type {
	case bson.TypeObjectID:
		return reference.ObjectID().Hex()
	case bson.TypeString:
		return reference.StringValue()
	case bson.TypeNull, bson.TypeUndefined:
		return nil
	}

	encoded, err := ToRawJSON(reference)
	if err != nil {
		return nil
	}

	var value any
	if json.Unmarshal(encoded, &value) != nil {
		return nil
	}

	return value
}

// questionPassthrough encodes the stored questions array untouched, in
// stored field order.
func questionPassthrough(raw bson.Raw) (json.RawMessage, error) {
	questions := raw.Lookup("questions")
	if questions.Type == bson.TypeNull {
		return json.RawMessage("null"), nil
	}

	return ToRawJSON(questions)
}
