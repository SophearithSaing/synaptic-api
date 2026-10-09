package audit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/inference"
	"github.com/SophearithSaing/synaptic-api/internal/together"
)

type fakeProvider struct {
	generation    inference.GenerationResult
	generationErr error
	grading       inference.GradeWrittenResult
	gradingErr    error
	calls         int
}

func (p *fakeProvider) GenerateQuestion(context.Context, inference.GenerationRequest) (inference.GenerationResult, error) {
	p.calls++
	return p.generation, p.generationErr
}
func (p *fakeProvider) GradeWritten(context.Context, inference.GradeWrittenRequest) (inference.GradeWrittenResult, error) {
	p.calls++
	return p.grading, p.gradingErr
}

type fakeRepository struct {
	records     []Record
	err         error
	contextErr  error
	hasDeadline bool
}

func (r *fakeRepository) Create(ctx context.Context, record Record) (string, error) {
	r.contextErr = ctx.Err()
	_, r.hasDeadline = ctx.Deadline()
	r.records = append(r.records, record)
	return "audit-id", r.err
}

func TestProviderAuditsGradingAndPersistenceFailure(t *testing.T) {
	repo := &fakeRepository{}
	provider := &fakeProvider{grading: inference.GradeWrittenResult{Completion: inference.CompletionMetadata{Model: "m", UserPrompt: "p", RawOutput: "o"}}}
	result, err := NewProvider(provider, repo).GradeWritten(context.Background(), inference.GradeWrittenRequest{})
	if err != nil || result.AuditID != "audit-id" || len(repo.records) != 1 || repo.records[0].Operation != OperationWrittenGrading {
		t.Fatalf("result=%#v err=%v records=%#v", result, err, repo.records)
	}
	repo.err = errors.New("store")
	_, err = NewProvider(provider, repo).GradeWritten(context.Background(), inference.GradeWrittenRequest{})
	if !errors.Is(err, repo.err) {
		t.Fatalf("error=%v", err)
	}
	provider.calls = 0
	_, err = NewProvider(provider, repo).GradeWritten(context.Background(), inference.GradeWrittenRequest{Answers: []inference.WrittenAnswer{{}}})
	if err == nil || provider.calls != 0 {
		t.Fatalf("invalid grading error=%v calls=%d", err, provider.calls)
	}
}

func TestProviderAuditsGradingFailureAndRetriedTogetherCallOnce(t *testing.T) {
	gradingErr := &inference.Error{Kind: inference.ErrorInvalidResponse, Completion: inference.CompletionMetadata{Model: "m", UserPrompt: "p", RawOutput: "bad"}}
	repo := &fakeRepository{}
	_, err := NewProvider(&fakeProvider{gradingErr: gradingErr}, repo).GradeWritten(context.Background(), inference.GradeWrittenRequest{})
	if !errors.Is(err, gradingErr) || len(repo.records) != 1 || repo.records[0].Output != "bad" {
		t.Fatalf("error=%v records=%#v", err, repo.records)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		if calls == 1 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"question\":{\"id\":\"x\",\"type\":\"written\",\"prompt\":\"p\",\"targetConcepts\":[\"c\"],\"feedback\":{\"correct\":\"y\",\"incorrect\":\"n\"},\"rubrics\":{\"keyPoints\":[],\"misconceptions\":[]}}}"}}]}`))
	}))
	defer server.Close()
	repo = &fakeRepository{}
	provider := together.NewClient(together.Config{APIKey: "key", Endpoint: server.URL, MaxRetries: 1, Backoff: func(int) time.Duration { return 0 }})
	result, err := NewProvider(provider, repo).GenerateQuestion(context.Background(), inference.GenerationRequest{QuestionType: inference.QuestionTypeWritten})
	if err != nil || calls != 2 || len(repo.records) != 1 || result.AuditID == "" || repo.records[0].Model == "" || repo.records[0].Prompt == "" || repo.records[0].Output == "" {
		t.Fatalf("result=%#v error=%v calls=%d records=%#v", result, err, calls, repo.records)
	}
}
func (r *fakeRepository) LinkLiveQuestion(context.Context, string, string) error { return nil }
func (r *fakeRepository) List(context.Context, int64, int64) (Page, error)       { return Page{}, nil }

func TestProviderAuditsSuccessAndFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeRepository{}
		wrapped := NewProvider(&fakeProvider{generation: inference.GenerationResult{Completion: inference.CompletionMetadata{Model: "m", UserPrompt: "p", RawOutput: "o"}}}, repo)
		result, err := wrapped.GenerateQuestion(context.Background(), inference.GenerationRequest{QuestionType: inference.QuestionTypeMCQ})
		if err != nil || result.AuditID != "audit-id" || len(repo.records) != 1 {
			t.Fatalf("result=%#v err=%v records=%#v", result, err, repo.records)
		}
	})
	t.Run("failure", func(t *testing.T) {
		callErr := &inference.Error{Kind: inference.ErrorInvalidResponse, Completion: inference.CompletionMetadata{Model: "m", UserPrompt: "p", RawOutput: "bad"}}
		repo := &fakeRepository{}
		wrapped := NewProvider(&fakeProvider{generationErr: callErr}, repo)
		_, err := wrapped.GenerateQuestion(context.Background(), inference.GenerationRequest{QuestionType: inference.QuestionTypeMCQ})
		if !errors.Is(err, callErr) || len(repo.records) != 1 || repo.records[0].Output != "bad" {
			t.Fatalf("err=%v records=%#v", err, repo.records)
		}
	})
}

func TestProviderSkipsInvalidInputAndPreservesPersistenceFailure(t *testing.T) {
	provider := &fakeProvider{}
	repo := &fakeRepository{err: errors.New("store unavailable")}
	wrapped := NewProvider(provider, repo)
	_, err := wrapped.GenerateQuestion(context.Background(), inference.GenerationRequest{QuestionType: "invalid"})
	if err == nil || provider.calls != 0 || len(repo.records) != 0 {
		t.Fatalf("err=%v calls=%d records=%d", err, provider.calls, len(repo.records))
	}
	callErr := errors.New("provider failed")
	provider.generationErr = callErr
	_, err = wrapped.GenerateQuestion(context.Background(), inference.GenerationRequest{QuestionType: inference.QuestionTypeMCQ})
	if !errors.Is(err, callErr) || !errors.Is(err, repo.err) {
		t.Fatalf("joined error=%v", err)
	}
}

func TestProviderAuditsCancelledCallsWithDetachedContext(t *testing.T) {
	callErr := &inference.Error{Kind: inference.ErrorUpstream,
		Cause:      context.Canceled,
		Completion: inference.CompletionMetadata{Model: "m", UserPrompt: "p"}}
	repo := &fakeRepository{}
	wrapped := NewProvider(&fakeProvider{generationErr: callErr}, repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := wrapped.GenerateQuestion(ctx, inference.GenerationRequest{
		QuestionType: inference.QuestionTypeMCQ,
	})
	if !errors.Is(err, context.Canceled) || len(repo.records) != 1 ||
		repo.contextErr != nil {
		t.Fatalf("error=%v records=%d audit context=%v", err, len(repo.records), repo.contextErr)
	}
	if !repo.hasDeadline {
		t.Fatal("detached audit context has no deadline")
	}
}
