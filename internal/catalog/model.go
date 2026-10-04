// Package catalog implements the protected catalog read paths for
// categories, topics, and question sets.
package catalog

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrInvalidObjectID     = errors.New("Invalid MongoDB ObjectId")
	ErrCategoryNotFound    = errors.New("Category not found")
	ErrTopicNotFound       = errors.New("Topic not found")
	ErrQuestionSetNotFound = errors.New("Question set not found")
)

// Category is the pinned category response shape.
type Category struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// Topic is the pinned topic response shape with the nested category.
type Topic struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Tags        []string  `json:"tags"`
	CategoryID  string    `json:"categoryId"`
	Category    *Category `json:"category"`
}

// QuestionSet is the question-set response shape. Questions are the
// stored embedded documents passed through raw.
type QuestionSet struct {
	ID        string          `json:"id"`
	TopicID   string          `json:"topicId"`
	Topic     *Topic          `json:"topic"`
	SetType   string          `json:"setType"`
	Level     int64           `json:"level"`
	Questions json.RawMessage `json:"questions"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}
