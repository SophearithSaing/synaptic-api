package together

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/inference"
)

func TestClientGenerateQuestionSendsTogetherPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		var payload completionRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Model != Model || payload.Temperature != GenerationTemperature ||
			payload.ResponseFormat.JSONSchema.Name != "generated_live_question" {
			t.Fatalf("payload = %#v", payload)
		}
		writeCompletion(writer, generatedQuestionJSON("mcq"))
	}))
	defer server.Close()

	result, err := NewClient(Config{APIKey: "test-key", Endpoint: server.URL}).GenerateQuestion(
		context.Background(), transportGenerationRequest(),
	)
	if err != nil {
		t.Fatalf("GenerateQuestion() error = %v", err)
	}
	if result.Question.ID != "automata-l2-q1" ||
		result.Question.CorrectOptionID != "automata-l2-q1-o2" {
		t.Fatalf("Question = %#v", result.Question)
	}
	if result.Completion.Model != Model || result.Completion.RawOutput == "" {
		t.Fatalf("Completion = %#v", result.Completion)
	}
}

func TestClientGradeWrittenSendsTogetherPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload completionRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Temperature != GradingTemperature ||
			payload.ResponseFormat.JSONSchema.Name != "written_answer_evaluations" {
			t.Fatalf("payload = %#v", payload)
		}
		writeCompletion(writer, `{"evaluations":[{"questionId":"q1","score":0.5,"correctAnswer":"answer","feedback":"good","strengths":[],"weaknesses":[]}]}`)
	}))
	defer server.Close()

	result, err := NewClient(Config{APIKey: "test-key", Endpoint: server.URL}).GradeWritten(
		context.Background(), gradingRequest(),
	)
	if err != nil || len(result.Evaluations) != 1 || result.Evaluations[0].Score != 0.5 {
		t.Fatalf("GradeWritten() = %#v, %v", result, err)
	}
}

func TestClientRetriesOnlyTransientStatuses(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if calls.Add(1) == 1 {
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writeCompletion(writer, generatedQuestionJSON("mcq"))
	}))
	defer server.Close()
	client := NewClient(Config{APIKey: "key", Endpoint: server.URL, MaxRetries: 1,
		Backoff: func(int) time.Duration { return 0 }})
	if _, err := client.GenerateQuestion(context.Background(), transportGenerationRequest()); err != nil {
		t.Fatalf("GenerateQuestion() error = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}

	calls.Store(0)
	server.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		writer.WriteHeader(http.StatusUnauthorized)
	})
	_, err := client.GenerateQuestion(context.Background(), transportGenerationRequest())
	if !hasErrorKind(err, inference.ErrorUpstream) || calls.Load() != 1 {
		t.Fatalf("error = %v, calls = %d", err, calls.Load())
	}
}

func TestClientRetriesServerFailuresAndHonorsBackoffCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := NewClient(Config{APIKey: "key", Endpoint: server.URL, MaxRetries: 1,
		Backoff: func(int) time.Duration { return 0 }})
	_, err := client.GenerateQuestion(context.Background(), transportGenerationRequest())
	if !hasErrorKind(err, inference.ErrorUpstream) || calls.Load() != 2 {
		t.Fatalf("error = %v, calls = %d", err, calls.Load())
	}

	started := make(chan struct{}, 1)
	server.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started <- struct{}{}
		writer.WriteHeader(http.StatusServiceUnavailable)
	})
	client = NewClient(Config{APIKey: "key", Endpoint: server.URL, MaxRetries: 2,
		Backoff: func(int) time.Duration { return time.Hour }})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := client.GenerateQuestion(ctx, transportGenerationRequest()); result <- err }()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestClientPreservesCancellationAndBoundsResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/slow" {
			time.Sleep(100 * time.Millisecond)
			return
		}
		_, _ = writer.Write(make([]byte, 32))
	}))
	defer server.Close()
	client := NewClient(Config{APIKey: "key", Endpoint: server.URL + "/slow",
		Timeout: time.Millisecond})
	_, err := client.GenerateQuestion(context.Background(), transportGenerationRequest())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v", err)
	}

	client = NewClient(Config{APIKey: "key", Endpoint: server.URL, MaxBodyBytes: 16})
	_, err = client.GenerateQuestion(context.Background(), transportGenerationRequest())
	if !hasErrorKind(err, inference.ErrorUpstream) {
		t.Fatalf("oversized error = %v", err)
	}
}

