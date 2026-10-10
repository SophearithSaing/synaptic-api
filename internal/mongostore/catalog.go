package mongostore

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// CatalogStore implements catalog.Repository using MongoDB.
type CatalogStore struct {
	categories         *mongo.Collection
	topics             *mongo.Collection
	questionSets       *mongo.Collection
	sessions           *mongo.Collection
	liveSessions       *mongo.Collection
	setAttempts        *mongo.Collection
	liveQuestions      *mongo.Collection
	sessionEvaluations *mongo.Collection
}

// NewCatalogStore builds a CatalogStore over a database.
func NewCatalogStore(database *mongo.Database) *CatalogStore {
	return &CatalogStore{
		categories:         database.Collection("categories"),
		topics:             database.Collection("topics"),
		questionSets:       database.Collection("questionSets"),
		sessions:           database.Collection("sessions"),
		liveSessions:       database.Collection("liveSessions"),
		setAttempts:        database.Collection("setAttempts"),
		liveQuestions:      database.Collection("liveQuestions"),
		sessionEvaluations: database.Collection("sessionEvaluations"),
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

// ListCategories lists every category sorted by title.
func (s *CatalogStore) ListCategories(
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

// GetCategoryByID resolves one category by hex ObjectId.
func (s *CatalogStore) GetCategoryByID(
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

// ListTopics lists every topic sorted by title with the nested category.
// All referenced categories load in one batched query instead of one
// lookup per topic.
func (s *CatalogStore) ListTopics(ctx context.Context) ([]catalog.Topic, error) {
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

// GetTopicByID resolves one topic by hex ObjectId with the nested
// category.
func (s *CatalogStore) GetTopicByID(
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

// GetQuestionSetByID resolves one question set by hex ObjectId.
func (s *CatalogStore) GetQuestionSetByID(
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

// ListQuestionSetsByTopicSlug lists question sets for a topic slug in
// stored order and reuses the joined topic for every result.
func (s *CatalogStore) ListQuestionSetsByTopicSlug(
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
		"topicId": topicRaw.Lookup("_id").ObjectID(),
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

// decodeTopic maps the stored topic document and joins its category.
func (s *CatalogStore) decodeTopic(
	ctx context.Context,
	raw bson.Raw,
	resolve categoryResolver,
) (*catalog.Topic, error) {
	var document TopicDocument
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, err
	}

	nested, err := resolve(ctx, document.CategoryID)
	if err != nil {
		return nil, err
	}

	return &catalog.Topic{
		ID:          document.ID.Hex(),
		Title:       document.Title,
		Slug:        document.Slug,
		Description: document.Description,
		Icon:        document.Icon,
		Tags:        document.Tags,
		CategoryID:  document.CategoryID.Hex(),
		Category:    nested,
		CreatedAt:   document.CreatedAt,
		UpdatedAt:   document.UpdatedAt,
		Version:     document.Version,
	}, nil
}

// batchCategories resolves every category reference of the stored
// topic documents in one query.
func (s *CatalogStore) batchCategories(
	ctx context.Context,
	raws []bson.Raw,
) (map[bson.ObjectID]*catalog.Category, error) {
	references := make([]bson.ObjectID, 0, len(raws))
	for _, raw := range raws {
		var document TopicDocument
		if err := bson.Unmarshal(raw, &document); err != nil {
			return nil, err
		}
		references = append(references, document.CategoryID)
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

	topic, err := s.questionSetTopic(ctx, document.TopicID, preloadedTopic)
	if err != nil {
		return nil, err
	}

	return &catalog.QuestionSet{
		ID:        document.ID.Hex(),
		TopicID:   document.TopicID.Hex(),
		Topic:     topic,
		SetType:   document.SetType,
		Level:     document.Level,
		Questions: document.Questions,
		CreatedAt: document.CreatedAt,
		UpdatedAt: document.UpdatedAt,
	}, nil
}

// questionSetTopic builds the stored identifier and its left-joined topic.
func (s *CatalogStore) questionSetTopic(
	ctx context.Context,
	topicID bson.ObjectID,
	preloadedTopic *catalog.Topic,
) (*catalog.Topic, error) {
	if preloadedTopic != nil && preloadedTopic.ID == topicID.Hex() {
		return preloadedTopic, nil
	}

	raw, err := s.findRaw(ctx, s.topics, topicID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	topic, err := s.decodeTopic(ctx, raw, s.categoryByObjectID)
	if err != nil {
		return nil, err
	}

	return topic, nil
}
