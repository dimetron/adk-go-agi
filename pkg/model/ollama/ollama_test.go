package ollama

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestConvertContentsToMessages(t *testing.T) {
	tests := []struct {
		name     string
		contents []*genai.Content
		wantLen  int
		wantErr  bool
	}{
		{
			name: "single user message",
			contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Hello"},
					},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "multiple messages with different roles",
			contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Hello"},
					},
				},
				{
					Role: "model",
					Parts: []*genai.Part{
						{Text: "Hi there!"},
					},
				},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:     "empty contents",
			contents: []*genai.Content{},
			wantLen:  0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := convertContentsToMessages(tt.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertContentsToMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(messages) != tt.wantLen {
				t.Errorf("convertContentsToMessages() got %d messages, want %d", len(messages), tt.wantLen)
			}
		})
	}
}

func TestNewModel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				ModelName: "llama3.2",
				BaseURL:   "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty model name",
			cfg: &Config{
				BaseURL: "http://localhost:11434",
			},
			wantErr: true,
		},
		{
			name: "invalid URL",
			cfg: &Config{
				ModelName: "llama3.2",
				BaseURL:   "://invalid-url",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdl, err := NewModel(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && mdl == nil {
				t.Error("NewModel() returned nil mdl without error")
			}
		})
	}
}

func TestNewSyncModel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				ModelName: "llama3.2",
				BaseURL:   "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty model name",
			cfg: &Config{
				BaseURL: "http://localhost:11434",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewSyncModel(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSyncModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && model == nil {
				t.Error("NewSyncModel() returned nil model without error")
			}
			if !tt.wantErr && model.name != tt.cfg.ModelName {
				t.Errorf("NewSyncModel() model name = %v, want %v", model.name, tt.cfg.ModelName)
			}
		})
	}
}

func TestNewStreamModel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				ModelName: "llama3.2",
				BaseURL:   "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty model name",
			cfg: &Config{
				BaseURL: "http://localhost:11434",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewStreamModel(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStreamModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && model == nil {
				t.Error("NewStreamModel() returned nil model without error")
			}
			if !tt.wantErr && model.name != tt.cfg.ModelName {
				t.Errorf("NewStreamModel() model name = %v, want %v", model.name, tt.cfg.ModelName)
			}
		})
	}
}

// mockClient is a mock implementation of the chatClient interface for testing.
type mockClient struct {
	chatFunc func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error
}

// Chat implements the chatClient interface.
func (m *mockClient) Chat(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req, fn)
	}
	return nil
}

// FuzzConvertContentsToMessages fuzzes the content-to-message conversion.
func FuzzConvertContentsToMessages(f *testing.F) {
	// Seed corpus
	f.Add("Hello world", "user", "How are you?", "model")
	f.Add("", "user", "", "assistant")
	f.Add("Test message", "system", "Another test", "user")

	f.Fuzz(func(t *testing.T, text1, role1, text2, role2 string) {
		contents := []*genai.Content{
			{
				Role: role1,
				Parts: []*genai.Part{
					{Text: text1},
				},
			},
			{
				Role: role2,
				Parts: []*genai.Part{
					{Text: text2},
				},
			},
		}

		messages, err := convertContentsToMessages(contents)

		// Should never error with valid input structure
		if err != nil {
			t.Errorf("convertContentsToMessages() unexpected error: %v", err)
		}

		// Should return same number of messages as contents
		if len(messages) != len(contents) {
			t.Errorf("convertContentsToMessages() got %d messages, want %d", len(messages), len(contents))
		}

		// Verify role mapping
		for i, msg := range messages {
			if contents[i].Role == "model" && msg.Role != "assistant" {
				t.Errorf("role 'model' should map to 'assistant', got %q", msg.Role)
			}
		}
	})
}

// FuzzConvertChatResponseToLLMResponse fuzzes the chat response conversion.
func FuzzConvertChatResponseToLLMResponse(f *testing.F) {
	// Seed corpus
	f.Add("Response text", int64(100), int64(50), true)
	f.Add("", int64(0), int64(0), false)
	f.Add("Multi-line\nresponse\ntext", int64(250), int64(125), true)

	f.Fuzz(func(t *testing.T, content string, promptEvalCount, evalCount int64, done bool) {
		resp := &api.ChatResponse{
			Message: api.Message{
				Role:    "assistant",
				Content: content,
			},
			Done: done,
		}
		// Set metrics fields directly (they are embedded from Metrics)
		resp.PromptEvalCount = int(promptEvalCount)
		resp.EvalCount = int(evalCount)

		llmResp := convertChatResponseToLLMResponse(resp)

		// Should never return nil
		if llmResp == nil {
			t.Fatal("convertChatResponseToLLMResponse() returned nil")
		}

		// Content should be preserved
		if llmResp.Content == nil {
			t.Fatal("convertChatResponseToLLMResponse() returned nil Content")
		}

		if len(llmResp.Content.Parts) != 1 {
			t.Errorf("expected 1 part, got %d", len(llmResp.Content.Parts))
		}

		if llmResp.Content.Parts[0].Text != content {
			t.Errorf("content text mismatch: got %q, want %q", llmResp.Content.Parts[0].Text, content)
		}

		// Usage metadata should be set if counts are positive
		if promptEvalCount > 0 || evalCount > 0 {
			if llmResp.UsageMetadata == nil {
				t.Error("expected UsageMetadata to be set")
			} else {
				if llmResp.UsageMetadata.PromptTokenCount != int32(promptEvalCount) {
					t.Errorf("PromptTokenCount = %d, want %d", llmResp.UsageMetadata.PromptTokenCount, promptEvalCount)
				}
				if llmResp.UsageMetadata.CandidatesTokenCount != int32(evalCount) {
					t.Errorf("CandidatesTokenCount = %d, want %d", llmResp.UsageMetadata.CandidatesTokenCount, evalCount)
				}
			}
		}

		// FinishReason should be set if done
		if done && llmResp.FinishReason != genai.FinishReasonStop {
			t.Errorf("expected FinishReason to be Stop when done=true, got %v", llmResp.FinishReason)
		}
	})
}

