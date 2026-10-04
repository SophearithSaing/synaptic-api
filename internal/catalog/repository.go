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
