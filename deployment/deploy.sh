#!/bin/bash

# AI Gateway 一键部署脚本 (国内环境)
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# 配置变量
DOCKER_REGISTRY="registry.cn-hangzhou.aliyuncs.com"
PROJECT_NAME="ai-gateway"
VERSION="1.0.0"

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

# 检查环境
check_environment() {
    log_info "检查部署环境..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    # 检查kubectl (如果是K8s部署)
    if [[ "$DEPLOY_MODE" == "kubernetes" ]]; then
        if ! command -v kubectl &> /dev/null; then
            log_error "kubectl 未安装"
            exit 1
        fi
    fi
    
    log_success "环境检查通过"
}

# 设置国内镜像源
setup_china_mirrors() {
    log_info "设置中国内地镜像源..."
    
    # 设置Go代理
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    
    # 设置NPM镜像
    export NPM_REGISTRY=https://registry.npmmirror.com
    
    # 设置PyPI镜像
    export PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple
    
    # 设置HuggingFace镜像
    export HF_ENDPOINT=https://hf-mirror.com
    
    log_success "镜像源配置完成"
}

# 构建镜像
build_images() {
    log_info "构建Docker镜像..."
    
    # 构建后端镜像
    docker build -t ${DOCKER_REGISTRY}/${PROJECT_NAME}/backend:${VERSION} \
        -f deployment/Dockerfile \
        --target production .
    
    # 构建前端镜像
    docker build -t ${DOCKER_REGISTRY}/${PROJECT_NAME}/frontend:${VERSION} \
        -f deployment/Dockerfile.react \
        --target production .
    
    # 构建Python模型镜像
    docker build -t ${DOCKER_REGISTRY}/${PROJECT_NAME}/python:${VERSION} \
        -f deployment/Dockerfile.python \
        --target production .
    
    log_success "镜像构建完成"
}

# 推送镜像到仓库
push_images() {
    if [[ "$PUSH_IMAGES" == "true" ]]; then
        log_info "推送镜像到仓库..."
        
        docker push ${DOCKER_REGISTRY}/${PROJECT_NAME}/backend:${VERSION}
        docker push ${DOCKER_REGISTRY}/${PROJECT_NAME}/frontend:${VERSION}
        docker push ${DOCKER_REGISTRY}/${PROJECT_NAME}/python:${VERSION}
        
        log_success "镜像推送完成"
    fi
}

# Docker Compose 部署
deploy_docker_compose() {
    log_info "使用Docker Compose部署..."
    
    # 复制环境配置
    if [[ ! -f "deployment/.env.production" ]]; then
        cp deployment/.env.production.example deployment/.env.production
        log_warning "请编辑 deployment/.env.production 文件配置生产环境参数"
        read -p "按回车键继续..."
    fi
    
    # 启动服务
    docker-compose -f deployment/docker-compose.prod.yml --env-file deployment/.env.production up -d
    
    log_success "Docker Compose 部署完成"
}

# Kubernetes 部署
deploy_kubernetes() {
    log_info "使用Kubernetes部署..."
    
    # 创建命名空间
    kubectl create namespace ${PROJECT_NAME} --dry-run=client -o yaml | kubectl apply -f -
    
    # 应用配置
    kubectl apply -f deployment/k8s/ -n ${PROJECT_NAME}
    
    log_success "Kubernetes 部署完成"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    local max_retries=30
    local retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        if curl -f http://localhost/health >/dev/null 2>&1; then
            log_success "服务健康检查通过"
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        sleep 10
    done
    
    log_error "健康检查失败"
    return 1
}

# 显示部署信息
show_deployment_info() {
    echo ""
    log_success "🎉 AI Gateway 部署完成!"
    echo ""
    echo -e "${BLUE}📋 访问地址:${NC}"
    echo -e "  前端: ${GREEN}https://yourdomain.com${NC}"
    echo -e "  API: ${GREEN}https://yourdomain.com/api${NC}"
    echo -e "  监控: ${GREEN}https://yourdomain.com:3001${NC}"
    echo ""
    echo -e "${BLUE}🔧 管理命令:${NC}"
    if [[ "$DEPLOY_MODE" == "kubernetes" ]]; then
        echo -e "  查看状态: ${YELLOW}kubectl get pods -n ${PROJECT_NAME}${NC}"
        echo -e "  查看日志: ${YELLOW}kubectl logs -f deployment/backend -n ${PROJECT_NAME}${NC}"
    else
        echo -e "  查看状态: ${YELLOW}docker-compose -f deployment/docker-compose.prod.yml ps${NC}"
        echo -e "  查看日志: ${YELLOW}docker-compose -f deployment/docker-compose.prod.yml logs -f${NC}"
    fi
}

# 主函数
main() {
    echo -e "${PURPLE}"
    echo "=============================================="
    echo "    🚀 AI Gateway 一键部署脚本"
    echo "    📅 $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=============================================="
    echo -e "${NC}"
    
    # 解析参数
    DEPLOY_MODE=${1:-docker-compose}  # docker-compose 或 kubernetes
    PUSH_IMAGES=${2:-false}
    
    log_info "部署模式: $DEPLOY_MODE"
    
    check_environment
    setup_china_mirrors
    build_images
    push_images
    
    case $DEPLOY_MODE in
        "kubernetes"|"k8s")
            deploy_kubernetes
            ;;
        "docker-compose"|"compose")
            deploy_docker_compose
            ;;
        *)
            log_error "不支持的部署模式: $DEPLOY_MODE"
            echo "支持的模式: docker-compose, kubernetes"
            exit 1
            ;;
    esac
    
    health_check
    show_deployment_info
}

# 错误处理
trap 'log_error "部署过程中出现错误"; exit 1' ERR

# 执行主函数
main "$@"
