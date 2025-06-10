#!/bin/bash

# AI Gateway 开发环境启动脚本 (中国内地优化版)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

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

# 检查函数
check_requirements() {
    log_info "检查环境要求..."
    
    # 检查 Docker
    if ! command -v docker >/dev/null 2>&1; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker 未运行，请先启动 Docker"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose >/dev/null 2>&1; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    # 检查可用内存 (推荐至少8GB)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        TOTAL_MEM_GB=$(($(sysctl -n hw.memsize) / 1024 / 1024 / 1024))
    else
        TOTAL_MEM_GB=$(($(grep MemTotal /proc/meminfo | awk '{print $2}') / 1024 / 1024))
    fi
    
    if [ "$TOTAL_MEM_GB" -lt 8 ]; then
        log_warning "可用内存不足8GB (当前: ${TOTAL_MEM_GB}GB)，可能影响模型服务性能"
    fi
    fi
    
    # 检查磁盘空间 (至少需要5GB)
    available_space=$(df -h . | awk 'NR==2 {print $4}' | sed 's/G.*//')
    if [ "$available_space" -lt 5 ]; then
        log_warning "磁盘空间不足，建议至少保留5GB空间"
    fi
    
    log_success "环境检查通过"
}

# 清理函数
cleanup_old_containers() {
    log_info "清理旧容器和资源..."
    
    # 停止并删除旧容器
    docker-compose -f docker-compose.dev.yml down --remove-orphans 2>/dev/null || true
    
    # 清理未使用的镜像和网络
    docker system prune -f >/dev/null 2>&1 || true
    
    log_success "清理完成"
}

# 环境变量设置
setup_environment() {
    log_info "设置环境变量..."
    
    # 检查.env文件
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            log_info "复制环境变量配置文件..."
            cp .env.example .env
            log_warning "请检查并修改 .env 文件中的配置"
        else
            log_error ".env.example 文件不存在"
            exit 1
        fi
    fi
    
    # 设置中国镜像源
    export DOCKER_BUILDKIT=1
    export COMPOSE_DOCKER_CLI_BUILD=1
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    
    log_success "环境变量设置完成"
}

# 构建服务
build_services() {
    log_info "构建服务镜像..."
    
    # 并行构建，提升速度
    docker-compose -f docker-compose.dev.yml build --parallel --progress=auto
    
    log_success "服务构建完成"
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    # 启动基础设施服务
    log_info "启动基础设施服务 (Redis, PostgreSQL)..."
    docker-compose -f docker-compose.dev.yml up -d redis postgres
    
    # 等待基础设施就绪
    log_info "等待基础设施服务就绪..."
    sleep 10
    
    # 启动应用服务
    log_info "启动应用服务..."
    docker-compose -f docker-compose.dev.yml up -d go-backend python-models react-frontend
    
    log_success "所有服务启动完成"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    # 等待服务完全启动
    sleep 15
    
    local max_retries=30
    local retry_count=0
    
    # 检查后端服务
    while [ $retry_count -lt $max_retries ]; do
        if curl -f http://localhost:8080/health >/dev/null 2>&1; then
            log_success "Go后端服务健康"
            break
        fi
        retry_count=$((retry_count + 1))
        sleep 2
    done
    
    if [ $retry_count -eq $max_retries ]; then
        log_warning "Go后端服务健康检查超时"
    fi
    
    # 检查Python模型服务
    retry_count=0
    while [ $retry_count -lt $max_retries ]; do
        if curl -f http://localhost:5000/health >/dev/null 2>&1; then
            log_success "Python模型服务健康"
            break
        fi
        retry_count=$((retry_count + 1))
        sleep 2
    done
    
    if [ $retry_count -eq $max_retries ]; then
        log_warning "Python模型服务健康检查超时"
    fi
    
    # 检查前端服务
    retry_count=0
    while [ $retry_count -lt $max_retries ]; do
        if curl -f http://localhost:3000 >/dev/null 2>&1; then
            log_success "React前端服务健康"
            break
        fi
        retry_count=$((retry_count + 1))
        sleep 2
    done
    
    if [ $retry_count -eq $max_retries ]; then
        log_warning "React前端服务健康检查超时"
    fi
}

# 显示服务状态
show_status() {
    echo ""
    log_success "🎉 AI Gateway 开发环境启动成功!"
    echo ""
    echo -e "${CYAN}📋 服务地址:${NC}"
    echo -e "  🌐 前端 (React):     ${GREEN}http://localhost:3000${NC}"
    echo -e "  🚪 后端 (Go):        ${GREEN}http://localhost:8080${NC}"
    echo -e "  🐍 Python 模型:      ${GREEN}http://localhost:5000${NC}"
    echo -e "  📊 Redis:            ${GREEN}localhost:6379${NC}"
    echo -e "  🗄️  PostgreSQL:      ${GREEN}localhost:5432${NC}"
    echo ""
    echo -e "${CYAN}🔧 管理端点:${NC}"
    echo -e "  📈 健康检查:         ${GREEN}http://localhost:8080/health${NC}"
    echo -e "  📊 监控指标:         ${GREEN}http://localhost:8080/metrics${NC}"
    echo -e "  🔐 API文档:          ${GREEN}http://localhost:8080/docs${NC} (待实现)"
    echo ""
    echo -e "${CYAN}🛠️  常用命令:${NC}"
    echo -e "  查看服务状态: ${YELLOW}docker-compose -f docker-compose.dev.yml ps${NC}"
    echo -e "  查看服务日志: ${YELLOW}docker-compose -f docker-compose.dev.yml logs -f [service_name]${NC}"
    echo -e "  重启服务:     ${YELLOW}docker-compose -f docker-compose.dev.yml restart [service_name]${NC}"
    echo -e "  停止服务:     ${YELLOW}./stop-dev.sh${NC}"
    echo -e "  进入容器:     ${YELLOW}docker-compose -f docker-compose.dev.yml exec [service_name] sh${NC}"
    echo ""
    echo -e "${CYAN}📁 日志查看:${NC}"
    echo -e "  所有服务: ${YELLOW}docker-compose -f docker-compose.dev.yml logs -f${NC}"
    echo -e "  后端日志: ${YELLOW}docker-compose -f docker-compose.dev.yml logs -f go-backend${NC}"
    echo -e "  模型日志: ${YELLOW}docker-compose -f docker-compose.dev.yml logs -f python-models${NC}"
    echo -e "  前端日志: ${YELLOW}docker-compose -f docker-compose.dev.yml logs -f react-frontend${NC}"
    echo ""
    echo -e "${GREEN}🎊 开发愉快!${NC}"
}

# 主函数
main() {
    echo -e "${PURPLE}"
    echo "=============================================="
    echo "    🚀 AI Gateway 开发环境启动器"
    echo "    📅 $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=============================================="
    echo -e "${NC}"
    
    # 进入部署目录
    cd "$(dirname "$0")"
    
    check_requirements
    setup_environment
    cleanup_old_containers
    build_services
    start_services
    health_check
    show_status
}

# 错误处理
trap 'log_error "启动过程中出现错误，请检查日志"; exit 1' ERR

# 执行主函数
main "$@"
