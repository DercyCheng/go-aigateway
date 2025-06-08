#!/bin/bash

# AI Gateway 中国大陆开发环境部署脚本
# 适用于Windows WSL、Linux和macOS

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

# 检查必要的工具
check_dependencies() {
    log_info "检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    log_success "系统依赖检查通过"
}

# 创建必要的目录
create_directories() {
    log_info "创建必要的目录..."
    
    mkdir -p logs
    mkdir -p model_cache
    mkdir -p python_logs
    mkdir -p ssl
    mkdir -p grafana/dashboards
    mkdir -p grafana/datasources
    
    log_success "目录创建完成"
}

# 设置环境变量
setup_env() {
    log_info "设置环境变量..."
    
    if [ ! -f .env.cn ]; then
        log_warning ".env.cn 文件不存在，请先配置环境变量"
        return 1
    fi
    
    # 检查关键配置项
    if ! grep -q "ZHIPU_API_KEY=" .env.cn || ! grep -q "QIANFAN_API_KEY=" .env.cn; then
        log_warning "请在 .env.cn 文件中配置至少一个国内AI服务的API密钥"
    fi
    
    log_success "环境变量设置完成"
}

# 拉取Docker镜像
pull_images() {
    log_info "拉取Docker镜像 (使用阿里云镜像加速)..."
    
    # 设置Docker镜像加速器
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        log_info "建议配置Docker镜像加速器："
        log_info "sudo mkdir -p /etc/docker"
        log_info "echo '{\"registry-mirrors\": [\"https://mirror.ccs.tencentyun.com\", \"https://registry.cn-hangzhou.aliyuncs.com\"]}' | sudo tee /etc/docker/daemon.json"
        log_info "sudo systemctl restart docker"
    fi
    
    docker-compose -f docker-compose.cn.yml pull
    
    log_success "Docker镜像拉取完成"
}

# 构建自定义镜像
build_images() {
    log_info "构建自定义镜像..."
    
    docker-compose -f docker-compose.cn.yml build
    
    log_success "镜像构建完成"
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    docker-compose -f docker-compose.cn.yml up -d
    
    log_success "服务启动完成"
}

# 检查服务状态
check_services() {
    log_info "检查服务状态..."
    
    sleep 10
    
    # 检查容器状态
    log_info "容器状态："
    docker-compose -f docker-compose.cn.yml ps
    
    # 检查服务健康状态
    log_info "检查服务健康状态..."
    
    # 检查主服务
    if curl -f http://localhost:8080/health &> /dev/null; then
        log_success "✓ AI Gateway 主服务运行正常"
    else
        log_error "✗ AI Gateway 主服务未响应"
    fi
    
    # 检查Python模型服务
    if curl -f http://localhost:5000/health &> /dev/null; then
        log_success "✓ Python 模型服务运行正常"
    else
        log_warning "✗ Python 模型服务未响应"
    fi
    
    # 检查Redis
    if docker exec redis-cn-dev redis-cli ping &> /dev/null; then
        log_success "✓ Redis 服务运行正常"
    else
        log_error "✗ Redis 服务未响应"
    fi
    
    # 检查PostgreSQL
    if docker exec postgres-cn-dev pg_isready -U postgres &> /dev/null; then
        log_success "✓ PostgreSQL 服务运行正常"
    else
        log_error "✗ PostgreSQL 服务未响应"
    fi
}

# 显示服务信息
show_services_info() {
    log_success "=== 中国大陆开发环境部署完成 ==="
    echo
    log_info "服务访问地址："
    echo "  🚀 AI Gateway API:    http://localhost:8080"
    echo "  🐍 Python 模型服务:   http://localhost:5000"
    echo "  📊 Prometheus:       http://localhost:9091"
    echo "  📈 Grafana:          http://localhost:3001 (admin/admin)"
    echo "  🔗 Nginx 代理:       http://localhost:80"
    echo
    log_info "数据库连接："
    echo "  🗄️  PostgreSQL:       localhost:5432 (postgres/postgres)"
    echo "  🔄 Redis:            localhost:6379"
    echo
    log_info "常用命令："
    echo "  查看日志:    docker-compose -f docker-compose.cn.yml logs -f [service_name]"
    echo "  停止服务:    docker-compose -f docker-compose.cn.yml down"
    echo "  重启服务:    docker-compose -f docker-compose.cn.yml restart [service_name]"
    echo "  查看状态:    docker-compose -f docker-compose.cn.yml ps"
    echo
    log_info "配置文件："
    echo "  环境变量:    .env.cn"
    echo "  Docker配置:  docker-compose.cn.yml"
    echo
}

# 主函数
main() {
    log_info "开始部署 AI Gateway 中国大陆开发环境..."
    echo
    
    check_dependencies
    create_directories
    setup_env
    pull_images
    build_images
    start_services
    check_services
    show_services_info
    
    log_success "部署完成！"
}

# 脚本选项
case "${1:-}" in
    "start")
        start_services
        ;;
    "stop")
        log_info "停止服务..."
        docker-compose -f docker-compose.cn.yml down
        log_success "服务已停止"
        ;;
    "restart")
        log_info "重启服务..."
        docker-compose -f docker-compose.cn.yml restart
        log_success "服务已重启"
        ;;
    "status")
        check_services
        ;;
    "logs")
        docker-compose -f docker-compose.cn.yml logs -f "${2:-}"
        ;;
    "clean")
        log_info "清理环境..."
        docker-compose -f docker-compose.cn.yml down -v --remove-orphans
        docker system prune -f
        log_success "环境清理完成"
        ;;
    *)
        main
        ;;
esac
