# AI Gateway

一个功能丰富的 AI 服务网关，提供统一的 API 接口来管理和代理多个 AI 服务。

## 功能特性

### 核心功能
- 🚀 **API 代理**: 统一代理多个 AI 服务 API
- 🔐 **多重认证**: 支持 JWT、API Key、RAM 认证
- 📊 **监控指标**: 实时监控和 Prometheus 指标
- 🔄 **负载均衡**: 智能请求分发和故障转移
- 🛡️ **安全防护**: 完整的安全中间件和防护措施

### 高级功能
- 🌐 **服务发现**: 支持 Consul、Etcd、Kubernetes、Nacos
- 🔄 **协议转换**: HTTP/HTTPS 到 gRPC 转换
- ☁️ **云原生**: 支持阿里云、AWS、Azure、GCP 集成
- 🤖 **本地模型**: Python 本地模型服务集成
- 📈 **自动扩缩容**: 基于指标的自动扩缩容
- 💾 **缓存系统**: Redis 分布式缓存
- 📋 **限流保护**: 多级限流保护机制

## 快速开始

### 环境要求
- Go 1.22+
- Redis (可选，用于高级功能)
- Python 3.8+ (可选，用于本地模型)

### 安装和运行

1. **克隆项目**
```bash
git clone <repository-url>
cd go-aigateway
```

2. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件，配置必要的参数
```

3. **安装依赖**
```bash
go mod download
```

4. **运行服务**
```bash
# 开发环境
make dev

# 生产环境
make build
./ai-gateway
```

### Docker 部署

```bash
# 单容器部署
docker build -t ai-gateway .
docker run -p 8080:8080 ai-gateway

# 完整环境部署
docker-compose up -d
```

## 配置说明

### 基础配置
| 环境变量 | 说明 | 默认值 |
|---------|------|-------|
| `PORT` | 服务端口 | `8080` |
| `TARGET_API_URL` | 目标 API 地址 | - |
| `TARGET_API_KEY` | 目标 API 密钥 | - |
| `JWT_SECRET` | JWT 签名密钥 | - |

### 高级配置
详细配置请参考 `.env.example` 文件。

## API 文档

### 认证端点
- `POST /auth/login` - 用户登录
- `POST /auth/refresh` - 刷新令牌

### API 代理端点
- `POST /v1/chat/completions` - 聊天补全
- `POST /v1/completions` - 文本补全
- `GET /v1/models` - 获取模型列表

### 管理端点
- `GET /health` - 健康检查
- `GET /metrics` - Prometheus 指标
- `GET /api/v1/monitoring/*` - 监控相关 API

### 本地模型端点 (可选)
- `POST /local/chat/completions` - 本地聊天补全
- `POST /local/completions` - 本地文本补全
- `POST /local/embeddings` - 本地向量嵌入

## 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend UI   │    │   Load Balancer │    │   AI Services   │
│                 │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼──────────────┐
                    │        AI Gateway          │
                    │  ┌─────────────────────┐   │
                    │  │   Authentication    │   │
                    │  └─────────────────────┘   │
                    │  ┌─────────────────────┐   │
                    │  │   Rate Limiting     │   │
                    │  └─────────────────────┘   │
                    │  ┌─────────────────────┐   │
                    │  │   Monitoring        │   │
                    │  └─────────────────────┘   │
                    │  ┌─────────────────────┐   │
                    │  │   Local Models      │   │
                    │  └─────────────────────┘   │
                    └────────────────────────────┘
                                 │
                    ┌─────────────▼──────────────┐
                    │        Redis Cache         │
                    └────────────────────────────┘
```

## 开发

### 项目结构
```
├── main.go                 # 主程序入口
├── internal/              # 内部包
│   ├── config/           # 配置管理
│   ├── handlers/         # HTTP 处理器
│   ├── middleware/       # 中间件
│   ├── security/         # 安全认证
│   ├── monitoring/       # 监控系统
│   └── localmodel/       # 本地模型
├── frontend/             # 前端界面 (React + TypeScript)
├── deployments/          # 部署配置
└── python/              # Python 模型服务
```

### 开发命令
```bash
# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 构建
make build

# 清理
make clean
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

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 联系方式

如有问题或建议，请提交 Issue 或联系维护团队。
