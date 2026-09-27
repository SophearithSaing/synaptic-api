// Package catalog implements the protected catalog read paths for
// categories, topics, and question sets, preserving the fixture-pinned
// HTTP shapes including legacy BSON toleration.
package catalog

import (
	"encoding/json"
	"time"
)

// Error is the catalog sentinel failure set.
type Error string

// Errors mapped to pinned HTTP bodies.
const (
	// ErrCategoryNotFound marks a missing category document.
	ErrCategoryNotFound Error = "Category not found"
	// ErrTopicNotFound marks a missing topic document.
	ErrTopicNotFound Error = "Topic not found"
	// ErrQuestionSetNotFound marks a missing question set document.
	ErrQuestionSetNotFound Error = "Question set not found"
)

// Error renders the sentinel message.
func (e Error) Error() string {
	return string(e)
}

// Category is the pinned category response shape.
type Category struct {
	// ID is the hex ObjectId.
	ID string `json:"id"`
	// Title is the display name.
	Title string `json:"title"`
	// Slug is the unique URL key.
	Slug string `json:"slug"`
	// Description is the summary text.
	Description string `json:"description"`
	// Icon is the visual key.
	Icon string `json:"icon"`
}

// Topic is the pinned topic response shape with the nested category.
type Topic struct {
	// ID is the hex ObjectId.
	ID string `json:"id"`
	// Title is the display name.
	Title string `json:"title"`
	// Slug is the unique URL key.
	Slug string `json:"slug"`
	// Description is the summary text.
	Description string `json:"description"`
	// Icon is the visual key.
	Icon string `json:"icon"`
	// Tags are the topic labels.
	Tags []string `json:"tags"`
	// Category embeds the populated category. A nil value serializes to
	// null only for unresolved legacy references.
	Category *Category `json:"category"`
}

// QuestionSet is the pinned question-set response shape. Questions are
// the stored embedded documents passed through raw.
type QuestionSet struct {
	// ID is the hex ObjectId.
	ID string `json:"id"`
	// Topic is the stored topic reference: a hex ObjectId string, or the
	// raw populated topic document when populated.
	Topic any `json:"topic"`
	// SetType is the stored set type, including legacy values outside
	// the regular/live enum.
	SetType string `json:"setType"`
	// Level is the numeric difficulty.
	Level int64 `json:"level"`
	// Questions holds the stored embedded questions as JSON in field
	// order.
	Questions json.RawMessage `json:"questions"`
	// CreatedAt is the stored creation timestamp in the pinned ISO-8601
	// millisecond shape.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is the stored update timestamp in the pinned ISO-8601
	// millisecond shape.
	UpdatedAt string `json:"updatedAt"`
}

// ISO8601 renders a stored instant the way a JS Date serializes:
// millisecond precision in UTC.
func ISO8601(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000") + "Z"
}
