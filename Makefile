# KubePulse Makefile

# Variables
BINARY_NAME=kubepulse
PACKAGE=github.com/kubepulse/kubepulse
VERSION?=0.1.0
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X ${PACKAGE}/cmd/kubepulse/commands.Version=${VERSION} -X ${PACKAGE}/cmd/kubepulse/commands.BuildDate=${BUILD_DATE} -X ${PACKAGE}/cmd/kubepulse/commands.GitCommit=${GIT_COMMIT}"
GO_FILES=$(shell find . -name '*.go' -type f)

# Build targets
.PHONY: all
all: build

.PHONY: build
build: clean frontend-build
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o bin/${BINARY_NAME} ./cmd/kubepulse

.PHONY: build-linux
build-linux: clean
	@echo "Building ${BINARY_NAME} for Linux..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 ./cmd/kubepulse

.PHONY: build-darwin
build-darwin: clean
	@echo "Building ${BINARY_NAME} for macOS..."
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 ./cmd/kubepulse
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-arm64 ./cmd/kubepulse

.PHONY: build-windows
build-windows: clean
	@echo "Building ${BINARY_NAME} for Windows..."
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe ./cmd/kubepulse

.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Frontend targets
.PHONY: frontend-install
frontend-install:
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install

.PHONY: frontend-build
frontend-build: frontend-install
	@echo "Building frontend..."
	@cd frontend && npm run build

.PHONY: frontend-dev
frontend-dev:
	@echo "Starting frontend development server..."
	@cd frontend && npm run dev

# Development targets
.PHONY: run
run:
	go run ./cmd/kubepulse monitor

.PHONY: dev
dev:
	@echo "Starting development servers..."
	@make -j2 run frontend-dev

.PHONY: install
install:
	go install ${LDFLAGS} ./cmd/kubepulse

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf frontend/dist/

# Testing targets
.PHONY: test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test -v -race -short ./...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -race -run Integration ./...

.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -run='^$$' ./...

.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.txt -o coverage.html

# Code quality targets
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	gofmt -w ${GO_FILES}

.PHONY: lint
lint:
	@echo "Running linters..."
	@if ! which golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

.PHONY: check
check: fmt vet lint test

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download

.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

.PHONY: vendor
vendor:
	@echo "Vendoring dependencies..."
	go mod vendor

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t kubepulse:${VERSION} -t kubepulse:latest .

.PHONY: docker-push
docker-push:
	@echo "Pushing Docker image..."
	docker push kubepulse:${VERSION}
	docker push kubepulse:latest

# Documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@if ! which godoc > /dev/null; then \
		echo "Installing godoc..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
	fi
	@echo "Documentation server starting at http://localhost:6060"
	godoc -http=:6060

# Development helpers
.PHONY: setup
setup: deps
	@echo "Setting up development environment..."
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/godoc@latest
	@echo "Setting up configuration..."
	@make config-init
	@echo "Development environment ready!"

# Configuration helpers
.PHONY: config-init
config-init:
	@echo "Initializing configuration..."
	@if [ ! -f ~/.kubepulse.yaml ]; then \
		echo "Creating default configuration file..."; \
		cp .kubepulse.yaml.example ~/.kubepulse.yaml; \
		echo "Configuration created at ~/.kubepulse.yaml"; \
		echo "Please edit this file to customize your settings"; \
	else \
		echo "Configuration already exists at ~/.kubepulse.yaml"; \
	fi
	@if [ ! -f frontend/.env ]; then \
		echo "Creating frontend environment file..."; \
		cp frontend/.env.example frontend/.env; \
		echo "Frontend configuration created at frontend/.env"; \
	fi

.PHONY: config-show
config-show:
	@echo "Current configuration:"
	@echo "====================="
	@if [ -f ~/.kubepulse.yaml ]; then \
		cat ~/.kubepulse.yaml; \
	else \
		echo "No configuration file found at ~/.kubepulse.yaml"; \
		echo "Run 'make config-init' to create one"; \
	fi

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary for current OS/arch"
	@echo "  build-all      - Build binaries for all platforms"
	@echo "  run            - Run the application"
	@echo "  install        - Install the binary"
	@echo "  test           - Run all tests with coverage"
	@echo "  test-unit      - Run unit tests only"
	@echo "  benchmark      - Run performance benchmarks"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo "  check          - Run all code quality checks"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docs           - Start documentation server"
	@echo "  setup          - Setup development environment"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Configuration targets:"
	@echo "  config-init    - Initialize configuration files"
	@echo "  config-show    - Display current configuration"
	@echo ""
	@echo "Frontend targets:"
	@echo "  frontend-build - Build the React frontend"
	@echo "  frontend-dev   - Start frontend development server"
	@echo "  dev            - Run both backend and frontend in development mode"