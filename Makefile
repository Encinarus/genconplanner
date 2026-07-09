.PHONY: fmt lint test security tidy

# Format code
fmt:
	go fmt ./...
	go run golang.org/x/tools/cmd/goimports@latest -w .

# Run linters
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found in PATH, running via go run..."; \
		go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...; \
	fi

# Run tests
test:
	go test ./...

# Run security scans locally
security:
	@echo "Running dependency vulnerability scan (govulncheck)..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "Running security AST check (gosec)..."
	go run github.com/securego/gosec/v2/cmd/gosec@latest ./...

# Tidy dependencies
tidy:
	go mod tidy
