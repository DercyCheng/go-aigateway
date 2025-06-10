# AI Gateway

一个基于**三层架构**的智能 AI 服务网关，采用 **React 前端 + Go 后端 + Python 模型服务** 的完整技术栈，提供统一的 API 接口来管理和代理多个 AI 服务。

## 🏗️ 三层架构完整支持

### 🎨 前端层 (React + TypeScript)

- **技术栈**: React 19 + TypeScript + Vite + Tailwind CSS
- **功能模块**:
  - 📊 Dashboard 仪表盘
  - 🎛️ 服务源管理 (ServiceSources)
  - 📋 服务列表 (ServiceList)
  - 🛣️ 路由配置 (RouteConfig)
  - 🌐 域名管理 (DomainManagement)
  - 🔒 证书管理 (CertificateManagement)
  - 🤖 **本地模型管理** (LocalModelManagement)
- **API 集成**: 完整的 RESTful API 客户端，支持本地模型和云端服务

### ⚙️ 后端层 (Go)

- **框架**: Gin Web Framework
- **核心功能**:
  - 🚀 **API 网关**: 统一代理多个 AI 服务 API
  - 🔐 **多重认证**: JWT、API Key、阿里云 RAM 认证
  - 📊 **监控指标**: 实时监控和 Prometheus 指标
  - 🔄 **负载均衡**: 智能请求分发和故障转移
  - 🛡️ **安全防护**: 完整的安全中间件和防护措施
  - 🤖 **本地模型桥接**: Go 后端与 Python 模型服务的无缝集成

### 🧠 模型服务层 (Python)

- **框架**: Flask + Transformers + PyTorch
- **支持的模型类型**:
  - 💬 **聊天补全** (Chat Completions): TinyLlama、Mistral-7B
  - 📝 **文本补全** (Text Completions): Phi-2、Gemma-2B
  - 🔍 **向量嵌入** (Embeddings): MiniLM、E5-Large
- **第三方集成**: 支持阿里云百炼 (Dashscope) API
- **模型规格**: 小型 (Small)、中型 (Medium)、大型 (Large) 多种规格

### 高级功能

- 🌐 **服务发现**: 支持 Consul、Etcd、Kubernetes、Nacos
- 🔄 **协议转换**: HTTP/HTTPS 到 gRPC 转换
- ☁️ **云原生**: 支持阿里云、AWS、Azure、GCP 集成
- 📈 **自动扩缩容**: 基于指标的自动扩缩容
- 💾 **缓存系统**: Redis 分布式缓存
- 📋 **限流保护**: 多级限流保护机制

## 快速开始

### 环境要求

- **前端**: Node.js 18+ (用于 React 开发环境)
- **后端**: Go 1.22+
- **模型服务**: Python 3.8+ (支持 PyTorch 和 Transformers)
- **缓存**: Redis (可选，用于高级功能)
- **容器**: Docker & Docker Compose (推荐)

### 🚀 一键启动完整三层架构

#### 方式一：Docker Compose (推荐)

```bash
# 克隆项目
git clone <repository-url>
cd go-aigateway

# 一键启动完整环境 (前端 + 后端 + Python 模型服务 + Redis + PostgreSQL)
cd deployment
docker-compose -f docker-compose.dev.yml up -d

# 访问服务
# 前端: http://localhost:3000
# 后端: http://localhost:8080  
# Python 模型服务: http://localhost:5000
# Redis: localhost:6379
```

#### 方式二：分别启动各层服务

1. **启动后端服务 (Go)**

```bash
# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，配置必要的参数

# 启动后端
go run main.go
```

2. **启动 Python 模型服务**

```bash
cd python

# 安装依赖
pip install -r requirements.txt

# 启动模型服务 (支持本地模型和第三方 API)
python server.py --host 0.0.0.0 --port 5000 --model-type chat --model-size small

# 启用第三方模型 (阿里云百炼)
export BAILIAN_API_KEY=your_api_key
python server.py --use-third-party
```

3. **启动前端服务 (React)**

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

### Docker 部署

#### 开发环境

```bash
# 启动完整开发环境
cd deployment
docker-compose -f docker-compose.dev.yml up -d

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f
```

#### 生产环境

```bash
# 构建生产镜像
docker build -t ai-gateway-backend -f deployment/Dockerfile .
docker build -t ai-gateway-frontend -f deployment/Dockerfile.react .
docker build -t ai-gateway-python -f deployment/Dockerfile.python .

# 运行生产容器
docker-compose up -d
```

## 配置说明

### 基础配置

| 环境变量           | 说明            | 默认值   |
| ------------------ | --------------- | -------- |
| `PORT`           | Go 后端服务端口 | `8080` |
| `TARGET_API_URL` | 目标 API 地址   | -        |
| `TARGET_API_KEY` | 目标 API 密钥   | -        |
| `JWT_SECRET`     | JWT 签名密钥    | -        |

### 本地模型配置

