# AI Gateway 部署管理 Makefile
# ============================================
# 变量定义
# ============================================

PROJECT_NAME = ai-gateway
DOCKER_COMPOSE_DEV = docker-compose -f deployment/docker-compose.dev.yml
DOCKER_COMPOSE_PROD = docker-compose -f deployment/docker-compose.prod.yml
ENV_DEV = deployment/.env.development
ENV_PROD = deployment/.env.production

# 颜色定义
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

# ============================================
# 帮助信息
# ============================================

.PHONY: help
help: ## 显示帮助信息
	@echo "$(BLUE)AI Gateway 部署管理命令$(NC)"
	@echo "======================================"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ============================================
# 环境检查
# ============================================

.PHONY: check-deps
check-deps: ## 检查依赖项
	@echo "$(BLUE)检查环境依赖...$(NC)"
	@command -v docker >/dev/null 2>&1 || { echo "$(RED)❌ Docker 未安装$(NC)"; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "$(RED)❌ Docker Compose 未安装$(NC)"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "$(RED)❌ Docker 未运行$(NC)"; exit 1; }
	@echo "$(GREEN)✅ 环境检查通过$(NC)"

.PHONY: check-env
check-env: ## 检查环境配置文件
	@test -f $(ENV_DEV) || { echo "$(YELLOW)⚠️  创建开发环境配置文件$(NC)"; cp deployment/.env.development.example $(ENV_DEV); }
	@test -f $(ENV_PROD) || { echo "$(YELLOW)⚠️  创建生产环境配置文件$(NC)"; cp deployment/.env.production.example $(ENV_PROD); }

# ============================================
# 开发环境
# ============================================

.PHONY: quick-start
quick-start: check-deps check-env dev-build dev-up ## 一键启动开发环境

.PHONY: dev-build
dev-build: ## 构建开发环境镜像
	@echo "$(BLUE)构建开发环境镜像...$(NC)"
	@$(DOCKER_COMPOSE_DEV) --env-file $(ENV_DEV) build --parallel
	@echo "$(GREEN)✅ 开发镜像构建完成$(NC)"

.PHONY: dev-up
dev-up: check-env ## 启动开发环境
	@echo "$(BLUE)启动开发环境...$(NC)"
	@$(DOCKER_COMPOSE_DEV) --env-file $(ENV_DEV) up -d
	@echo "$(GREEN)✅ 开发环境启动完成$(NC)"
	@echo "$(YELLOW)🌐 前端: http://localhost:3000$(NC)"
	@echo "$(YELLOW)🚪 后端: http://localhost:8080$(NC)"
	@echo "$(YELLOW)🐍 模型: http://localhost:5000$(NC)"

.PHONY: dev-down
dev-down: ## 停止开发环境
	@echo "$(BLUE)停止开发环境...$(NC)"
	@$(DOCKER_COMPOSE_DEV) --env-file $(ENV_DEV) down
	@echo "$(GREEN)✅ 开发环境已停止$(NC)"

.PHONY: dev-restart
dev-restart: dev-down dev-up ## 重启开发环境

.PHONY: dev-logs
dev-logs: ## 查看开发环境日志
	@$(DOCKER_COMPOSE_DEV) --env-file $(ENV_DEV) logs -f

.PHONY: dev-status
dev-status: ## 查看开发环境状态
	@$(DOCKER_COMPOSE_DEV) --env-file $(ENV_DEV) ps

# ============================================
# 生产环境
# ============================================

.PHONY: prod-build
prod-build: ## 构建生产环境镜像
	@echo "$(BLUE)构建生产环境镜像...$(NC)"
	@$(DOCKER_COMPOSE_PROD) --env-file $(ENV_PROD) build --parallel
	@echo "$(GREEN)✅ 生产镜像构建完成$(NC)"

.PHONY: prod-up
prod-up: check-env ## 启动生产环境
	@echo "$(BLUE)启动生产环境...$(NC)"
	@$(DOCKER_COMPOSE_PROD) --env-file $(ENV_PROD) up -d
	@echo "$(GREEN)✅ 生产环境启动完成$(NC)"

.PHONY: prod-down
prod-down: ## 停止生产环境
	@echo "$(BLUE)停止生产环境...$(NC)"
	@$(DOCKER_COMPOSE_PROD) --env-file $(ENV_PROD) down
	@echo "$(GREEN)✅ 生产环境已停止$(NC)"

.PHONY: prod-restart
prod-restart: prod-down prod-up ## 重启生产环境

.PHONY: prod-logs
prod-logs: ## 查看生产环境日志
	@$(DOCKER_COMPOSE_PROD) --env-file $(ENV_PROD) logs -f

.PHONY: prod-status
prod-status: ## 查看生产环境状态
	@$(DOCKER_COMPOSE_PROD) --env-file $(ENV_PROD) ps

# ============================================
# 清理
# ============================================

.PHONY: clean-dev
clean-dev: ## 清理开发环境资源
	@echo "$(BLUE)清理开发环境...$(NC)"
	@$(DOCKER_COMPOSE_DEV) --env-file $(ENV_DEV) down -v --remove-orphans
	@docker system prune -f
	@echo "$(GREEN)✅ 开发环境清理完成$(NC)"

