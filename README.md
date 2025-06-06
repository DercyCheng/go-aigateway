# Go AI Gateway

AI网关服务，提供统一的AI服务接入点。

## 项目结构

```
.
├── main.go                 # 应用程序入口点
├── go.mod                  # Go模块定义
├── go.sum                  # Go模块依赖锁定
├── Makefile               # 构建脚本
├── .env.example           # 环境变量示例
├── internal/              # 内部包（不对外暴露）
│   ├── cloud/            # 云服务集成
│   ├── config/           # 配置管理
│   ├── discovery/        # 服务发现
│   ├── handlers/         # HTTP处理器
│   ├── middleware/       # 中间件
│   ├── protocol/         # 协议转换
│   ├── ram/              # 访问控制
│   └── router/           # 路由配置
├── deployments/          # 部署相关文件
│   ├── docker-compose.yml    # Docker Compose配置
│   ├── Dockerfile           # Docker镜像构建文件
│   ├── k8s-deployment.yaml  # Kubernetes部署文件
│   ├── nginx.conf          # Nginx配置
│   └── monitoring/         # 监控配置
│       ├── docker-compose.monitoring.yml
│       ├── grafana-dashboard.json
│       ├── grafana-datasource.yml
│       └── prometheus.yml
├── scripts/              # 脚本文件
│   └── init-db.sql      # 数据库初始化脚本
├── build/               # 构建输出目录
└── docs/                # 文档目录
```

## 快速开始

### 本地开发

1. 安装依赖：
   ```bash
   go mod download
   ```

2. 配置环境变量：
   ```bash
   cp .env.example .env
   # 编辑 .env 文件，设置相应的配置
   ```

3. 运行应用：
   ```bash
   make run
   # 或者
   go run main.go
   ```

### Docker部署

```bash
# 构建并运行
make docker-build
make docker-run

# 或者使用docker-compose
cd deployments
docker-compose up -d
```

### Kubernetes部署

```bash
kubectl apply -f deployments/k8s-deployment.yaml
```

## 开发指南

### 目录说明

- `internal/`: 包含所有内部业务逻辑代码
- `deployments/`: 包含所有部署相关的配置文件
- `scripts/`: 包含数据库脚本和其他实用脚本
- `build/`: 构建输出目录（通常被.gitignore忽略）
- `docs/`: 项目文档

### 构建

```bash
# 构建应用
make build

# 清理构建文件
make clean

# 运行测试
make test
```

## 许可证

[添加许可证信息]
