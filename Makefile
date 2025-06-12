# AI Gateway Makefile
# 项目管理和部署自动化

.PHONY: help setup clean build dev prod test docker-dev docker-prod docker-clean env-setup

# 默认目标
help: ## 显示帮助信息
	@echo "AI Gateway - 可用的make命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==========================================
# 环境配置
# ==========================================

env-setup: ## 设置环境配置文件
	@echo "设置环境配置文件..."
	@if [ ! -f .env ]; then \
		cp configs/env.template .env; \
		echo "✅ 已创建 .env 文件，请编辑填入实际的API密钥"; \
	else \
		echo "⚠️  .env 文件已存在"; \
	fi

env-check: ## 检查环境配置
	@echo "检查环境配置..."
	@if [ -f .env ]; then \
		echo "✅ .env 文件存在"; \
		echo "检查必要的环境变量:"; \
		if grep -q "your_.*_here" .env; then \
			echo "⚠️  发现未配置的API密钥，请编辑 .env 文件"; \
			grep "your_.*_here" .env; \
		else \
			echo "✅ 环境变量配置完成"; \
		fi; \
	else \
		echo "❌ .env 文件不存在，请运行 make env-setup"; \
		exit 1; \
	fi

# ==========================================
# 安全性检查和审计
# ==========================================

security-check: ## 运行安全检查
	@echo "运行安全检查..."
	@echo "检查环境变量安全性..."
	@if grep -q "your_.*_here" .env 2>/dev/null; then \
		echo "⚠️  发现未配置的默认值，请检查 .env 文件"; \
		grep "your_.*_here" .env; \
	else \
		echo "✅ 环境变量检查通过"; \
	fi
	@echo "检查JWT密钥强度..."
	@if [ -f .env ] && grep -q "JWT_SECRET=your_super_secret" .env; then \
		echo "❌ JWT_SECRET 使用默认值，请更改为强密钥"; \
		exit 1; \
	else \
		echo "✅ JWT密钥检查通过"; \
	fi

vulnerability-scan: ## 扫描依赖漏洞
	@echo "扫描Go依赖漏洞..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "安装 govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

container-scan: ## 扫描容器镜像漏洞
	@echo "扫描容器镜像漏洞..."
	@if command -v trivy >/dev/null 2>&1; then \
		trivy image aigateway:latest; \
	else \
		echo "请安装 Trivy: https://aquasecurity.github.io/trivy/"; \
	fi

audit: security-check vulnerability-scan ## 完整安全审计
	@echo "✅ 安全审计完成"

setup: env-setup ## 初始化项目环境
	@echo "初始化项目环境..."
	@go mod download
	@echo "✅ Go依赖下载完成"
	@if command -v npm >/dev/null 2>&1; then \
		cd frontend && npm install; \
		echo "✅ 前端依赖安装完成"; \
	else \
		echo "⚠️  npm未安装，跳过前端依赖安装"; \
	fi
	@echo "✅ 项目初始化完成"

clean: ## 清理构建文件和缓存
	@echo "清理项目..."
	@go clean -cache
	@go clean -modcache
	@rm -rf ./build
	@rm -rf ./dist
	@if [ -d "frontend/node_modules" ]; then rm -rf frontend/node_modules; fi
	@if [ -d "frontend/dist" ]; then rm -rf frontend/dist; fi
	@echo "✅ 清理完成"

# ==========================================
# 构建相关
# ==========================================

build: ## 构建后端服务
	@echo "构建后端服务..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/aigateway .
	@echo "✅ 后端构建完成"

build-frontend: ## 构建前端
	@echo "构建前端..."
	@cd frontend && npm run build
	@echo "✅ 前端构建完成"

build-all: build build-frontend ## 构建所有组件
	@echo "✅ 全部构建完成"

# ==========================================
# 开发环境
# ==========================================

dev: env-check ## 启动开发环境
	@echo "启动开发环境..."
	@docker-compose -f docker-compose.dev.yml up -d
	@echo "✅ 开发环境启动完成"
	@echo "🌐 访问地址:"
	@echo "   前端: http://localhost:3000"
	@echo "   后端API: http://localhost:8080"
	@echo "   Prometheus: http://localhost:9090"
	@echo "   Grafana: http://localhost:3001 (admin/admin_dev_2024)"
	@echo "   Consul: http://localhost:8500"

dev-logs: ## 查看开发环境日志
	@docker-compose -f docker-compose.dev.yml logs -f

dev-stop: ## 停止开发环境
	@echo "停止开发环境..."
	@docker-compose -f docker-compose.dev.yml down
	@echo "✅ 开发环境已停止"

dev-restart: dev-stop dev ## 重启开发环境

