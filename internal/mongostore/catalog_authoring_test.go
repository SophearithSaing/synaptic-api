package mongostore_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
)

func TestCatalogAuthoringPersistsAndRestrictsReferences(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	validator := newCatalogAuthoringValidator(t)
	service := catalog.NewService(store, validator)

	category, messages, err := service.CreateCategory(ctx,
		catalog.CreateCategoryRequest{Title: "New", Slug: "new", Description: "New category", Icon: "book"},
	)
	if err != nil || len(messages) != 0 {
		t.Fatalf("create category: %v, %v", messages, err)
	}
	topic, messages, err := service.CreateTopic(ctx, catalog.CreateTopicRequest{
		Title: "New topic", Slug: "new-topic", Description: "New topic", Icon: "tag", Tags: []string{"tag"}, Category: category.ID,
	})
	if err != nil || len(messages) != 0 {
		t.Fatalf("create topic: %v, %v", messages, err)
	}
	level := int64(0)
	set, messages, err := service.CreateQuestionSets(ctx, []catalog.CreateQuestionSetRequest{{
		Topic: topic.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()},
	}})
	if err != nil || len(messages) != 0 || len(set) != 1 {
		t.Fatalf("create set: %v, %v", messages, err)
	}
	if set[0].CreatedAt.IsZero() || set[0].UpdatedAt.IsZero() {
		t.Fatal("question set timestamps were not set")
	}
	var storedSet struct {
		Version int `bson:"__v"`
	}
	if err := database.Collection("questionSets").FindOne(
		ctx, bson.M{"_id": mustID(t, set[0].ID)},
	).Decode(&storedSet); err != nil {
		t.Fatal(err)
	}
	if storedSet.Version != 0 {
		t.Fatalf("question set version %d, want 0", storedSet.Version)
	}

	selected, err := store.SelectQuestionSet(ctx, topic.ID, 0, "regular")
	if err != nil || selected.ID != set[0].ID {
		t.Fatalf("select set: %#v, %v", selected, err)
	}
	if err := store.DeleteCategory(ctx, category.ID); !errors.Is(err, catalog.ErrCategoryReferenced) {
		t.Fatalf("delete referenced category: %v", err)
	}
	if err := store.DeleteTopic(ctx, topic.ID); !errors.Is(err, catalog.ErrTopicReferenced) {
		t.Fatalf("delete referenced topic: %v", err)
	}

	setID, err := bson.ObjectIDFromHex(set[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Collection("setAttempts").InsertOne(ctx, bson.M{"questionSet": setID}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteQuestionSet(ctx, set[0].ID); !errors.Is(err, catalog.ErrQuestionSetReferenced) {
		t.Fatalf("delete referenced set: %v", err)
	}
	if _, err := database.Collection("setAttempts").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteQuestionSet(ctx, set[0].ID); err != nil {
		t.Fatalf("delete set: %v", err)
	}
	if err := store.DeleteTopic(ctx, topic.ID); err != nil {
		t.Fatalf("delete topic: %v", err)
	}
	if err := store.DeleteCategory(ctx, category.ID); err != nil {
		t.Fatalf("delete category: %v", err)
	}
}

func TestCatalogDeletesHonorHistoricalAndCanonicalReferences(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{
		Title: "Category", Slug: "category", Description: "Category", Icon: "book",
	})
	if err != nil {
		t.Fatal(err)
	}
	topic, err := store.CreateTopic(ctx, catalog.CreateTopicRequest{
		Title: "Topic", Slug: "topic", Description: "Topic", Icon: "tag",
		Tags: []string{"tag"}, Category: category.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	topicID := mustID(t, topic.ID)
	for _, reference := range []struct {
		collection string
		field      string
		value      any
	}{
		{"sessions", "topic", topicID},
		{"liveSessions", "topic", topic.ID},
		{"setAttempts", "topicId", topicID},
		{"sessionEvaluations", "topic", topic.ID},
	} {
		collection := database.Collection(reference.collection)
		if _, err := collection.InsertOne(ctx, bson.M{reference.field: reference.value}); err != nil {
			t.Fatal(err)
		}
		if err := store.DeleteTopic(ctx, topic.ID); !errors.Is(err, catalog.ErrTopicReferenced) {
			t.Fatalf("%s/%s delete: %v", reference.collection, reference.field, err)
		}
		if _, err := collection.DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatal(err)
		}
	}

	level := int64(0)
	set, err := store.CreateQuestionSet(ctx, catalog.CreateQuestionSetRequest{Topic: topic.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}})
	if err != nil {
		t.Fatal(err)
	}
	setID := mustID(t, set.ID)
	for _, reference := range []struct {
		collection string
		field      string
		value      any
	}{
		{"setAttempts", "questionSet", setID},
		{"liveQuestions", "questionSetId", set.ID},
	} {
		collection := database.Collection(reference.collection)
		if _, err := collection.InsertOne(ctx, bson.M{reference.field: reference.value}); err != nil {
			t.Fatal(err)
		}
		if err := store.DeleteQuestionSet(ctx, set.ID); !errors.Is(err, catalog.ErrQuestionSetReferenced) {
			t.Fatalf("%s/%s delete: %v", reference.collection, reference.field, err)
		}
		if _, err := collection.DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCatalogWriteDeleteRacesPreserveReferences(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	for attempt := 0; attempt < 3; attempt++ {
		suffix := fmt.Sprintf("race-%d", attempt)
		category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{
			Title: suffix, Slug: suffix + "-category", Description: suffix,
			Icon: "book",
		})
		if err != nil {
			t.Fatal(err)
		}
		outcomes := runTogether(
			func() error {
				_, err := store.CreateTopic(ctx, topicRequest(
					suffix+"-child", category.ID,
				))
				return err
			},
			func() error { return store.DeleteCategory(ctx, category.ID) },
		)
		assertRaceOutcome(t, outcomes, catalog.ErrCategoryNotFound,
			catalog.ErrCategoryReferenced)
		assertNoDanglingReference(t, ctx, database, "topics", "categoryId",
			mustID(t, category.ID), "categories")

		_, topic := createCategoryAndTopic(t, ctx, store, suffix+"-set")
		level := int64(0)
		outcomes = runTogether(
			func() error {
				_, err := store.CreateQuestionSet(ctx, questionSetRequest(
					topic.ID, level,
				))
				return err
			},
			func() error { return store.DeleteTopic(ctx, topic.ID) },
		)
		assertRaceOutcome(t, outcomes, catalog.ErrTopicNotFound,
			catalog.ErrTopicReferenced)
		assertNoDanglingReference(t, ctx, database, "questionSets", "topicId",
			mustID(t, topic.ID), "topics")

		_, source := createCategoryAndTopic(t, ctx, store, suffix+"-source")
		_, target := createCategoryAndTopic(t, ctx, store, suffix+"-target")
		set, err := store.CreateQuestionSet(ctx, questionSetRequest(source.ID, level))
		if err != nil {
			t.Fatal(err)
		}
		targetID := target.ID
		outcomes = runTogether(
			func() error {
				_, err := store.UpdateQuestionSet(ctx, set.ID,
					catalog.UpdateQuestionSetRequest{Topic: &targetID},
				)
				return err
			},
			func() error { return store.DeleteTopic(ctx, target.ID) },
		)
		assertRaceOutcome(t, outcomes, catalog.ErrTopicNotFound,
			catalog.ErrTopicReferenced)
		assertNoDanglingReference(t, ctx, database, "questionSets", "topicId",
			mustID(t, target.ID), "topics")
	}
}

func TestCatalogBulkWritesUseDocumentedPartialBehavior(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	validator := newCatalogAuthoringValidator(t)
	service := catalog.NewService(store, validator)
	category, _, err := service.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "New", Slug: "new", Description: "New category", Icon: "book"})
	if err != nil {
		t.Fatal(err)
	}
	topic, _, err := service.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "Topic", Slug: "topic", Description: "Topic", Icon: "tag", Tags: []string{"tag"}, Category: category.ID})
	if err != nil {
		t.Fatal(err)
	}
	level := int64(0)
	missingTopic := "665f1e2b9d1a2c3b4d5e9999"
	_, _, err = service.CreateQuestionSets(ctx, []catalog.CreateQuestionSetRequest{
		{Topic: topic.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}},
		{Topic: missingTopic, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}},
	})
	if !errors.Is(err, catalog.ErrTopicNotFound) {
		t.Fatalf("bulk create error: %v", err)
	}
	if count, err := database.Collection("questionSets").CountDocuments(ctx, bson.M{"topicId": mustID(t, topic.ID)}); err != nil || count != 1 {
		t.Fatalf("bulk create count %d, %v", count, err)
	}

	var created struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := database.Collection("questionSets").FindOne(ctx, bson.M{"topicId": mustID(t, topic.ID)}).Decode(&created); err != nil {
		t.Fatal(err)
	}
	nextLevel := int64(2)
	_, _, err = service.UpdateQuestionSets(ctx, []catalog.BulkUpdateQuestionSetRequest{
		{ID: "665f1e2b9d1a2c3b4d5e9998", UpdateQuestionSetRequest: catalog.UpdateQuestionSetRequest{Level: &nextLevel}},
		{ID: created.ID.Hex(), UpdateQuestionSetRequest: catalog.UpdateQuestionSetRequest{Level: &nextLevel}},
	})
	if !errors.Is(err, catalog.ErrQuestionSetNotFound) {
		t.Fatalf("bulk update error: %v", err)
	}
	var updated struct {
		Level int64 `bson:"level"`
	}
	if err := database.Collection("questionSets").FindOne(ctx, bson.M{"_id": created.ID}).Decode(&updated); err != nil || updated.Level != nextLevel {
		t.Fatalf("bulk update saved %#v, %v", updated, err)
	}
}

