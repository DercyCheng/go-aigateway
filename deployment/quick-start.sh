#!/bin/bash

# AI Gateway 快速启动脚本
# 一键检查、构建、启动完整开发环境

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Logo显示
show_logo() {
    echo -e "${PURPLE}"
    cat << "EOF"
    ╔═══════════════════════════════════════════════════════════════╗
    ║                                                               ║
    ║     █████╗ ██╗     ██████╗  █████╗ ████████╗███████╗██╗    ██╗ ║
    ║    ██╔══██╗██║    ██╔════╝ ██╔══██╗╚══██╔══╝██╔════╝██║    ██║ ║
    ║    ███████║██║    ██║  ███╗███████║   ██║   █████╗  ██║ █╗ ██║ ║
    ║    ██╔══██║██║    ██║   ██║██╔══██║   ██║   ██╔══╝  ██║███╗██║ ║
    ║    ██║  ██║██║    ╚██████╔╝██║  ██║   ██║   ███████╗╚███╔███╔╝ ║
    ║    ╚═╝  ╚═╝╚═╝     ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝ ╚══╝╚══╝  ║
    ║                                                               ║
    ║               🚀 AI Gateway 快速启动器 🚀                      ║
    ║                   针对中国内地环境优化                          ║
    ║                                                               ║
    ╚═══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

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

# 主菜单
show_menu() {
    echo -e "${CYAN}请选择操作:${NC}"
    echo "1. 🚀 快速启动 (检查依赖 + 构建 + 启动)"
    echo "2. 🔧 仅检查环境依赖"
    echo "3. 🏗️  仅构建服务"
    echo "4. ▶️  仅启动服务"
    echo "5. 🛑 停止服务"
    echo "6. 🔄 重启服务"
    echo "7. 📊 查看服务状态"
    echo "8. 📋 查看服务日志"
    echo "9. 🧪 运行验证测试"
    echo "10. 🗑️  清理环境"
    echo "11. 📖 显示帮助信息"
    echo "0. ❌ 退出"
    echo ""
    read -p "请输入选择 (0-11): " choice
}

# 检查依赖
check_dependencies() {
    log_info "检查环境依赖..."
    
    # 检查是否在正确的目录
    if [[ ! -f "go.mod" ]]; then
        log_error "请在项目根目录运行此脚本"
        exit 1
    fi
    
    # 调用检查脚本
    make check-deps
    
    log_success "依赖检查完成"
}

# 构建服务
build_services() {
    log_info "构建服务镜像..."
    make dev-build
    log_success "服务构建完成"
}

# 启动服务
start_services() {
    log_info "启动开发环境..."
    make dev-up
    
    # 等待服务启动
    log_info "等待服务启动完成..."
    sleep 10
    
    # 显示服务信息
    show_service_info
}

# 停止服务
stop_services() {
    log_info "停止开发环境..."
    make dev-down
    log_success "服务已停止"
}

# 重启服务
restart_services() {
    log_info "重启开发环境..."
    make dev-restart
    sleep 10
    show_service_info
}

# 查看状态
show_status() {
    log_info "查看服务状态..."
    make dev-status
}

# 查看日志
show_logs() {
    echo -e "${CYAN}选择要查看的日志:${NC}"
    echo "1. 所有服务"
    echo "2. Go 后端"
    echo "3. Python 模型"
    echo "4. React 前端"
    echo "5. PostgreSQL"
    echo "6. Redis"
    read -p "请选择 (1-6): " log_choice
    
    case $log_choice in
        1) docker-compose -f deployment/docker-compose.dev.yml logs -f ;;
        2) docker-compose -f deployment/docker-compose.dev.yml logs -f go-backend ;;
        3) docker-compose -f deployment/docker-compose.dev.yml logs -f python-models ;;
        4) docker-compose -f deployment/docker-compose.dev.yml logs -f react-frontend ;;
        5) docker-compose -f deployment/docker-compose.dev.yml logs -f postgres ;;
        6) docker-compose -f deployment/docker-compose.dev.yml logs -f redis ;;
        *) log_error "无效选择" ;;
    esac
}

# 运行验证
run_validation() {
    log_info "运行部署验证..."
    ./deployment/validate.sh
}

