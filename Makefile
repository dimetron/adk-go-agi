
.PHONY: gomod
gomod:
	@go mod tidy

.PHONY: build
build: gomod
	@CGO_ENABLED=0 go build -o bin/agi 	 cmd/agi/main.go

.PHONY: test
test:
	@go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test
	@go tool cover -html=coverage.out

.PHONY: fuzz
fuzz:
	@echo "Running all fuzz tests for 30 seconds each..."
	@go test -fuzz=FuzzConvertContentsToMessages -fuzztime=30s ./pkg/model/ollama/
	@go test -fuzz=FuzzConvertChatResponseToLLMResponse -fuzztime=30s ./pkg/model/ollama/
	@go test -fuzz=FuzzSyncGeneratorWithMock -fuzztime=30s ./pkg/model/ollama/

.PHONY: fuzz-short
fuzz-short:
	@echo "Running quick fuzz tests (10 seconds each)..."
	@go test -fuzz=. -fuzztime=10s ./pkg/model/ollama/

.PHONY: fuzz-long
fuzz-long:
	@echo "Running extended fuzz tests (5 minutes)..."
	@go test -fuzz=. -fuzztime=5m ./pkg/model/ollama/

.PHONY: fuzz-contents
fuzz-contents:
	@echo "Fuzzing content-to-message conversion..."
	@go test -fuzz=FuzzConvertContentsToMessages -fuzztime=1m ./pkg/model/ollama/

.PHONY: fuzz-response
fuzz-response:
	@echo "Fuzzing chat response conversion..."
	@go test -fuzz=FuzzConvertChatResponseToLLMResponse -fuzztime=1m ./pkg/model/ollama/

.PHONY: fuzz-generator
fuzz-generator:
	@echo "Fuzzing synchronous generator..."
	@go test -fuzz=FuzzSyncGeneratorWithMock -fuzztime=1m ./pkg/model/ollama/

.PHONY: e2e
e2e:
	@echo "Running E2E tests..."
	@go test -v -timeout 10m ./test/e2e/...

.PHONY: e2e-verbose
e2e-verbose:
	@echo "Running E2E tests with verbose output..."
	@go test -v -ginkgo.v -timeout 10m ./test/e2e/...

.PHONY: lint
lint:
	@echo "Running linters..."
	@go vet ./...
	@gofmt -l -s .

.PHONY: run-with-gateway
run-with-gateway: PORT=9090
run-with-gateway:
	curl https://raw.githubusercontent.com/agentgateway/agentgateway/refs/heads/main/common/scripts/get-agentgateway | bash
	agentgateway -f agentgateway/config.yaml


.PHONY: run
run: PORT=9090
run: build
	@killall  agi || true
	@echo "Running AGI Agent on port $(PORT)"
	@echo "Using Ollama model: $${OLLAMA_MODEL:-llama3.2}"
	@echo "Ollama endpoint: $${OLLAMA_BASE_URL:-http://localhost:11434}"
	@open http://localhost:9090/ui/?app=CodePipelineAgent
	@./bin/agi  web -port $(PORT) api -webui_address localhost a2a -a2a_agent_url http://localhost:$(PORT) webui -api_server_address http://localhost:$(PORT)/api

.PHONY: ollama-check
ollama-check:
	@echo "Checking Ollama installation..."
	@if command -v ollama >/dev/null 2>&1; then \
		echo "✓ Ollama is installed"; \
		if curl -s http://localhost:11434/api/version >/dev/null 2>&1; then \
			echo "✓ Ollama is running"; \
			echo "\nInstalled models:"; \
			ollama list; \
		else \
			echo "✗ Ollama is not running. Start it with: ollama serve"; \
			exit 1; \
		fi \
	else \
		echo "✗ Ollama is not installed. Install from: https://ollama.ai"; \
		exit 1; \
	fi

#govulncheck
.PHONY: govulncheck
govulncheck:
	which govulncheck || @go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Running govulncheck..."
	@govulncheck ./...