func TestCatalogIndexesRejectDuplicateSlugs(t *testing.T) {
	ctx, _, store := newCatalogAuthoringStore(t)
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "One", Slug: "same", Description: "One", Icon: "one"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "Two", Slug: "same", Description: "Two", Icon: "two"})
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatal("duplicate category slug was accepted")
	}
	_, err = store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "One", Slug: "topic", Description: "One", Icon: "one", Tags: []string{"tag"}, Category: category.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "Two", Slug: "topic", Description: "Two", Icon: "two", Tags: []string{"tag"}, Category: category.ID})
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatal("duplicate topic slug was accepted")
	}
}

func TestCatalogSelectionAllowsDuplicateGroupsInObjectIDOrder(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "One", Slug: "one", Description: "One", Icon: "one"})
	if err != nil {
		t.Fatal(err)
	}
	topic, err := store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "Topic", Slug: "topic", Description: "Topic", Icon: "tag", Tags: []string{"tag"}, Category: category.ID})
	if err != nil {
		t.Fatal(err)
	}
	topicID := mustID(t, topic.ID)
	low := mustID(t, "665f1e2b9d1a2c3b4d5e0001")
	high := mustID(t, "665f1e2b9d1a2c3b4d5e0002")
	live := mustID(t, "665f1e2b9d1a2c3b4d5e0003")
	otherLevel := mustID(t, "665f1e2b9d1a2c3b4d5e0004")
	if _, err := database.Collection("questionSets").InsertMany(ctx, []any{
		questionSetDocument(high, topicID, "regular", 1),
		questionSetDocument(low, topicID, "regular", 1),
		questionSetDocument(live, topicID, "live", 1),
		questionSetDocument(otherLevel, topicID, "regular", 2),
	}); err != nil {
		t.Fatal(err)
	}
	assertSelectedID(t, ctx, store, topic.ID, 1, "regular", low.Hex())
	assertSelectedID(t, ctx, store, topic.ID, 1, "live", live.Hex())
	assertSelectedID(t, ctx, store, topic.ID, 2, "regular", otherLevel.Hex())
	if _, err := store.SelectQuestionSet(ctx, topic.ID, 3, "regular"); !errors.Is(err, catalog.ErrQuestionSetNotFound) {
		t.Fatalf("no match: %v", err)
	}
}