// TestSyncGeneratorWithMock tests synchronous generation with a mock client.
func TestSyncGeneratorWithMock(t *testing.T) {
	tests := []struct {
		name     string
		chatFunc func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error
		wantErr  bool
	}{
		{
			name: "successful response",
			chatFunc: func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
				resp := api.ChatResponse{
					Message: api.Message{
						Role:    "assistant",
						Content: "Hello from mock!",
					},
					Done: true,
				}
				resp.PromptEvalCount = 10
				resp.EvalCount = 5
				return fn(resp)
			},
			wantErr: false,
		},
		{
			name: "api error",
			chatFunc: func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
				return errors.New("mock API error")
			},
			wantErr: true,
		},
		{
			name: "context canceled",
			chatFunc: func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
				return context.Canceled
			},
			wantErr: false, // Context cancellation doesn't yield anything (no error, no response)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{chatFunc: tt.chatFunc}

			gen := &SyncGenerator{
				baseModel: baseModel{
					client:  mock,
					name:    "test-model",
					baseURL: "http://localhost:11434",
					options: make(map[string]interface{}),
				},
			}

			// Create a simple request
			req := &model.LLMRequest{
				Contents: []*genai.Content{
					{
						Role: "user",
						Parts: []*genai.Part{
							{Text: "Test message"},
						},
					},
				},
			}

			ctx := context.Background()
			if tt.name == "context canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			var gotResponse *model.LLMResponse
			var gotError error

			for resp, err := range gen.generate(ctx, req) {
				gotResponse = resp
				gotError = err
			}

			if (gotError != nil) != tt.wantErr {
				t.Errorf("generate() error = %v, wantErr %v", gotError, tt.wantErr)
			}

			// For context canceled case, we expect no response and no error
			if !tt.wantErr && gotResponse == nil && tt.name != "context canceled" {
				t.Error("generate() returned nil response without error")
			}
		})
	}
}

// TestStreamGeneratorWithMock tests streaming generation with a mock client.
func TestStreamGeneratorWithMock(t *testing.T) {
	tests := []struct {
		name       string
		chatFunc   func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error
		wantChunks int
		wantErr    bool
	}{
		{
			name: "successful streaming",
			chatFunc: func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
				// Simulate 3 streaming chunks
				chunk1 := api.ChatResponse{
					Message: api.Message{Role: "assistant", Content: "Hello"},
					Done:    false,
				}
				chunk2 := api.ChatResponse{
					Message: api.Message{Role: "assistant", Content: " world"},
					Done:    false,
				}
				chunk3 := api.ChatResponse{
					Message: api.Message{Role: "assistant", Content: "!"},
					Done:    true,
				}
				chunk3.PromptEvalCount = 10
				chunk3.EvalCount = 5

				for _, chunk := range []api.ChatResponse{chunk1, chunk2, chunk3} {
					if err := fn(chunk); err != nil {
						return err
					}
				}
				return nil
			},
			wantChunks: 3,
			wantErr:    false,
		},
		{
			name: "streaming error mid-stream",
			chatFunc: func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
				fn(api.ChatResponse{
					Message: api.Message{Role: "assistant", Content: "Start"},
					Done:    false,
				})
				return errors.New("stream interrupted")
			},
			wantChunks: 1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{chatFunc: tt.chatFunc}

			gen := &StreamGenerator{
				baseModel: baseModel{
					client:  mock,
					name:    "test-model",
					baseURL: "http://localhost:11434",
					options: make(map[string]interface{}),
				},
			}

			req := &model.LLMRequest{
				Contents: []*genai.Content{
					{
						Role: "user",
						Parts: []*genai.Part{
							{Text: "Test streaming"},
						},
					},
				},
			}

			ctx := context.Background()
			var chunkCount int
			var lastError error

			for resp, err := range gen.generate(ctx, req) {
				if err != nil {
					lastError = err
					break
				}
				if resp != nil {
					chunkCount++
				}
			}

			if (lastError != nil) != tt.wantErr {
				t.Errorf("generate() error = %v, wantErr %v", lastError, tt.wantErr)
			}

			if chunkCount != tt.wantChunks {
				t.Errorf("generate() got %d chunks, want %d", chunkCount, tt.wantChunks)
			}
		})
	}
}

