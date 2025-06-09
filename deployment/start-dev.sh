#!/bin/bash

# 开发环境启动脚本

set -e

echo "🚀 启动 AI Gateway 开发环境..."

# 检查 Docker 是否运行
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

# 检查 Docker Compose 是否可用
if ! command -v docker-compose >/dev/null 2>&1; then
    echo "❌ Docker Compose 未安装"
    exit 1
fi

# 进入部署目录
cd "$(dirname "$0")"

echo "📦 拉取最新镜像..."
docker-compose -f docker-compose.dev.yml pull

echo "🔨 构建服务..."
docker-compose -f docker-compose.dev.yml build

echo "🧹 清理旧容器..."
docker-compose -f docker-compose.dev.yml down --remove-orphans

echo "🌟 启动开发环境..."
docker-compose -f docker-compose.dev.yml up -d

echo "⏳ 等待服务启动..."
sleep 10

echo "🔍 检查服务状态..."
docker-compose -f docker-compose.dev.yml ps

echo ""
echo "✅ 开发环境启动成功!"
echo ""
echo "📋 服务地址:"
echo "  🌐 前端 (React):     http://localhost:3000"
echo "  🚪 后端 (Go):        http://localhost:8080"
echo "  🐍 Python 模型:      http://localhost:5000"
echo "  🔄 Nginx 代理:       http://localhost"
echo "  📊 Redis:            localhost:6379"
echo "  🗄️  PostgreSQL:      localhost:5432"
echo ""
echo "🛠️  常用命令:"
echo "  查看日志: docker-compose -f docker-compose.dev.yml logs -f [service_name]"
echo "  重启服务: docker-compose -f docker-compose.dev.yml restart [service_name]"
echo "  停止服务: docker-compose -f docker-compose.dev.yml down"
echo "  进入容器: docker-compose -f docker-compose.dev.yml exec [service_name] sh"
echo ""
echo "🎉 开发愉快!"
