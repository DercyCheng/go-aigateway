# AI Gateway

一个基于 OpenAI API 兼容的云原生 AI 网关，支持代理请求到阿里云通义千问等 AI 服务。

## 功能特性

- 🔑 OpenAI API 兼容接口
- 🛡️ API 密钥认证
- 🚦 请求频率限制
- 🔄 请求代理和转发
- 📊 健康检查
- 🐳 Docker 容器化支持
- ☸️ Kubernetes 云原生部署
- 📋 结构化日志记录

## 快速开始

### 1. 环境配置

复制环境变量模板：
```bash
cp .env.example .env
```

编辑 `.env` 文件，配置必要的参数：
```bash
# 目标 API 配置
TARGET_API_KEY=your-dashscope-api-key-here

# 网关 API 密钥（多个用逗号分隔）
GATEWAY_API_KEYS=sk-gateway-key1,sk-gateway-key2
```

### 2. 本地运行

#### 方法一：直接运行
```bash
# 安装依赖
go mod tidy

# 运行服务
go run main.go
```

#### 方法二：Docker 运行
```bash
# 构建镜像
docker build -t ai-gateway .

# 运行容器
docker run -p 8080:8080 --env-file .env ai-gateway
```

#### 方法三：Docker Compose
```bash
# 设置环境变量
export DASHSCOPE_API_KEY="your-dashscope-key"
export GATEWAY_API_KEYS="sk-gateway-key1,sk-gateway-key2"

# 启动服务
docker-compose up -d
```

### 3. Kubernetes 部署

```bash
# 更新 Secret 中的 API 密钥
kubectl apply -f k8s-deployment.yaml
```

## API 使用

### 健康检查
```bash
curl http://localhost:8080/health
```

### Chat Completions
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-gateway-key1" \
  -d '{
    "model": "qwen-turbo",
    "messages": [
      {
        "role": "user",
        "content": "Hello, how are you?"
      }
    ]
  }'
```

### 获取模型列表
```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-gateway-key1"
```

## 配置说明

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `GIN_MODE` | `release` | Gin 运行模式 |
| `TARGET_API_URL` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | 目标 API 地址 |
| `TARGET_API_KEY` | - | 目标 API 密钥（必填） |
| `GATEWAY_API_KEYS` | - | 网关 API 密钥列表（必填） |
| `LOG_LEVEL` | `info` | 日志级别 |
| `LOG_FORMAT` | `json` | 日志格式 |
| `RATE_LIMIT_REQUESTS_PER_MINUTE` | `60` | 每分钟请求限制 |
| `HEALTH_CHECK_ENABLED` | `true` | 启用健康检查 |

## 项目结构

```
go-aigateway/
├── main.go                 # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go      # 配置管理
│   ├── handlers/
│   │   └── handlers.go    # 请求处理器
│   ├── middleware/
│   │   └── middleware.go  # 中间件
│   └── router/
│       └── router.go      # 路由配置
├── Dockerfile             # Docker 构建文件
├── docker-compose.yml     # Docker Compose 配置
├── k8s-deployment.yaml    # Kubernetes 部署配置
├── .env.example          # 环境变量模板
├── go.mod                # Go 模块定义
└── README.md             # 项目说明
```

## 安全特性

- ✅ API 密钥认证
- ✅ 请求速率限制
- ✅ CORS 支持
- ✅ 请求日志记录
- ✅ 健康检查端点

## 监控和日志

- 结构化 JSON 日志输出
- 请求/响应日志记录
- 健康检查端点
- Kubernetes 就绪性和存活性探针

## 开发

### 添加新的中间件

在 `internal/middleware/middleware.go` 中添加新的中间件函数。

### 添加新的处理器

在 `internal/handlers/handlers.go` 中添加新的请求处理器。

### 修改路由

在 `internal/router/router.go` 中添加或修改路由配置。

## 许可证

MIT License
