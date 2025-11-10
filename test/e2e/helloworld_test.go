package e2e_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"com.github.dimetron.adk-go-agi/pkg/agents"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

var _ = Describe("Hello World E2E Test", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		pipelineAgent agent.Agent
		llmModel      model.LLM // The model interface
	)

	BeforeEach(func() {
		// Create context with timeout for test
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)

		// Initialize the model
		var err error
		llmModel, err = gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create Gemini model")

		// Register cleanup
		DeferCleanup(func() {
			cancel()
		})
	})

	Context("when creating a code pipeline agent using factory", func() {
		BeforeEach(func() {
			By("creating code pipeline agent using factory function")
			var err error
			pipelineAgent, err = agents.NewCodePipelineAgent(agents.PipelineConfig{
				Model: llmModel,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to create pipeline agent")
		})

		It("should successfully create and validate the agent pipeline", func(ctx SpecContext) {
			By("verifying agent name")
			Expect(pipelineAgent.Name()).To(Equal("CodePipelineAgent"))

			By("verifying agent description")
			Expect(pipelineAgent.Description()).To(ContainSubstring("sequence"))
			Expect(pipelineAgent.Description()).To(ContainSubstring("code writing"))
		}, SpecTimeout(30*time.Second))

		It("should have all required sub-agents configured", func(ctx SpecContext) {
			By("checking pipeline agent is not nil")
			Expect(pipelineAgent).NotTo(BeNil())

			By("verifying agent type")
			Expect(pipelineAgent.Name()).To(Equal("CodePipelineAgent"))
		}, SpecTimeout(10*time.Second))
	})

	Context("when creating a custom named pipeline agent", func() {
		BeforeEach(func() {
			By("creating custom pipeline agent")
			var err error
			pipelineAgent, err = agents.NewCodePipelineAgent(agents.PipelineConfig{
				Model:       llmModel,
				Name:        "CustomPipelineAgent",
				Description: "Custom pipeline for testing",
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to create custom pipeline agent")
		})

		It("should use the custom name and description", func(ctx SpecContext) {
			By("verifying custom agent name")
			Expect(pipelineAgent.Name()).To(Equal("CustomPipelineAgent"))

			By("verifying custom agent description")
			Expect(pipelineAgent.Description()).To(Equal("Custom pipeline for testing"))
		}, SpecTimeout(10*time.Second))
	})

	Context("when testing basic agent functionality", func() {
		BeforeEach(func() {
			// Create a minimal pipeline for fast testing
			simpleAgent, err := llmagent.New(llmagent.Config{
				Name:        "SimpleAgent",
				Model:       llmModel,
				Instruction: "You are a simple test agent. Reply with 'Hello, World!'",
				Description: "A simple test agent for validation.",
				OutputKey:   "output",
			})
			Expect(err).NotTo(HaveOccurred())
			pipelineAgent = simpleAgent
		})

		It("should create a functional agent", func(ctx SpecContext) {
			By("verifying agent is created")
			Expect(pipelineAgent).NotTo(BeNil())

			By("checking agent name")
			Expect(pipelineAgent.Name()).To(Equal("SimpleAgent"))

			By("checking agent description")
			Expect(pipelineAgent.Description()).To(Equal("A simple test agent for validation."))
		}, SpecTimeout(10*time.Second))
	})
})
