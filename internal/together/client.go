package together

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/inference"
)

const (
	defaultEndpoint     = "https://api.together.xyz/v1/chat/completions"
	defaultTimeout      = 30 * time.Second
	defaultMaxBodyBytes = int64(1 << 20)
	defaultMaxRetries   = 2
)

// Config configures a Together inference provider.
type Config struct {
	APIKey       string
	Endpoint     string
	HTTPClient   *http.Client
	Timeout      time.Duration
	MaxBodyBytes int64
	MaxRetries   int
	Backoff      func(int) time.Duration
}

// Client implements inference.Provider with Together's chat completions API.
type Client struct {
	apiKey       string
	endpoint     string
	httpClient   *http.Client
	timeout      time.Duration
	maxBodyBytes int64
	maxRetries   int
	backoff      func(int) time.Duration
}

// NewClient creates a Together provider with bounded defaults.
func NewClient(config Config) *Client {
	if config.Endpoint == "" {
		config.Endpoint = defaultEndpoint
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = defaultMaxBodyBytes
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.Backoff == nil {
		config.Backoff = defaultBackoff
	}
	return &Client{
		apiKey: config.APIKey, endpoint: config.Endpoint,
		httpClient: config.HTTPClient, timeout: config.Timeout,
		maxBodyBytes: config.MaxBodyBytes, maxRetries: config.MaxRetries,
		backoff: config.Backoff,
	}
}

// GenerateQuestion generates and validates one question.
func (client *Client) GenerateQuestion(
	ctx context.Context,
	request inference.GenerationRequest,
) (inference.GenerationResult, error) {
	if err := inference.ValidateGenerationRequest(request); err != nil {
		return inference.GenerationResult{}, err
	}
	prompt, err := CreateGenerationUserPrompt(request)
	if err != nil {
		return inference.GenerationResult{}, err
	}
	metadata := inference.CompletionMetadata{Model: Model, UserPrompt: prompt}
	output, err := client.complete(ctx, generationPayload(prompt), metadata)
	metadata.RawOutput = output
	if err != nil {
		return inference.GenerationResult{}, providerError(err, metadata)
	}
	question, err := decodeGeneratedQuestion(output, request)
	if err != nil {
		return inference.GenerationResult{}, invalidResponseError(err, metadata)
	}
	return inference.GenerationResult{Question: question, Completion: metadata}, nil
}

// GradeWritten grades and validates one batch of written answers.
func (client *Client) GradeWritten(
	ctx context.Context,
	request inference.GradeWrittenRequest,
) (inference.GradeWrittenResult, error) {
	if err := inference.ValidateGradeWrittenRequest(request); err != nil {
		return inference.GradeWrittenResult{}, err
	}
	prompt, err := CreateWrittenGradingUserPrompt(request)
	if err != nil {
		return inference.GradeWrittenResult{}, err
	}
	metadata := inference.CompletionMetadata{Model: Model, UserPrompt: prompt}
	output, err := client.complete(ctx, gradingPayload(prompt), metadata)
	metadata.RawOutput = output
	if err != nil {
		return inference.GradeWrittenResult{}, providerError(err, metadata)
	}
	evaluations, err := decodeWrittenEvaluations(output, request)
	if err != nil {
		return inference.GradeWrittenResult{}, invalidResponseError(err, metadata)
	}
	return inference.GradeWrittenResult{
		Evaluations: evaluations, Completion: metadata,
	}, nil
}

// generationPayload builds one Together generation request.
func generationPayload(prompt string) completionRequest {
	return completionRequest{
		Model: Model, ResponseFormat: GeneratedQuestionResponseFormat,
		Temperature: GenerationTemperature,
		Messages: []message{
			{Role: "system", Content: QuestionGenerationSystemPrompt},
			{Role: "user", Content: prompt},
		},
	}
}

// gradingPayload builds one Together written-grading request.
func gradingPayload(prompt string) completionRequest {
	return completionRequest{
		Model: Model, ResponseFormat: WrittenEvaluationResponseFormat,
		Temperature: GradingTemperature,
		Messages: []message{
			{Role: "system", Content: WrittenEvaluationSystemPrompt},
			{Role: "user", Content: prompt},
		},
	}
}

// complete sends a bounded completion request with limited transient retries.
func (client *Client) complete(
	ctx context.Context,
	payload completionRequest,
	metadata inference.CompletionMetadata,
) (string, error) {
	if client.apiKey == "" {
		return "", &inference.Error{
			Kind: inference.ErrorUnavailable, Message: "AI is not configured",
			Completion: metadata,
		}
	}
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal Together request: %w", err)
	}
	for attempt := 0; ; attempt++ {
		output, retry, err := client.doCompletion(ctx, body)
		if err == nil {
			return output, nil
		}
		if !retry || attempt >= client.maxRetries {
			return output, err
		}
		if err := wait(ctx, client.backoff(attempt)); err != nil {
			return output, err
		}
	}
}

