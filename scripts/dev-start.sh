#!/bin/bash

# AI Gateway 开发环境启动脚本
# 自动检查依赖、初始化服务并启动开发环境

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker和Docker Compose
check_dependencies() {
    log_info "检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker Desktop"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    # 检查Docker是否运行
    if ! docker info &> /dev/null; then
        log_error "Docker 未运行，请启动 Docker Desktop"
        exit 1
    fi
    
    log_success "系统依赖检查完成"
}

# 创建必要的目录
create_directories() {
    log_info "创建必要的目录..."
    
    mkdir -p configs/{consul,grafana,nginx,prometheus,redis}
    mkdir -p scripts
    mkdir -p logs
    
    log_success "目录创建完成"
}

# 检查配置文件
check_configs() {
    log_info "检查配置文件..."
    
    # 检查必要的配置文件
    local configs=(
        "configs/prometheus.yml"
        "configs/redis.conf"
        "configs/nginx-dev.conf"
        "configs/consul/consul.json"
        "configs/grafana/datasources.yml"
    )
    
    local missing_configs=()
    
    for config in "${configs[@]}"; do
        if [[ ! -f "$config" ]]; then
            missing_configs+=("$config")
        fi
    done
    
    if [[ ${#missing_configs[@]} -gt 0 ]]; then
        log_warning "以下配置文件缺失："
        for config in "${missing_configs[@]}"; do
            echo "  - $config"
        done
        log_info "将使用默认配置继续启动..."
    else
        log_success "配置文件检查完成"
    fi
}

# 清理旧容器和网络
cleanup() {
    log_info "清理旧的容器和网络..."
    
    # 停止并删除开发环境容器
    docker-compose -f docker-compose.dev.yml down --remove-orphans 2>/dev/null || true
    
    # 清理未使用的网络
    docker network prune -f 2>/dev/null || true
    
    log_success "清理完成"
}

# 启动基础设施服务
start_infrastructure() {
    log_info "启动基础设施服务..."
    
    # 分阶段启动服务
    log_info "第一阶段：启动数据存储服务..."
    docker-compose -f docker-compose.dev.yml up -d postgres redis consul
    
    # 等待服务就绪
    log_info "等待数据存储服务启动..."
    sleep 10
    
    # 检查服务健康状态
    log_info "检查数据存储服务健康状态..."
    for i in {1..30}; do
        if docker-compose -f docker-compose.dev.yml ps | grep -E "(postgres|redis|consul)" | grep -q "healthy\|Up"; then
            log_success "数据存储服务启动成功"
            break
        fi
        if [[ $i -eq 30 ]]; then
            log_error "数据存储服务启动超时"
            exit 1
        fi
        sleep 2
    done
    
    log_info "第二阶段：启动监控服务..."
    docker-compose -f docker-compose.dev.yml up -d prometheus grafana node-exporter cadvisor
    
    log_success "基础设施服务启动完成"
}

# 启动应用服务
start_applications() {
    log_info "启动应用服务..."
    
    # 构建并启动后端服务
    log_info "构建并启动后端服务..."
    docker-compose -f docker-compose.dev.yml up -d --build backend
    
    # 启动Python模型服务
    log_info "启动Python模型服务..."
    docker-compose -f docker-compose.dev.yml up -d --build python-models
    
    # 启动前端服务
    log_info "启动前端服务..."
    docker-compose -f docker-compose.dev.yml up -d --build frontend
    
    # 启动Nginx代理
    log_info "启动Nginx代理..."
    docker-compose -f docker-compose.dev.yml up -d nginx
    
    log_success "应用服务启动完成"
}

# 显示服务状态
show_status() {
    log_info "服务状态："
    echo ""
    docker-compose -f docker-compose.dev.yml ps
    echo ""
    
    log_info "服务访问地址："
    echo -e "  🌐 前端界面:     ${GREEN}http://localhost${NC}"
    echo -e "  🔧 后端API:      ${GREEN}http://localhost:8080${NC}"
    echo -e "  🤖 Python模型:   ${GREEN}http://localhost:5000${NC}"
    echo -e "  📊 Grafana:      ${GREEN}http://localhost:3001${NC} (admin/admin_dev_2024)"
    echo -e "  📈 Prometheus:   ${GREEN}http://localhost:9090${NC}"
    echo -e "  🔍 Consul:       ${GREEN}http://localhost:8500${NC}"
    echo -e "  💾 Redis:        ${GREEN}localhost:6379${NC}"
    echo -e "  🗄️  PostgreSQL:   ${GREEN}localhost:5432${NC} (aigateway/dev_password_2024)"
    echo ""
}

# 显示日志
show_logs() {
    log_info "显示服务日志..."
    docker-compose -f docker-compose.dev.yml logs -f --tail=100
}

# 主函数
main() {
    echo -e "${BLUE}===========================================${NC}"
    echo -e "${BLUE}    AI Gateway 开发环境启动脚本${NC}"
    echo -e "${BLUE}===========================================${NC}"
    echo ""
    
    # 解析命令行参数
    case "${1:-start}" in
        "start")
            check_dependencies
            create_directories
            check_configs
            cleanup
            start_infrastructure
            start_applications
            show_status
            
            read -p "是否查看服务日志？(y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                show_logs
            fi
            ;;
        "stop")
            log_info "停止所有服务..."
            docker-compose -f docker-compose.dev.yml down
            log_success "所有服务已停止"
            ;;
        "restart")
            log_info "重启所有服务..."
            docker-compose -f docker-compose.dev.yml restart
            show_status
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs
            ;;
        "clean")
            log_info "清理所有数据..."
            docker-compose -f docker-compose.dev.yml down -v --remove-orphans
            docker system prune -f
            log_success "清理完成"
            ;;
        *)
            echo "用法: $0 {start|stop|restart|status|logs|clean}"
            echo ""
            echo "命令说明："
            echo "  start   - 启动开发环境（默认）"
            echo "  stop    - 停止所有服务"
            echo "  restart - 重启所有服务"
            echo "  status  - 显示服务状态"
            echo "  logs    - 显示服务日志"
            echo "  clean   - 清理所有数据和容器"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
