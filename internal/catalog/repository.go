package catalog

import (
	"context"
)

// Repository resolves catalog documents for the read routes.
// Implementations preserve stored shapes and tolerate legacy BSON.
type Repository interface {
	// Categories lists every category sorted by title.
	Categories(ctx context.Context) ([]Category, error)
	// CategoryByID resolves one category by hex ObjectId.
	CategoryByID(ctx context.Context, id string) (*Category, error)
	// Topics lists every topic sorted by title with the nested category.
	// Legacy string-typed category references stay unresolved: their
	// nested category is nil.
	Topics(ctx context.Context) ([]Topic, error)
	// TopicByID resolves one topic by hex ObjectId with the nested
	// category.
	TopicByID(ctx context.Context, id string) (*Topic, error)
	// QuestionSetByID resolves one question set by hex ObjectId.
	QuestionSetByID(
		ctx context.Context, id string,
	) (*QuestionSet, error)
	// QuestionSetsByTopicSlug lists the question sets for a topic slug
	// in stored order.
	QuestionSetsByTopicSlug(
		ctx context.Context, slug string,
	) ([]QuestionSet, error)
}
