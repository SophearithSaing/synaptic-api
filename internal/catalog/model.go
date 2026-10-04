// Package catalog implements the protected catalog read paths for
// categories, topics, and question sets.
package catalog

import "time"

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

// QuestionOption is one selectable answer for a question.
type QuestionOption struct {
	ID   string `bson:"id" json:"id"`
	Text string `bson:"text" json:"text"`
}

// QuestionFeedback describes responses to correct and incorrect answers.
type QuestionFeedback struct {
	Correct   string `bson:"correct" json:"correct"`
	Incorrect string `bson:"incorrect" json:"incorrect"`
}

// QuestionRubric describes expected points and common misconceptions.
type QuestionRubric struct {
	KeyPoints      []string `bson:"keyPoints" json:"keyPoints"`
	Misconceptions []string `bson:"misconceptions" json:"misconceptions"`
}

// Question is one multiple-choice or written question.
type Question struct {
	ID              string           `bson:"id" json:"id"`
	Type            string           `bson:"type" json:"type"`
	Prompt          string           `bson:"prompt" json:"prompt"`
	Options         []QuestionOption `bson:"options,omitempty" json:"options,omitempty"`
	CorrectOptionID string           `bson:"correctOptionId,omitempty" json:"correctOptionId,omitempty"`
	TargetConcepts  []string         `bson:"targetConcepts" json:"targetConcepts"`
	Feedback        QuestionFeedback `bson:"feedback" json:"feedback"`
	Rubrics         QuestionRubric   `bson:"rubrics" json:"rubrics"`
}

// QuestionSet is the question-set response shape.
type QuestionSet struct {
	ID        string     `json:"id"`
	TopicID   string     `json:"topicId"`
	Topic     *Topic     `json:"topic"`
	SetType   string     `json:"setType"`
	Level     int64      `json:"level"`
	Questions []Question `json:"questions"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}
