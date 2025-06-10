#!/bin/bash

# AI Gateway 部署验证脚本
# ============================================

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

# 验证函数
validate_service() {
    local service_name=$1
    local url=$2
    local expected_status=${3:-200}
    
    log_info "验证 $service_name..."
    
    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "$expected_status"; then
        log_success "$service_name 运行正常"
        return 0
    else
        log_error "$service_name 验证失败"
        return 1
    fi
}

# 等待服务启动
wait_for_service() {
    local service_name=$1
    local url=$2
    local max_attempts=${3:-30}
    local attempt=1
    
    log_info "等待 $service_name 启动..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" >/dev/null 2>&1; then
            log_success "$service_name 已启动"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_error "$service_name 启动超时"
    return 1
}

# 主验证流程
main() {
    log_info "开始验证 AI Gateway 部署..."
    echo "=========================================="
    
    # 检查 Docker 容器状态
    log_info "检查容器状态..."
    if docker-compose -f deployment/docker-compose.dev.yml ps | grep -q "Up"; then
        log_success "Docker 容器运行正常"
    else
        log_error "Docker 容器状态异常"
        docker-compose -f deployment/docker-compose.dev.yml ps
        return 1
    fi
    
    # 等待服务启动
    wait_for_service "Redis" "http://localhost:6379" 10
    wait_for_service "PostgreSQL" "http://localhost:5432" 15
    wait_for_service "Go 后端" "http://localhost:8080/health" 30
    wait_for_service "Python 模型" "http://localhost:5000/health" 60
    wait_for_service "React 前端" "http://localhost:3000" 30
    
    # 验证各服务API
    log_info "验证服务API..."
    
    # 验证后端健康检查
    if validate_service "后端健康检查" "http://localhost:8080/health"; then
        # 验证后端API
        validate_service "后端API" "http://localhost:8080/api/v1/status"
    fi
    
    # 验证模型服务
    if validate_service "模型服务健康检查" "http://localhost:5000/health"; then
        # 验证模型推理
        validate_service "模型推理" "http://localhost:5000/predict" 405  # POST方法，GET返回405
    fi
    
    # 验证前端
    validate_service "前端页面" "http://localhost:3000"
    
    # 验证监控指标
    validate_service "Prometheus指标" "http://localhost:9091/metrics"
    
    # 验证数据库连接
    log_info "验证数据库连接..."
    if docker exec aigateway-postgres-dev pg_isready -U aigateway -d ai_gateway >/dev/null 2>&1; then
        log_success "数据库连接正常"
    else
        log_error "数据库连接失败"
    fi
    
    # 验证Redis连接
    log_info "验证Redis连接..."
    if docker exec aigateway-redis-dev redis-cli ping | grep -q "PONG"; then
        log_success "Redis连接正常"
    else
        log_error "Redis连接失败"
    fi
    
    # 性能测试
    log_info "执行简单性能测试..."
    
    # 测试后端响应时间
    response_time=$(curl -o /dev/null -s -w "%{time_total}" "http://localhost:8080/health")
    if (( $(echo "$response_time < 1.0" | bc -l) )); then
        log_success "后端响应时间正常: ${response_time}s"
    else
        log_warning "后端响应时间较慢: ${response_time}s"
    fi
    
    # 测试前端响应时间
    response_time=$(curl -o /dev/null -s -w "%{time_total}" "http://localhost:3000")
    if (( $(echo "$response_time < 2.0" | bc -l) )); then
        log_success "前端响应时间正常: ${response_time}s"
    else
        log_warning "前端响应时间较慢: ${response_time}s"
    fi
    
    echo "=========================================="
    log_success "AI Gateway 部署验证完成！"
    
    echo ""
    log_info "服务访问地址:"
    echo "前端: http://localhost:3000"
    echo "后端: http://localhost:8080"
    echo "模型: http://localhost:5000"
    echo "监控: http://localhost:9091/metrics"
    
    echo ""
    log_info "常用命令:"
    echo "查看日志: make dev-logs"
    echo "查看状态: make dev-status" 
    echo "停止服务: make dev-down"
    echo "重启服务: make dev-restart"
}

# 错误处理
trap 'log_error "验证过程中发生错误"; exit 1' ERR

# 执行主函数
main "$@"