// FuzzSyncGeneratorWithMock fuzzes synchronous generation with various inputs.
func FuzzSyncGeneratorWithMock(f *testing.F) {
	// Seed corpus
	f.Add("Hello", "user", "Hi there!", int(10), int(5))
	f.Add("", "system", "Response", int(0), int(0))
	f.Add("Test\nmultiline", "user", "Multi\nline\nresponse", int(100), int(50))

	f.Fuzz(func(t *testing.T, inputText, inputRole, responseText string, promptTokens, completionTokens int) {
		// Create mock that returns the fuzzy response
		chatFunc := func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
			resp := api.ChatResponse{
				Message: api.Message{
					Role:    "assistant",
					Content: responseText,
				},
				Done: true,
			}
			resp.PromptEvalCount = promptTokens
			resp.EvalCount = completionTokens
			return fn(resp)
		}

		mock := &mockClient{chatFunc: chatFunc}

		gen := &SyncGenerator{
			baseModel: baseModel{
				client:  mock,
				name:    "fuzz-model",
				baseURL: "http://localhost:11434",
				options: make(map[string]interface{}),
			},
		}

		req := &model.LLMRequest{
			Contents: []*genai.Content{
				{
					Role: inputRole,
					Parts: []*genai.Part{
						{Text: inputText},
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var gotResponse *model.LLMResponse
		for resp, err := range gen.generate(ctx, req) {
			if err != nil {
				// Errors are acceptable in fuzzing
				return
			}
			gotResponse = resp
		}

		// Verify response structure
		if gotResponse == nil {
			t.Fatal("expected non-nil response")
		}

		if gotResponse.Content == nil {
			t.Fatal("expected non-nil Content")
		}

		if len(gotResponse.Content.Parts) == 0 {
			t.Error("expected at least one part in response")
		}
	})
}

// TestContextCancellation verifies that context cancellation doesn't cause panics.
// This test ensures the fix for the nil pointer dereference when context is cancelled.
func TestContextCancellation(t *testing.T) {
	chatFunc := func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
		return ctx.Err()
	}

	mock := &mockClient{chatFunc: chatFunc}

	gen := &SyncGenerator{
		baseModel: baseModel{
			client:  mock,
			name:    "test-model",
			baseURL: "http://localhost:11434",
			options: make(map[string]interface{}),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: "Test"},
				},
			},
		},
	}

	// Should not panic
	for range gen.generate(ctx, req) {
		// Context cancelled, should exit gracefully
	}
}

// TestModelName verifies the Name() method returns the correct model name.
func TestModelName(t *testing.T) {
	model, err := NewModel(context.Background(), &Config{
		ModelName: "test-model-123",
		BaseURL:   "http://localhost:11434",
	})
	if err != nil {
		t.Fatalf("NewModel() failed: %v", err)
	}

	if model.Name() != "test-model-123" {
		t.Errorf("Name() = %q, want %q", model.Name(), "test-model-123")
	}
}

// TestGenerateContentStreamingSwitch verifies the Model switches between sync and stream modes.
func TestGenerateContentStreamingSwitch(t *testing.T) {
	tests := []struct {
		name   string
		stream bool
	}{
		{
			name:   "sync mode",
			stream: false,
		},
		{
			name:   "stream mode",
			stream: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatFunc := func(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
				if tt.stream {
					// Return multiple chunks for streaming
					fn(api.ChatResponse{
						Message: api.Message{Role: "assistant", Content: "Part 1"},
						Done:    false,
					})
					return fn(api.ChatResponse{
						Message: api.Message{Role: "assistant", Content: "Part 2"},
						Done:    true,
					})
				}
				// Return single response for sync
				return fn(api.ChatResponse{
					Message: api.Message{Role: "assistant", Content: "Complete"},
					Done:    true,
				})
			}

			mock := &mockClient{chatFunc: chatFunc}

			base := &baseModel{
				client:  mock,
				name:    "test-model",
				baseURL: "http://localhost:11434",
				options: make(map[string]interface{}),
			}

			ollamaModel := &Model{
				syncGen:   &SyncGenerator{baseModel: *base},
				streamGen: &StreamGenerator{baseModel: *base},
			}

			req := &model.LLMRequest{
				Contents: []*genai.Content{
					{
						Role: "user",
						Parts: []*genai.Part{
							{Text: "Test"},
						},
					},
				},
			}

			ctx := context.Background()
			var responseCount int

			for resp, err := range ollamaModel.GenerateContent(ctx, req, tt.stream) {
				if err != nil {
					t.Fatalf("GenerateContent() error = %v", err)
				}
				if resp != nil {
					responseCount++
				}
			}

			if tt.stream && responseCount != 2 {
				t.Errorf("stream mode: got %d responses, want 2", responseCount)
			}
			if !tt.stream && responseCount != 1 {
				t.Errorf("sync mode: got %d responses, want 1", responseCount)
			}
		})
	}
}
