package mongostore_test

import (
	"context"
	"errors"
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
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "Category", Slug: "category", Description: "Category", Icon: "book"})
	if err != nil {
		t.Fatal(err)
	}
	runTogether(
		func() {
			_, _ = store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "Created", Slug: "created", Description: "Created", Icon: "tag", Tags: []string{"tag"}, Category: category.ID})
		},
		func() { _ = store.DeleteCategory(ctx, category.ID) },
	)
	assertNoDanglingReference(t, ctx, database, "topics", "categoryId", mustID(t, category.ID), "categories")

	category, topic := createCategoryAndTopic(t, ctx, store, "set-race")
	level := int64(0)
	runTogether(
		func() {
			_, _ = store.CreateQuestionSet(ctx, catalog.CreateQuestionSetRequest{Topic: topic.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}})
		},
		func() { _ = store.DeleteTopic(ctx, topic.ID) },
	)
	assertNoDanglingReference(t, ctx, database, "questionSets", "topicId", mustID(t, topic.ID), "topics")
	_ = category

	_, source := createCategoryAndTopic(t, ctx, store, "move-source")
	_, target := createCategoryAndTopic(t, ctx, store, "move-target")
	set, err := store.CreateQuestionSet(ctx, catalog.CreateQuestionSetRequest{Topic: source.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}})
	if err != nil {
		t.Fatal(err)
	}
	targetID := target.ID
	runTogether(
		func() {
			_, _ = store.UpdateQuestionSet(ctx, set.ID, catalog.UpdateQuestionSetRequest{Topic: &targetID})
		},
		func() { _ = store.DeleteTopic(ctx, target.ID) },
	)
	assertNoDanglingReference(t, ctx, database, "questionSets", "topicId", mustID(t, target.ID), "topics")
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
	if err == nil {
		t.Fatal("duplicate category slug was accepted")
	}
	_, err = store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "One", Slug: "topic", Description: "One", Icon: "one", Tags: []string{"tag"}, Category: category.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "Two", Slug: "topic", Description: "Two", Icon: "two", Tags: []string{"tag"}, Category: category.ID})
	if err == nil {
		t.Fatal("duplicate topic slug was accepted")
	}
}

func TestCatalogSelectionAllowsDuplicateGroupsInObjectIDOrder(t *testing.T) {
	ctx, _, store := newCatalogAuthoringStore(t)
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: "One", Slug: "one", Description: "One", Icon: "one"})
	if err != nil {
		t.Fatal(err)
	}
	topic, err := store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: "Topic", Slug: "topic", Description: "Topic", Icon: "tag", Tags: []string{"tag"}, Category: category.ID})
	if err != nil {
		t.Fatal(err)
	}
	level := int64(1)
	first, err := store.CreateQuestionSet(ctx, catalog.CreateQuestionSetRequest{Topic: topic.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateQuestionSet(ctx, catalog.CreateQuestionSetRequest{Topic: topic.ID, SetType: "regular", Level: &level, Questions: []catalog.Question{validMCQ()}})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := store.SelectQuestionSet(ctx, topic.ID, level, "regular")
	if err != nil {
		t.Fatal(err)
	}
	want := first.ID
	if second.ID < want {
		want = second.ID
	}
	if selected.ID != want {
		t.Fatalf("selected %s, want %s", selected.ID, want)
	}
	if _, err := store.SelectQuestionSet(ctx, topic.ID, level, "live"); !errors.Is(err, catalog.ErrQuestionSetNotFound) {
		t.Fatalf("exact type selection: %v", err)
	}
	if _, err := store.SelectQuestionSet(ctx, topic.ID, level+1, "regular"); !errors.Is(err, catalog.ErrQuestionSetNotFound) {
		t.Fatalf("exact level selection: %v", err)
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
	return catalog.Question{ID: "q", Type: "mcq", Prompt: "Question", Options: []catalog.QuestionOption{{ID: "a", Text: "Answer"}}, CorrectOptionID: "a", TargetConcepts: []string{"concept"}, Feedback: catalog.QuestionFeedback{Correct: "Correct", Incorrect: "Incorrect"}, Rubrics: catalog.QuestionRubric{KeyPoints: []string{"point"}, Misconceptions: []string{"mistake"}}}
}

func mustID(t *testing.T, value string) bson.ObjectID {
	t.Helper()
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func runTogether(first, second func()) {
	ready := make(chan struct{})
	var group sync.WaitGroup
	group.Add(2)
	for _, operation := range []func(){first, second} {
		go func(operation func()) {
			defer group.Done()
			<-ready
			operation()
		}(operation)
	}
	close(ready)
	group.Wait()
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
	category, err := store.CreateCategory(ctx, catalog.CreateCategoryRequest{Title: suffix, Slug: suffix + "-category", Description: suffix, Icon: "book"})
	if err != nil {
		t.Fatal(err)
	}
	topic, err := store.CreateTopic(ctx, catalog.CreateTopicRequest{Title: suffix, Slug: suffix + "-topic", Description: suffix, Icon: "tag", Tags: []string{"tag"}, Category: category.ID})
	if err != nil {
		t.Fatal(err)
	}
	return category, topic
}