# 清理环境
clean_environment() {
    echo -e "${YELLOW}选择清理级别:${NC}"
    echo "1. 🧹 轻度清理 (停止容器)"
    echo "2. 🗑️  中度清理 (删除容器和镜像)"
    echo "3. 💥 深度清理 (删除所有数据，包括数据库)"
    read -p "请选择 (1-3): " clean_choice
    
    case $clean_choice in
        1) make dev-down ;;
        2) make clean-dev ;;
        3) 
            echo -e "${RED}⚠️  警告: 这将删除所有数据，包括数据库！${NC}"
            read -p "确认继续? (输入 'yes' 确认): " confirm
            if [[ "$confirm" == "yes" ]]; then
                make clean-dev
                docker volume prune -f
            else
                log_info "操作已取消"
            fi
            ;;
        *) log_error "无效选择" ;;
    esac
}

# 显示服务信息
show_service_info() {
    echo ""
    log_success "🎉 AI Gateway 开发环境已启动!"
    echo ""
    echo -e "${CYAN}📋 服务地址:${NC}"
    echo -e "  🌐 前端 (React):     ${GREEN}http://localhost:3000${NC}"
    echo -e "  🚪 后端 (Go):        ${GREEN}http://localhost:8080${NC}"
    echo -e "  🐍 Python 模型:      ${GREEN}http://localhost:5000${NC}"
    echo -e "  📊 监控面板:         ${GREEN}http://localhost:3001${NC}"
    echo ""
    echo -e "${CYAN}🔧 管理端点:${NC}"
    echo -e "  📈 健康检查:         ${GREEN}http://localhost:8080/health${NC}"
    echo -e "  📊 监控指标:         ${GREEN}http://localhost:8080/metrics${NC}"
    echo ""
    echo -e "${CYAN}💡 快捷命令:${NC}"
    echo -e "  查看状态: ${YELLOW}make dev-status${NC}"
    echo -e "  查看日志: ${YELLOW}make dev-logs${NC}"
    echo -e "  重启服务: ${YELLOW}make dev-restart${NC}"
    echo -e "  停止服务: ${YELLOW}make dev-down${NC}"
}

# 显示帮助
show_help() {
    echo -e "${CYAN}AI Gateway 快速启动器帮助${NC}"
    echo ""
    echo "此工具提供以下功能:"
    echo "• 一键检查环境依赖"
    echo "• 自动构建Docker镜像"
    echo "• 启动完整的三层架构开发环境"
    echo "• 实时监控和日志查看"
    echo "• 部署验证和测试"
    echo ""
    echo "支持的服务:"
    echo "• Go 后端 (端口 8080)"
    echo "• React 前端 (端口 3000)"
    echo "• Python 模型服务 (端口 5000)"
    echo "• PostgreSQL 数据库 (端口 5432)"
    echo "• Redis 缓存 (端口 6379)"
    echo "• Consul 服务发现 (端口 8500)"
    echo "• Prometheus 监控 (端口 9090)"
    echo "• Grafana 面板 (端口 3001)"
    echo ""
    echo "更多信息请查看: deployment/README.md"
}

# 快速启动
quick_start() {
    log_info "开始快速启动流程..."
    check_dependencies
    build_services
    start_services
    run_validation
}

# 主函数
main() {
    show_logo
    
    while true; do
        show_menu
        
        case $choice in
            1) quick_start ;;
            2) check_dependencies ;;
            3) build_services ;;
            4) start_services ;;
            5) stop_services ;;
            6) restart_services ;;
            7) show_status ;;
            8) show_logs ;;
            9) run_validation ;;
            10) clean_environment ;;
            11) show_help ;;
            0) 
                log_info "感谢使用 AI Gateway 快速启动器!"
                exit 0 
                ;;
            *) 
                log_error "无效选择，请重新输入"
                sleep 1
                ;;
        esac
        
        echo ""
        read -p "按回车键继续..."
        clear
        show_logo
    done
}

# 如果传递了参数，直接执行对应功能
if [[ $# -gt 0 ]]; then
    case $1 in
        "quick"|"start") quick_start ;;
        "stop") stop_services ;;
        "restart") restart_services ;;
        "status") show_status ;;
        "logs") show_logs ;;
        "test") run_validation ;;
        "clean") clean_environment ;;
        "help") show_help ;;
        *) 
            echo "用法: $0 [quick|start|stop|restart|status|logs|test|clean|help]"
            exit 1
            ;;
    esac
else
    main
fi
