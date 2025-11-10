// Package agents provides factory functions for creating ADK agent pipelines
package agents

import (
	"fmt"
	"log/slog"

	"com.github.dimetron.adk-go-agi/pkg/tools"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
)

// PipelineConfig holds configuration for creating a code pipeline agent
type PipelineConfig struct {
	// Model is the LLM model to use for all agents in the pipeline
	Model model.LLM
	// Name is the name of the pipeline agent (defaults to "CodePipelineAgent")
	Name string
	// Description is the description of the pipeline agent
	Description string
}

// NewCodePipelineAgent creates a sequential agent pipeline for code generation, testing, and review
func NewCodePipelineAgent(config PipelineConfig) (agent.Agent, error) {
	// Validate config
	if config.Model == nil {
		return nil, fmt.Errorf("model cannot be nil")
	}

	slog.Info("Creating code pipeline agent",
		"name", config.Name,
		"model", config.Model.Name())

	// Set defaults
	if config.Name == "" {
		config.Name = "CodePipelineAgent"
	}

	if config.Description == "" {
		config.Description = "Executes a sequence of code writing, test generation, and reviewing."
	}

	// Create sub-agents
	slog.Info("Creating design agent")
	designAgent, err := newDesignAgent(config.Model)
	if err != nil {
		slog.Error("Failed to create design agent", "error", err)
		return nil, err
	}
	if designAgent == nil {
		slog.Error("Design agent is nil despite no error")
		return nil, fmt.Errorf("design agent creation returned nil")
	}
	slog.Info("Design agent created successfully")

	slog.Info("Creating code writer agent")
	codeWriterAgent, err := newCodeWriterAgent(config.Model)
	if err != nil {
		slog.Error("Failed to create code writer agent", "error", err)
		return nil, err
	}
	if codeWriterAgent == nil {
		slog.Error("Code writer agent is nil despite no error")
		return nil, fmt.Errorf("code writer agent creation returned nil")
	}
	slog.Info("Code writer agent created successfully")

	slog.Info("Creating TDD expert agent")
	tddExpertAgent, err := newTDDExpertAgent(config.Model)
	if err != nil {
		slog.Error("Failed to create TDD expert agent", "error", err)
		return nil, err
	}
	if tddExpertAgent == nil {
		slog.Error("TDD expert agent is nil despite no error")
		return nil, fmt.Errorf("TDD expert agent creation returned nil")
	}
	slog.Info("TDD expert agent created successfully")

	slog.Info("Creating code reviewer agent")
	codeReviewerAgent, err := newCodeReviewerAgent(config.Model)
	if err != nil {
		slog.Error("Failed to create code reviewer agent", "error", err)
		return nil, err
	}
	if codeReviewerAgent == nil {
		slog.Error("Code reviewer agent is nil despite no error")
		return nil, fmt.Errorf("code reviewer agent creation returned nil")
	}
	slog.Info("Code reviewer agent created successfully")

	// Validate all agents are non-nil before assembling pipeline
	subAgents := []agent.Agent{
		designAgent,
		codeWriterAgent,
		tddExpertAgent,
		codeReviewerAgent,
	}

	for i, ag := range subAgents {
		if ag == nil {
			err := fmt.Errorf("sub-agent at index %d is nil", i)
			slog.Error("Agent validation failed", "error", err, "index", i)
			return nil, err
		}
	}

	slog.Info("Assembling sequential pipeline agent",
		"sub_agents", len(subAgents),
		"pipeline_name", config.Name)

	// Log each sub-agent for debugging
	for i, ag := range subAgents {
		slog.Info("Sub-agent details",
			"index", i,
			"name", ag.Name(),
			"description", ag.Description())
	}

	// Create the sequential pipeline agent
	pipelineAgent, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        config.Name,
			SubAgents:   subAgents,
			Description: config.Description,
		},
	})
	if err != nil {
		slog.Error("Failed to create sequential pipeline agent", "error", err)
		return nil, fmt.Errorf("sequential agent creation failed: %w", err)
	}
	if pipelineAgent == nil {
		slog.Error("Sequential pipeline agent is nil despite no error")
		return nil, fmt.Errorf("sequential pipeline agent creation returned nil")
	}

	slog.Info("Sequential pipeline agent created successfully",
		"name", pipelineAgent.Name(),
		"description", pipelineAgent.Description())

	return pipelineAgent, nil
}

// newDesignAgent creates a design agent that creates a new design for the code
func newDesignAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "DesignAgent",
		Model: model,
		Instruction: `You are a Go Software Architect. Create a high-level design for a Go application. Work completely autonomously without asking for clarification or user input.

**Required Sections:**
1. Architecture Overview - brief description
2. Package Structure - list packages and key files (pkg/, internal/, cmd/)
3. Design Patterns - which patterns to use and where
4. Key Interfaces - main abstractions for testability
5. Dependencies - only essential external packages with justification
6. Error Handling & Concurrency - strategies

**Format Example:**
## Architecture Overview
[description]

## Package Structure
- pkg/user/
  - user.go - domain model
  - repository.go - data access interface

## Design Patterns
- Repository: abstract data access

## Key Interfaces
- UserRepository: CRUD operations

## Dependencies
- none (use stdlib)

**Constraints:**
- Follow Go standard layout
- Minimize dependencies
- Target >85% test coverage
- Include concurrency where beneficial

**IMPORTANT: Complete the entire design now. Do not ask for clarification. Provide a complete, detailed design document covering all required sections.**`,
		Description: "Creates a new design for the code.",
		OutputKey:   "design",
	})
}

