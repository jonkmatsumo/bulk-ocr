.PHONY: fmt fmt-fix vet test test-race test-coverage lint build docker-build docker-test clean

# Formatting
fmt:
	@echo "Checking formatting..."
	@test -z $$(go fmt ./... | tee /dev/stderr) || (echo "Code is not formatted. Run 'make fmt-fix' to fix." && exit 1)

fmt-fix:
	@echo "Fixing formatting..."
	@go fmt ./...

# Static analysis
vet:
	@echo "Running go vet..."
	@go vet ./...

# Testing
test:
	@echo "Running tests..."
	@go test ./... -v

test-race:
	@echo "Running tests with race detector..."
	@go test ./... -race

test-coverage:
	@echo "Generating coverage report..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Linting
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Building
build:
	@echo "Building binary..."
	@go build ./cmd/pipeline

# Docker
docker-build:
	@echo "Building Docker image..."
	@docker build -t bulk-ocr:latest .

docker-test: docker-build
	@echo "Testing Docker image..."
	@docker run --rm bulk-ocr:latest version
	@docker run --rm bulk-ocr:latest doctor
	@echo "Docker tests passed"

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@rm -f pipeline
	@rm -f coverage.out coverage.html
	@go clean ./...

