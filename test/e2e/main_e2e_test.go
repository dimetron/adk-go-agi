package e2e_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	mainTestBinaryPath string
)

var _ = BeforeSuite(func() {
	By("building the AGI binary for main CLI tests")
	// Get the workspace root (3 levels up from test/e2e/)
	workspaceRoot, err := filepath.Abs(filepath.Join("..", ".."))
	Expect(err).NotTo(HaveOccurred(), "Failed to get workspace root")

	mainTestBinaryPath = filepath.Join(workspaceRoot, "bin", "agi")

	// Build the binary
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = workspaceRoot
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		GinkgoWriter.Printf("Build output: %s\n", string(buildOutput))
	}
	Expect(err).NotTo(HaveOccurred(), "Failed to build AGI binary")
	Expect(mainTestBinaryPath).To(BeAnExistingFile(), "Binary not found at expected path")
})

var _ = Describe("Main CLI E2E Test", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		binaryPath string
		port       int
		baseURL    string
	)

	BeforeEach(func() {
		// Create context with timeout for each test
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)

		// Use the global binary path from BeforeSuite
		binaryPath = mainTestBinaryPath

		// Use a dynamic port for parallel test execution
		port = 9090 + GinkgoParallelProcess()
		baseURL = fmt.Sprintf("http://localhost:%d", port)

		DeferCleanup(func() {
			cancel()
		})
	})

	Context("when starting the AGI server", func() {
		var (
			cmd    *exec.Cmd
			cmdCtx context.Context
		)

		BeforeEach(func() {
			By("starting AGI server in background")
			// Create a context for the command that we can cancel
			cmdCtx = ctx

			// Build the command arguments
			args := []string{
				"web",
				"-port", fmt.Sprintf("%d", port),
				"api",
				"-webui_address", "localhost",
				"a2a",
				"-a2a_agent_url", fmt.Sprintf("http://localhost:%d", port),
				"webui",
				"-api_server_address", fmt.Sprintf("http://localhost:%d/api", port),
			}

			cmd = exec.CommandContext(cmdCtx, binaryPath, args...)

			// Capture stdout and stderr for debugging
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter

			// Start the server
			err := cmd.Start()
			Expect(err).NotTo(HaveOccurred(), "Failed to start AGI server")

			By("waiting for server to be ready")
			// Wait for the server to be ready by polling the API endpoint
			Eventually(func() error {
				client := &http.Client{Timeout: 2 * time.Second}
				resp, err := client.Get(fmt.Sprintf("%s/api/list-apps", baseURL))
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				// Accept any 2xx or 3xx status code as "ready"
				if resp.StatusCode >= 400 {
					return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				}
				return nil
			}).WithTimeout(30 * time.Second).WithPolling(1 * time.Second).Should(Succeed())

			// Clean up the process
			DeferCleanup(func() {
				if cmd.Process != nil {
					By("stopping AGI server")
					// Send interrupt signal
					_ = cmd.Process.Signal(os.Interrupt)

					// Wait for graceful shutdown with timeout
					done := make(chan error, 1)
					go func() {
						done <- cmd.Wait()
					}()

					select {
					case <-done:
						// Process exited
					case <-time.After(5 * time.Second):
						// Force kill if not stopped
						_ = cmd.Process.Kill()
						<-done
					}
				}
			})
		})

		It("should respond to list-apps endpoint", func(ctx SpecContext) {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(fmt.Sprintf("%s/api/list-apps", baseURL))
			Expect(err).NotTo(HaveOccurred(), "List apps request failed")
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(BeNumerically("<", 400), "List apps should return success")

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred(), "Failed to read list apps response")
			GinkgoWriter.Printf("List apps response: %s\n", string(body))
		}, SpecTimeout(10*time.Second))

		It("should expose the API endpoint", func(ctx SpecContext) {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(fmt.Sprintf("%s/api", baseURL))
			Expect(err).NotTo(HaveOccurred(), "API request failed")
			defer resp.Body.Close()

			// API endpoint should be accessible (even if it returns different status codes)
			Expect(resp.StatusCode).To(BeNumerically("<", 500), "API endpoint should be accessible")
		}, SpecTimeout(10*time.Second))

		It("should expose the WebUI", func(ctx SpecContext) {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(fmt.Sprintf("%s/ui/?app=CodePipelineAgent", baseURL))
			Expect(err).NotTo(HaveOccurred(), "WebUI request failed")
			defer resp.Body.Close()

			// WebUI should be accessible
			Expect(resp.StatusCode).To(BeNumerically("<", 500), "WebUI should be accessible")
		}, SpecTimeout(10*time.Second))

		It("should have CodePipelineAgent available", func(ctx SpecContext) {
			By("checking agent endpoint")
			client := &http.Client{Timeout: 5 * time.Second}

			// Try to get agent info (endpoint may vary based on ADK API)
			resp, err := client.Get(fmt.Sprintf("%s/api/agents", baseURL))
			if err != nil {
				// If agents endpoint doesn't exist, skip this test
				Skip("Agent listing endpoint not available")
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				bodyStr := string(body)

				// Check if CodePipelineAgent is mentioned in the response
				Expect(bodyStr).To(ContainSubstring("CodePipelineAgent"),
					"CodePipelineAgent should be listed in available agents")

				GinkgoWriter.Printf("Agents response: %s\n", bodyStr)
			}
		}, SpecTimeout(10*time.Second))
	})

	Context("when testing binary existence and permissions", func() {
		It("should have executable permissions", func(ctx SpecContext) {
			info, err := os.Stat(binaryPath)
			Expect(err).NotTo(HaveOccurred(), "Failed to stat binary")

			// Check if file is executable
			mode := info.Mode()
			Expect(mode.IsRegular()).To(BeTrue(), "Binary should be a regular file")
			Expect(mode.Perm()&0111).NotTo(BeZero(), "Binary should have execute permissions")
		}, SpecTimeout(5*time.Second))

		It("should display help when run with --help", func(ctx SpecContext) {
			cmd := exec.CommandContext(ctx, binaryPath, "--help")
			output, _ := cmd.CombinedOutput()

			// --help should exit with code 0 or 1 (depends on implementation)
			// We just check that it produces output
			Expect(output).NotTo(BeEmpty(), "Help output should not be empty")

			GinkgoWriter.Printf("Help output:\n%s\n", string(output))
		}, SpecTimeout(10*time.Second))
	})
})