# ==========================================
# 生产环境
# ==========================================

prod-secrets: ## 创建生产环境密钥目录和模板
	@echo "创建生产环境密钥目录..."
	@mkdir -p deployment/secrets
	@mkdir -p deployment/ssl
	@echo "请在deployment/secrets/目录下创建以下密钥文件:"
	@echo "  - postgres_password.txt"
	@echo "  - jwt_secret.txt"
	@echo "  - grafana_password.txt"
	@echo "  - tongyi_api_key.txt"
	@echo "  - openai_api_key.txt"
	@echo "  - openai_org_id.txt"
	@echo "  - wenxin_api_key.txt"
	@echo "  - wenxin_secret_key.txt"
	@echo "  - zhipu_api_key.txt"
	@echo "  - hunyuan_secret_id.txt"
	@echo "  - hunyuan_secret_key.txt"
	@echo "  - moonshot_api_key.txt"

prod: build-all prod-secrets ## 启动生产环境
	@echo "启动生产环境..."
	@docker-compose -f docker-compose.prod.yml up -d
	@echo "✅ 生产环境启动完成"

prod-logs: ## 查看生产环境日志
	@docker-compose -f docker-compose.prod.yml logs -f

prod-stop: ## 停止生产环境
	@echo "停止生产环境..."
	@docker-compose -f docker-compose.prod.yml down
	@echo "✅ 生产环境已停止"

# ==========================================
# Docker管理
# ==========================================

docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	@docker build -t aigateway:latest .
	@cd frontend && docker build -t aigateway-frontend:latest .
	@cd python && docker build -t aigateway-python:latest .
	@echo "✅ Docker镜像构建完成"

docker-clean: ## 清理Docker资源
	@echo "清理Docker资源..."
	@docker-compose -f docker-compose.dev.yml down -v --remove-orphans
	@docker-compose -f docker-compose.prod.yml down -v --remove-orphans
	@docker system prune -f
	@echo "✅ Docker资源清理完成"

# ==========================================
# 测试相关
# ==========================================

test: ## 运行测试
	@echo "运行测试..."
	@go test -v ./...
	@echo "✅ 测试完成"

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告生成完成: coverage.html"

# ==========================================
# 数据库管理
# ==========================================

db-migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	@docker-compose -f docker-compose.dev.yml exec postgres psql -U aigateway -d aigateway_dev -f /docker-entrypoint-initdb.d/init-db.sql
	@echo "✅ 数据库迁移完成"

db-reset: ## 重置数据库
	@echo "重置数据库..."
	@docker-compose -f docker-compose.dev.yml down -v
	@docker-compose -f docker-compose.dev.yml up -d postgres
	@sleep 10
	@make db-migrate
	@echo "✅ 数据库重置完成"

# ==========================================
# 监控和调试
# ==========================================

health-check: ## 检查服务健康状态
	@echo "检查服务健康状态..."
	@curl -f http://localhost:8080/health || echo "❌ 后端服务不可用"
	@curl -f http://localhost:3000 || echo "❌ 前端服务不可用"
	@curl -f http://localhost:9090/-/healthy || echo "❌ Prometheus不可用"
	@curl -f http://localhost:3001/api/health || echo "❌ Grafana不可用"
	@echo "✅ 健康检查完成"

logs: dev-logs ## 查看服务日志 (dev-logs的别名)

status: ## 查看服务状态
	@echo "开发环境服务状态:"
	@docker-compose -f docker-compose.dev.yml ps

# ==========================================
# 代码质量
# ==========================================

lint: ## 运行代码检查
	@echo "运行Go代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint未安装，请运行: brew install golangci-lint"; \
	fi
	@echo "运行前端代码检查..."
	@if [ -d "frontend" ]; then \
		cd frontend && npm run lint; \
	fi

format: ## 格式化代码
	@echo "格式化Go代码..."
	@go fmt ./...
	@echo "格式化前端代码..."
	@if [ -d "frontend" ]; then \
		cd frontend && npm run format; \
	fi
	@echo "✅ 代码格式化完成"

# ==========================================
# 部署相关
# ==========================================

deploy-dev: dev ## 部署到开发环境
	@echo "✅ 开发环境部署完成"

deploy-prod: prod ## 部署到生产环境
	@echo "✅ 生产环境部署完成"

# ==========================================
# 工具命令
# ==========================================

install-tools: ## 安装开发工具
	@echo "安装开发工具..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ 开发工具安装完成"

docs: ## 生成API文档
	@echo "生成API文档..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init; \
		echo "✅ API文档生成完成"; \
	else \
		echo "⚠️  swag未安装，请运行: make install-tools"; \
	fi
