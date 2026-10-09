package catalog

// CreateCategoryRequest is the input for creating a category.
type CreateCategoryRequest struct {
	Title       string `json:"title" validate:"required,min=1"`
	Slug        string `json:"slug" validate:"required,min=1"`
	Description string `json:"description" validate:"required,min=1"`
	Icon        string `json:"icon" validate:"required,min=1"`
}

// CreateTopicRequest is the input for creating a topic.
type CreateTopicRequest struct {
	Title       string   `json:"title" validate:"required,min=1"`
	Slug        string   `json:"slug" validate:"required,min=1"`
	Description string   `json:"description" validate:"required,min=1"`
	Icon        string   `json:"icon" validate:"required,min=1"`
	Tags        []string `json:"tags" validate:"required,min=1,max=2,dive,required"`
	Category    string   `json:"category" validate:"required,min=1"`
}

// CreateQuestionSetRequest is the input for creating one question set.
type CreateQuestionSetRequest struct {
	Topic     string     `json:"topic" validate:"required,min=1"`
	SetType   string     `json:"setType" validate:"required,oneof=regular live"`
	Level     int64      `json:"level" validate:"gte=0"`
	Questions []Question `json:"questions" validate:"required,min=1,dive"`
}

// UpdateQuestionSetRequest is the input for updating one question set.
// Pointers preserve whether the client omitted a field or supplied its zero
// value.
type UpdateQuestionSetRequest struct {
	Topic     *string     `json:"topic,omitempty" validate:"omitempty,min=1"`
	SetType   *string     `json:"setType,omitempty" validate:"omitempty,oneof=regular live"`
	Level     *int64      `json:"level,omitempty" validate:"omitempty,gte=0"`
	Questions *[]Question `json:"questions,omitempty" validate:"omitempty,min=1,dive"`
}

// BulkUpdateQuestionSetRequest is one item in a bulk question-set update.
type BulkUpdateQuestionSetRequest struct {
	ID string `json:"id" validate:"required,min=1"`
	UpdateQuestionSetRequest
}
