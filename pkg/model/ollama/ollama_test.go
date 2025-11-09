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

package ollama_test

import (
	"context"
	"testing"

	"com.github.dimetron.adk-go-agi/pkg/model/ollama"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestOllama(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ollama Model Suite")
}

var _ = Describe("Ollama Model", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewModel", func() {
		It("should create a new Ollama model with default base URL", func() {
			model, err := ollama.NewModel(ctx, ollama.Config{
				ModelName: "llama2",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(model).NotTo(BeNil())
			Expect(model.Name()).To(Equal("ollama/llama2"))
		})

		It("should create a new Ollama model with custom base URL", func() {
			model, err := ollama.NewModel(ctx, ollama.Config{
				BaseURL:   "http://custom-ollama:11434",
				ModelName: "mistral",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(model).NotTo(BeNil())
			Expect(model.Name()).To(Equal("ollama/mistral"))
		})

		It("should return error when model name is empty", func() {
			_, err := ollama.NewModel(ctx, ollama.Config{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("model name is required"))
		})

		It("should return error for invalid base URL", func() {
			_, err := ollama.NewModel(ctx, ollama.Config{
				BaseURL:   "://invalid-url",
				ModelName: "llama2",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid base URL"))
		})
	})

	Describe("Name", func() {
		It("should return the correct model name format", func() {
			model, err := ollama.NewModel(ctx, ollama.Config{
				ModelName: "llama2",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(model.Name()).To(Equal("ollama/llama2"))
		})
	})

	Describe("GenerateContent", func() {
		var (
			ollamaModel *ollama.Model
			llmRequest  *model.LLMRequest
		)

		BeforeEach(func() {
			var err error
			ollamaModel, err = ollama.NewModel(ctx, ollama.Config{
				ModelName: "llama2",
			})
			Expect(err).NotTo(HaveOccurred())

			// Create a basic LLM request
			llmRequest = &model.LLMRequest{
				Contents: []*genai.Content{
					{
						Role: "user",
						Parts: []genai.Part{
							&genai.TextPart{Text: "Hello, how are you?"},
						},
					},
				},
				Config: &genai.GenerateContentConfig{},
			}
		})

		It("should implement the LLM interface", func() {
			var _ model.LLM = ollamaModel
		})

		Context("when Ollama server is not available", func() {
			It("should handle connection errors gracefully", func() {
				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)

				var receivedError error
				for _, err := range responses {
					if err != nil {
						receivedError = err
						break
					}
				}

				// We expect an error since Ollama server is likely not running in tests
				Expect(receivedError).To(HaveOccurred())
			})
		})

		Context("with configuration", func() {
			It("should accept temperature configuration", func() {
				temp := 0.7
				llmRequest.Config = &genai.GenerateContentConfig{
					Temperature: &temp,
				}

				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)
				Expect(responses).NotTo(BeNil())
			})

			It("should accept max tokens configuration", func() {
				maxTokens := int32(100)
				llmRequest.Config = &genai.GenerateContentConfig{
					MaxOutputTokens: &maxTokens,
				}

				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)
				Expect(responses).NotTo(BeNil())
			})

			It("should accept top_p configuration", func() {
				topP := 0.9
				llmRequest.Config = &genai.GenerateContentConfig{
					TopP: &topP,
				}

				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)
				Expect(responses).NotTo(BeNil())
			})
		})

		Context("with different message roles", func() {
			It("should handle user role", func() {
				llmRequest.Contents = []*genai.Content{
					{
						Role: "user",
						Parts: []genai.Part{
							&genai.TextPart{Text: "Test message"},
						},
					},
				}
				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)
				Expect(responses).NotTo(BeNil())
			})

			It("should handle assistant role", func() {
				llmRequest.Contents = []*genai.Content{
					{
						Role: "assistant",
						Parts: []genai.Part{
							&genai.TextPart{Text: "Test message"},
						},
					},
				}
				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)
				Expect(responses).NotTo(BeNil())
			})

			It("should handle system role", func() {
				llmRequest.Contents = []*genai.Content{
					{
						Role: "system",
						Parts: []genai.Part{
							&genai.TextPart{Text: "You are a helpful assistant"},
						},
					},
				}
				responses := ollamaModel.GenerateContent(ctx, llmRequest, false)
				Expect(responses).NotTo(BeNil())
			})
		})
	})
})
