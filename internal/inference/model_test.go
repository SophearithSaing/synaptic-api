package inference

import (
	"errors"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

func TestValidateGenerationRequest(t *testing.T) {
	if err := ValidateGenerationRequest(GenerationRequest{QuestionType: "nope"}); err == nil {
		t.Fatal("ValidateGenerationRequest() error = nil")
	}
	if err := ValidateGenerationRequest(GenerationRequest{QuestionType: QuestionTypeMCQ}); err != nil {
		t.Fatalf("ValidateGenerationRequest() error = %v", err)
	}
}

func TestValidateGradeWrittenRequest(t *testing.T) {
	written := func(id string) WrittenAnswer {
		return WrittenAnswer{Question: catalog.Question{ID: id, Type: QuestionTypeWritten}}
	}
	tests := []struct {
		name    string
		request GradeWrittenRequest
		wantErr bool
	}{
		{"valid", GradeWrittenRequest{Answers: []WrittenAnswer{written("q1")}}, false},
		{"duplicate IDs", GradeWrittenRequest{Answers: []WrittenAnswer{written("q1"), written("q1")}}, true},
		{"MCQ", GradeWrittenRequest{Answers: []WrittenAnswer{{Question: catalog.Question{ID: "q1", Type: QuestionTypeMCQ}}}}, true},
		{"missing ID", GradeWrittenRequest{Answers: []WrittenAnswer{written("")}}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateGradeWrittenRequest(test.request)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, want error %t", err, test.wantErr)
			}
		})
	}
}

func TestCanonicalizeGeneratedQuestion(t *testing.T) {
	question, err := CanonicalizeGeneratedQuestion(catalog.Question{
		Type: QuestionTypeMCQ, CorrectOptionID: "o2",
		Options: []catalog.QuestionOption{{ID: "o1"}, {ID: "o2"}},
	}, GenerationRequest{TopicSlug: "automata", Level: 2, QuestionNumber: 3})
	if err != nil {
		t.Fatalf("CanonicalizeGeneratedQuestion() error = %v", err)
	}
	if question.ID != "automata-l2-q3" || question.Options[0].ID != "automata-l2-q3-o1" || question.CorrectOptionID != "automata-l2-q3-o2" {
		t.Fatalf("canonical question = %#v", question)
	}
}

func TestErrorUnwrapsCause(t *testing.T) {
	cause := errors.New("cancelled")
	err := &Error{Kind: ErrorUpstream, Cause: cause}
	if !errors.Is(err, cause) {
		t.Fatal("Error did not unwrap cause")
	}
}
