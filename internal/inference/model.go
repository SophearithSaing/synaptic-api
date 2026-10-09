// Package inference defines provider-neutral AI inference contracts.
package inference

import (
	"context"
	"fmt"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

const (
	// QuestionTypeMCQ identifies multiple-choice questions.
	QuestionTypeMCQ = "mcq"
	// QuestionTypeWritten identifies written questions.
	QuestionTypeWritten = "written"
)

// CompletionMetadata preserves the source material required to audit a call.
type CompletionMetadata struct {
	Model      string
	UserPrompt string
	RawOutput  string
}

// Provider generates questions and grades written answers.
type Provider interface {
	GenerateQuestion(context.Context, GenerationRequest) (GenerationResult, error)
	GradeWritten(context.Context, GradeWrittenRequest) (GradeWrittenResult, error)
}

// RecentQuestion is compact prior-question context for generation.
type RecentQuestion struct {
	Level          int64    `json:"level"`
	Prompt         string   `json:"prompt"`
	TargetConcepts []string `json:"targetConcepts"`
}

// GenerationRequest supplies all context needed to generate one question.
type GenerationRequest struct {
	TopicSlug               string
	TopicTitle              string
	TopicDescription        string
	TopicTags               []string
	Level                   int64
	QuestionNumber          int64
	QuestionType            string
	RecentAcceptedQuestions []RecentQuestion
	RejectedQuestion        *catalog.Question
	RejectionReason         string
}

// GenerationResult is one generated question and its completion metadata.
type GenerationResult struct {
	Question   catalog.Question
	Completion CompletionMetadata
}

// WrittenAnswer supplies the question and answer context needed for grading.
type WrittenAnswer struct {
	Question      catalog.Question
	StudentAnswer string
}

// GradeWrittenRequest supplies every answer in one batch grading call.
type GradeWrittenRequest struct {
	Answers []WrittenAnswer
}

// WrittenEvaluation is one AI evaluation for a written answer.
type WrittenEvaluation struct {
	QuestionID    string
	Score         float64
	CorrectAnswer string
	Feedback      string
	Strengths     []string
	Weaknesses    []string
}

// GradeWrittenResult contains all evaluations and completion metadata.
type GradeWrittenResult struct {
	Evaluations []WrittenEvaluation
	Completion  CompletionMetadata
}

// ErrorKind identifies provider failures without coupling them to HTTP.
type ErrorKind string

const (
	// ErrorUnavailable indicates an unavailable or unconfigured provider.
	ErrorUnavailable ErrorKind = "unavailable"
	// ErrorInvalidResponse indicates a malformed provider response.
	ErrorInvalidResponse ErrorKind = "invalid_response"
	// ErrorEmptyResponse indicates a provider response without content.
	ErrorEmptyResponse ErrorKind = "empty_response"
	// ErrorUpstream indicates an upstream or transport failure.
	ErrorUpstream ErrorKind = "upstream"
)

// Error is a metadata-preserving provider error.
type Error struct {
	Kind       ErrorKind
	Message    string
	Completion CompletionMetadata
	Cause      error
}

// Error returns the normalized failure message.
func (err *Error) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return string(err.Kind)
}

// Unwrap exposes cancellation and other underlying causes to errors.Is.
func (err *Error) Unwrap() error {
	return err.Cause
}

// ValidateGenerationRequest rejects unsupported generated-question requests.
func ValidateGenerationRequest(request GenerationRequest) error {
	if request.QuestionType != QuestionTypeMCQ &&
		request.QuestionType != QuestionTypeWritten {
		return fmt.Errorf("unsupported question type %q", request.QuestionType)
	}
	return nil
}

// ValidateGradeWrittenRequest rejects invalid or ambiguous grading inputs.
func ValidateGradeWrittenRequest(request GradeWrittenRequest) error {
	questionIDs := make(map[string]struct{}, len(request.Answers))
	for _, answer := range request.Answers {
		if answer.Question.Type != QuestionTypeWritten {
			return fmt.Errorf("question %q is not written", answer.Question.ID)
		}
		if answer.Question.ID == "" {
			return fmt.Errorf("written question ID is required")
		}
		if _, exists := questionIDs[answer.Question.ID]; exists {
			return fmt.Errorf("duplicate written question ID %q", answer.Question.ID)
		}
		questionIDs[answer.Question.ID] = struct{}{}
	}
	return nil
}

// CanonicalizeGeneratedQuestion applies legacy deterministic question IDs.
func CanonicalizeGeneratedQuestion(
	question catalog.Question,
	request GenerationRequest,
) (catalog.Question, error) {
	questionID := fmt.Sprintf(
		"%s-l%d-q%d", request.TopicSlug, request.Level, request.QuestionNumber,
	)
	question.ID = questionID
	if question.Type != QuestionTypeMCQ {
		return question, nil
	}

	correctIndex := -1
	options := make([]catalog.QuestionOption, len(question.Options))
	for index, option := range question.Options {
		if option.ID == question.CorrectOptionID {
			correctIndex = index
		}
		option.ID = fmt.Sprintf("%s-o%d", questionID, index+1)
		options[index] = option
	}
	if correctIndex < 0 {
		return catalog.Question{}, fmt.Errorf("generated correct option was invalid")
	}
	question.Options = options
	question.CorrectOptionID = options[correctIndex].ID
	return question, nil
}
