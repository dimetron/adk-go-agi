
.PHONY: gomod
gomod:
	@go mod tidy

.PHONY: build
build: gomod
	@CGO_ENABLED=0 go build -o bin/agi cmd/agi.go

.PHONY: test
test:
	@go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test
	@go tool cover -html=coverage.out

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

.PHONY: run
run: PORT=9090
run: build
	@killall  agi || true
	@echo "Running AGI Agent on port $(PORT)"
	@open http://localhost:9090/ui/?app=CodePipelineAgent
	@./bin/agi web -port $(PORT) api -webui_address localhost a2a -a2a_agent_url http://localhost:$(PORT) webui -api_server_address http://localhost:$(PORT)/api