.PHONY: clean-prod
clean-prod: ## 清理生产环境资源 (谨慎使用)
	@echo "$(BLUE)清理生产环境...$(NC)"
	@read -p "确认清理生产环境? [y/N] " confirm && [ "$$confirm" = "y" ]
	@$(DOCKER_COMPOSE_PROD) --env-file $(ENV_PROD) down -v --remove-orphans
	@echo "$(GREEN)✅ 生产环境清理完成$(NC)"

.PHONY: clean-images
clean-images: ## 清理Docker镜像
	@echo "$(BLUE)清理Docker镜像...$(NC)"
	@docker image prune -f
	@docker rmi $(shell docker images -f "dangling=true" -q) 2>/dev/null || true
	@echo "$(GREEN)✅ 镜像清理完成$(NC)"

# ============================================
# 监控与测试
# ============================================

.PHONY: health-check
health-check: ## 检查服务健康状态
	@echo "$(BLUE)检查服务健康状态...$(NC)"
	@curl -f http://localhost:8080/health && echo "$(GREEN)✅ 后端服务正常$(NC)" || echo "$(RED)❌ 后端服务异常$(NC)"
	@curl -f http://localhost:5000/health && echo "$(GREEN)✅ 模型服务正常$(NC)" || echo "$(RED)❌ 模型服务异常$(NC)"
	@curl -f http://localhost:3000 && echo "$(GREEN)✅ 前端服务正常$(NC)" || echo "$(RED)❌ 前端服务异常$(NC)"

.PHONY: test
test: ## 运行测试
	@echo "$(BLUE)运行测试...$(NC)"
	@docker exec aigateway-backend-dev go test ./... -v
	@echo "$(GREEN)✅ 测试完成$(NC)"

.PHONY: benchmark
benchmark: ## 运行性能测试
	@echo "$(BLUE)运行性能测试...$(NC)"
	@docker exec aigateway-backend-dev go test ./... -bench=. -benchmem
	@echo "$(GREEN)✅ 性能测试完成$(NC)"

# ============================================
# 实用工具
# ============================================

.PHONY: shell-backend
shell-backend: ## 进入后端容器Shell
	@docker exec -it aigateway-backend-dev sh

.PHONY: shell-model
shell-model: ## 进入模型容器Shell
	@docker exec -it aigateway-python-dev bash

.PHONY: shell-frontend
shell-frontend: ## 进入前端容器Shell
	@docker exec -it aigateway-frontend-dev sh

.PHONY: shell-db
shell-db: ## 进入数据库Shell
	@docker exec -it aigateway-postgres-dev psql -U aigateway ai_gateway

.PHONY: shell-redis
shell-redis: ## 进入Redis Shell
	@docker exec -it aigateway-redis-dev redis-cli

.PHONY: update-deps
update-deps: ## 更新依赖
	@echo "$(BLUE)更新Go依赖...$(NC)"
	@docker exec aigateway-backend-dev go mod tidy
	@echo "$(BLUE)更新Python依赖...$(NC)"
	@docker exec aigateway-python-dev pip install --upgrade -r requirements.txt
	@echo "$(BLUE)更新前端依赖...$(NC)"
	@docker exec aigateway-frontend-dev npm update
	@echo "$(GREEN)✅ 依赖更新完成$(NC)"

.PHONY: db-migrate
db-migrate: ## 运行数据库迁移
	@echo "$(BLUE)运行数据库迁移...$(NC)"
	@docker exec aigateway-backend-dev ./scripts/migrate.sh
	@echo "$(GREEN)✅ 数据库迁移完成$(NC)"

.PHONY: db-backup
db-backup: ## 备份数据库
	@echo "$(BLUE)备份数据库...$(NC)"
	@mkdir -p backups
	@docker exec aigateway-postgres-dev pg_dump -U aigateway ai_gateway > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✅ 数据库备份完成$(NC)"

.PHONY: db-restore
db-restore: ## 恢复数据库 (需要指定 BACKUP_FILE)
	@test -n "$(BACKUP_FILE)" || (echo "$(RED)❌ 请指定 BACKUP_FILE$(NC)"; exit 1)
	@echo "$(BLUE)恢复数据库...$(NC)"
	@docker exec -i aigateway-postgres-dev psql -U aigateway ai_gateway < $(BACKUP_FILE)
	@echo "$(GREEN)✅ 数据库恢复完成$(NC)"

# ============================================
# 监控
# ============================================

.PHONY: monitor-up
monitor-up: ## 启动监控服务
	@echo "$(BLUE)启动监控服务...$(NC)"
	@docker-compose -f deployment/docker-compose.monitor.yml up -d
	@echo "$(GREEN)✅ 监控服务启动完成$(NC)"
	@echo "$(YELLOW)📊 Prometheus: http://localhost:9090$(NC)"
	@echo "$(YELLOW)📈 Grafana: http://localhost:3001$(NC)"

.PHONY: monitor-down
monitor-down: ## 停止监控服务
	@echo "$(BLUE)停止监控服务...$(NC)"
	@docker-compose -f deployment/docker-compose.monitor.yml down
	@echo "$(GREEN)✅ 监控服务已停止$(NC)"

# ============================================
# 默认目标
# ============================================

.DEFAULT_GOAL := help