| 环境变量                | 说明                | 默认值        |
| ----------------------- | ------------------- | ------------- |
| `LOCAL_MODEL_ENABLED` | 启用本地模型服务    | `true`      |
| `LOCAL_MODEL_HOST`    | Python 模型服务地址 | `localhost` |
| `LOCAL_MODEL_PORT`    | Python 模型服务端口 | `5000`      |
| `PYTHON_PATH`         | Python 解释器路径   | `python`    |
| `MODEL_PATH`          | 模型存储路径        | `./models`  |

### 第三方模型配置

| 环境变量                  | 说明                | 默认值    |
| ------------------------- | ------------------- | --------- |
| `BAILIAN_API_KEY`       | 阿里云百炼 API 密钥 | -         |
| `USE_THIRD_PARTY_MODEL` | 启用第三方模型      | `false` |

### Redis 配置

| 环境变量           | 说明         | 默认值             |
| ------------------ | ------------ | ------------------ |
| `REDIS_ENABLED`  | 启用 Redis   | `false`          |
| `REDIS_ADDR`     | Redis 地址   | `localhost:6379` |
| `REDIS_PASSWORD` | Redis 密码   | -                  |
| `REDIS_DB`       | Redis 数据库 | `0`              |

### 高级配置

详细配置请参考 `.env.example` 文件。

## API 文档

### 前端 API 端点

- `GET /` - React 前端主页面
- `GET /dashboard` - 仪表盘页面
- `GET /local-models` - 本地模型管理页面
- `GET /service-sources` - 服务源管理页面

### 后端认证端点

- `POST /auth/login` - 用户登录
- `POST /auth/refresh` - 刷新令牌

### API 代理端点 (Go 后端)

- `POST /v1/chat/completions` - 聊天补全代理
- `POST /v1/completions` - 文本补全代理
- `GET /v1/models` - 获取模型列表代理

### 本地模型端点 (Go 后端 → Python 服务)

- `POST /local/chat/completions` - 本地聊天补全
- `POST /local/completions` - 本地文本补全
- `POST /local/embeddings` - 本地向量嵌入
- `GET /local/models` - 获取本地模型列表

### 本地模型管理端点 (Go 后端)

- `GET /api/local/models` - 列出可用模型
- `POST /api/local/models/{id}/start` - 启动模型
- `POST /api/local/models/{id}/stop` - 停止模型
- `POST /api/local/models/{id}/download` - 下载模型
- `GET /api/local/models/{id}/status` - 模型状态
- `PUT /api/local/models/{id}/settings` - 更新模型设置

### Python 模型服务原生端点

- `GET /health` - 健康检查
- `POST /v1/chat/completions` - 聊天补全 (支持本地和第三方)
- `POST /v1/completions` - 文本补全 (支持本地和第三方)
- `POST /v1/embeddings` - 向量嵌入 (支持本地和第三方)
- `GET /v1/models` - 可用模型列表

### 管理端点

- `GET /health` - 健康检查
- `GET /metrics` - Prometheus 指标
- `GET /api/v1/monitoring/*` - 监控相关 API

## 🏗️ 三层架构设计

### 架构总览

```
                    ┌─────────────────────────────────────────┐
                    │          🎨 前端层 (React)              │
                    │  ┌─────────────────────────────────┐    │
                    │  │  • Dashboard 仪表盘              │    │
                    │  │  • LocalModelManagement 模型管理 │    │
                    │  │  • ServiceSources 服务源管理     │    │
                    │  │  • 统一 API 客户端              │    │
                    │  └─────────────────────────────────┘    │
                    └──────────────────┬──────────────────────┘
                                       │ HTTP/RESTful API
                    ┌──────────────────▼──────────────────────┐
                    │         ⚙️ 后端层 (Go Gateway)          │
                    │  ┌─────────────────────────────────┐    │
                    │  │  • 🔐 Authentication & Security │    │
                    │  │  • 🚀 API Proxy & Rate Limiting │    │
                    │  │  • 📊 Monitoring & Metrics      │    │
                    │  │  • 🤖 Local Model Bridge        │    │
                    │  │  • ☁️ Cloud Integration         │    │
                    │  └─────────────────────────────────┘    │
                    └──────────────────┬──────────────────────┘
                                       │ HTTP API Calls
                    ┌──────────────────▼──────────────────────┐
                    │        🧠 模型服务层 (Python)           │
                    │  ┌─────────────────────────────────┐    │
                    │  │  • 💬 Chat Completions         │    │
                    │  │  • 📝 Text Completions         │    │
                    │  │  • 🔍 Vector Embeddings        │    │
                    │  │  • 🌐 Third-party API 集成     │    │
                    │  │  • 🔄 Model Management         │    │
                    │  └─────────────────────────────────┘    │
                    └─────────────────────────────────────────┘
                                       │
            ┌──────────────────────────┼──────────────────────────┐
            │                          │                          │
    ┌───────▼────────┐    ┌───────────▼──────────┐    ┌─────────▼──────┐
    │  🗄️ Redis Cache │    │ 🤖 本地模型存储      │    │ ☁️ 第三方 API   │
    │  • 缓存         │    │ • Transformers      │    │ • 阿里云百炼     │
    │  • 限流         │    │ • PyTorch Models    │    │ • OpenAI 兼容   │
    │  • 监控数据     │    │ • Hugging Face      │    │ • 云端模型       │
    └────────────────┘    └─────────────────────┘    └────────────────┘
```

