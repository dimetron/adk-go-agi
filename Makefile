
.PHONY: gomod
gomod:
	@go mod tidy

.PHONY: build
build: gomod
	@go build -o bin/agi cmd/agi.go

.PHONY: run
run: PORT=9090
run: build
	@killall  agi || true
	@echo "Running AGI Agent on port $(PORT)"
	@open http://localhost:9090/ui/?app=CodePipelineAgent
	@./bin/agi web -port $(PORT) api -webui_address localhost a2a -a2a_agent_url http://localhost:$(PORT) webui -api_server_address http://localhost:$(PORT)/api
