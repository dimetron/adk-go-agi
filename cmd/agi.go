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

package main

import (
	"context"
	"log"
	"os"

	"com.github.dimetron.adk-go-agi/pkg/agents"
	"com.github.dimetron.adk-go-agi/pkg/model/ollama"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/server/restapi/services"
)

func main() {
	ctx := context.Background()

	// Get model name from environment variable or use default
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama2" // Default model
	}

	// Get Ollama base URL from environment variable or use default
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Initialize the Ollama model
	model, err := ollama.NewModel(ctx, ollama.Config{
		BaseURL:   baseURL,
		ModelName: modelName,
	})
	if err != nil {
		log.Fatalf("failed to create Ollama model: %s", err)
	}

	log.Printf("Using Ollama model: %s at %s", modelName, baseURL)

	// Create the code pipeline agent using the factory function
	rootAgent, err := agents.NewCodePipelineAgent(agents.PipelineConfig{
		Model: model,
	})
	if err != nil {
		log.Fatalf("failed to create code pipeline agent: %s", err)
	}

	// The rootAgent can now be used by the ADK framework.
	log.Printf("Successfully created root agent: %s", rootAgent.Name())

	config := &adk.Config{
		AgentLoader: services.NewSingleAgentLoader(rootAgent),
	}
	l := full.NewLauncher()
	err = l.Execute(ctx, config, os.Args[1:])
	if err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}

}