### 数据流向

1. **用户请求**: React 前端 → Go 后端 API Gateway
2. **请求处理**: Go 后端进行认证、限流、监控
3. **模型调用**: Go 后端 → Python 模型服务 (本地/第三方)
4. **响应返回**: Python 服务 → Go 后端 → React 前端
5. **数据缓存**: Redis 缓存热点数据和监控指标

### 服务通信

- **前后端**: HTTP/HTTPS RESTful API
- **后端与Python**: HTTP API (localhost:5000)
- **第三方集成**: HTTPS API (阿里云百炼等)
- **缓存通信**: Redis 协议

## 开发

### 项目结构

```
├── main.go                 # Go 后端主程序入口
├── frontend/              # 🎨 前端层 (React + TypeScript)
│   ├── src/
│   │   ├── pages/         # 页面组件
│   │   │   ├── Dashboard.tsx
│   │   │   ├── LocalModelManagement.tsx
│   │   │   └── ServiceSources.tsx
│   │   ├── services/      # API 服务层
│   │   │   └── api.ts
│   │   └── components/    # 通用组件
│   ├── package.json       # 前端依赖
│   └── vite.config.ts     # Vite 构建配置
├── internal/              # ⚙️ 后端层 (Go)
│   ├── config/           # 配置管理
│   ├── handlers/         # HTTP 处理器
│   │   ├── localmodel.go        # 本地模型 API
│   │   └── localmodel_manager.go # 模型管理 API
│   ├── middleware/       # 中间件
│   ├── security/         # 安全认证
│   ├── monitoring/       # 监控系统
│   ├── localmodel/       # 本地模型集成
│   │   ├── manager.go           # 模型管理器
│   │   ├── python_model.go      # Python 服务桥接
│   │   └── model_manager.go     # 模型生命周期管理
│   └── router/           # 路由配置
├── python/               # 🧠 模型服务层 (Python)
│   ├── server.py         # Flask 模型服务器
│   ├── error_handling.py # 错误处理和验证
│   ├── requirements.txt  # Python 依赖
│   └── test_server.py    # 模型服务测试
└── deployment/           # 🐳 部署配置
    ├── docker-compose.dev.yml    # 开发环境
    ├── Dockerfile               # Go 后端镜像
    ├── Dockerfile.react         # React 前端镜像
    └── Dockerfile.python        # Python 模型服务镜像
```

### 开发命令

#### 🎨 前端开发

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 代码检查
npm run lint
```

#### ⚙️ 后端开发

```bash
# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 构建
make build

# 开发模式运行
make dev

# 清理
make clean
```

#### 🧠 Python 模型服务开发

```bash
cd python

# 安装依赖
pip install -r requirements.txt

# 运行测试
python test_server.py

# 启动服务 (本地模型)
python server.py --model-type chat --model-size small

# 启动服务 (第三方模型)
python server.py --use-third-party
```

#### 🐳 Docker 开发

```bash
# 启动完整开发环境
cd deployment
docker-compose -f docker-compose.dev.yml up -d

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看实时日志
docker-compose -f docker-compose.dev.yml logs -f

# 停止服务
docker-compose -f docker-compose.dev.yml down
```

## 监控和观测

- **Prometheus 指标**: `/metrics` 端点提供详细的系统指标
- **健康检查**: `/health` 端点提供服务状态
- **日志**: 支持 JSON 和文本格式的结构化日志
- **分布式追踪**: 集成 OpenTelemetry (计划中)

## 安全特性

- 🔐 **多重认证**: JWT、API Key、阿里云 RAM
- 🛡️ **请求验证**: 请求大小限制、内容验证
- 🚫 **安全头**: 完整的 HTTP 安全头设置
- 🔒 **HTTPS**: 强制 HTTPS (生产环境)
- 📊 **审计日志**: 详细的操作审计日志

## 性能优化

- ⚡ **连接池**: HTTP 客户端连接池优化
- 💾 **缓存**: Redis 分布式缓存
- 🔄 **批处理**: 请求批处理优化
- 📊 **压缩**: Gzip 响应压缩
- 🎯 **限流**: 多级限流保护

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/new-feature`)
3. 提交更改 (`git commit -am 'Add new feature'`)
4. 推送分支 (`git push origin feature/new-feature`)
5. 创建 Pull Request

## 联系方式

如有问题或建议，请提交 Issue 或联系维护团队。
