// Package ollama implements the model.LLM interface for Ollama models using the official Ollama Go API client.
package ollama

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// chatClient defines the interface for chat operations, allowing for testing with mocks.
type chatClient interface {
	Chat(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error
}

// baseModel holds shared configuration and client for Ollama models.
type baseModel struct {
	client  chatClient
	name    string
	baseURL string
	options map[string]interface{}
}

// SyncGenerator generates content synchronously (non-streaming).
type SyncGenerator struct {
	baseModel
}

// StreamGenerator generates content with streaming.
type StreamGenerator struct {
	baseModel
}

// Model implements the model.LLM interface for Ollama models.
// It can operate in either sync or stream mode based on the stream parameter.
type Model struct {
	syncGen   *SyncGenerator
	streamGen *StreamGenerator
}

// Config holds configuration for creating an Ollama model.
type Config struct {
	// ModelName is the name of the Ollama model to use (e.g., "llama3.2", "mistral", "codellama")
	ModelName string
	// BaseURL is the Ollama API endpoint (default: "http://localhost:11434")
	BaseURL string
	// HTTPClient is an optional custom HTTP client
	HTTPClient *http.Client
	// Options are model-specific options (temperature, top_p, etc.)
	Options map[string]interface{}
}

// NewModel creates a new Ollama model that implements model.LLM interface.
func NewModel(ctx context.Context, cfg *Config) (model.LLM, error) {
	base, err := newBaseModel(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Model{
		syncGen:   &SyncGenerator{baseModel: *base},
		streamGen: &StreamGenerator{baseModel: *base},
	}, nil
}

// NewSyncModel creates a model optimized for synchronous (non-streaming) generation.
func NewSyncModel(ctx context.Context, cfg *Config) (*SyncGenerator, error) {
	base, err := newBaseModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &SyncGenerator{baseModel: *base}, nil
}

// NewStreamModel creates a model optimized for streaming generation.
func NewStreamModel(ctx context.Context, cfg *Config) (*StreamGenerator, error) {
	base, err := newBaseModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &StreamGenerator{baseModel: *base}, nil
}

// newBaseModel creates the shared base model configuration.
func newBaseModel(ctx context.Context, cfg *Config) (*baseModel, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("model name is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Ensure we have an HTTP client with proper timeouts to prevent indefinite hangs
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Minute, // Overall request timeout
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second, // Connection timeout
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second, // TLS handshake timeout
				ResponseHeaderTimeout: 30 * time.Second, // Wait for response headers
				ExpectContinueTimeout: 1 * time.Second,
				IdleConnTimeout:       90 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
			},
		}
	}

	// Create Ollama client
	client := api.NewClient(parsedURL, httpClient)

	return &baseModel{
		client:  client,
		name:    cfg.ModelName,
		baseURL: baseURL,
		options: cfg.Options,
	}, nil
}

// Name returns the model name.
func (m *Model) Name() string {
	return m.syncGen.name
}

// GenerateContent implements the model.LLM interface.
// It delegates to the appropriate generator based on the stream parameter.
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	if stream {
		return m.streamGen.generate(ctx, req)
	}
	return m.syncGen.generate(ctx, req)
}

// generate implements synchronous (non-streaming) content generation.
func (g *SyncGenerator) generate(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Check context before starting - early cancellation detection
		if err := ctx.Err(); err != nil {
			slog.WarnContext(ctx, "Context already canceled before starting generation",
				"model", g.name,
				"error", err)
			return // Don't yield, just return early
		}

		// Convert genai contents to Ollama messages
		messages, err := convertContentsToMessages(req.Contents)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert contents: %w", err))
			return
		}

		// Build Ollama chat request
		chatReq := &api.ChatRequest{
			Model:    g.name,
			Messages: messages,
			Options:  g.options,
			Stream:   new(bool), // false
		}

		// Log start of API call
		slog.InfoContext(ctx, "Starting Ollama API call",
			"model", g.name,
			"stream", false,
			"message_count", len(messages))
		start := time.Now()

		var response api.ChatResponse
		err = g.client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
			response = resp
			return nil
		})

		duration := time.Since(start)

		if err != nil {
			slog.ErrorContext(ctx, "Ollama API call failed",
				"model", g.name,
				"duration_ms", duration.Milliseconds(),
				"error", err)
			// Check if context was canceled - don't yield in this case as consumer may have stopped
			if ctx.Err() != nil {
				return
			}
			yield(nil, fmt.Errorf("ollama chat failed: %w", err))
			return
		}

		// Log successful completion
		slog.InfoContext(ctx, "Ollama API call completed",
			"model", g.name,
			"duration_ms", duration.Milliseconds(),
			"prompt_tokens", response.PromptEvalCount,
			"completion_tokens", response.EvalCount,
			"total_tokens", response.PromptEvalCount+response.EvalCount)

		// Convert Ollama response to LLMResponse
		llmResp := convertChatResponseToLLMResponse(&response)
		yield(llmResp, nil)
	}
}

