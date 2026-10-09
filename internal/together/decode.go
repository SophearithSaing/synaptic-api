package together

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/inference"
)

// decodeGeneratedQuestion strictly decodes and validates a generated question.
func decodeGeneratedQuestion(
	output string,
	request inference.GenerationRequest,
) (catalog.Question, error) {
	var response generatedQuestionResponse
	if err := decodeCompletionJSON(output, &response); err != nil {
		return catalog.Question{}, err
	}
	if response.Question == nil {
		return catalog.Question{}, fmt.Errorf("missing question")
	}
	question, err := response.Question.question()
	if err != nil {
		return catalog.Question{}, err
	}
	if question.Type != request.QuestionType {
		return catalog.Question{}, fmt.Errorf("generated question type %q", question.Type)
	}
	if err := validateGeneratedQuestion(question); err != nil {
		return catalog.Question{}, err
	}
	return inference.CanonicalizeGeneratedQuestion(question, request)
}

// decodeWrittenEvaluations strictly decodes and validates written evaluations.
func decodeWrittenEvaluations(
	output string,
	request inference.GradeWrittenRequest,
) ([]inference.WrittenEvaluation, error) {
	var response writtenEvaluationResponse
	if err := decodeCompletionJSON(output, &response); err != nil {
		return nil, err
	}
	if response.Evaluations == nil {
		return nil, fmt.Errorf("missing evaluations")
	}
	expected := make(map[string]struct{}, len(request.Answers))
	for _, answer := range request.Answers {
		expected[answer.Question.ID] = struct{}{}
	}
	if len(*response.Evaluations) != len(expected) {
		return nil, fmt.Errorf("incomplete evaluations")
	}
	evaluations := make([]inference.WrittenEvaluation, len(*response.Evaluations))
	for index, evaluation := range *response.Evaluations {
		result, err := evaluation.evaluation()
		if err != nil {
			return nil, err
		}
		if _, exists := expected[result.QuestionID]; !exists {
			return nil, fmt.Errorf("foreign evaluation question ID %q", result.QuestionID)
		}
		delete(expected, result.QuestionID)
		evaluations[index] = result
	}
	if len(expected) != 0 {
		return nil, fmt.Errorf("incomplete evaluations")
	}
	return evaluations, nil
}

// decodeCompletionJSON accepts optional fenced JSON and rejects trailing data.
func decodeCompletionJSON(output string, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader([]byte(unfenceJSON(output))))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("trailing JSON")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// unfenceJSON preserves legacy optional markdown JSON fence compatibility.
func unfenceJSON(output string) string {
	output = strings.TrimSpace(output)
	if strings.HasPrefix(strings.ToLower(output), "```json") {
		output = output[len("```json"):]
	}
	output = strings.TrimPrefix(output, "```")
	output = strings.TrimSuffix(output, "```")
	return strings.TrimSpace(output)
}

// validateGeneratedQuestion checks generated-question semantic constraints.
func validateGeneratedQuestion(question catalog.Question) error {
	if len(question.TargetConcepts) == 0 {
		return fmt.Errorf("question target concepts are required")
	}
	if question.Type != inference.QuestionTypeMCQ {
		return nil
	}
	if len(question.Options) != 3 || question.CorrectOptionID == "" {
		return fmt.Errorf("invalid MCQ options")
	}
	optionIDs := make(map[string]struct{}, len(question.Options))
	correct := false
	for _, option := range question.Options {
		if option.ID == "" || option.Text == "" {
			return fmt.Errorf("invalid MCQ option")
		}
		if _, exists := optionIDs[option.ID]; exists {
			return fmt.Errorf("duplicate MCQ option ID")
		}
		optionIDs[option.ID] = struct{}{}
		correct = correct || option.ID == question.CorrectOptionID
	}
	if !correct {
		return fmt.Errorf("MCQ correct option was not found")
	}
	return nil
}

type generatedQuestionResponse struct {
	Question *generatedQuestion `json:"question"`
}

type generatedQuestion struct {
	ID              *string            `json:"id"`
	Type            *string            `json:"type"`
	Prompt          *string            `json:"prompt"`
	Options         *[]generatedOption `json:"options"`
	CorrectOptionID *string            `json:"correctOptionId"`
	TargetConcepts  *strictStrings     `json:"targetConcepts"`
	Feedback        *generatedFeedback `json:"feedback"`
	Rubrics         *generatedRubrics  `json:"rubrics"`
}

