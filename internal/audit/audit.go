// Package audit records AI inference calls and exposes administrator log reads.
package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/inference"
)

const auditTimeout = 5 * time.Second

// Operation identifies one auditable AI operation.
type Operation string

const (
	// OperationQuestionGeneration records a generated question completion.
	OperationQuestionGeneration Operation = "question-generation"
	// OperationWrittenGrading records a batch written-answer grading completion.
	OperationWrittenGrading Operation = "written-grading"
)

// Record is a raw AI completion retained for audit.
type Record struct {
	ID           string        `json:"id"`
	Operation    Operation     `json:"operation"`
	Model        string        `json:"aiModel"`
	Prompt       string        `json:"prompt"`
	Output       string        `json:"output"`
	LiveQuestion *LiveQuestion `json:"liveQuestion"`
	CreatedAt    time.Time     `json:"createdAt"`
}

// LiveQuestion is the populated live-question log relation.
type LiveQuestion struct {
	ID             string           `json:"id"`
	Question       catalog.Question `json:"question"`
	Level          int64            `json:"level"`
	QuestionNumber int64            `json:"questionNumber"`
	Status         string           `json:"status"`
}

// Page is one deterministic page of audit records.
type Page struct {
	Items []Record
	Total int64
	Page  int64
	Limit int64
}

// Repository persists and queries audit records.
type Repository interface {
	Create(context.Context, Record) (string, error)
	LinkLiveQuestion(context.Context, string, string) error
	List(context.Context, int64, int64) (Page, error)
}

// Provider audits an inference provider without changing its interface.
type Provider struct {
	provider   inference.Provider
	repository Repository
}

// NewProvider wraps provider with required audit persistence.
func NewProvider(provider inference.Provider, repository Repository) *Provider {
	return &Provider{provider: provider, repository: repository}
}

// GenerateQuestion generates one question and records exactly one logical call.
func (provider *Provider) GenerateQuestion(ctx context.Context, request inference.GenerationRequest) (inference.GenerationResult, error) {
	if err := inference.ValidateGenerationRequest(request); err != nil {
		return inference.GenerationResult{}, err
	}
	result, callErr := provider.provider.GenerateQuestion(ctx, request)
	auditID, auditErr := provider.record(ctx, OperationQuestionGeneration, result.Completion, callErr)
	if callErr != nil {
		return inference.GenerationResult{}, joinErrors(callErr, auditErr)
	}
	if auditErr != nil {
		return inference.GenerationResult{}, auditErr
	}
	result.AuditID = auditID
	return result, nil
}

// GradeWritten grades a batch and records exactly one logical call.
func (provider *Provider) GradeWritten(ctx context.Context, request inference.GradeWrittenRequest) (inference.GradeWrittenResult, error) {
	if err := inference.ValidateGradeWrittenRequest(request); err != nil {
		return inference.GradeWrittenResult{}, err
	}
	result, callErr := provider.provider.GradeWritten(ctx, request)
	auditID, auditErr := provider.record(ctx, OperationWrittenGrading, result.Completion, callErr)
	if callErr != nil {
		return inference.GradeWrittenResult{}, joinErrors(callErr, auditErr)
	}
	if auditErr != nil {
		return inference.GradeWrittenResult{}, auditErr
	}
	result.AuditID = auditID
	return result, nil
}

// record persists completion metadata even after caller context cancellation.
func (provider *Provider) record(ctx context.Context, operation Operation, completion inference.CompletionMetadata, callErr error) (string, error) {
	if callErr != nil {
		var inferenceErr *inference.Error
		if errors.As(callErr, &inferenceErr) {
			completion = inferenceErr.Completion
		}
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditTimeout)
	defer cancel()
	id, err := provider.repository.Create(auditCtx, Record{Operation: operation,
		Model: completion.Model, Prompt: completion.UserPrompt, Output: completion.RawOutput})
	if err != nil {
		return "", fmt.Errorf("record AI audit: %w", err)
	}
	return id, nil
}

// LinkLiveQuestion links an audit record to a persisted live question.
func LinkLiveQuestion(ctx context.Context, repository Repository, auditID string, liveQuestionID string) error {
	return repository.LinkLiveQuestion(ctx, auditID, liveQuestionID)
}

// joinErrors preserves both inference and persistence failures.
func joinErrors(first error, second error) error {
	if second == nil {
		return first
	}
	return errors.Join(first, second)
}