// generate implements streaming content generation.
func (g *StreamGenerator) generate(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Check context before starting - early cancellation detection
		if err := ctx.Err(); err != nil {
			slog.WarnContext(ctx, "Context already canceled before starting streaming generation",
				"model", g.name,
				"error", err)
			return // Don't yield, just return early
		}

		// Convert genai contents to Ollama messages
		messages, err := convertContentsToMessages(req.Contents)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert contents: %w", err))
			return
		}

		// Build Ollama chat request with streaming
		chatReq := &api.ChatRequest{
			Model:    g.name,
			Messages: messages,
			Options:  g.options,
			Stream:   ptrBool(true),
		}

		// Log start of streaming API call
		slog.InfoContext(ctx, "Starting Ollama streaming API call",
			"model", g.name,
			"stream", true,
			"message_count", len(messages))
		start := time.Now()

		var chunkCount int
		var lastResponse *api.ChatResponse

		err = g.client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
			// Check if context is canceled before processing each chunk
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			chunkCount++
			lastResponse = &resp
			llmResp := convertChatResponseToLLMResponse(&resp)
			llmResp.Partial = !resp.Done
			llmResp.TurnComplete = resp.Done

			if !yield(llmResp, nil) {
				// Consumer stopped - signal to stop the stream immediately
				slog.InfoContext(ctx, "Consumer stopped streaming",
					"model", g.name,
					"chunks_received", chunkCount)
				return fmt.Errorf("consumer stopped")
			}
			return nil
		})

		duration := time.Since(start)

		if err != nil {
			slog.ErrorContext(ctx, "Ollama streaming API call failed",
				"model", g.name,
				"duration_ms", duration.Milliseconds(),
				"chunks_received", chunkCount,
				"error", err)
			// Check if context was canceled - don't yield in this case as consumer may have stopped
			if ctx.Err() != nil {
				return
			}
			yield(nil, fmt.Errorf("ollama streaming failed: %w", err))
			return
		}

		// Log successful completion with statistics
		logArgs := []any{
			"model", g.name,
			"duration_ms", duration.Milliseconds(),
			"chunks_received", chunkCount,
		}
		if lastResponse != nil {
			logArgs = append(logArgs,
				"prompt_tokens", lastResponse.PromptEvalCount,
				"completion_tokens", lastResponse.EvalCount,
				"total_tokens", lastResponse.PromptEvalCount+lastResponse.EvalCount)
		}
		slog.InfoContext(ctx, "Ollama streaming API call completed", logArgs...)
	}
}

// convertContentsToMessages converts genai.Content to Ollama messages.
func convertContentsToMessages(contents []*genai.Content) ([]api.Message, error) {
	messages := make([]api.Message, 0, len(contents))

	for _, content := range contents {
		if content == nil {
			continue
		}

		// Determine role (user, assistant, system)
		role := content.Role
		if role == "" {
			role = "user"
		}
		if role == "model" {
			role = "assistant"
		}

		// Extract text from parts
		var textContent string
		for _, part := range content.Parts {
			if part == nil {
				continue
			}
			// Part is a struct with Text field
			if part.Text != "" {
				textContent += part.Text
			}
			if part.InlineData != nil {
				// Ollama supports images - could be extended
				textContent += "[Inline data not yet supported]"
			}
			if part.FunctionCall != nil {
				textContent += fmt.Sprintf("[FunctionCall: %s]", part.FunctionCall.Name)
			}
		}

		messages = append(messages, api.Message{
			Role:    role,
			Content: textContent,
		})
	}

	return messages, nil
}

// convertChatResponseToLLMResponse converts Ollama ChatResponse to model.LLMResponse.
func convertChatResponseToLLMResponse(resp *api.ChatResponse) *model.LLMResponse {
	// Create genai.Content from Ollama response
	content := &genai.Content{
		Role: "model",
		Parts: []*genai.Part{
			{
				Text: resp.Message.Content,
			},
		},
	}

	llmResp := &model.LLMResponse{
		Content: content,
	}

	// Add usage metadata if available
	// Check if metrics data is available
	if resp.PromptEvalCount > 0 || resp.EvalCount > 0 {
		llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(resp.PromptEvalCount),
			CandidatesTokenCount: int32(resp.EvalCount),
			TotalTokenCount:      int32(resp.PromptEvalCount + resp.EvalCount),
		}
	}

	// Map finish reason
	if resp.Done {
		llmResp.FinishReason = genai.FinishReasonStop
	}

	return llmResp
}

// ptrBool returns a pointer to a bool value.
func ptrBool(b bool) *bool {
	return &b
}
