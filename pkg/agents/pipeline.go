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

// Package agents provides factory functions for creating ADK agent pipelines
package agents

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
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

// NewCodePipelineAgent creates a sequential agent pipeline for code generation, testing, review, and refactoring
func NewCodePipelineAgent(config PipelineConfig) (agent.Agent, error) {
	// Set defaults
	if config.Name == "" {
		config.Name = "CodePipelineAgent"
	}
	if config.Description == "" {
		config.Description = "Executes a sequence of code writing, test generation, reviewing, and refactoring."
	}

	// Create sub-agents
	codeWriterAgent, err := newCodeWriterAgent(config.Model)
	if err != nil {
		return nil, err
	}

	tddExpertAgent, err := newTDDExpertAgent(config.Model)
	if err != nil {
		return nil, err
	}

	codeReviewerAgent, err := newCodeReviewerAgent(config.Model)
	if err != nil {
		return nil, err
	}

	codeRefactorerAgent, err := newCodeRefactorerAgent(config.Model)
	if err != nil {
		return nil, err
	}

	// Create the sequential pipeline agent
	return sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name: config.Name,
			SubAgents: []agent.Agent{
				codeWriterAgent,
				tddExpertAgent,
				codeReviewerAgent,
				codeRefactorerAgent,
			},
			Description: config.Description,
		},
	})
}

// newCodeWriterAgent creates a code writer agent that generates Go code from specifications
func newCodeWriterAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "CodeWriterAgent",
		Model: model,
		Instruction: `You are a Go Code Generator.
Based *only* on the user's request, write Go code that fulfills the requirement.
Output *only* the complete Go code block, enclosed in triple backticks ('''go ... ''').
Do not add any other text before or after the code block.`,
		Description: "Writes initial Go code based on a specification.",
		OutputKey:   "generated_code",
	})
}

// newTDDExpertAgent creates a TDD expert agent that writes comprehensive tests
func newTDDExpertAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "TDDExpertAgent",
		Model: model,
		Instruction: `You are a Go TDD (Test-Driven Development) Expert.
Your task is to write comprehensive, high-quality tests for the provided Go code.

**Code to Test:**
'''go
{generated_code}
'''

**Testing Requirements:**
1.  **Test Coverage:** Write tests that cover all major functions, methods, and code paths.
2.  **Table-Driven Tests:** Use table-driven tests where appropriate (Go best practice).
3.  **Edge Cases:** Include tests for edge cases, boundary conditions, and error scenarios.
4.  **Clear Names:** Use descriptive test function names following the pattern Test<FunctionName>_<Scenario>.
5.  **Assertions:** Use clear assertions and helpful error messages.
6.  **Test Structure:** Follow the Arrange-Act-Assert (AAA) pattern.
7.  **Subtests:** Use t.Run() for organizing related test cases.
8.  **Benchmarks:** Include benchmark tests for performance-critical functions (optional but recommended).

**Output:**
Output *only* the complete Go test code block, enclosed in triple backticks ('''go ... ''').
The test file should include:
- Proper package declaration (package <name>_test or package <name>)
- All necessary imports
- Well-structured test functions
- Helper functions if needed
Do not add any other text before or after the code block.`,
		Description: "Writes comprehensive Go tests following TDD best practices.",
		OutputKey:   "test_code",
	})
}

// newCodeReviewerAgent creates a code reviewer agent that provides feedback
func newCodeReviewerAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "CodeReviewerAgent",
		Model: model,
		Instruction: `You are an expert Go Code Reviewer.
Your task is to provide constructive feedback on the provided code.

**Code to Review:**
'''go
{generated_code}
'''

**Review Criteria:**
1.  **Correctness:** Does the code work as intended? Are there logic errors?
2.  **Readability:** Is the code clear and easy to understand? Follows Go style guidelines and gofmt formatting?
3.  **Efficiency:** Is the code reasonably efficient? Any obvious performance bottlenecks?
4.  **Edge Cases:** Does the code handle potential edge cases or invalid inputs gracefully? Proper error handling?
5.  **Best Practices:** Does the code follow common Go best practices and idioms?

**Output:**
Provide your feedback as a concise, bulleted list. Focus on the most important points for improvement.
If the code is excellent and requires no changes, simply state: "No major issues found."
Output *only* the review comments or the "No major issues" statement.`,
		Description: "Reviews code and provides feedback.",
		OutputKey:   "review_comments",
	})
}

// newCodeRefactorerAgent creates a code refactorer agent that improves code based on feedback
func newCodeRefactorerAgent(model model.LLM) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:  "CodeRefactorerAgent",
		Model: model,
		Instruction: `You are a Go Code Refactoring AI.
Your goal is to improve the given Go code based on the provided review comments.

**Original Code:**
'''go
{generated_code}
'''

**Review Comments:**
{review_comments}

**Task:**
Carefully apply the suggestions from the review comments to refactor the original code.
If the review comments state "No major issues found," return the original code unchanged.
Ensure the final code is complete, functional, and includes necessary imports and proper documentation.

**Output:**
Output *only* the final, refactored Go code block, enclosed in triple backticks ('''go ... ''').
Do not add any other text before or after the code block.`,
		Description: "Refactors code based on review comments.",
		OutputKey:   "refactored_code",
	})
}
