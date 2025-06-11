#!/bin/bash
# 从 deployment/ 迁移到 configs/ 的迁移脚本

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployment"
CONFIGS_DIR="$PROJECT_ROOT/configs"

echo -e "${BLUE}AI Gateway 配置迁移工具${NC}"
echo "从 deployment/ 迁移到 configs/"
echo "================================"

# 检查是否存在旧的 deployment 目录
if [ ! -d "$DEPLOYMENT_DIR" ]; then
    echo -e "${GREEN}✓ 未发现旧的 deployment 目录，无需迁移${NC}"
    exit 0
fi

echo -e "${YELLOW}发现旧的 deployment 目录${NC}"

# 停止旧的服务
echo "1. 停止旧服务..."
if [ -f "$DEPLOYMENT_DIR/docker-compose.dev.yml" ]; then
    cd "$DEPLOYMENT_DIR"
    docker-compose -f docker-compose.dev.yml down 2>/dev/null || true
    docker-compose -f docker-compose.prod.yml down 2>/dev/null || true
    docker-compose -f docker-compose.monitor.yml down 2>/dev/null || true
fi

# 备份数据
echo "2. 创建数据备份..."
BACKUP_DIR="$PROJECT_ROOT/backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# 备份环境变量文件
if [ -f "$DEPLOYMENT_DIR/.env.development" ]; then
    cp "$DEPLOYMENT_DIR/.env.development" "$BACKUP_DIR/"
fi
if [ -f "$DEPLOYMENT_DIR/.env.production" ]; then
    cp "$DEPLOYMENT_DIR/.env.production" "$BACKUP_DIR/"
fi

# 备份数据库 (如果容器还在运行)
if docker ps | grep -q postgres; then
    echo "  备份数据库..."
    docker exec -t $(docker ps -q -f name=postgres) pg_dumpall -c -U postgres > "$BACKUP_DIR/postgres_backup.sql" 2>/dev/null || true
fi

# 备份 Redis 数据
if docker ps | grep -q redis; then
    echo "  备份 Redis..."
    docker exec -t $(docker ps -q -f name=redis) redis-cli BGSAVE 2>/dev/null || true
fi

echo -e "${GREEN}✓ 数据备份完成: $BACKUP_DIR${NC}"

# 迁移自定义配置
echo "3. 检查自定义配置..."

# 检查是否有自定义的环境变量需要迁移
if [ -f "$DEPLOYMENT_DIR/.env.development" ]; then
    echo -e "${YELLOW}  发现自定义开发环境配置，请手动检查差异：${NC}"
    echo "    旧文件: $DEPLOYMENT_DIR/.env.development"
    echo "    新文件: $CONFIGS_DIR/.env.development"
fi

if [ -f "$DEPLOYMENT_DIR/.env.production" ]; then
    echo -e "${YELLOW}  发现自定义生产环境配置，请手动检查差异：${NC}"
    echo "    旧文件: $DEPLOYMENT_DIR/.env.production"
    echo "    新文件: $CONFIGS_DIR/.env.production"
fi

# 检查自定义服务配置
CUSTOM_CONFIGS=""
if [ -f "$DEPLOYMENT_DIR/nginx/nginx.conf" ] && ! cmp -s "$DEPLOYMENT_DIR/nginx/nginx.conf" "$CONFIGS_DIR/services/nginx/nginx.conf" 2>/dev/null; then
    CUSTOM_CONFIGS+="nginx.conf "
fi

if [ -f "$DEPLOYMENT_DIR/postgres/postgresql-prod.conf" ]; then
    CUSTOM_CONFIGS+="postgresql.conf "
fi

if [ -n "$CUSTOM_CONFIGS" ]; then
    echo -e "${YELLOW}  发现自定义服务配置: $CUSTOM_CONFIGS${NC}"
    echo "  请手动检查并迁移这些配置到 configs/services/ 目录"
fi

# 更新项目引用
echo "4. 更新项目引用..."

# 更新 Makefile 中的引用
if [ -f "$PROJECT_ROOT/Makefile" ]; then
    if grep -q "deployment/" "$PROJECT_ROOT/Makefile"; then
        echo "  更新 Makefile..."
        sed -i.bak 's|deployment/|configs/|g' "$PROJECT_ROOT/Makefile"
    fi
fi

# 更新 README.md 中的引用
if [ -f "$PROJECT_ROOT/README.md" ]; then
    if grep -q "deployment/" "$PROJECT_ROOT/README.md"; then
        echo "  更新 README.md..."
        sed -i.bak 's|deployment/|configs/|g' "$PROJECT_ROOT/README.md"
    fi
fi

# 更新 .gitignore
if [ -f "$PROJECT_ROOT/.gitignore" ]; then
    if grep -q "deployment/" "$PROJECT_ROOT/.gitignore"; then
        echo "  更新 .gitignore..."
        sed -i.bak 's|deployment/|configs/|g' "$PROJECT_ROOT/.gitignore"
    fi
fi

echo -e "${GREEN}✓ 项目引用更新完成${NC}"

# 测试新配置
echo "5. 测试新配置..."
cd "$CONFIGS_DIR"

# 验证 docker-compose 文件语法
if docker-compose config > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Docker Compose 配置语法正确${NC}"
else
    echo -e "${RED}✗ Docker Compose 配置语法错误${NC}"
    docker-compose config
fi

# 提供清理建议
echo ""
echo -e "${BLUE}迁移完成！${NC}"
echo ""
echo "后续步骤："
echo "1. 使用新的管理脚本启动服务:"
echo "   ./configs/scripts/manage.sh dev"
echo ""
echo "2. 验证服务正常运行后，可以删除旧目录:"
echo "   rm -rf deployment/"
echo ""
echo "3. 备份文件位置:"
echo "   $BACKUP_DIR"
echo ""
echo "4. 查看新的配置文档:"
echo "   cat configs/README.md"

# 询问是否立即启动新服务
echo ""
read -p "是否立即启动开发环境进行测试? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}启动新的开发环境...${NC}"
    "$CONFIGS_DIR/scripts/manage.sh" dev
fi
