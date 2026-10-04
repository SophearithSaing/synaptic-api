package catalog_test

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// repoState is the in-memory catalog backend for handler tests.
// Question sets are keyed by id and topic slug.
type repoState struct {
	categories   []catalog.Category
	categoryById map[string]*catalog.Category
	topics       []catalog.Topic
	topicById    map[string]*catalog.Topic
	setsById     map[string]*catalog.QuestionSet
	setsBySlug   map[string][]catalog.QuestionSet
}

// newRepoState builds an empty store.
func newRepoState() *repoState {
	return &repoState{
		categoryById: map[string]*catalog.Category{},
		topicById:    map[string]*catalog.Topic{},
		setsById:     map[string]*catalog.QuestionSet{},
		setsBySlug:   map[string][]catalog.QuestionSet{},
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

// seedQuestionSet adds a question set.
func (r *repoState) seedQuestionSet(questionSet catalog.QuestionSet) {
	r.setsById[questionSet.ID] = &questionSet
}

// seedQuestionSetsForSlug wires a slug to its sets.
func (r *repoState) seedQuestionSetsForSlug(
	slug string, sets []catalog.QuestionSet,
) {
	r.setsBySlug[slug] = sets
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
	_ context.Context, id string,
) (*catalog.QuestionSet, error) {
	if !objectID(id) {
		return nil, catalog.ErrInvalidObjectID
	}
	if questionSet := r.setsById[id]; questionSet != nil {
		return questionSet, nil
	}

	return nil, catalog.ErrQuestionSetNotFound
}

// QuestionSetsByTopicSlug implements catalog.Repository.
func (r *repoState) QuestionSetsByTopicSlug(
	_ context.Context, slug string,
) ([]catalog.QuestionSet, error) {
	if sets, ok := r.setsBySlug[slug]; ok {
		return sets, nil
	}

	return nil, catalog.ErrTopicNotFound
}