// doCompletion performs one HTTP completion attempt.
func (client *Client) doCompletion(
	ctx context.Context,
	body []byte,
) (string, bool, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, client.endpoint, bytes.NewReader(body),
	)
	if err != nil {
		return "", false, err
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return "", false, ctx.Err()
		}
		return "", true, fmt.Errorf("Together request failed: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := readBounded(response.Body, client.maxBodyBytes)
	if err != nil {
		return "", false, err
	}
	output := string(responseBody)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return output, response.StatusCode == http.StatusTooManyRequests ||
				response.StatusCode >= http.StatusInternalServerError,
			fmt.Errorf("Together returned HTTP %d", response.StatusCode)
	}
	if strings.TrimSpace(output) == "" {
		return output, false, errEmptyCompletion
	}
	var completion completionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return output, false, fmt.Errorf("%w: %v", errInvalidCompletion, err)
	}
	if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == nil {
		return output, false, errEmptyCompletion
	}
	if strings.TrimSpace(*completion.Choices[0].Message.Content) == "" {
		return *completion.Choices[0].Message.Content, false, errEmptyCompletion
	}
	return *completion.Choices[0].Message.Content, false, nil
}

// providerError preserves normalized errors and their completion metadata.
func providerError(err error, metadata inference.CompletionMetadata) error {
	var providerErr *inference.Error
	if errors.As(err, &providerErr) {
		providerErr.Completion = metadata
		return providerErr
	}
	if errors.Is(err, errEmptyCompletion) {
		return &inference.Error{Kind: inference.ErrorEmptyResponse,
			Message: "AI response was empty", Completion: metadata, Cause: err}
	}
	if errors.Is(err, errInvalidCompletion) {
		return &inference.Error{Kind: inference.ErrorInvalidResponse,
			Message: "AI response was invalid", Completion: metadata, Cause: err}
	}
	return &inference.Error{Kind: inference.ErrorUpstream,
		Message: "AI provider request failed", Completion: metadata, Cause: err}
}

// invalidResponseError creates a metadata-preserving invalid response error.
func invalidResponseError(err error, metadata inference.CompletionMetadata) error {
	return &inference.Error{Kind: inference.ErrorInvalidResponse,
		Message: "AI response was invalid", Completion: metadata, Cause: err}
}

// readBounded reads no more than limit bytes plus a size sentinel.
func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("Together response exceeded %d bytes", limit)
	}
	return body, nil
}

// wait blocks for duration or until context cancellation.
func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// defaultBackoff returns a bounded exponential retry delay.
func defaultBackoff(attempt int) time.Duration {
	delay := 100 * time.Millisecond * time.Duration(1<<attempt)
	if delay > time.Second {
		return time.Second
	}
	return delay
}

var errEmptyCompletion = errors.New("empty Together completion")
var errInvalidCompletion = errors.New("invalid Together completion")

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model          string                   `json:"model"`
	ResponseFormat JSONSchemaResponseFormat `json:"response_format"`
	Temperature    float64                  `json:"temperature"`
	Messages       []message                `json:"messages"`
}

type completionResponse struct {
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
