# AI Gateway 开发环境部署文档

## 概述

这是一个完整的 Docker 开发环境配置，专门针对中国内地网络环境优化，包含以下组件：

- **Go 后端服务** (端口 8080) - 主要的 API 网关服务
- **Python 模型服务** (端口 5000) - AI 模型推理服务
- **React 前端** (端口 3000) - Web 用户界面
- **Redis** (端口 6379) - 缓存和会话存储
- **PostgreSQL** (端口 5432) - 主数据库
- **Nginx** (端口 80) - 反向代理和负载均衡

## 中国内地优化配置

### 镜像源配置
- **Go**: 使用 `goproxy.cn` 作为模块代理
- **Python**: 使用清华大学 PyPI 镜像源
- **Node.js**: 使用 npmmirror.com 镜像源
- **Alpine Linux**: 使用阿里云镜像源

### 网络优化
- 所有依赖下载都使用国内镜像源
- Docker 镜像使用官方或阿里云镜像
- 支持离线开发和本地缓存

## 快速开始

### 1. 前置要求

确保已安装以下工具：
- Docker (>=20.10)
- Docker Compose (>=2.0)

### 2. 启动开发环境

```bash
# 进入项目根目录
cd /Users/dercyc/go/src/pro/go-aigateway

# 给脚本执行权限
chmod +x deployment/start-dev.sh deployment/stop-dev.sh

# 启动开发环境
./deployment/start-dev.sh
```

### 3. 访问服务

启动成功后，可以通过以下地址访问：

- **前端界面**: http://localhost:3000
- **API 网关**: http://localhost:8080
- **Python 模型**: http://localhost:5000
- **完整应用**: http://localhost (通过 Nginx)

### 4. 停止环境

```bash
./deployment/stop-dev.sh
```

## 开发工作流

### 热重载支持

- **Go 后端**: 使用 Air 实现热重载，代码修改后自动重新构建
- **React 前端**: 使用 Vite 开发服务器，支持 HMR
- **Python 服务**: 使用 Flask 开发模式，代码修改后自动重启

### 数据持久化

- **Redis 数据**: 存储在 `redis_data` volume
- **PostgreSQL 数据**: 存储在 `postgres_data` volume
- **Go 模块缓存**: 存储在 `go_modules` volume
- **Python 包缓存**: 存储在 `python_packages` volume
- **Node.js 模块**: 存储在 `node_modules` volume

### 日志查看

```bash
# 查看所有服务日志
docker-compose -f deployment/docker-compose.dev.yml logs -f

# 查看特定服务日志
docker-compose -f deployment/docker-compose.dev.yml logs -f go-backend
docker-compose -f deployment/docker-compose.dev.yml logs -f python-models
docker-compose -f deployment/docker-compose.dev.yml logs -f react-frontend
```

### 服务管理

```bash
# 重启特定服务
docker-compose -f deployment/docker-compose.dev.yml restart go-backend

# 进入容器调试
docker-compose -f deployment/docker-compose.dev.yml exec go-backend sh
docker-compose -f deployment/docker-compose.dev.yml exec python-models bash

# 查看服务状态
docker-compose -f deployment/docker-compose.dev.yml ps
```

## 配置说明

### 环境变量

开发环境配置文件: `deployment/.env.dev`

主要配置项：
- 数据库连接信息
- Redis 连接信息
- JWT 密钥配置
- CORS 跨域配置
- 日志级别设置

### 网络配置

所有服务运行在 `aigateway-network` 网络中，内部通信使用容器名称作为主机名。

### 端口映射

| 服务 | 内部端口 | 外部端口 | 说明 |
|------|---------|---------|------|
| Go Backend | 8080 | 8080 | API 服务 |
| Python Models | 5000 | 5000 | 模型推理 |
| React Frontend | 3000 | 3000 | 前端开发服务器 |
| PostgreSQL | 5432 | 5432 | 数据库 |
| Redis | 6379 | 6379 | 缓存 |
| Nginx | 80 | 80 | 反向代理 |

## 故障排除

### 常见问题

1. **容器启动失败**
   ```bash
   # 检查日志
   docker-compose -f deployment/docker-compose.dev.yml logs [service_name]
   
   # 重新构建镜像
   docker-compose -f deployment/docker-compose.dev.yml build --no-cache [service_name]
   ```

2. **端口冲突**
   - 修改 `docker-compose.dev.yml` 中的端口映射
   - 或停止占用端口的其他服务

3. **网络连接问题**
   ```bash
   # 检查网络状态
   docker network ls
   docker network inspect deployment_aigateway-network
   ```

4. **权限问题**
   ```bash
   # 给脚本执行权限
   chmod +x deployment/*.sh
   ```

### 性能优化

1. **增加 Docker 资源分配**
   - CPU: 建议 4+ 核心
   - 内存: 建议 8GB+
   - 磁盘: 建议 SSD

2. **使用本地缓存**
   - Go modules 缓存会持久化
   - Python packages 缓存会持久化
   - Node.js modules 缓存会持久化

## 生产环境部署

生产环境部署请参考：
- `docker-compose.prod.yml` (待创建)
- 使用环境变量管理敏感信息
- 配置 HTTPS 和域名
- 设置监控和日志收集
- 配置备份策略

## 贡献指南

1. 修改配置文件前请备份
2. 测试配置更改是否正常工作
3. 更新相关文档
4. 提交 Pull Request

## 许可证

[在此添加许可证信息]
