

# AGI Agent - ADK-GO AI Agent

## Official ADK Go Repository

https://github.com/google/adk-go

## Overview

This project provides an AGI agent implementation using Google's ADK framework with support for Ollama LLMs. The agent implements a code generation pipeline with specialized sub-agents for writing, testing, reviewing, and refactoring code.

## Features

- **Ollama LLM Integration**: Native support for Ollama models (llama2, mistral, etc.)
- **Code Pipeline Agent**: Sequential workflow with specialized sub-agents:
  - Code Writer Agent: Generates Go code from specifications
  - TDD Expert Agent: Creates comprehensive tests following TDD best practices
  - Code Reviewer Agent: Provides constructive feedback on code quality
  - Code Refactorer Agent: Improves code based on review comments

## Prerequisites

1. **Ollama**: Install and run Ollama locally
   ```bash
   # Install Ollama (see https://ollama.ai)
   curl -fsSL https://ollama.ai/install.sh | sh

   # Pull a model (e.g., llama2)
   ollama pull llama2

   # Start Ollama server (runs on http://localhost:11434 by default)
   ollama serve
   ```

2. **Go**: Version 1.25.3 or higher

## Getting Started

### Configuration

Configure the agent using environment variables:

```bash
# Set the Ollama model to use (default: llama2)
export OLLAMA_MODEL=llama2

# Set the Ollama server URL (default: http://localhost:11434)
export OLLAMA_BASE_URL=http://localhost:11434
```

### Running the Agent

Use the Makefile to run the agent:

```bash
make run
```

Or run directly:

```bash
go run cmd/agi.go
```

## Supported Ollama Models

The agent works with any Ollama model. Popular choices include:

- `llama2` - Meta's Llama 2 model
- `mistral` - Mistral 7B model
- `codellama` - Code-specialized Llama model
- `phi` - Microsoft's Phi model
- `neural-chat` - Intel's Neural Chat model

To use a different model:

```bash
export OLLAMA_MODEL=mistral
make run
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

### Project Structure

```
.
├── cmd/
│   └── agi.go              # Main application entry point
├── pkg/
│   ├── agents/             # Agent implementations
│   │   ├── pipeline.go     # Code pipeline agent
│   │   └── pipeline_test.go
│   └── model/
│       └── ollama/         # Ollama LLM integration
│           ├── ollama.go
│           └── ollama_test.go
└── test/
    └── e2e/                # End-to-end tests
```

## Architecture

The Ollama LLM integration implements the `model.LLM` interface from Google's ADK framework, providing:

- **Request conversion**: Transforms ADK LLMRequest to Ollama ChatRequest
- **Response conversion**: Converts Ollama responses to ADK LLMResponse format
- **Streaming support**: Handles both streaming and non-streaming modes
- **Configuration mapping**: Maps ADK config (temperature, max_tokens, top_p) to Ollama options

## License

Copyright 2025 Google LLC - Apache License 2.0
