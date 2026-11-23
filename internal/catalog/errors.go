package catalog

import "errors"

var (
	ErrInvalidObjectID       = errors.New("Invalid MongoDB ObjectId")
	ErrCategoryNotFound      = errors.New("Category not found")
	ErrTopicNotFound         = errors.New("Topic not found")
	ErrQuestionSetNotFound   = errors.New("Question set not found")
	ErrCategoryReferenced    = errors.New("Category is referenced")
	ErrTopicReferenced       = errors.New("Topic is referenced")
	ErrQuestionSetReferenced = errors.New("Question set is referenced")
)
