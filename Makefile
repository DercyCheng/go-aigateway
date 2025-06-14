# Makefile for Go AI Gateway
.PHONY: help build start stop restart logs clean dev prod test lint format

# Default target
help: ## Show this help message
	@echo "Go AI Gateway - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Environment setup
ENV_FILE := .env
COMPOSE_FILE := docker-compose.yml
COMPOSE_CMD := docker-compose -f $(COMPOSE_FILE)

# Load environment variables if .env exists
ifneq (,$(wildcard $(ENV_FILE)))
    include $(ENV_FILE)
    export
endif

# Development commands
dev: ## Start in development mode with hot reload
	@echo "🚀 Starting Go AI Gateway in development mode..."
	@GIN_MODE=debug LOG_LEVEL=debug $(COMPOSE_CMD) up --build

dev-detached: ## Start in development mode (detached)
	@echo "🚀 Starting Go AI Gateway in development mode (detached)..."
	@GIN_MODE=debug LOG_LEVEL=debug $(COMPOSE_CMD) up -d --build

# Production commands
prod: ## Start in production mode
	@echo "🚀 Starting Go AI Gateway in production mode..."
	@GIN_MODE=release LOG_LEVEL=info $(COMPOSE_CMD) up -d --build

start: prod ## Alias for prod

# Build commands
build: ## Build all Docker images
	@echo "🔨 Building all Docker images..."
	@$(COMPOSE_CMD) build --no-cache

build-gateway: ## Build only the gateway service
	@echo "🔨 Building gateway service..."
	@$(COMPOSE_CMD) build --no-cache gateway

build-frontend: ## Build only the frontend service
	@echo "🔨 Building frontend service..."
	@$(COMPOSE_CMD) build --no-cache frontend

build-python: ## Build only the python model service
	@echo "🔨 Building python model service..."
	@$(COMPOSE_CMD) build --no-cache python-model

# Service management
stop: ## Stop all services
	@echo "⏹️  Stopping all services..."
	@$(COMPOSE_CMD) down

restart: ## Restart all services
	@echo "🔄 Restarting all services..."
	@$(COMPOSE_CMD) restart

restart-gateway: ## Restart only the gateway service
	@echo "🔄 Restarting gateway service..."
	@$(COMPOSE_CMD) restart gateway

restart-frontend: ## Restart only the frontend service
	@echo "🔄 Restarting frontend service..."
	@$(COMPOSE_CMD) restart frontend

restart-python: ## Restart only the python model service
	@echo "🔄 Restarting python model service..."
	@$(COMPOSE_CMD) restart python-model

# Logs and monitoring
logs: ## Show logs from all services
	@$(COMPOSE_CMD) logs -f

logs-gateway: ## Show logs from gateway service
	@$(COMPOSE_CMD) logs -f gateway

logs-frontend: ## Show logs from frontend service
	@$(COMPOSE_CMD) logs -f frontend

logs-python: ## Show logs from python model service
	@$(COMPOSE_CMD) logs -f python-model

logs-redis: ## Show logs from redis service
	@$(COMPOSE_CMD) logs -f redis

# Health checks
health: ## Check health of all services
	@echo "🏥 Checking service health..."
	@docker ps --filter "name=go-aigateway" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

status: health ## Alias for health

# Testing
test: ## Run Go tests
	@echo "🧪 Running Go tests..."
	@go test ./... -v

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@go test ./tests/integration/... -v

# Code quality
lint: ## Run linters
	@echo "🔍 Running linters..."
	@golangci-lint run ./...

format: ## Format Go code
	@echo "🎨 Formatting Go code..."
	@go fmt ./...
	@goimports -w .

# Database and cache
redis-cli: ## Connect to Redis CLI
	@docker exec -it go-aigateway-redis redis-cli

redis-flush: ## Flush Redis cache
	@docker exec -it go-aigateway-redis redis-cli FLUSHALL

# Cleanup
clean: ## Clean up containers, volumes, and images
	@echo "🧹 Cleaning up..."
	@$(COMPOSE_CMD) down -v --remove-orphans
	@docker system prune -f

clean-volumes: ## Clean up volumes only
	@echo "🧹 Cleaning up volumes..."
	@$(COMPOSE_CMD) down -v

clean-images: ## Clean up Docker images
	@echo "🧹 Cleaning up Docker images..."
	@docker rmi $(shell docker images go-aigateway* -q) 2>/dev/null || true

# Environment setup
env-example: ## Create example .env file
	@echo "📄 Creating example .env file..."
	@cat > .env.example << 'EOF'
# Environment configuration for Docker
# Copy this to .env and modify as needed

# Basic Configuration
GIN_MODE=debug
LOG_LEVEL=debug
LOG_FORMAT=text
REDIS_PASSWORD=your_redis_password_here