func TestCatalogAuthoringRejectsMissingAndInvalidReferences(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	missing := "665f1e2b9d1a2c3b4d5e9999"
	if _, err := store.CreateTopic(ctx, catalog.CreateTopicRequest{Category: missing}); !errors.Is(err, catalog.ErrCategoryNotFound) {
		t.Fatalf("missing category: %v", err)
	}
	level := int64(0)
	if _, err := store.CreateQuestionSet(ctx, catalog.CreateQuestionSetRequest{Topic: missing, Level: &level}); !errors.Is(err, catalog.ErrTopicNotFound) {
		t.Fatalf("missing topic: %v", err)
	}
	for _, deletion := range []func(string) error{
		func(id string) error { return store.DeleteCategory(ctx, id) },
		func(id string) error { return store.DeleteTopic(ctx, id) },
		func(id string) error { return store.DeleteQuestionSet(ctx, id) },
	} {
		if err := deletion("bad-id"); !errors.Is(err, catalog.ErrInvalidObjectID) {
			t.Fatalf("invalid delete id: %v", err)
		}
	}
	for _, deletion := range []struct {
		delete func(string) error
		want   error
	}{
		{func(id string) error { return store.DeleteCategory(ctx, id) }, catalog.ErrCategoryNotFound},
		{func(id string) error { return store.DeleteTopic(ctx, id) }, catalog.ErrTopicNotFound},
		{func(id string) error { return store.DeleteQuestionSet(ctx, id) }, catalog.ErrQuestionSetNotFound},
	} {
		if err := deletion.delete(missing); !errors.Is(err, deletion.want) {
			t.Fatalf("missing delete: %v", err)
		}
	}
	legacyID := bson.NewObjectID()
	if _, err := database.Collection("questionSets").InsertOne(ctx, bson.M{"_id": legacyID, "topicId": mustID(t, missing), "setType": "regular", "level": int64(0), "questions": bson.A{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateQuestionSet(ctx, legacyID.Hex(), catalog.UpdateQuestionSetRequest{Topic: &missing}); !errors.Is(err, catalog.ErrTopicNotFound) {
		t.Fatalf("same missing topic patch: %v", err)
	}
}

func TestCatalogDuplicateTopicRollsBackParentVersion(t *testing.T) {
	ctx, database, store := newCatalogAuthoringStore(t)
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "Category", Slug: "category", Description: "Category", Icon: "book"})
	if err != nil {
		t.Fatal(err)
	}
	request := catalog.CreateTopicRequest{Title: "Topic", Slug: "topic", Description: "Topic", Icon: "tag", Tags: []string{"tag"}, Category: category.ID}
	if _, err := store.CreateTopic(ctx, request); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTopic(ctx, request); err == nil {
		t.Fatal("duplicate topic was accepted")
	}
	var document struct {
		Version int `bson:"__v"`
	}
	if err := database.Collection("categories").FindOne(ctx, bson.M{"_id": mustID(t, category.ID)}).Decode(&document); err != nil {
		t.Fatal(err)
	}
	if document.Version != 1 {
		t.Fatalf("category version %d, want 1", document.Version)
	}
}

func newCatalogAuthoringStore(t *testing.T) (context.Context, *mongo.Database, *mongostore.CatalogStore) {
	t.Helper()
	ctx := context.Background()
	client, err := mongostore.Connect(ctx, startMongo(t, ctx))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	database := client.Database("catalogauthoring")
	if err := mongostore.EnsureCatalogIndexes(ctx, database); err != nil {
		t.Fatal(err)
	}
	return ctx, database, mongostore.NewCatalogStore(database)
}

func newCatalogAuthoringValidator(t *testing.T) *catalog.AuthoringValidator {
	t.Helper()
	validator, err := catalog.NewAuthoringValidator()
	if err != nil {
		t.Fatal(err)
	}
	return validator
}

func validMCQ() catalog.Question {
	return catalog.Question{
		ID: "q", Type: "mcq", Prompt: "Question", CorrectOptionID: "a",
		Options:        []catalog.QuestionOption{{ID: "a", Text: "Answer"}},
		TargetConcepts: []string{"concept"},
		Feedback: catalog.QuestionFeedback{
			Correct: "Correct", Incorrect: "Incorrect",
		},
		Rubrics: catalog.QuestionRubric{
			KeyPoints: []string{"point"}, Misconceptions: []string{"mistake"},
		},
	}
}

func topicRequest(slug, categoryID string) catalog.CreateTopicRequest {
	return catalog.CreateTopicRequest{
		Title: slug, Slug: slug, Description: slug, Icon: "tag",
		Tags: []string{"tag"}, Category: categoryID,
	}
}

func questionSetRequest(
	topicID string,
	level int64,
) catalog.CreateQuestionSetRequest {
	return catalog.CreateQuestionSetRequest{
		Topic: topicID, SetType: "regular", Level: &level,
		Questions: []catalog.Question{validMCQ()},
	}
}

func mustID(t *testing.T, value string) bson.ObjectID {
	t.Helper()
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func runTogether(first, second func() error) [2]error {
	ready := make(chan struct{})
	var outcomes [2]error
	var group sync.WaitGroup
	group.Add(2)
	for index, operation := range []func() error{first, second} {
		go func(index int, operation func() error) {
			defer group.Done()
			<-ready
			outcomes[index] = operation()
		}(index, operation)
	}
	close(ready)
	group.Wait()
	return outcomes
}

func assertRaceOutcome(
	t *testing.T,
	outcomes [2]error,
	createNotFound, deleteReferenced error,
) {
	t.Helper()
	if outcomes[0] == nil && errors.Is(outcomes[1], deleteReferenced) {
		return
	}
	if errors.Is(outcomes[0], createNotFound) && outcomes[1] == nil {
		return
	}
	t.Fatalf("race outcomes %v and %v", outcomes[0], outcomes[1])
}

func assertSelectedID(
	t *testing.T,
	ctx context.Context,
	store *mongostore.CatalogStore,
	topic string,
	level int64,
	setType, want string,
) {
	t.Helper()
	selected, err := store.SelectQuestionSet(ctx, topic, level, setType)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != want {
		t.Fatalf("selected %s, want %s", selected.ID, want)
	}
}

func questionSetDocument(
	id, topicID bson.ObjectID,
	setType string,
	level int64,
) bson.M {
	return bson.M{
		"_id": id, "topicId": topicID, "setType": setType, "level": level,
		"questions": bson.A{}, "__v": 0,
	}
}

func assertNoDanglingReference(
	t *testing.T,
	ctx context.Context,
	database *mongo.Database,
	children, field string,
	parentID bson.ObjectID,
	parents string,
) {
	t.Helper()
	count, err := database.Collection(children).CountDocuments(ctx, bson.M{field: parentID})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		return
	}
	parentsCount, err := database.Collection(parents).CountDocuments(ctx, bson.M{"_id": parentID})
	if err != nil {
		t.Fatal(err)
	}
	if parentsCount != 1 {
		t.Fatalf("%s references deleted %s", children, parents)
	}
}

func createCategoryAndTopic(
	t *testing.T,
	ctx context.Context,
	store *mongostore.CatalogStore,
	suffix string,
) (*catalog.Category, *catalog.Topic) {
	t.Helper()
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{
		Title: suffix, Slug: suffix + "-category", Description: suffix,
		Icon: "book",
	})
	if err != nil {
		t.Fatal(err)
	}
	topic, err := store.CreateTopic(ctx, catalog.CreateTopicRequest{
		Title: suffix, Slug: suffix + "-topic", Description: suffix,
		Icon: "tag", Tags: []string{"tag"}, Category: category.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return category, topic
}
