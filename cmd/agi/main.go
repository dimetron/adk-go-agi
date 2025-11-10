package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"com.github.dimetron.adk-go-agi/pkg/agents"
	ollamamodel "com.github.dimetron.adk-go-agi/pkg/model/ollama"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/server/restapi/services"
)

func main() {
	// Create context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize the Ollama model using the official Ollama Go API client
	// You can change the model name to any model you have installed in Ollama
	// Examples: "llama3.2", "mistral", "codellama", "gemma2", "qwen2.5-coder", etc.
	ollamaBaseURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaBaseURL == "" {
		ollamaBaseURL = "http://localhost:11434"
	}

	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		//modelName = "gpt-oss:120b-cloud" // Default Ollama model
		modelName = "gpt-oss:120b-cloud"
	}

	log.Printf("Initializing Ollama model: %s at %s", modelName, ollamaBaseURL)

	model, err := ollamamodel.NewModel(ctx, &ollamamodel.Config{
		ModelName: modelName,
		BaseURL:   ollamaBaseURL,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.9,
		},
	})
	if err != nil {
		log.Fatalf("failed to create Ollama model: %s", err)
	}

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
