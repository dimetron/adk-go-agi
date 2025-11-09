# Ollama Integration Guide

This document provides detailed information about the Ollama LLM integration in the ADK-GO AGI Agent.

## Overview

The Ollama integration enables the ADK-GO AGI Agent to use locally-running Ollama models instead of cloud-based LLMs. This provides:

- **Privacy**: All data stays on your local machine
- **Cost**: No API costs for LLM usage
- **Flexibility**: Support for any Ollama model
- **Speed**: Low-latency responses from local inference

## Architecture

The Ollama integration is implemented in `pkg/model/ollama/` and consists of:

### Core Components

1. **Model struct**: Implements the `model.LLM` interface from ADK
2. **Config struct**: Configuration options for Ollama connection
3. **Conversion functions**: Transform between ADK and Ollama formats

### Interface Implementation

```go
type Model struct {
    client    *api.Client
    modelName string
}

func (m *Model) Name() string
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error]
```

## Request Flow

1. **Receive LLMRequest**: ADK framework provides request with genai.Content
2. **Convert to Ollama format**: Transform genai.Content to Ollama messages
3. **Call Ollama API**: Use Ollama Go client to send request
4. **Convert response**: Transform Ollama response back to genai.Content
5. **Return LLMResponse**: Provide response to ADK framework

## Configuration Options

### Environment Variables

- `OLLAMA_MODEL`: Model name (default: `llama2`)
- `OLLAMA_BASE_URL`: Server URL (default: `http://localhost:11434`)

### Programmatic Configuration

```go
model, err := ollama.NewModel(ctx, ollama.Config{
    BaseURL:   "http://localhost:11434",
    ModelName: "llama2",
    HTTPClient: &http.Client{
        Timeout: 5 * time.Minute,
    },
})
```

## Supported Features

### ADK Features

- ✅ Temperature configuration
- ✅ Max tokens (num_predict)
- ✅ Top-p sampling
- ✅ Streaming responses
- ✅ Non-streaming responses
- ✅ Multiple message roles (user, assistant, system)
- ✅ Usage metadata

### Limitations

- ⚠️ Multi-part messages: Currently uses only first text part
- ⚠️ Image/media parts: Not yet supported
- ⚠️ Function calling: Depends on Ollama model support
- ⚠️ Citations and grounding: Not applicable to Ollama

## Model Selection

Choose an Ollama model based on your needs:

### Code Generation
- `codellama:7b` - Fast, good for simple code tasks
- `codellama:13b` - Better quality, slower
- `deepseek-coder` - Excellent for code

### General Purpose
- `llama2:7b` - Fast, general purpose
- `llama2:13b` - Better quality
- `mistral:7b` - Good balance of speed/quality
- `mixtral:8x7b` - High quality, requires more resources

### Specialized
- `phi:2.7b` - Very fast, small model
- `neural-chat:7b` - Optimized for chat
- `orca-mini` - Reasoning and explanation

## Performance Tuning

### GPU Acceleration

Ollama automatically uses GPU if available. Check with:

```bash
ollama list
nvidia-smi  # For NVIDIA GPUs
```

### Memory Management

- Smaller models (7B): ~8GB RAM
- Medium models (13B): ~16GB RAM
- Large models (70B): ~64GB RAM

### Streaming vs Non-Streaming

- **Streaming**: Better UX for long responses, progressive output
- **Non-streaming**: Simpler error handling, complete response at once

## Troubleshooting

### Connection Errors

```
Error: failed to connect to Ollama server
```

**Solution**: Ensure Ollama is running:
```bash
ollama serve
```

### Model Not Found

```
Error: model 'xyz' not found
```

**Solution**: Pull the model first:
```bash
ollama pull llama2
```

### Out of Memory

```
Error: failed to load model
```

**Solution**: Use a smaller model or increase system memory

### Slow Responses

- Use smaller models
- Enable GPU acceleration
- Reduce max_tokens
- Use streaming mode

## Testing

The Ollama integration includes comprehensive tests:

```bash
# Run all Ollama tests
go test -v ./pkg/model/ollama/

# Run with coverage
go test -v -coverprofile=coverage.out ./pkg/model/ollama/
go tool cover -html=coverage.out
```

### Test Coverage

- Model creation and configuration
- Name formatting
- Request conversion
- Response conversion
- Error handling
- Configuration mapping (temperature, max_tokens, top_p)
- Message role handling (user, assistant, system)

## Examples

### Basic Usage

```go
ctx := context.Background()

// Create Ollama model
model, err := ollama.NewModel(ctx, ollama.Config{
    ModelName: "llama2",
})
if err != nil {
    log.Fatal(err)
}

// Create request
req := &model.LLMRequest{
    Contents: []*genai.Content{
        {
            Role: "user",
            Parts: []genai.Part{
                &genai.TextPart{Text: "Write a hello world function in Go"},
            },
        },
    },
}

// Generate response (non-streaming)
for resp, err := range model.GenerateContent(ctx, req, false) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Content)
}
```

### With Configuration

```go
temp := 0.7
maxTokens := int32(500)

req := &model.LLMRequest{
    Contents: []*genai.Content{
        {
            Role: "user",
            Parts: []genai.Part{
                &genai.TextPart{Text: "Explain how Go channels work"},
            },
        },
    },
    Config: &genai.GenerateContentConfig{
        Temperature:     &temp,
        MaxOutputTokens: &maxTokens,
    },
}
```

### Streaming Mode

```go
// Generate response (streaming)
for resp, err := range model.GenerateContent(ctx, req, true) {
    if err != nil {
        log.Fatal(err)
    }
    if resp.Partial {
        fmt.Print(resp.Content)  // Print partial response
    } else {
        fmt.Println(resp.Content)  // Final response
    }
}
```

## Future Enhancements

Potential improvements for the Ollama integration:

1. **Multi-modal support**: Handle image and audio parts
2. **Function calling**: Support for Ollama models with function calling
3. **Model management**: Auto-pull models if not available
4. **Health checks**: Verify Ollama server availability
5. **Retry logic**: Automatic retry with exponential backoff
6. **Metrics**: Track token usage and latency
7. **Model switching**: Dynamic model selection based on task
8. **Embeddings**: Support for Ollama embedding models

## Contributing

To improve the Ollama integration:

1. Add tests for new features
2. Update documentation
3. Follow Go best practices
4. Ensure backward compatibility
5. Add examples for new functionality

## Resources

- [Ollama Documentation](https://github.com/ollama/ollama)
- [Ollama API Reference](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [ADK Documentation](https://github.com/google/adk-go)
- [Available Ollama Models](https://ollama.ai/library)
