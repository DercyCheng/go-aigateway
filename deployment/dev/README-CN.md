# AI Gateway 中国大陆开发环境部署指南

## 概述

本指南提供了针对中国大陆网络环境优化的 AI Gateway 开发环境部署方案。包含了国内镜像源、AI服务配置和网络优化等特性。

## 特性

### 🇨🇳 中国大陆优化
- 使用阿里云 Docker 镜像源
- 配置清华大学 PyPI 镜像源
- 使用 goproxy.cn Go 模块代理
- 优化网络超时和重试设置

### 🤖 国内AI服务支持
- 智谱AI (ChatGLM)
- 百度千帆大模型平台
- 阿里云通义千问
- 腾讯混元大模型
- 讯飞星火认知大模型
- 字节跳动豆包大模型

### 🔧 开发工具
- 热重载支持
- 完整的监控栈 (Prometheus + Grafana)
- Nginx 反向代理
- Redis 缓存
- PostgreSQL 数据库

## 快速开始

### 1. 环境准备

确保您的系统已安装：
- Docker (>= 20.10)
- Docker Compose (>= 2.0)
- Git

### 2. 克隆项目

```bash
git clone <repository-url>
cd go-aigateway/deployment/dev
```

### 3. 配置环境变量

复制并编辑环境配置文件：

```bash
cp .env.cn .env.cn.local
```

编辑 `.env.cn.local` 文件，配置您的AI服务API密钥：

```bash
# 智谱AI
ZHIPU_API_KEY=your_actual_api_key_here

# 百度千帆
QIANFAN_API_KEY=your_actual_api_key_here
QIANFAN_SECRET_KEY=your_actual_secret_key_here

# 阿里云通义千问
DASHSCOPE_API_KEY=your_actual_api_key_here

# 其他服务...
```

### 4. 部署服务

#### 使用部署脚本 (推荐)

```bash
# 给脚本执行权限
chmod +x deploy-cn.sh

# 运行完整部署
./deploy-cn.sh

# 或者使用具体命令
./deploy-cn.sh start    # 启动服务
./deploy-cn.sh stop     # 停止服务
./deploy-cn.sh restart  # 重启服务
./deploy-cn.sh status   # 检查状态
./deploy-cn.sh logs     # 查看日志
./deploy-cn.sh clean    # 清理环境
```

#### 手动部署

```bash
# 创建必要目录
mkdir -p logs model_cache python_logs ssl

# 构建并启动服务
docker-compose -f docker-compose.cn.yml up -d --build

# 查看服务状态
docker-compose -f docker-compose.cn.yml ps
```

## 服务访问

部署完成后，您可以访问以下服务：

| 服务 | 地址 | 说明 |
|------|------|------|
| AI Gateway API | http://localhost:8080 | 主API服务 |
| Python 模型服务 | http://localhost:5000 | 本地模型服务 |
| Nginx 代理 | http://localhost:80 | 反向代理入口 |
| Prometheus | http://localhost:9091 | 监控指标 |
| Grafana | http://localhost:3001 | 监控仪表板 (admin/admin) |
| PostgreSQL | localhost:5432 | 数据库 (postgres/postgres) |
| Redis | localhost:6379 | 缓存服务 |

## 网络配置

### Docker 镜像加速器

为了加快镜像拉取速度，建议配置 Docker 镜像加速器：

**Linux:**
```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": [
    "https://mirror.ccs.tencentyun.com",
    "https://registry.cn-hangzhou.aliyuncs.com"
  ]
}
EOF
sudo systemctl restart docker
```

**Windows (Docker Desktop):**
1. 打开 Docker Desktop
2. 进入 Settings > Docker Engine
3. 添加镜像加速器配置
4. 重启 Docker Desktop

### 代理设置

如果您的网络环境需要代理，请在 `.env.cn` 文件中配置：

```bash
HTTP_PROXY=http://your-proxy:port
HTTPS_PROXY=http://your-proxy:port
NO_PROXY=localhost,127.0.0.1,::1
```

## 开发模式

### 热重载

开发环境默认启用热重载功能：
- Go 代码变更会自动重新编译
- Python 代码变更会自动重新加载
- 前端代码可通过 Vite 开发服务器实现热重载

### 调试

#### 查看日志
```bash
# 查看所有服务日志
docker-compose -f docker-compose.cn.yml logs -f

# 查看特定服务日志
docker-compose -f docker-compose.cn.yml logs -f go-aigateway
docker-compose -f docker-compose.cn.yml logs -f python-model-cn
```

#### 进入容器
```bash
# 进入主服务容器
docker exec -it go-aigateway-cn-dev sh

# 进入Python服务容器
docker exec -it python-model-cn-dev bash
```

### 性能监控

访问 Grafana (http://localhost:3001) 查看性能指标：
- API 请求延迟
- 错误率统计
- 资源使用情况
- AI 服务响应时间

## 故障排除

### 常见问题

#### 1. 端口冲突
如果遇到端口冲突，请修改 `docker-compose.cn.yml` 中的端口映射。

#### 2. 镜像拉取失败
- 检查网络连接
- 配置 Docker 镜像加速器
- 使用代理设置

#### 3. AI 服务调用失败
- 检查 API 密钥配置
- 验证网络连接
- 查看服务日志

#### 4. 数据库连接失败
```bash
# 重启数据库服务
docker-compose -f docker-compose.cn.yml restart postgres-cn

# 检查数据库状态
docker exec postgres-cn-dev pg_isready -U postgres
```

### 日志分析

重要日志位置：
- 应用日志: `./logs/`
- Python 服务日志: `./python_logs/`
- Nginx 日志: 容器内 `/var/log/nginx/`

### 重置环境

如需完全重置环境：

```bash
# 停止并删除所有容器和卷
./deploy-cn.sh clean

# 重新部署
./deploy-cn.sh
```

## 生产环境迁移

当您准备将应用迁移到生产环境时：

1. 修改环境变量（移除调试选项）
2. 配置 HTTPS 和 SSL 证书
3. 设置更严格的安全策略
4. 配置外部数据库和缓存
5. 设置日志收集和监控

## 支持

如遇到问题，请：
1. 查看日志文件
2. 检查环境配置
3. 参考本文档的故障排除部分
4. 提交 Issue 到项目仓库

## 更新

要更新部署环境：

```bash
# 拉取最新代码
git pull

# 重新构建和部署
./deploy-cn.sh clean
./deploy-cn.sh
```
