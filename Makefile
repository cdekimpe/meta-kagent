# KMeta-Agent Makefile

# Variables
BINARY_NAME := kmeta-agent-server
IMAGE_NAME := ghcr.io/kagent-dev/meta-kagent
VERSION ?= latest
GO := go
GOFLAGS := -v
LDFLAGS := -s -w

# Directories
BUILD_DIR := bin
CMD_DIR := cmd/mcp-server

.PHONY: all build clean test lint fmt docker-build docker-push deploy undeploy help

# Default target
all: build

## Build targets

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Build for Linux (for container)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./$(CMD_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GO) clean

## Development targets

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Lint the code
lint:
	@echo "Linting..."
	golangci-lint run ./...

# Format the code
fmt:
	@echo "Formatting..."
	$(GO) fmt ./...
	gofumpt -w .

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify

## Docker targets

# Build Docker image
docker-build: build-linux
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME):$(VERSION) .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push $(IMAGE_NAME):$(VERSION)

# Build and push
docker-release: docker-build docker-push

## Kubernetes targets

# Deploy to Kubernetes
deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -k deploy/kubernetes/

# Undeploy from Kubernetes
undeploy:
	@echo "Undeploying from Kubernetes..."
	kubectl delete -k deploy/kubernetes/ --ignore-not-found

# Show diff before deploy
deploy-diff:
	@echo "Showing deployment diff..."
	kubectl diff -k deploy/kubernetes/ || true

# Check deployment status
status:
	@echo "Checking deployment status..."
	kubectl get agents,modelconfigs,mcpservers -n kagent -l app.kubernetes.io/part-of=kmeta-agent

## Local development

# Run locally (requires kubeconfig)
run:
	@echo "Running locally..."
	KAGENT_NAMESPACE=kagent $(GO) run ./$(CMD_DIR)

# Run with debug logging
run-debug:
	@echo "Running with debug logging..."
	LOG_LEVEL=debug KAGENT_NAMESPACE=kagent $(GO) run ./$(CMD_DIR)

## Help

# Show help
help:
	@echo "KMeta-Agent Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build          Build the binary"
	@echo "  build-linux    Build for Linux (container)"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Development targets:"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage"
	@echo "  lint           Lint the code"
	@echo "  fmt            Format the code"
	@echo "  deps           Download dependencies"
	@echo "  verify         Verify dependencies"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-push    Push Docker image"
	@echo "  docker-release Build and push Docker image"
	@echo ""
	@echo "Kubernetes targets:"
	@echo "  deploy         Deploy to Kubernetes"
	@echo "  undeploy       Undeploy from Kubernetes"
	@echo "  deploy-diff    Show deployment diff"
	@echo "  status         Check deployment status"
	@echo ""
	@echo "Local development:"
	@echo "  run            Run locally"
	@echo "  run-debug      Run with debug logging"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION        Image tag (default: latest)"
	@echo "  IMAGE_NAME     Docker image name"
