package mongostore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

// parseObjectID parses a hex route id, reporting the distinct
// invalid-id sentinel so handlers map it to the pinned 400 body
// instead of duplicating the driver's own parsing.
func parseObjectID(id string) (bson.ObjectID, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.NilObjectID, catalog.ErrInvalidObjectID
	}

	return objectID, nil
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

	results := make([]catalog.Category, 0)
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
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, err
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
// All referenced categories load in one batched query instead of one
// lookup per topic.
func (s *CatalogStore) Topics(ctx context.Context) ([]catalog.Topic, error) {
	cursor, err := s.topics.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "title", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var raws []bson.Raw
	for cursor.Next(ctx) {
		raws = append(raws, copyRaw(cursor.Current))
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	categories, err := s.batchCategories(ctx, raws)
	if err != nil {
		return nil, err
	}

	results := make([]catalog.Topic, 0, len(raws))
	for _, raw := range raws {
		topic, err := s.decodeTopic(ctx, raw, categoryLookup(categories))
		if err != nil {
			return nil, err
		}
		results = append(results, *topic)
	}

	return results, nil
}

// TopicByID resolves one topic by hex ObjectId with the nested
// category.
func (s *CatalogStore) TopicByID(
	ctx context.Context,
	id string,
) (*catalog.Topic, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	raw, err := s.findRaw(ctx, s.topics, objectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, catalog.ErrTopicNotFound
	}

	return s.decodeTopic(ctx, raw, s.categoryByObjectID)
}

// QuestionSetByID resolves one question set by hex ObjectId.
func (s *CatalogStore) QuestionSetByID(
	ctx context.Context,
	id string,
) (*catalog.QuestionSet, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	raw, err := s.findRaw(ctx, s.questionSets, objectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, catalog.ErrQuestionSetNotFound
	}

	return s.decodeQuestionSet(ctx, raw, nil)
}

// QuestionSetsByTopicSlug lists the question sets for a topic slug in
// stored order and reuses the joined topic for every result.
func (s *CatalogStore) QuestionSetsByTopicSlug(
	ctx context.Context,
	slug string,
) ([]catalog.QuestionSet, error) {
	topicRaw, err := s.topics.FindOne(ctx, bson.M{"slug": slug}).Raw()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, catalog.ErrTopicNotFound
		}
		return nil, err
	}
	topic, err := s.decodeTopic(ctx, topicRaw, s.categoryByObjectID)
	if err != nil {
		return nil, err
	}

	cursor, err := s.questionSets.Find(ctx, bson.M{
		"topic": topicRaw.Lookup("_id").ObjectID(),
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	results := make([]catalog.QuestionSet, 0)
	for cursor.Next(ctx) {
		questionSet, err := s.decodeQuestionSet(
			ctx, cursor.Current, topic,
		)
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

// copyRaw copies a stored document so it survives the cursor's buffer.
func copyRaw(raw bson.Raw) bson.Raw {
	return bson.Raw(append([]byte(nil), raw...))
}

// categoryResolver resolves one category reference; a missing
// reference resolves nil so nested shapes stay pinned.
type categoryResolver func(
	ctx context.Context,
	objectID bson.ObjectID,
) (*catalog.Category, error)

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
// ObjectId category through the resolver. A missing or legacy
// string-typed category keeps the nested response nil.
func (s *CatalogStore) decodeTopic(
	ctx context.Context,
	raw bson.Raw,
	resolve categoryResolver,
) (*catalog.Topic, error) {
	var document TopicDocument
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, err
	}

	var nested *catalog.Category
	category := raw.Lookup("category")
	if category.Type == bson.TypeObjectID {
		resolved, err := resolve(ctx, category.ObjectID())
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

// batchTopics resolves every ObjectId category reference of the stored
// topic documents in one query.
func (s *CatalogStore) batchCategories(
	ctx context.Context,
	raws []bson.Raw,
) (map[bson.ObjectID]*catalog.Category, error) {
	references := make([]bson.ObjectID, 0, len(raws))
	for _, raw := range raws {
		category := raw.Lookup("category")
		if category.Type != bson.TypeObjectID {
			continue
		}
		references = append(references, category.ObjectID())
	}

	if len(references) == 0 {
		return map[bson.ObjectID]*catalog.Category{}, nil
	}

	cursor, err := s.categories.Find(ctx, bson.M{"_id": bson.M{
		"$in": references,
	}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	categories := make(map[bson.ObjectID]*catalog.Category)
	for cursor.Next(ctx) {
		category, err := decodeCategory(cursor.Current)
		if err != nil {
			return nil, err
		}
		id := cursor.Current.Lookup("_id").ObjectID()
		categories[id] = category
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// categoryLookup adapts a resolved category map to the resolver shape.
func categoryLookup(
	categories map[bson.ObjectID]*catalog.Category,
) categoryResolver {
	return func(_ context.Context, objectID bson.ObjectID) (*catalog.Category, error) {
		return categories[objectID], nil
	}
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

// decodeQuestionSet maps a stored question set and joins its topic.
func (s *CatalogStore) decodeQuestionSet(
	ctx context.Context,
	raw bson.Raw,
	preloadedTopic *catalog.Topic,
) (*catalog.QuestionSet, error) {
	var document QuestionSetDocument
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, err
	}

	topicID, topic, err := s.questionSetTopic(
		ctx, raw.Lookup("topic"), preloadedTopic,
	)
	if err != nil {
		return nil, err
	}

	questions, err := questionPassthrough(raw)
	if err != nil {
		return nil, err
	}

	return &catalog.QuestionSet{
		ID:        document.ID.Hex(),
		TopicID:   topicID,
		Topic:     topic,
		SetType:   document.SetType,
		Level:     document.Level,
		Questions: questions,
		CreatedAt: catalog.ISO8601(document.CreatedAt),
		UpdatedAt: catalog.ISO8601(document.UpdatedAt),
	}, nil
}

// questionSetTopic builds the stored identifier and its left-joined topic.
func (s *CatalogStore) questionSetTopic(
	ctx context.Context,
	reference bson.RawValue,
	preloadedTopic *catalog.Topic,
) (string, *catalog.Topic, error) {
	topicID, err := questionSetTopicID(reference)
	if err != nil {
		return "", nil, err
	}

	if preloadedTopic != nil && preloadedTopic.ID == topicID {
		return topicID, preloadedTopic, nil
	}
	if reference.Type != bson.TypeObjectID {
		return topicID, nil, nil
	}

	raw, err := s.findRaw(ctx, s.topics, reference.ObjectID())
	if err != nil {
		return "", nil, err
	}
	if raw == nil {
		return topicID, nil, nil
	}

	topic, err := s.decodeTopic(ctx, raw, s.categoryByObjectID)
	if err != nil {
		return "", nil, err
	}

	return topicID, topic, nil
}

// questionSetTopicID returns the string form of a stored topic reference.
func questionSetTopicID(reference bson.RawValue) (string, error) {
	switch reference.Type {
	case bson.TypeObjectID:
		return reference.ObjectID().Hex(), nil
	case bson.TypeString:
		return reference.StringValue(), nil
	default:
		return "", fmt.Errorf(
			"unsupported question set topic type %s", reference.Type,
		)
	}
}

// questionPassthrough encodes the stored questions array untouched, in
// stored field order.
func questionPassthrough(raw bson.Raw) (json.RawMessage, error) {
	questions := raw.Lookup("questions")
	if questions.Type == bson.TypeNull {
		return json.RawMessage("null"), nil
	}

	return rawValueJSON(questions)
}
