#!/bin/bash

# AI Gateway 开发环境停止脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 主函数
main() {
    echo -e "${BLUE}"
    echo "=============================================="
    echo "    🛑 AI Gateway 开发环境停止器"
    echo "    📅 $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=============================================="
    echo -e "${NC}"
    
    # 进入部署目录
    cd "$(dirname "$0")"
    
    log_info "停止 AI Gateway 开发环境..."
    
    # 停止所有服务
    docker-compose -f docker-compose.dev.yml down
    
    log_success "所有服务已停止"
    
    # 询问是否清理数据卷
    echo ""
    read -p "是否清理数据卷? (会删除数据库数据) [y/N]: " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理数据卷..."
        docker-compose -f docker-compose.dev.yml down -v
        log_warning "数据卷已清理，所有数据已删除"
    fi
    
    # 询问是否清理镜像
    echo ""
    read -p "是否清理未使用的镜像? [y/N]: " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理未使用的镜像..."
        docker image prune -f
        log_success "未使用的镜像已清理"
    fi
    
    echo ""
    log_success "🎉 AI Gateway 开发环境已完全停止!"
    echo ""
    echo -e "${YELLOW}💡 重新启动: ./start-dev.sh${NC}"
}

# 执行主函数
main "$@"
