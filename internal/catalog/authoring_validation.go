package catalog

import (
	"fmt"

	"github.com/SophearithSaing/synaptic-api/internal/validation"
)

// AuthoringValidator validates catalog authoring requests and question
// content invariants.
type AuthoringValidator struct {
	validator *validation.Validator
}

// NewAuthoringValidator builds an AuthoringValidator.
func NewAuthoringValidator() (*AuthoringValidator, error) {
	validator, err := validation.New()
	if err != nil {
		return nil, fmt.Errorf("new request validator: %w", err)
	}

	return &AuthoringValidator{validator: validator}, nil
}

// ValidateCreateCategory validates a category create request.
func (v *AuthoringValidator) ValidateCreateCategory(
	request CreateCategoryRequest,
) []string {
	return categoryMessages(request, v.validator.Validate(request))
}

// ValidateCreateTopic validates a topic create request.
func (v *AuthoringValidator) ValidateCreateTopic(
	request CreateTopicRequest,
) []string {
	messages := v.validator.Validate(request)
	return replaceMessage(messages, "tags must contain at most 2 items",
		"tags must contain no more than 2 elements")
}

// ValidateCreateQuestionSet validates a question-set create request.
func (v *AuthoringValidator) ValidateCreateQuestionSet(
	request CreateQuestionSetRequest,
) []string {
	messages := v.validator.Validate(request)
	if request.Questions == nil {
		messages = replaceMessage(messages, "questions is a required field",
			"questions should not be empty", "questions must be an array")
	}
	return append(messages, ValidateQuestions(request.Questions)...)
}

// categoryMessages preserves the legacy fixture messages for missing strings.
func categoryMessages(
	request CreateCategoryRequest,
	messages []string,
) []string {
	fields := []struct {
		name  string
		value string
	}{
		{"title", request.Title},
		{"slug", request.Slug},
		{"description", request.Description},
		{"icon", request.Icon},
	}
	for _, field := range fields {
		if field.value != "" {
			continue
		}
		messages = replaceMessage(messages,
			field.name+" is a required field",
			field.name+" should not be empty",
			field.name+" must be a string")
	}

	return messages
}

// replaceMessage replaces one validation message without changing the order of
// the remaining messages.
func replaceMessage(messages []string, old string, replacements ...string) []string {
	for index, message := range messages {
		if message != old {
			continue
		}
		result := make([]string, 0, len(messages)+len(replacements)-1)
		result = append(result, messages[:index]...)
		result = append(result, replacements...)
		return append(result, messages[index+1:]...)
	}

	return messages
}

// ValidateUpdateQuestionSet validates a question-set patch request.
func (v *AuthoringValidator) ValidateUpdateQuestionSet(
	request UpdateQuestionSetRequest,
) []string {
	messages := v.validator.Validate(request)
	if request.Questions != nil {
		messages = append(messages, ValidateQuestions(*request.Questions)...)
	}

	return messages
}

// ValidateBulkUpdateQuestionSet validates one bulk patch item.
func (v *AuthoringValidator) ValidateBulkUpdateQuestionSet(
	request BulkUpdateQuestionSetRequest,
) []string {
	messages := v.validator.Validate(request)
	if request.Questions != nil {
		messages = append(messages, ValidateQuestions(*request.Questions)...)
	}

	return messages
}

// ValidateQuestions validates question content that depends on sibling
// fields. Structural validation remains in the request DTO tags.
func ValidateQuestions(questions []Question) []string {
	messages := make([]string, 0)
	questionIDs := make(map[string]struct{}, len(questions))

	for _, question := range questions {
		if question.ID != "" {
			if _, exists := questionIDs[question.ID]; exists {
				messages = append(messages, "question IDs must be unique")
			} else {
				questionIDs[question.ID] = struct{}{}
			}
		}

		switch question.Type {
		case "mcq":
			messages = append(messages, validateMCQ(question)...)
		case "written":
			if len(question.Options) != 0 || question.CorrectOptionID != "" {
				messages = append(messages,
					"written questions must not include options or correctOptionId",
				)
			}
		}
	}

	return messages
}

// validateMCQ validates MCQ-specific option invariants.
func validateMCQ(question Question) []string {
	messages := make([]string, 0)
	if len(question.Options) == 0 {
		messages = append(messages, "mcq questions must include options")
	}

	optionIDs := make(map[string]struct{}, len(question.Options))
	for _, option := range question.Options {
		if option.ID == "" {
			continue
		}
		if _, exists := optionIDs[option.ID]; exists {
			messages = append(messages, "option IDs must be unique")
			continue
		}
		optionIDs[option.ID] = struct{}{}
	}

	if question.CorrectOptionID == "" {
		return append(messages, "mcq questions must include correctOptionId")
	}
	if _, exists := optionIDs[question.CorrectOptionID]; !exists {
		messages = append(messages,
			"correctOptionId must identify an option",
		)
	}

	return messages
}
