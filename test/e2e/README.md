# E2E Tests for ADK-Go-AGI

This directory contains end-to-end (E2E) tests for the ADK-Go-AGI code generation agent pipeline.

## Overview

The E2E tests validate the functionality of the agent pipeline, including:
- Agent creation and configuration
- Sequential agent pipeline orchestration
- LLM model integration
- Multi-agent workflows (code writing, testing, reviewing, refactoring)

## Test Structure

The tests follow Ginkgo/Gomega best practices as outlined in `.cursor/rules/e2e.test.mdc`:

- **Suite File**: `e2e_suite_test.go` - Initializes the Ginkgo test runner
- **Test Files**: `*_test.go` - Contains test specifications

### Current Tests

#### Hello World Test (`helloworld_test.go`)
Tests the basic functionality of creating and configuring the agent pipeline:

1. **Agent Pipeline Creation**
   - Tests creating a sequential pipeline with multiple sub-agents
   - Validates agent configuration and initialization
   - Verifies agent naming and descriptions

2. **Simple Agent Functionality**
   - Tests basic agent creation
   - Validates agent properties and configuration

## Running Tests

### Run All E2E Tests
```bash
make e2e
```

### Run with Verbose Output
```bash
make e2e-verbose
```

### Run Specific Test Files
```bash
go test -v ./test/e2e/helloworld_test.go ./test/e2e/e2e_suite_test.go
```

### Run with Ginkgo CLI
```bash
ginkgo -v ./test/e2e/
```

## Test Best Practices

Following the project's e2e testing guidelines:

1. **Declare in Containers, Initialize in Setup**
   - Variables declared in `Describe` blocks
   - Initialized in `BeforeEach` blocks

2. **Resource Cleanup**
   - Use `DeferCleanup` for automatic resource cleanup
   - Context cancellation for timeout management

3. **Async Operations**
   - Use `SpecTimeout` for long-running operations
   - Context-aware test execution

4. **Clear Test Documentation**
   - Use `By` statements to document test steps
   - Descriptive test names and descriptions

## Dependencies

- **Ginkgo v2**: BDD-style test framework
- **Gomega**: Matcher/assertion library
- **ADK Go SDK**: Agent Development Kit
- **Gemini Model**: LLM integration

## Environment Requirements

- Go 1.25.3 or later
- Google Cloud credentials (for Gemini model access)
- Internet connection (for LLM API calls)

## Test Timeout

Default timeout for E2E tests is 10 minutes (configurable in Makefile).
Individual test specs can override with `SpecTimeout()`.

## Future Test Additions

Planned test scenarios:
- Full pipeline execution with actual code generation
- Error handling and edge cases
- Performance and load testing
- Integration with different LLM models
- Agent state management and persistence

