package catalog

import (
	"context"
)

// Repository resolves catalog documents for the read routes.
type Repository interface {
	// ListCategories lists every category sorted by title.
	ListCategories(ctx context.Context) ([]Category, error)
	// GetCategoryByID resolves one category by hex ObjectId.
	GetCategoryByID(ctx context.Context, id string) (*Category, error)
	// ListTopics lists every topic sorted by title with the nested category.
	ListTopics(ctx context.Context) ([]Topic, error)
	// GetTopicByID resolves one topic by hex ObjectId with the nested
	// category.
	GetTopicByID(ctx context.Context, id string) (*Topic, error)
	// GetQuestionSetByID resolves one question set by hex ObjectId.
	GetQuestionSetByID(
		ctx context.Context, id string,
	) (*QuestionSet, error)
	// ListQuestionSetsByTopicSlug lists question sets for a topic slug
	// in stored order.
	ListQuestionSetsByTopicSlug(
		ctx context.Context, slug string,
	) ([]QuestionSet, error)
}

// AuthoringRepository persists catalog authoring operations.
type AuthoringRepository interface {
	// CreateCategory persists a category.
	CreateCategory(ctx context.Context, request CreateCategoryRequest) (*Category, error)
	// CreateTopic persists a topic.
	CreateTopic(ctx context.Context, request CreateTopicRequest) (*Topic, error)
	// CreateQuestionSet persists a question set.
	CreateQuestionSet(ctx context.Context, request CreateQuestionSetRequest) (*QuestionSet, error)
	// UpdateQuestionSet applies a question-set patch.
	UpdateQuestionSet(ctx context.Context, id string, request UpdateQuestionSetRequest) (*QuestionSet, error)
	// DeleteCategory removes an unreferenced category.
	DeleteCategory(ctx context.Context, id string) error
	// DeleteTopic removes an unreferenced topic.
	DeleteTopic(ctx context.Context, id string) error
	// DeleteQuestionSet removes an unreferenced question set.
	DeleteQuestionSet(ctx context.Context, id string) error
	// SelectQuestionSet resolves one exact question-set selection.
	SelectQuestionSet(ctx context.Context, topic string, level int64, setType string) (*QuestionSet, error)
}
