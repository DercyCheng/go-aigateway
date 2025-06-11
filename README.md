# AI Gateway

## 📋 概述

AI Gateway 是一个高性能的AI服务网关，支持多种协议转换、服务发现和负载均衡。针对中国内地环境进行了优化，提供了完整的开发和生产环境配置。

## 🏗️ 架构图

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React 前端    │    │   Go 后端       │    │  Python 模型   │
│   (Port: 3000)  │───▶│  (Port: 8080)   │───▶│  (Port: 5000)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Nginx 代理    │    │  服务发现(Consul)│    │  监控(Prometheus)│
│   (Port: 80)    │    │  (Port: 8500)   │    │  (Port: 9090)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  PostgreSQL     │    │     Redis       │    │    Grafana      │
│  (Port: 5432)   │    │  (Port: 6379)   │    │  (Port: 3001)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 快速开始

### 系统要求

- **操作系统**: macOS, Linux, Windows (WSL2)
- **内存**: 推荐 8GB+ (最低 4GB)
- **存储**: 可用空间 10GB+
- **Docker**: 20.10+
- **Docker Compose**: 2.0+

### 一键启动开发环境

```bash
# 克隆项目
git clone <repository-url>
cd go-aigateway

# 快速启动 (包含构建和启动)
make quick-start

# 或者分步执行
make check-deps    # 检查依赖
make dev-build     # 构建镜像
make dev-up        # 启动服务
```

### 服务访问地址

| 服务       | 地址                          | 描述            |
| ---------- | ----------------------------- | --------------- |
| 前端应用   | http://localhost:80           | React 前端界面  |
| 后端API    | http://localhost:8080         | Go 后端服务     |
| 模型服务   | http://localhost:5000         | Python 模型推理 |
| 监控面板   | http://localhost:9091/metrics | Prometheus 指标 |
| Redis      | localhost:6379                | 缓存服务        |
| PostgreSQL | localhost:5432                | 数据库服务      |

## 🔧 配置说明

### 环境变量配置

#### 开发环境 (`.env.development`)

```bash
# 服务发现配置
SERVICE_DISCOVERY_ENABLED=true
SERVICE_DISCOVERY_TYPE=static

# 协议转换配置
PROTOCOL_CONVERSION_ENABLED=true
HTTP_TO_GRPC_ENABLED=true

# 监控配置
MONITORING_ENABLED=true
PROMETHEUS_ENABLED=true
```

#### 生产环境 (`.env.production`)

```bash
# 使用 Consul 服务发现
SERVICE_DISCOVERY_TYPE=consul
CONSUL_ADDR=consul:8500

# 安全配置
JWT_SECRET=your_production_jwt_secret
TLS_ENABLED=true

# 性能优化
CACHE_ENABLED=true
GZIP_ENABLED=true
```

### 服务发现配置

系统支持多种服务发现方式：

1. **静态配置** (开发环境推荐)

```json
{
  "python-models": {
    "host": "python-models",
    "port": 5000,
    "protocol": "http",
    "health_check": "/health"
  }
}
```

2. **Consul** (生产环境推荐)

```bash
SERVICE_DISCOVERY_TYPE=consul
CONSUL_ADDR=consul:8500
```

3. **Kubernetes** (云原生部署)

```bash
SERVICE_DISCOVERY_TYPE=kubernetes
K8S_NAMESPACE=default
```

### 协议转换功能

- **HTTP to gRPC**: 自动转换 HTTP 请求为 gRPC 调用
- **WebSocket**: 支持实时双向通信
- **GraphQL**: 提供统一的查询接口
- **REST API**: 标准的 RESTful API

## 🐳 Docker 配置

### 多阶段构建优化

每个服务都使用多阶段构建来优化镜像大小和构建时间：

```dockerfile
# 构建阶段 - 下载依赖和编译
FROM golang:1.22-alpine AS builder
# ... 构建逻辑

# 开发阶段 - 包含开发工具
FROM golang:1.22-alpine AS development
# ... 开发配置

# 生产阶段 - 最小运行时镜像
FROM alpine:3.18 AS production
# ... 生产配置
```

### 中国内地优化

