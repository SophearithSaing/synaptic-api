package together

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/inference"
)

func TestGenerationPrompts(t *testing.T) {
	initial := generationRequest()
	recent := generationRequest()
	recent.QuestionNumber = 2
	recent.RecentAcceptedQuestions = []inference.RecentQuestion{{
		Level: 1, Prompt: "Define a DFA.", TargetConcepts: []string{"dfa"},
	}}
	rejected := generationRequest()
	rejected.QuestionNumber = 3
	rejected.RejectedQuestion = &catalog.Question{
		ID: "automata-l2-q2", Type: inference.QuestionTypeMCQ, Prompt: "Old?",
		Options: []catalog.QuestionOption{}, TargetConcepts: []string{},
		Feedback: catalog.QuestionFeedback{},
		Rubrics:  catalog.QuestionRubric{KeyPoints: []string{}, Misconceptions: []string{}},
	}
	rejected.RejectionReason = "Too similar"

	tests := []struct {
		name    string
		request inference.GenerationRequest
	}{
		{"initial", initial}, {"recent", recent}, {"rejected", rejected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CreateGenerationUserPrompt(test.request)
			if err != nil {
				t.Fatalf("CreateGenerationUserPrompt() error = %v", err)
			}
			want := golden(t, "generation-"+test.name+".golden")
			if got != want {
				t.Errorf("prompt = %s\nwant %s", got, want)
			}
		})
	}
}

func TestWrittenGradingPrompt(t *testing.T) {
	request := inference.GradeWrittenRequest{Answers: []inference.WrittenAnswer{{
		Question: catalog.Question{ID: "q1", Type: inference.QuestionTypeWritten,
			Prompt: "Explain a DFA.", TargetConcepts: []string{"dfa"},
			Rubrics: catalog.QuestionRubric{KeyPoints: []string{"states"}, Misconceptions: []string{}},
		}, StudentAnswer: "It has states.",
	}}}
	got, err := CreateWrittenGradingUserPrompt(request)
	if err != nil {
		t.Fatalf("CreateWrittenGradingUserPrompt() error = %v", err)
	}
	if want := golden(t, "grading.golden"); got != want {
		t.Errorf("prompt = %s\nwant %s", got, want)
	}
}

func TestTogetherRequestConstantsAndSchemas(t *testing.T) {
	if inference.Model != "openai/gpt-oss-120b" || GenerationTemperature != 0.7 || GradingTemperature != 0 {
		t.Fatal("legacy model or temperatures changed")
	}
	for _, format := range []JSONSchemaResponseFormat{GeneratedQuestionResponseFormat, WrittenEvaluationResponseFormat} {
		if !format.JSONSchema.Strict {
			t.Fatalf("schema %q is not strict", format.JSONSchema.Name)
		}
		encoded, err := json.Marshal(format)
		if err != nil {
			t.Fatalf("Marshal(%q): %v", format.JSONSchema.Name, err)
		}
		if len(encoded) == 0 {
			t.Fatal("empty schema")
		}
	}
}

func generationRequest() inference.GenerationRequest {
	return inference.GenerationRequest{TopicSlug: "automata", TopicTitle: "Automata", TopicDescription: "Finite-state machines.", TopicTags: []string{}, Level: 2, QuestionNumber: 1, QuestionType: inference.QuestionTypeMCQ, RecentAcceptedQuestions: []inference.RecentQuestion{}}
}

func golden(t *testing.T, name string) string {
	t.Helper()
	value, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(string(value), "\n")
}
