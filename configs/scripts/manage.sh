#!/bin/bash
# AI Gateway 统一管理脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIGS_DIR="$PROJECT_ROOT/configs"

# Logo
show_logo() {
    echo -e "${BLUE}"
    cat << "EOF"
    ╔═══════════════════════════════════════════════════╗
    ║                AI Gateway                         ║
    ║            统一配置管理工具                        ║
    ╚═══════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# 帮助信息
show_help() {
    echo "用法: $0 [命令] [选项]"
    echo ""
    echo "命令:"
    echo "  dev          启动开发环境"
    echo "  prod         启动生产环境 (带监控)"
    echo "  stop         停止所有服务"
    echo "  restart      重启服务"
    echo "  logs         查看日志"
    echo "  clean        清理容器和数据卷"
    echo "  build        构建镜像"
    echo "  deploy       部署到 Kubernetes"
    echo "  health       检查服务健康状态"
    echo ""
    echo "选项:"
    echo "  -h, --help   显示帮助信息"
    echo "  -v, --verbose 详细输出"
    echo ""
    echo "示例:"
    echo "  $0 dev                    # 启动开发环境"
    echo "  $0 prod                   # 启动生产环境"
    echo "  $0 logs backend           # 查看后端日志"
    echo "  $0 deploy                 # 部署到 k8s"
}

# 检查依赖
check_dependencies() {
    local missing_deps=()
    
    if ! command -v docker &> /dev/null; then
        missing_deps+=(docker)
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        missing_deps+=(docker-compose)
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        echo -e "${RED}错误: 缺少依赖: ${missing_deps[*]}${NC}"
        echo "请先安装所需依赖"
        exit 1
    fi
}

# 启动开发环境
start_dev() {
    echo -e "${GREEN}启动开发环境...${NC}"
    cd "$CONFIGS_DIR"
    
    # 复制环境变量文件
    cp .env.development .env
    
    # 启动服务
    docker-compose up -d backend postgres redis consul
    
    echo -e "${GREEN}开发环境启动成功!${NC}"
    echo "服务地址:"
    echo "  后端: http://localhost:8080"
    echo "  Consul: http://localhost:8500"
    echo "  监控: docker-compose --profile monitoring up -d"
}

# 启动生产环境
start_prod() {
    echo -e "${GREEN}启动生产环境...${NC}"
    cd "$CONFIGS_DIR"
    
    # 复制环境变量文件
    cp .env.production .env
    
    # 启动完整服务栈
    docker-compose --profile production --profile monitoring up -d
    
    echo -e "${GREEN}生产环境启动成功!${NC}"
    echo "服务地址:"
    echo "  应用: http://localhost"
    echo "  监控: http://localhost:3001"
    echo "  Prometheus: http://localhost:9090"
}

# 停止服务
stop_services() {
    echo -e "${YELLOW}停止所有服务...${NC}"
    cd "$CONFIGS_DIR"
    docker-compose --profile production --profile monitoring down
    echo -e "${GREEN}服务已停止${NC}"
}

# 重启服务
restart_services() {
    echo -e "${YELLOW}重启服务...${NC}"
    stop_services
    sleep 2
    start_dev
}

# 查看日志
show_logs() {
    local service="${1:-}"
    cd "$CONFIGS_DIR"
    
    if [ -z "$service" ]; then
        docker-compose logs -f
    else
        docker-compose logs -f "$service"
    fi
}

# 清理环境
clean_env() {
    echo -e "${YELLOW}清理环境...${NC}"
    cd "$CONFIGS_DIR"
    
    read -p "确定要删除所有容器和数据卷吗? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker-compose --profile production --profile monitoring down -v --remove-orphans
        docker system prune -f
        echo -e "${GREEN}环境清理完成${NC}"
    else
        echo "已取消"
    fi
}

# 构建镜像
build_images() {
    echo -e "${GREEN}构建镜像...${NC}"
    cd "$CONFIGS_DIR"
    docker-compose build --no-cache
    echo -e "${GREEN}镜像构建完成${NC}"
}

# 部署到 Kubernetes
deploy_k8s() {
    echo -e "${GREEN}部署到 Kubernetes...${NC}"
    
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}错误: kubectl 未安装${NC}"
        exit 1
    fi
    
    cd "$CONFIGS_DIR/k8s"
    
    # 应用配置
    kubectl apply -f networking.yaml
    kubectl apply -f infrastructure.yaml
    kubectl apply -f applications.yaml
    
    echo -e "${GREEN}Kubernetes 部署完成${NC}"
    echo "查看状态: kubectl get pods -l app=aigateway"
}

# 健康检查
health_check() {
    echo -e "${BLUE}检查服务健康状态...${NC}"
    
    local services=("backend:8080" "consul:8500")
    
    for service in "${services[@]}"; do
        local name="${service%:*}"
        local port="${service#*:}"
        
        if curl -sf "http://localhost:$port/health" &> /dev/null; then
            echo -e "${GREEN}✓${NC} $name 服务正常"
        else
            echo -e "${RED}✗${NC} $name 服务异常"
        fi
    done
}

# 主函数
main() {
    show_logo
    
    case "${1:-}" in
        "dev")
            check_dependencies
            start_dev
            ;;
        "prod")
            check_dependencies
            start_prod
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            check_dependencies
            restart_services
            ;;
        "logs")
            show_logs "${2:-}"
            ;;
        "clean")
            clean_env
            ;;
        "build")
            check_dependencies
            build_images
            ;;
        "deploy")
            deploy_k8s
            ;;
        "health")
            health_check
            ;;
        "-h"|"--help"|"help")
            show_help
            ;;
        "")
            show_help
            ;;
        *)
            echo -e "${RED}错误: 未知命令 '$1'${NC}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