- **Go**: 使用 `goproxy.cn` 代理
- **Python**: 使用清华大学 PyPI 镜像
- **Node.js**: 使用 npmmirror 镜像
- **Alpine**: 使用阿里云镜像源
- **HuggingFace**: 使用 `hf-mirror.com` 镜像

## 📊 监控与日志

### Prometheus 监控指标

- **应用指标**: 请求量、响应时间、错误率
- **系统指标**: CPU、内存、磁盘使用率
- **业务指标**: 模型推理次数、用户活跃度

### 健康检查

```bash
# 检查所有服务状态
make health-check

# 查看服务详情
make dev-status
```

### 日志管理

```bash
# 查看所有服务日志
make dev-logs

# 查看特定服务日志
docker logs aigateway-backend-dev -f
```

## 🔒 安全配置

### JWT 认证

```bash
# 设置 JWT 密钥 (生产环境必须修改)
JWT_SECRET=your_super_secret_jwt_key_change_in_production_2024
JWT_EXPIRY=24h
```

### CORS 配置

```bash
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://yourdomain.com
```

### Rate Limiting

```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=10
```

## 🚀 生产环境部署

### 1. 环境准备

```bash
# 复制并修改生产环境配置
cp deployment/.env.production.example deployment/.env.production
# 编辑配置文件，修改密码和密钥
vim deployment/.env.production
```

### 2. SSL 证书配置

```bash
# 创建 SSL 证书目录
mkdir -p deployment/ssl

# 复制证书文件
cp server.crt deployment/ssl/
cp server.key deployment/ssl/
```

### 3. 启动生产环境

```bash
# 构建生产镜像
make prod-build

# 启动生产服务
make prod-up

# 检查服务状态
make prod-status
```

### 4. 数据库迁移

```bash
# 运行数据库迁移
make db-migrate

# 备份数据库
make db-backup
```

## 🛠️ 常用命令

### 开发环境

```bash
make dev-up          # 启动开发环境
make dev-down        # 停止开发环境
make dev-restart     # 重启开发环境
make dev-logs        # 查看日志
make dev-status      # 查看状态
```

### 生产环境

```bash
make prod-up         # 启动生产环境
make prod-down       # 停止生产环境
make prod-restart    # 重启生产环境
make prod-logs       # 查看日志
make prod-status     # 查看状态
```

### 实用工具

```bash
make shell-backend   # 进入后端容器
make shell-model     # 进入模型容器
make shell-db        # 进入数据库
make test           # 运行测试
make benchmark      # 性能测试
```

## 🔧 故障排除

### 常见问题

1. **容器启动失败**

   ```bash
   # 检查日志
   docker logs <container-name>

   # 检查资源使用
   docker stats
   ```
2. **数据库连接失败**

   ```bash
   # 检查数据库状态
   docker exec aigateway-postgres-dev pg_isready

   # 查看数据库日志
   docker logs aigateway-postgres-dev
   ```
3. **模型加载缓慢**

   ```bash
   # 检查 HuggingFace 镜像配置
   echo $HF_ENDPOINT

   # 使用国内模型源
   HF_ENDPOINT=https://hf-mirror.com
   ```

### 性能优化

1. **启用缓存**

   ```bash
   CACHE_ENABLED=true
   REDIS_ENABLED=true
   ```
2. **调整资源限制**

   ```yaml
   deploy:
     resources:
       limits:
         memory: 4G
         cpus: '2.0'
   ```
3. **使用多实例**

   ```yaml
   deploy:
     replicas: 3
   ```

## 📈 扩展部署

### Kubernetes 部署

项目包含 Kubernetes 配置文件，支持云原生部署：

```bash
# 应用 Kubernetes 配置
kubectl apply -f k8s/

# 查看部署状态
kubectl get pods -n ai-gateway
```

### Docker Swarm 部署

```bash
# 初始化 Swarm
docker swarm init

# 部署服务栈
docker stack deploy -c docker-compose.prod.yml ai-gateway
```

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](../LICENSE) 文件了解详情。

## 📞 支持

如果您遇到问题或需要帮助，请：

1. 查看本文档的故障排除部分
2. 提交 [Issue](https://github.com/your-repo/go-aigateway/issues)
3. 加入我们的讨论社区