type generatedOption struct {
	ID   *string `json:"id"`
	Text *string `json:"text"`
}

type generatedFeedback struct {
	Correct   *string `json:"correct"`
	Incorrect *string `json:"incorrect"`
}

type generatedRubrics struct {
	KeyPoints      *strictStrings `json:"keyPoints"`
	Misconceptions *strictStrings `json:"misconceptions"`
}

// question converts a strict generated DTO into neutral content.
func (source *generatedQuestion) question() (catalog.Question, error) {
	if source.ID == nil || source.Type == nil || source.Prompt == nil ||
		source.TargetConcepts == nil || source.Feedback == nil || source.Rubrics == nil {
		return catalog.Question{}, fmt.Errorf("missing required question field")
	}
	if source.Feedback.Correct == nil || source.Feedback.Incorrect == nil ||
		source.Rubrics.KeyPoints == nil || source.Rubrics.Misconceptions == nil {
		return catalog.Question{}, fmt.Errorf("missing required question field")
	}
	question := catalog.Question{
		ID: *source.ID, Type: *source.Type, Prompt: *source.Prompt,
		TargetConcepts: []string(*source.TargetConcepts),
		Feedback: catalog.QuestionFeedback{Correct: *source.Feedback.Correct,
			Incorrect: *source.Feedback.Incorrect},
		Rubrics: catalog.QuestionRubric{KeyPoints: []string(*source.Rubrics.KeyPoints),
			Misconceptions: []string(*source.Rubrics.Misconceptions)},
	}
	if source.Options != nil {
		question.Options = make([]catalog.QuestionOption, len(*source.Options))
		for index, option := range *source.Options {
			if option.ID == nil || option.Text == nil {
				return catalog.Question{}, fmt.Errorf("missing option field")
			}
			question.Options[index] = catalog.QuestionOption{ID: *option.ID, Text: *option.Text}
		}
	}
	if source.CorrectOptionID != nil {
		question.CorrectOptionID = *source.CorrectOptionID
	}
	return question, nil
}

type writtenEvaluationResponse struct {
	Evaluations *[]writtenEvaluation `json:"evaluations"`
}

type writtenEvaluation struct {
	QuestionID    *string        `json:"questionId"`
	Score         *float64       `json:"score"`
	CorrectAnswer *string        `json:"correctAnswer"`
	Feedback      *string        `json:"feedback"`
	Strengths     *strictStrings `json:"strengths"`
	Weaknesses    *strictStrings `json:"weaknesses"`
}

// evaluation converts a strict grading DTO into a neutral evaluation.
func (source writtenEvaluation) evaluation() (inference.WrittenEvaluation, error) {
	if source.QuestionID == nil || source.Score == nil ||
		source.CorrectAnswer == nil || source.Feedback == nil ||
		source.Strengths == nil || source.Weaknesses == nil {
		return inference.WrittenEvaluation{}, fmt.Errorf("missing evaluation field")
	}
	if *source.Score < 0 || *source.Score > 1 {
		return inference.WrittenEvaluation{}, fmt.Errorf("evaluation score out of range")
	}
	return inference.WrittenEvaluation{QuestionID: *source.QuestionID,
		Score: *source.Score, CorrectAnswer: *source.CorrectAnswer,
		Feedback: *source.Feedback, Strengths: []string(*source.Strengths),
		Weaknesses: []string(*source.Weaknesses)}, nil
}

// strictStrings rejects null and non-string elements in a required array.
type strictStrings []string

// UnmarshalJSON decodes an array while rejecting null elements.
func (values *strictStrings) UnmarshalJSON(data []byte) error {
	var rawValues []json.RawMessage
	if err := json.Unmarshal(data, &rawValues); err != nil {
		return err
	}
	decoded := make([]string, len(rawValues))
	for index, raw := range rawValues {
		if bytes.Equal(raw, []byte("null")) {
			return fmt.Errorf("null string array member")
		}
		if err := json.Unmarshal(raw, &decoded[index]); err != nil {
			return err
		}
	}
	*values = decoded
	return nil
}
