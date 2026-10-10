package catalog

import "time"

// topicResponse is the public topic boundary. Internal references stay out of
// HTTP responses.
type topicResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Tags        []string  `json:"tags"`
	Category    *Category `json:"category"`
}

// questionSetResponse is the public question-set boundary.
type questionSetResponse struct {
	ID        string     `json:"id"`
	Topic     any        `json:"topic"`
	SetType   string     `json:"setType"`
	Level     int64      `json:"level"`
	Questions []Question `json:"questions"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// populatedTopicResponse is the legacy populated topic response boundary.
type populatedTopicResponse struct {
	ID          string    `json:"_id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Tags        []string  `json:"tags"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Version     int       `json:"__v"`
}

// topicHTTP maps a topic to its standard public response boundary.
func topicHTTP(topic Topic) topicResponse {
	return topicResponse{ID: topic.ID, Title: topic.Title, Slug: topic.Slug,
		Description: topic.Description, Icon: topic.Icon, Tags: topic.Tags,
		Category: topic.Category}
}

// populatedTopicHTTP maps a topic to the question-set populated boundary.
func populatedTopicHTTP(topic Topic) populatedTopicResponse {
	return populatedTopicResponse{ID: topic.ID, Title: topic.Title,
		Slug: topic.Slug, Description: topic.Description, Icon: topic.Icon,
		Tags: topic.Tags, Category: topic.CategoryID,
		CreatedAt: topic.CreatedAt, UpdatedAt: topic.UpdatedAt,
		Version: topic.Version}
}

// questionSetHTTP maps a question set to its public response boundary.
func questionSetHTTP(questionSet QuestionSet, populated bool) questionSetResponse {
	topic := any(questionSet.TopicID)
	if populated && questionSet.Topic != nil {
		topic = populatedTopicHTTP(*questionSet.Topic)
	}
	if populated && questionSet.Topic == nil {
		topic = nil
	}
	return questionSetResponse{ID: questionSet.ID, Topic: topic,
		SetType: questionSet.SetType, Level: questionSet.Level,
		Questions: questionSet.Questions, CreatedAt: questionSet.CreatedAt,
		UpdatedAt: questionSet.UpdatedAt}
}