# Security (IMPORTANT: Change in production!)
JWT_SECRET=your_super_secret_jwt_key_change_in_production_2024

# Gateway API Keys (for external access)
GATEWAY_API_KEYS=your_gateway_api_key_1,your_gateway_api_key_2

# AI Service Providers Configuration
# =================================

# 通义千问 (Tongyi/Qwen - 阿里云)
TONGYI_API_KEY=your_tongyi_api_key_here
TONGYI_BASE_URL=https://dashscope.aliyuncs.com/api/v1
TONGYI_ENABLED=false

# OpenAI
OPENAI_API_KEY=your_openai_api_key_here
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_ENABLED=false

# Anthropic Claude
ANTHROPIC_API_KEY=your_anthropic_api_key_here
ANTHROPIC_BASE_URL=https://api.anthropic.com
ANTHROPIC_ENABLED=false

# 百度文心一言 (Wenxin)
WENXIN_API_KEY=your_wenxin_api_key_here
WENXIN_SECRET_KEY=your_wenxin_secret_key_here
WENXIN_BASE_URL=https://aip.baidubce.com
WENXIN_ENABLED=false

# 智谱 AI (Zhipu)
ZHIPU_API_KEY=your_zhipu_api_key_here
ZHIPU_BASE_URL=https://open.bigmodel.cn/api/paas/v4
ZHIPU_ENABLED=false

# 腾讯混元 (Hunyuan)
HUNYUAN_SECRET_ID=your_hunyuan_secret_id_here
HUNYUAN_SECRET_KEY=your_hunyuan_secret_key_here
HUNYUAN_BASE_URL=https://hunyuan.tencentcloudapi.com
HUNYUAN_ENABLED=false

# 月之暗面 (Moonshot/Kimi)
MOONSHOT_API_KEY=your_moonshot_api_key_here
MOONSHOT_BASE_URL=https://api.moonshot.cn/v1
MOONSHOT_ENABLED=false

# 阿里百炼 (通过兼容模式)
BAILIAN_API_KEY=your_bailian_api_key_here
BAILIAN_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1

# Local Model Configuration
LOCAL_MODEL_ENABLED=true
LOCAL_MODEL_HOST=python-model
LOCAL_MODEL_PORT=5000
LOCAL_MODEL_TYPE=chat
LOCAL_MODEL_SIZE=small

# Third-party Model Configuration (Optional)
THIRD_PARTY_MODEL_ENABLED=false
THIRD_PARTY_MODEL_PROVIDER=bailian
THIRD_PARTY_MODEL_API_KEY=your_bailian_api_key_here
THIRD_PARTY_MODEL_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
THIRD_PARTY_MODEL_DEFAULT=qwen-turbo

# Auto Scaling (Optional)
AUTO_SCALING_ENABLED=false

# Cloud Integration (Optional)
CLOUD_INTEGRATION_ENABLED=false
CLOUD_PROVIDER=aws
CLOUD_REGION=us-west-2
CLOUD_ACCESS_KEY_ID=your_access_key_here
CLOUD_ACCESS_KEY_SECRET=your_secret_key_here

# RAM Authentication (阿里云)
RAM_AUTH_ENABLED=false
RAM_ACCESS_KEY_ID=your_ram_access_key_id
RAM_ACCESS_KEY_SECRET=your_ram_access_key_secret
RAM_REGION=cn-hangzhou

# Hugging Face Mirror (for China users)
HF_ENDPOINT=https://hf-mirror.com
USE_THIRD_PARTY_MODEL=false

# Frontend Configuration
VITE_API_BASE_URL=http://localhost:8080
EOF
	@echo "✅ Created .env.example file"

env-copy: ## Copy .env.example to .env
	@cp .env.example .env
	@echo "✅ Copied .env.example to .env"

# Backup and restore
backup: ## Backup Redis data
	@echo "💾 Backing up Redis data..."
	@docker exec go-aigateway-redis redis-cli BGSAVE
	@docker cp go-aigateway-redis:/data/dump.rdb ./backup-$(shell date +%Y%m%d-%H%M%S).rdb
	@echo "✅ Backup completed"

# Development utilities
shell-gateway: ## Open shell in gateway container
	@docker exec -it go-aigateway-api sh

shell-python: ## Open shell in python container
	@docker exec -it go-aigateway-python bash

shell-frontend: ## Open shell in frontend container
	@docker exec -it go-aigateway-frontend sh

# Quick commands
up: dev ## Quick start (development mode)
down: stop ## Quick stop

# Security scan
security-scan: ## Run security scan on Docker images
	@echo "🔒 Running security scan..."
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy image go-aigateway-api:latest || true
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy image go-aigateway-frontend:latest || true
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy image go-aigateway-python:latest || true
