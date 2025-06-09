#!/bin/bash

# 停止开发环境脚本

set -e

echo "🛑 停止 AI Gateway 开发环境..."

# 进入部署目录
cd "$(dirname "$0")"

echo "📦 停止所有服务..."
docker-compose -f docker-compose.dev.yml down

echo "🧹 清理未使用的 Docker 资源..."
docker system prune -f --volumes

echo "✅ 开发环境已停止并清理完成!"