func TestClientResponseFailuresRetainMetadata(t *testing.T) {
	tests := []struct {
		name string
		body string
		kind inference.ErrorKind
	}{
		{"empty", `{"choices":[{"message":{"content":""}}]}`, inference.ErrorEmptyResponse},
		{"invalid", `{"choices":[{"message":{"content":"not json"}}]}`, inference.ErrorInvalidResponse},
		{"fenced", completionEnvelope("```json\n" + generatedQuestionJSON("mcq") + "\n```"), ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			result, err := NewClient(Config{APIKey: "key", Endpoint: server.URL}).GenerateQuestion(context.Background(), transportGenerationRequest())
			if test.kind == "" {
				if err != nil || result.Question.ID == "" {
					t.Fatalf("result = %#v, error = %v", result, err)
				}
				return
			}
			if !hasErrorKind(err, test.kind) {
				t.Fatalf("error = %v", err)
			}
			var providerErr *inference.Error
			if !errors.As(err, &providerErr) ||
				providerErr.Completion.Model == "" ||
				providerErr.Completion.UserPrompt == "" {
				t.Fatalf("metadata = %#v", providerErr)
			}
		})
	}
}

func TestDecodeWrittenEvaluationsRejectsInvalidResults(t *testing.T) {
	request := gradingRequest()
	for _, output := range []string{
		`{"evaluations":[]}`,
		`{"evaluations":[{"questionId":"q1","score":2,"correctAnswer":"a","feedback":"f","strengths":[],"weaknesses":[]}]}`,
		`{"evaluations":[{"questionId":"other","score":0,"correctAnswer":"a","feedback":"f","strengths":[],"weaknesses":[]}]}`,
		`{"evaluations":[{"questionId":"q1","score":0,"correctAnswer":"a","feedback":"f","strengths":[],"weaknesses":[],"extra":true}]}`,
	} {
		if _, err := decodeWrittenEvaluations(output, request); err == nil {
			t.Fatalf("output accepted: %s", output)
		}
	}
}

func TestDecodeGeneratedQuestionRejectsInvalidMCQ(t *testing.T) {
	for _, output := range []string{
		`{"question":{"id":"x","type":"written","prompt":"p","targetConcepts":[],"feedback":{"correct":"","incorrect":""},"rubrics":{"keyPoints":[],"misconceptions":[]}}}`,
		`{"question":{"id":"x","type":"mcq","prompt":"p","targetConcepts":[],"feedback":{"correct":"","incorrect":""},"rubrics":{"keyPoints":[],"misconceptions":[]}}}`,
		generatedQuestionJSON("mcq") + ` trailing`,
	} {
		if _, err := decodeGeneratedQuestion(output, transportGenerationRequest()); err == nil {
			t.Fatalf("output accepted: %s", output)
		}
	}
}

func transportGenerationRequest() inference.GenerationRequest {
	return inference.GenerationRequest{TopicSlug: "automata", Level: 2, QuestionNumber: 1,
		QuestionType: inference.QuestionTypeMCQ, TopicTags: []string{}, RecentAcceptedQuestions: []inference.RecentQuestion{}}
}

func gradingRequest() inference.GradeWrittenRequest {
	return inference.GradeWrittenRequest{Answers: []inference.WrittenAnswer{{
		Question: catalog.Question{ID: "q1", Type: inference.QuestionTypeWritten},
	}}}
}

func generatedQuestionJSON(questionType string) string {
	return `{"question":{"id":"temporary","type":"` + questionType + `","prompt":"What is a DFA?","options":[{"id":"o1","text":"one"},{"id":"o2","text":"two"},{"id":"o3","text":"three"}],"correctOptionId":"o2","targetConcepts":["dfa"],"feedback":{"correct":"yes","incorrect":"no"},"rubrics":{"keyPoints":[],"misconceptions":[]}}}`
}

func writeCompletion(writer http.ResponseWriter, content string) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(completionEnvelope(content)))
}

func completionEnvelope(content string) string {
	return `{"choices":[{"message":{"content":` + quote(content) + `}}]}`
}

func quote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func hasErrorKind(err error, kind inference.ErrorKind) bool {
	var providerErr *inference.Error
	return errors.As(err, &providerErr) && providerErr.Kind == kind
}
