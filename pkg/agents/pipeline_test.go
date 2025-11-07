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

package agents

import (
	"context"
	"testing"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

func TestNewCodePipelineAgent(t *testing.T) {
	ctx := context.Background()

	// Initialize the Gemini model for testing
	model, err := gemini.NewModel(ctx, "gemini-2.5-pro", &genai.ClientConfig{})
	if err != nil {
		t.Skipf("Skipping test: failed to create model: %v", err)
	}

	tests := []struct {
		name     string
		config   PipelineConfig
		wantName string
		wantDesc string
		wantErr  bool
	}{
		{
			name: "default configuration",
			config: PipelineConfig{
				Model: model,
			},
			wantName: "CodePipelineAgent",
			wantDesc: "Executes a sequence of code writing, test generation, reviewing, and refactoring.",
			wantErr:  false,
		},
		{
			name: "custom name and description",
			config: PipelineConfig{
				Model:       model,
				Name:        "CustomPipeline",
				Description: "Custom pipeline description",
			},
			wantName: "CustomPipeline",
			wantDesc: "Custom pipeline description",
			wantErr:  false,
		},
		{
			name: "empty name uses default",
			config: PipelineConfig{
				Model: model,
				Name:  "",
			},
			wantName: "CodePipelineAgent",
			wantDesc: "Executes a sequence of code writing, test generation, reviewing, and refactoring.",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewCodePipelineAgent(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCodePipelineAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if agent == nil {
					t.Fatal("NewCodePipelineAgent() returned nil agent")
				}

				if got := agent.Name(); got != tt.wantName {
					t.Errorf("Agent.Name() = %v, want %v", got, tt.wantName)
				}

				if got := agent.Description(); got != tt.wantDesc {
					t.Errorf("Agent.Description() = %v, want %v", got, tt.wantDesc)
				}
			}
		})
	}
}

func TestNewCodePipelineAgent_NilModel(t *testing.T) {
	config := PipelineConfig{
		Model: nil,
	}

	// This should panic or return an error when the model is used
	// We test that the function doesn't panic during creation
	agent, err := NewCodePipelineAgent(config)

	// The agent creation might succeed but sub-agent creation will fail
	if err == nil && agent != nil {
		t.Log("Agent creation succeeded with nil model, sub-agents may fail at runtime")
	}
}

func TestSubAgentCreation(t *testing.T) {
	ctx := context.Background()

	llmModel, err := gemini.NewModel(ctx, "gemini-2.5-pro", &genai.ClientConfig{})
	if err != nil {
		t.Skipf("Skipping test: failed to create model: %v", err)
	}

	tests := []struct {
		name    string
		factory func(model.LLM) (agent.Agent, error)
		wantErr bool
	}{
		{
			name:    "code writer agent",
			factory: newCodeWriterAgent,
			wantErr: false,
		},
		{
			name:    "TDD expert agent",
			factory: newTDDExpertAgent,
			wantErr: false,
		},
		{
			name:    "code reviewer agent",
			factory: newCodeReviewerAgent,
			wantErr: false,
		},
		{
			name:    "code refactorer agent",
			factory: newCodeRefactorerAgent,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag, err := tt.factory(llmModel)
			if (err != nil) != tt.wantErr {
				t.Errorf("factory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && ag == nil {
				t.Fatal("factory() returned nil agent")
			}
		})
	}
}

// Benchmark for agent creation
func BenchmarkNewCodePipelineAgent(b *testing.B) {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-pro", &genai.ClientConfig{})
	if err != nil {
		b.Skipf("Skipping benchmark: failed to create model: %v", err)
	}

	config := PipelineConfig{
		Model: model,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewCodePipelineAgent(config)
		if err != nil {
			b.Fatalf("NewCodePipelineAgent() failed: %v", err)
		}
	}
}