// newCodeWriterAgent creates a code writer agent that generates Go code from specifications
func newCodeWriterAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "CodeWriterAgent",
		Model: model,
		Tools: []tool.Tool{
			tools.FileReadTool(),
			tools.FileWriteTool(),
		},
		Instruction: `You are a Go Developer. Implement code from the design below. Use fileWrite to save files. Work completely autonomously without asking questions or waiting for approval.

**Design:**
{design}

**Tools:**
- fileRead: Read existing files
- fileWrite: Save code files (use this for ALL code)

**Process:**
1. Read design to identify files
2. For each file, generate complete Go code
3. Use fileWrite with path and content
4. List all files created at the end

**File Paths:**
- pkg/packagename/file.go - public packages
- internal/packagename/file.go - private packages
- cmd/appname/main.go - main executables

**Code Standards:**
- Add godoc comments for exported items
- Return errors as last value, wrap with %w
- Use interfaces for abstraction
- Prefer composition over inheritance
- Use defer for cleanup
- Keep functions <50 lines
- Validate inputs

**Example fileWrite:**
path: "pkg/user/user.go"
content: "package user\n\n// User represents...\ntype User struct {...}"

**CRITICAL: You MUST generate and save ALL files now. Do not stop until every file from the design is created. Do not ask for confirmation. Complete the entire implementation.**`,
		Description: "Writes initial Go code based on a specification.",
		OutputKey:   "generated_code",
	})
}

// newTDDExpertAgent creates a TDD expert agent that writes comprehensive tests
func newTDDExpertAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "TDDExpertAgent",
		Model: model,
		Tools: []tool.Tool{
			tools.FileReadTool(),
			tools.FileWriteTool(),
		},
		Instruction: `You are a Go Testing Expert. Write tests for code files. Target >85% coverage. Use fileRead to read code, fileWrite to save tests. Work completely autonomously without requesting input.

**Code Reference:**
{generated_code}

**Tools:**
- fileRead: Read .go files
- fileWrite: Save test files

**Process:**
1. Use fileRead on each .go file (skip _test.go)
2. Write tests for each file
3. Use fileWrite to save as filename_test.go in same directory
4. List all test files created

**Test Requirements:**
- Package: use package_test for black-box tests
- Naming: TestFunction_Scenario
- Structure: table-driven tests with t.Run()
- Coverage: all exported items, success/error paths, edge cases
- Format: Arrange-Act-Assert (AAA)

**Table-Driven Test Template:**
tests := []struct {
    name    string
    input   Type
    want    Type
    wantErr bool
}{
    {"valid", validInput, expected, false},
    {"invalid", badInput, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {...})
}

**Test Cases:**
- Happy path and errors
- Nil/empty/zero values
- Boundary conditions
- Use errors.Is() for error checks

**Example fileWrite:**
path: "pkg/user/user_test.go"
content: "package user_test\n\nimport \"testing\"\n\nfunc TestUser_Valid(t *testing.T) {...}"

**MANDATORY: Create ALL test files now. Do not stop until every code file has corresponding tests. Do not ask for permission. Complete all test generation immediately.**`,
		Description: "Writes comprehensive Go tests following TDD best practices.",
		OutputKey:   "test_code",
	})
}

// newCodeReviewerAgent creates a code reviewer agent that provides feedback
func newCodeReviewerAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "CodeReviewerAgent",
		Model: model,
		Tools: []tool.Tool{
			tools.FileReadTool(),
		},
		Instruction: `You are a Senior Go Code Reviewer. Review all code files for correctness, quality, and best practices. Use fileRead to examine files. Work completely autonomously without asking questions.

**Tools:**
- fileRead: Read code files for review

**Process:**
1. Use fileRead on all .go files (code and tests)
2. Check each file against review criteria
3. Provide structured feedback

**Code Reference:**
{generated_code}

**Review Criteria:**
- Correctness: logic errors, bugs, proper error handling
- Go Idioms: interfaces, composition, error wrapping (%w), defer usage
- Quality: readable code, descriptive names, functions <50 lines, no duplication
- Documentation: godoc comments for all exported items
- Edge Cases: nil/empty/zero values, input validation
- Performance: unnecessary allocations, efficient data structures
- Concurrency: proper goroutine/channel usage, race condition checks
- Security: input validation, injection prevention
- Testability: dependency injection, minimal side effects

**Output Format:**
## Critical Issues (Must Fix)
- [file:function] [specific issue and fix]

## Suggestions (Should Consider)
- [file] [improvement with rationale]

## Positive Observations
- [what works well]

If no issues: "No major issues found. Code follows Go best practices."

Be specific, constructive, and actionable.

**REQUIRED: Complete the full review now. Read ALL files and provide comprehensive feedback. Do not ask for clarification. Finish the entire code review process immediately.**`,
		Description: "Reviews code and provides feedback.",
		OutputKey:   "review_comments",
	})
}
