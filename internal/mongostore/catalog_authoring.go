package mongostore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// CreateCategory creates a category with legacy timestamps and version.
func (s *CatalogStore) CreateCategory(
	ctx context.Context,
	request catalog.CreateCategoryRequest,
) (*catalog.Category, error) {
	now := time.Now().UTC()
	document := CategoryDocument{
		ID: bson.NewObjectID(), Title: request.Title, Slug: request.Slug,
		Description: request.Description, Icon: request.Icon,
		CreatedAt: now, UpdatedAt: now, Version: 0,
	}
	if _, err := s.categories.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return categoryFromDocument(document), nil
}

// CreateTopic creates a topic after transactionally guarding its category.
func (s *CatalogStore) CreateTopic(
	ctx context.Context,
	request catalog.CreateTopicRequest,
) (*catalog.Topic, error) {
	categoryID, err := parseObjectID(request.Category)
	if err != nil {
		return nil, err
	}

	result, err := s.withTransaction(ctx, func(ctx context.Context) (any, error) {
		category, err := s.categoryDocument(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, catalog.ErrCategoryNotFound
		}
		if err := s.bumpVersion(ctx, s.categories, categoryID); err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		document := TopicDocument{
			ID: bson.NewObjectID(), Title: request.Title, Slug: request.Slug,
			Description: request.Description, Icon: request.Icon, Tags: request.Tags,
			CategoryID: categoryID, CreatedAt: now, UpdatedAt: now, Version: 0,
		}
		if _, err := s.topics.InsertOne(ctx, document); err != nil {
			return nil, err
		}

		return topicFromDocument(document, categoryFromDocument(*category)), nil
	})
	if err != nil {
		return nil, err
	}
	topic, ok := result.(*catalog.Topic)
	if !ok {
		return nil, fmt.Errorf("create topic returned %T", result)
	}

	return topic, nil
}

// CreateQuestionSet creates a question set after transactionally guarding its
// referenced topic.
func (s *CatalogStore) CreateQuestionSet(
	ctx context.Context,
	request catalog.CreateQuestionSetRequest,
) (*catalog.QuestionSet, error) {
	if request.Level == nil {
		return nil, errors.New("question set level is required")
	}
	topicID, err := parseObjectID(request.Topic)
	if err != nil {
		return nil, err
	}

	result, err := s.withTransaction(ctx, func(ctx context.Context) (any, error) {
		topic, err := s.topicDocument(ctx, topicID)
		if err != nil {
			return nil, err
		}
		if topic == nil {
			return nil, catalog.ErrTopicNotFound
		}
		if err := s.bumpVersion(ctx, s.topics, topicID); err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		document := QuestionSetDocument{
			ID: bson.NewObjectID(), TopicID: topicID, SetType: request.SetType,
			Level: *request.Level, Questions: request.Questions,
			CreatedAt: now, UpdatedAt: now, Version: 0,
		}
		if _, err := s.questionSets.InsertOne(ctx, document); err != nil {
			return nil, err
		}
		return document, nil
	})
	if err != nil {
		return nil, err
	}
	document, ok := result.(QuestionSetDocument)
	if !ok {
		return nil, fmt.Errorf("create question set returned %T", result)
	}

	return s.questionSetFromDocument(ctx, document)
}

// UpdateQuestionSet applies a partial update to one question set.
func (s *CatalogStore) UpdateQuestionSet(
	ctx context.Context,
	id string,
	request catalog.UpdateQuestionSetRequest,
) (*catalog.QuestionSet, error) {
	questionSetID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	var nextTopicID bson.ObjectID
	if request.Topic != nil {
		nextTopicID, err = parseObjectID(*request.Topic)
		if err != nil {
			return nil, err
		}
	}

	result, err := s.withTransaction(ctx, func(ctx context.Context) (any, error) {
		document, err := s.questionSetDocument(ctx, questionSetID)
		if err != nil {
			return nil, err
		}
		if document == nil {
			return nil, catalog.ErrQuestionSetNotFound
		}

		set := bson.M{"updatedAt": time.Now().UTC()}
		if request.Topic != nil {
			topic, err := s.topicDocument(ctx, nextTopicID)
			if err != nil {
				return nil, err
			}
			if topic == nil {
				return nil, catalog.ErrTopicNotFound
			}
			if err := s.bumpVersion(ctx, s.topics, nextTopicID); err != nil {
				return nil, err
			}
			if nextTopicID != document.TopicID {
				set["topicId"] = nextTopicID
			}
		}
		if request.SetType != nil {
			set["setType"] = *request.SetType
		}
		if request.Level != nil {
			set["level"] = *request.Level
		}
		if request.Questions != nil {
			set["questions"] = *request.Questions
		}
		if _, err := s.questionSets.UpdateOne(ctx,
			bson.M{"_id": questionSetID}, bson.M{"$set": set}); err != nil {
			return nil, err
		}
		for key, value := range set {
			switch key {
			case "topicId":
				document.TopicID = value.(bson.ObjectID)
			case "setType":
				document.SetType = value.(string)
			case "level":
				document.Level = value.(int64)
			case "questions":
				document.Questions = value.([]catalog.Question)
			case "updatedAt":
				document.UpdatedAt = value.(time.Time)
			}
		}
		return *document, nil
	})
	if err != nil {
		return nil, err
	}
	document, ok := result.(QuestionSetDocument)
	if !ok {
		return nil, fmt.Errorf("update question set returned %T", result)
	}

	return s.questionSetFromDocument(ctx, document)
}

