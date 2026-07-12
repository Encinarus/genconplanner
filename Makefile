.PHONY: fmt lint backend-lint test security sec tidy ui-lint ui-audit docker-lint docker-scan

# Format code
fmt:
	go fmt ./...
	go run golang.org/x/tools/cmd/goimports@latest -w .

# Run all linters (Backend & UI)
lint: backend-lint ui-lint docker-lint

backend-lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found in PATH, running via go run..."; \
		go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...; \
	fi

# Run tests and security scans
test: lint security
	go test ./...

# Run security scans locally (Backend & UI)
security:
	@echo "Running backend dependency vulnerability scan (govulncheck)..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "Running backend security AST check (gosec)..."
	go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
	@echo "Running UI dependency audit..."
	npm audit --prefix ui
	@echo "Running Docker configuration security scan (trivy)..."
	@make docker-scan

sec: security
# Run UI linting
ui-lint:
	npm run lint --prefix ui

# Run UI dependency audit
ui-audit:
	npm audit --prefix ui

# Run Docker linting
docker-lint:
	@echo "Running Hadolint..."
	@if command -v hadolint >/dev/null 2>&1; then \
		hadolint Dockerfile; \
	else \
		echo "hadolint not found in PATH, running via Docker container..."; \
		docker run --rm -i hadolint/hadolint < Dockerfile; \
	fi

# Run Docker configuration security scans
docker-scan:
	@echo "Running Trivy Config Scan..."
	@if command -v trivy >/dev/null 2>&1; then \
		trivy config .; \
	else \
		echo "trivy not found in PATH, running via Docker container..."; \
		docker run --rm -v $$(pwd):/apps aquasecurity/trivy config /apps; \
	fi

# Tidy dependencies
tidy:
	go mod tidy
