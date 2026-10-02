package catalog_test

import (
	"context"
	"encoding/json"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// repoState is the in-memory catalog backend for handler tests.
// Question sets are keyed by id and topic slug in populated and
// unpopulated variants.
type repoState struct {
	categories        []catalog.Category
	categoryById      map[string]*catalog.Category
	topics            []catalog.Topic
	topicById         map[string]*catalog.Topic
	setsById          map[string]*catalog.QuestionSet
	setsByIdPopulated map[string]*catalog.QuestionSet
	setsBySlug        map[string][]catalog.QuestionSet
	setsSlugPopulated map[string][]catalog.QuestionSet
}

// newRepoState builds an empty store.
func newRepoState() *repoState {
	return &repoState{
		categoryById:      map[string]*catalog.Category{},
		topicById:         map[string]*catalog.Topic{},
		setsById:          map[string]*catalog.QuestionSet{},
		setsByIdPopulated: map[string]*catalog.QuestionSet{},
		setsBySlug:        map[string][]catalog.QuestionSet{},
		setsSlugPopulated: map[string][]catalog.QuestionSet{},
	}
}

// seedCategory adds a category.
func (r *repoState) seedCategory(category catalog.Category) {
	r.categories = append(r.categories, category)
	r.categoryById[category.ID] = &category
}

// seedTopic adds a topic.
func (r *repoState) seedTopic(topic catalog.Topic) {
	r.topics = append(r.topics, topic)
	r.topicById[topic.ID] = &topic
}

// seedQuestionSet adds a question set for both populate variants. The
// seeded topic reference is always the stored hex string shape.
func (r *repoState) seedQuestionSet(questionSet catalog.QuestionSet) {
	unpopulated := questionSet
	r.setsById[questionSet.ID] = &unpopulated

	populated := questionSet
	populated.Topic = rawPopulatedTopic(questionSet.Topic)
	r.setsByIdPopulated[questionSet.ID] = &populated
}

// seedQuestionSetsForSlug wires a slug to its sets in both variants.
func (r *repoState) seedQuestionSetsForSlug(
	slug string, sets []catalog.QuestionSet,
) {
	r.setsBySlug[slug] = sets

	populated := make([]catalog.QuestionSet, 0, len(sets))
	for _, questionSet := range sets {
		populatedVariant := questionSet
		populatedVariant.Topic = rawPopulatedTopic(questionSet.Topic)
		populated = append(populated, populatedVariant)
	}
	r.setsSlugPopulated[slug] = populated
}

// rawPopulatedTopic renders the stored topic document for the pinned
// seeded hex reference; other references render as JSON null.
func rawPopulatedTopic(
	reference json.RawMessage,
) json.RawMessage {
	if string(reference) == `"665f1e2b9d1a2c3b4d5e0004"` {
		return json.RawMessage(`{"_id":"665f1e2b9d1a2c3b4d5e0004",` +
			`"title":"Binary Basics","slug":"binary-basics",` +
			`"description":"Binary numbers and arithmetic.",` +
			`"icon":"binary","tags":["binary","arithmetic"],` +
			`"category":"665f1e2b9d1a2c3b4d5e0003",` +
			`"createdAt":"2026-01-01T00:00:00.000Z",` +
			`"updatedAt":"2026-01-01T00:00:00.000Z","__v":0}`)
	}

	return json.RawMessage("null")
}

// Categories implements catalog.Repository sorted by title. Results
// stay non-nil so empty stores still render [] instead of null.
func (r *repoState) Categories(
	_ context.Context,
) ([]catalog.Category, error) {
	sorted := make([]catalog.Category, 0, len(r.categories))
	sorted = append(sorted, r.categories...)

	sort.Slice(sorted, func(position, other int) bool {
		return sorted[position].Title < sorted[other].Title
	})

	return sorted, nil
}

// objectID parses the fake ids with the same contract as the store:
// unparsable ids report the distinct invalid-id sentinel mapped to
// HTTP 400.
func objectID(id string) bool {
	_, err := bson.ObjectIDFromHex(id)

	return err == nil
}

// CategoryByID implements catalog.Repository.
func (r *repoState) CategoryByID(
	_ context.Context, id string,
) (*catalog.Category, error) {
	if !objectID(id) {
		return nil, catalog.ErrInvalidObjectID
	}
	if category := r.categoryById[id]; category != nil {
		return category, nil
	}

	return nil, catalog.ErrCategoryNotFound
}

// Topics implements catalog.Repository sorted by title. Results stay
// non-nil so empty stores render [] instead of null.
func (r *repoState) Topics(_ context.Context) ([]catalog.Topic, error) {
	sorted := make([]catalog.Topic, 0, len(r.topics))
	sorted = append(sorted, r.topics...)

	sort.Slice(sorted, func(position, other int) bool {
		return sorted[position].Title < sorted[other].Title
	})

	return sorted, nil
}

// TopicByID implements catalog.Repository.
func (r *repoState) TopicByID(
	_ context.Context, id string,
) (*catalog.Topic, error) {
	if !objectID(id) {
		return nil, catalog.ErrInvalidObjectID
	}
	if topic := r.topicById[id]; topic != nil {
		return topic, nil
	}

	return nil, catalog.ErrTopicNotFound
}

// QuestionSetByID implements catalog.Repository.
func (r *repoState) QuestionSetByID(
	_ context.Context, id string, populate bool,
) (*catalog.QuestionSet, error) {
	if !objectID(id) {
		return nil, catalog.ErrInvalidObjectID
	}
	if populate {
		if questionSet := r.setsByIdPopulated[id]; questionSet != nil {
			return questionSet, nil
		}

		return nil, catalog.ErrQuestionSetNotFound
	}
	if questionSet := r.setsById[id]; questionSet != nil {
		return questionSet, nil
	}

	return nil, catalog.ErrQuestionSetNotFound
}

// QuestionSetsByTopicSlug implements catalog.Repository.
func (r *repoState) QuestionSetsByTopicSlug(
	_ context.Context, slug string, populate bool,
) ([]catalog.QuestionSet, error) {
	if sets, ok := r.setsBySlug[slug]; ok {
		if populate {
			return r.setsSlugPopulated[slug], nil
		}

		return sets, nil
	}

	return nil, catalog.ErrTopicNotFound
}