// DeleteCategory deletes a category only when no topic references it.
func (s *CatalogStore) DeleteCategory(ctx context.Context, id string) error {
	objectID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	_, err = s.withTransaction(ctx, func(ctx context.Context) (any, error) {
		if exists, err := s.exists(ctx, s.categories, "_id", objectID); err != nil || !exists {
			if err != nil {
				return nil, err
			}
			return nil, catalog.ErrCategoryNotFound
		}
		if referenced, err := s.hasReferences(
			ctx, s.topics, []string{"categoryId", "category"}, objectID,
		); err != nil || referenced {
			if err != nil {
				return nil, err
			}
			return nil, catalog.ErrCategoryReferenced
		}
		_, err := s.categories.DeleteOne(ctx, bson.M{"_id": objectID})
		return nil, err
	})
	return err
}

// DeleteTopic deletes a topic only when no persisted record references it.
func (s *CatalogStore) DeleteTopic(ctx context.Context, id string) error {
	objectID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	_, err = s.withTransaction(ctx, func(ctx context.Context) (any, error) {
		if exists, err := s.exists(ctx, s.topics, "_id", objectID); err != nil || !exists {
			if err != nil {
				return nil, err
			}
			return nil, catalog.ErrTopicNotFound
		}
		for _, reference := range []struct {
			collection *mongo.Collection
			fields     []string
		}{
			{s.questionSets, []string{"topicId", "topic"}},
			{s.sessions, []string{"topic", "topicId"}},
			{s.liveSessions, []string{"topic", "topicId"}},
			{s.setAttempts, []string{"topic", "topicId"}},
			{s.sessionEvaluations, []string{"topic", "topicId"}},
		} {
			if referenced, err := s.hasReferences(
				ctx, reference.collection, reference.fields, objectID,
			); err != nil || referenced {
				if err != nil {
					return nil, err
				}
				return nil, catalog.ErrTopicReferenced
			}
		}
		_, err := s.topics.DeleteOne(ctx, bson.M{"_id": objectID})
		return nil, err
	})
	return err
}

// DeleteQuestionSet deletes a set only when no persisted record references it.
func (s *CatalogStore) DeleteQuestionSet(ctx context.Context, id string) error {
	objectID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	_, err = s.withTransaction(ctx, func(ctx context.Context) (any, error) {
		if exists, err := s.exists(ctx, s.questionSets, "_id", objectID); err != nil || !exists {
			if err != nil {
				return nil, err
			}
			return nil, catalog.ErrQuestionSetNotFound
		}
		for _, reference := range []struct {
			collection *mongo.Collection
			fields     []string
		}{
			{s.setAttempts, []string{"questionSet", "questionSetId"}},
			{s.liveQuestions, []string{"questionSet", "questionSetId"}},
		} {
			if referenced, err := s.hasReferences(
				ctx, reference.collection, reference.fields, objectID,
			); err != nil || referenced {
				if err != nil {
					return nil, err
				}
				return nil, catalog.ErrQuestionSetReferenced
			}
		}
		_, err := s.questionSets.DeleteOne(ctx, bson.M{"_id": objectID})
		return nil, err
	})
	return err
}

