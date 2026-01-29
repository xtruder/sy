.PHONY: build install test lint clean release help

# Variables
BINARY_NAME := sy
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target
all: build

# Build binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/sy

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/sy

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf dist/
	rm -f coverage.out coverage.html

# Build for all platforms
release: clean
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 go build $(LDFLAGS) \
		-o dist/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$$([ "$${platform%/*}" = "windows" ] && echo ".exe") \
		./cmd/sy; \
		echo "Built dist/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done
	@cd dist && sha256sum * > checksums.txt
	@echo "Release binaries built in dist/"

# Run the binary
run: build
	./$(BINARY_NAME)

# Development: build and run with args
dev: build
	./$(BINARY_NAME) $(ARGS)

# Check everything before commit
check: fmt lint test build
	@echo "All checks passed!"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  install       - Install to GOPATH/bin"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format code"
	@echo "  tidy          - Tidy go.mod"
	@echo "  clean         - Remove build artifacts"
	@echo "  release       - Build for all platforms"
	@echo "  run           - Build and run"
	@echo "  dev ARGS=...  - Build and run with arguments"
	@echo "  check         - Run fmt, lint, test, build"
	@echo "  help          - Show this help"
