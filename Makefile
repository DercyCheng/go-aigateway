# Makefile for AI Gateway

# Variables
APP_NAME := ai-gateway
VERSION := v1.0.0
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_IMAGE_LATEST := $(APP_NAME):latest

# Go variables
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0

.PHONY: help build run test clean docker-build docker-run docker-push k8s-deploy k8s-delete

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p build
	@go build -o build/$(APP_NAME) main.go

# Local deployment targets
local-setup: env deps ## Setup local deployment environment
	@echo "Setting up local deployment environment..."
	@mkdir -p python/model logs data/local
	@python -m pip install --upgrade pip
	@python -m pip install flask==2.3.3 transformers==4.36.2 torch==2.1.0 sentencepiece==0.1.99 accelerate==0.25.0 psutil
	@if [ ! -f .env.local ]; then cp .env.local .env || cp .env.example .env.local; fi
	@echo "Local environment setup complete!"

local-run: build ## Run the application in local mode
	@echo "Starting $(APP_NAME) in local mode..."
	@LOCAL_MODEL_ENABLED=true LOCAL_AUTH_ENABLED=true ./build/$(APP_NAME)

local-start: ## Start local deployment (Linux/macOS)
	@echo "Starting local deployment..."
	@chmod +x start-local.sh
	@./start-local.sh

local-start-windows: ## Start local deployment (Windows)
	@echo "Starting local deployment on Windows..."
	@start-local.bat
	@echo "Building $(APP_NAME)..."
	@mkdir -p build/
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -a -installsuffix cgo -o build/$(APP_NAME) .

run: ## Run the application locally
	@echo "Running $(APP_NAME)..."
	@go run main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf build/
	@docker rmi $(DOCKER_IMAGE) $(DOCKER_IMAGE_LATEST) 2>/dev/null || true

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -f deployments/Dockerfile -t $(DOCKER_IMAGE) -t $(DOCKER_IMAGE_LATEST) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE_LATEST)

docker-push: docker-build ## Push Docker image to registry
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE)
	@docker push $(DOCKER_IMAGE_LATEST)

# Docker Compose targets
compose-up: ## Start services with Docker Compose
	@echo "Starting services with Docker Compose..."
	@docker-compose -f deployments/docker-compose.yml up -d

compose-down: ## Stop services with Docker Compose
	@echo "Stopping services with Docker Compose..."
	@docker-compose -f deployments/docker-compose.yml down

compose-logs: ## View Docker Compose logs
	@docker-compose -f deployments/docker-compose.yml logs -f

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f deployments/k8s-deployment.yaml

k8s-delete: ## Delete from Kubernetes
	@echo "Deleting from Kubernetes..."
	@kubectl delete -f deployments/k8s-deployment.yaml

k8s-status: ## Check Kubernetes deployment status
	@echo "Checking Kubernetes status..."
	@kubectl get pods -l app=$(APP_NAME)
	@kubectl get services -l app=$(APP_NAME)

k8s-logs: ## View Kubernetes logs
	@kubectl logs -f -l app=$(APP_NAME)

# Development helpers
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

# Environment setup
env: ## Copy environment template
	@if [ ! -f .env ]; then \
		echo "Copying .env.example to .env..."; \
		cp .env.example .env; \
		echo "Please edit .env file with your configuration"; \
	else \
		echo ".env file already exists"; \
	fi

# Health check
health: ## Check application health
	@echo "Checking application health..."
	@curl -f http://localhost:8080/health || echo "Health check failed"

# Quick development setup
dev-setup: env deps ## Setup development environment
	@echo "Development environment setup complete!"
	@echo "Please edit .env file with your API keys and run 'make run'"