// SelectQuestionSet selects the first exact topic, level, and type match.
func (s *CatalogStore) SelectQuestionSet(
	ctx context.Context,
	topic string,
	level int64,
	setType string,
) (*catalog.QuestionSet, error) {
	topicID, err := parseObjectID(topic)
	if err != nil {
		return nil, err
	}
	raw, err := s.questionSets.FindOne(
		ctx,
		bson.M{"topicId": topicID, "level": level, "setType": setType},
		options.FindOne().SetSort(bson.D{{Key: "_id", Value: 1}}),
	).Raw()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, catalog.ErrQuestionSetNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.decodeQuestionSet(ctx, raw, nil)
}

// withTransaction runs callback in a MongoDB transaction.
func (s *CatalogStore) withTransaction(
	ctx context.Context,
	callback func(context.Context) (any, error),
) (any, error) {
	session, err := s.categories.Database().Client().StartSession()
	if err != nil {
		return nil, fmt.Errorf("start catalog transaction: %w", err)
	}
	defer session.EndSession(ctx)
	return session.WithTransaction(ctx, callback)
}

// exists reports whether collection has an ObjectID document or reference.
func (s *CatalogStore) exists(
	ctx context.Context,
	collection *mongo.Collection,
	field string,
	value bson.ObjectID,
) (bool, error) {
	err := collection.FindOne(ctx, bson.M{field: value}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

// hasReferences reports whether collection has a historical or canonical
// reference to value.
func (s *CatalogStore) hasReferences(
	ctx context.Context,
	collection *mongo.Collection,
	fields []string,
	value bson.ObjectID,
) (bool, error) {
	filters := make(bson.A, 0, len(fields))
	for _, field := range fields {
		filters = append(filters, bson.M{field: bson.M{
			"$in": bson.A{value, value.Hex()},
		}})
	}
	err := collection.FindOne(ctx, bson.M{"$or": filters}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

// bumpVersion atomically advances a parent document's legacy version.
func (s *CatalogStore) bumpVersion(
	ctx context.Context,
	collection *mongo.Collection,
	id bson.ObjectID,
) error {
	result, err := collection.UpdateOne(
		ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"__v": 1}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("catalog parent changed during transaction")
	}
	return nil
}

// categoryDocument loads one category document or nil when absent.
func (s *CatalogStore) categoryDocument(
	ctx context.Context,
	id bson.ObjectID,
) (*CategoryDocument, error) {
	var document CategoryDocument
	err := s.categories.FindOne(ctx, bson.M{"_id": id}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &document, err
}

// topicDocument loads one topic document or nil when absent.
func (s *CatalogStore) topicDocument(
	ctx context.Context,
	id bson.ObjectID,
) (*TopicDocument, error) {
	var document TopicDocument
	err := s.topics.FindOne(ctx, bson.M{"_id": id}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &document, err
}

// questionSetDocument loads one question-set document or nil when absent.
func (s *CatalogStore) questionSetDocument(
	ctx context.Context,
	id bson.ObjectID,
) (*QuestionSetDocument, error) {
	var document QuestionSetDocument
	err := s.questionSets.FindOne(ctx, bson.M{"_id": id}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &document, err
}

// categoryFromDocument maps an in-memory category document.
func categoryFromDocument(document CategoryDocument) *catalog.Category {
	return &catalog.Category{
		ID: document.ID.Hex(), Title: document.Title, Slug: document.Slug,
		Description: document.Description, Icon: document.Icon,
	}
}

// topicFromDocument maps an in-memory topic document.
func topicFromDocument(document TopicDocument, category *catalog.Category) *catalog.Topic {
	return &catalog.Topic{
		ID: document.ID.Hex(), Title: document.Title, Slug: document.Slug,
		Description: document.Description, Icon: document.Icon, Tags: document.Tags,
		CategoryID: document.CategoryID.Hex(), Category: category,
	}
}

// questionSetFromDocument maps an in-memory question-set document.
func (s *CatalogStore) questionSetFromDocument(
	ctx context.Context,
	document QuestionSetDocument,
) (*catalog.QuestionSet, error) {
	topic, err := s.questionSetTopic(ctx, document.TopicID, nil)
	if err != nil {
		return nil, err
	}
	return &catalog.QuestionSet{
		ID: document.ID.Hex(), TopicID: document.TopicID.Hex(), Topic: topic,
		SetType: document.SetType, Level: document.Level,
		Questions: document.Questions, CreatedAt: document.CreatedAt,
		UpdatedAt: document.UpdatedAt,
	}, nil
}
