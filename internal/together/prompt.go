// Package together implements Together-specific request formatting.
package together

import (
	"encoding/json"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/inference"
)

const (
	// GenerationTemperature preserves the legacy question generation temperature.
	GenerationTemperature = 0.7
	// GradingTemperature preserves the legacy written-answer grading temperature.
	GradingTemperature = 0

	// QuestionGenerationSystemPrompt instructs the model to produce one question.
	QuestionGenerationSystemPrompt = "You generate one computing theory learning question for a live session. Return exactly one question matching the requested type. Keep prompts clear and concise. MCQ questions must include three options, one correctOptionId, target concepts, feedback, and rubrics. Written questions must include target concepts, feedback, and rubrics, and may omit options and correctOptionId."

	// WrittenEvaluationSystemPrompt instructs the model to evaluate answers.
	WrittenEvaluationSystemPrompt = "You are evaluating written answers for a computing theory learning session. Score each answer from 0 to 1 based on how well it covers the expected key points while avoiding listed misconceptions. Return one evaluation for every submitted answer, using the provided questionId for each result. Include a concise correctAnswer for each question. Keep feedback concise and helpful, and use short concept labels for strengths and weaknesses."
)

// CreateGenerationUserPrompt creates the legacy-compatible generation prompt.
func CreateGenerationUserPrompt(request inference.GenerationRequest) (string, error) {
	instructions := []string{
		"Generate exactly one question for the questionType.",
		"Avoid repeats or close paraphrases in recentAcceptedQuestions.",
		"Use recentAcceptedQuestions targetConcepts to vary focus.",
		"Use short option IDs such as o1, o2, and o3 for MCQ questions.",
	}
	if request.RejectedQuestion != nil && request.RejectionReason != "" {
		instructions = append(instructions,
			"Use rejectedQuestion and rejectionReason to avoid the same issue.")
	}

	prompt := struct {
		TopicSlug               string                     `json:"topicSlug"`
		TopicTitle              string                     `json:"topicTitle"`
		TopicDescription        string                     `json:"topicDescription"`
		TopicTags               []string                   `json:"topicTags"`
		Level                   int64                      `json:"level"`
		QuestionNumber          int64                      `json:"questionNumber"`
		QuestionType            string                     `json:"questionType"`
		RecentAcceptedQuestions []inference.RecentQuestion `json:"recentAcceptedQuestions"`
		RejectedQuestion        *catalog.Question          `json:"rejectedQuestion,omitempty"`
		RejectionReason         string                     `json:"rejectionReason,omitempty"`
		Instructions            []string                   `json:"instructions"`
	}{
		TopicSlug: request.TopicSlug, TopicTitle: request.TopicTitle,
		TopicDescription: request.TopicDescription, TopicTags: stringSlice(request.TopicTags),
		Level: request.Level, QuestionNumber: request.QuestionNumber,
		QuestionType:            request.QuestionType,
		RecentAcceptedQuestions: recentQuestions(request.RecentAcceptedQuestions),
		RejectedQuestion:        request.RejectedQuestion,
		RejectionReason:         request.RejectionReason, Instructions: instructions,
	}
	value, err := json.Marshal(prompt)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

// CreateWrittenGradingUserPrompt creates the legacy-compatible grading prompt.
func CreateWrittenGradingUserPrompt(request inference.GradeWrittenRequest) (string, error) {
	answers := make([]struct {
		QuestionID     string   `json:"questionId"`
		Prompt         string   `json:"prompt"`
		TargetConcepts []string `json:"targetConcepts"`
		KeyPoints      []string `json:"keyPoints"`
		Misconceptions []string `json:"misconceptions"`
		StudentAnswer  string   `json:"studentAnswer"`
	}, len(request.Answers))
	for index, answer := range request.Answers {
		answers[index].QuestionID = answer.Question.ID
		answers[index].Prompt = answer.Question.Prompt
		answers[index].TargetConcepts = stringSlice(answer.Question.TargetConcepts)
		answers[index].KeyPoints = stringSlice(answer.Question.Rubrics.KeyPoints)
		answers[index].Misconceptions = stringSlice(answer.Question.Rubrics.Misconceptions)
		answers[index].StudentAnswer = answer.StudentAnswer
	}
	value, err := json.Marshal(struct {
		Answers any `json:"answers"`
	}{Answers: answers})
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func stringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func recentQuestions(values []inference.RecentQuestion) []inference.RecentQuestion {
	if values == nil {
		return []inference.RecentQuestion{}
	}
	return values
}
