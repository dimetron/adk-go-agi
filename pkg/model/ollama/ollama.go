// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ollama provides an Ollama LLM implementation for the ADK framework
package ollama

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model implements the model.LLM interface for Ollama
type Model struct {
	client    *api.Client
	modelName string
}

// Config holds configuration for creating an Ollama model
type Config struct {
	// BaseURL is the URL of the Ollama server (defaults to http://localhost:11434)
	BaseURL string
	// ModelName is the name of the Ollama model to use (e.g., "llama2", "mistral")
	ModelName string
	// HTTPClient is an optional custom HTTP client
	HTTPClient *http.Client
}

// NewModel creates a new Ollama LLM model
func NewModel(ctx context.Context, config Config) (*Model, error) {
	if config.ModelName == "" {
		return nil, fmt.Errorf("model name is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := api.NewClient(parsedURL, httpClient)

	return &Model{
		client:    client,
		modelName: config.ModelName,
	}, nil
}

// Name returns the name of the model
func (m *Model) Name() string {
	return fmt.Sprintf("ollama/%s", m.modelName)
}

// GenerateContent generates content using the Ollama API
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to Ollama request
		ollamaReq, err := m.convertToOllamaRequest(req)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert request: %w", err))
			return
		}

		if stream {
			// Streaming mode
			err := m.client.Chat(ctx, ollamaReq, func(resp api.ChatResponse) error {
				adkResp := m.convertFromOllamaResponse(&resp)
				if !yield(adkResp, nil) {
					return fmt.Errorf("iteration stopped")
				}
				return nil
			})
			if err != nil {
				yield(nil, fmt.Errorf("streaming failed: %w", err))
			}
		} else {
			// Non-streaming mode
			var finalResp api.ChatResponse
			err := m.client.Chat(ctx, ollamaReq, func(resp api.ChatResponse) error {
				finalResp = resp
				return nil
			})
			if err != nil {
				yield(nil, fmt.Errorf("chat failed: %w", err))
				return
			}
			adkResp := m.convertFromOllamaResponse(&finalResp)
			adkResp.TurnComplete = true
			yield(adkResp, nil)
		}
	}
}

// convertToOllamaRequest converts an ADK LLMRequest to an Ollama ChatRequest
func (m *Model) convertToOllamaRequest(req *model.LLMRequest) (*api.ChatRequest, error) {
	messages := make([]api.Message, 0, len(req.Contents))

	for _, content := range req.Contents {
		msg, err := m.convertContentToMessage(content)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	ollamaReq := &api.ChatRequest{
		Model:    m.modelName,
		Messages: messages,
	}

	// Apply configuration if available
	if req.Config != nil {
		if req.Config.Temperature != nil {
			temp := float32(*req.Config.Temperature)
			ollamaReq.Options = map[string]interface{}{
				"temperature": temp,
			}
		}
		if req.Config.MaxOutputTokens != nil {
			if ollamaReq.Options == nil {
				ollamaReq.Options = make(map[string]interface{})
			}
			ollamaReq.Options["num_predict"] = *req.Config.MaxOutputTokens
		}
		if req.Config.TopP != nil {
			if ollamaReq.Options == nil {
				ollamaReq.Options = make(map[string]interface{})
			}
			ollamaReq.Options["top_p"] = float32(*req.Config.TopP)
		}
	}

	return ollamaReq, nil
}

// convertContentToMessage converts a genai.Content to an Ollama Message
func (m *Model) convertContentToMessage(content *genai.Content) (api.Message, error) {
	msg := api.Message{}

	// Map role
	switch content.Role {
	case "user":
		msg.Role = "user"
	case "model", "assistant":
		msg.Role = "assistant"
	case "system":
		msg.Role = "system"
	default:
		msg.Role = "user" // Default to user
	}

	// Extract text from parts
	var textParts []string
	for _, part := range content.Parts {
		if textPart, ok := part.(*genai.TextPart); ok {
			textParts = append(textParts, textPart.Text)
		}
	}

	if len(textParts) > 0 {
		msg.Content = textParts[0] // For now, just use the first text part
		// TODO: Handle multiple parts and non-text parts
	}

	return msg, nil
}

// convertFromOllamaResponse converts an Ollama ChatResponse to an ADK LLMResponse
func (m *Model) convertFromOllamaResponse(resp *api.ChatResponse) *model.LLMResponse {
	adkResp := &model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []genai.Part{
				&genai.TextPart{
					Text: resp.Message.Content,
				},
			},
		},
		TurnComplete: resp.Done,
		Partial:      !resp.Done,
	}

	// Add usage metadata if available
	if resp.Done {
		adkResp.UsageMetadata = &genai.UsageMetadata{
			PromptTokenCount:     int32(resp.PromptEvalCount),
			CandidatesTokenCount: int32(resp.EvalCount),
			TotalTokenCount:      int32(resp.PromptEvalCount + resp.EvalCount),
		}
	}

	return adkResp
}
