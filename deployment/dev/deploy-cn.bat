@echo off
REM AI Gateway 中国大陆开发环境部署脚本 - Windows 批处理版本

setlocal enabledelayedexpansion

REM 颜色定义 (Windows 10+ 支持ANSI颜色)
set "RED=[91m"
set "GREEN=[92m"
set "YELLOW=[93m"
set "BLUE=[94m"
set "NC=[0m"

REM 获取脚本参数
set "ACTION=%~1"
set "SERVICE=%~2"

REM 日志函数
:log_info
echo %BLUE%[INFO]%NC% %~1
exit /b

:log_success
echo %GREEN%[SUCCESS]%NC% %~1
exit /b

:log_warning
echo %YELLOW%[WARNING]%NC% %~1
exit /b

:log_error
echo %RED%[ERROR]%NC% %~1
exit /b

REM 检查依赖
:check_dependencies
call :log_info "检查系统依赖..."

docker --version >nul 2>&1
if errorlevel 1 (
    call :log_error "Docker 未安装，请先安装 Docker Desktop"
    exit /b 1
)

docker-compose --version >nul 2>&1
if errorlevel 1 (
    call :log_error "Docker Compose 未安装，请先安装 Docker Compose"
    exit /b 1
)

call :log_success "系统依赖检查通过"
exit /b

REM 创建目录
:create_directories
call :log_info "创建必要的目录..."

if not exist "logs" mkdir logs
if not exist "model_cache" mkdir model_cache
if not exist "python_logs" mkdir python_logs
if not exist "ssl" mkdir ssl
if not exist "grafana\dashboards" mkdir grafana\dashboards
if not exist "grafana\datasources" mkdir grafana\datasources

call :log_success "目录创建完成"
exit /b

REM 设置环境变量
:setup_env
call :log_info "设置环境变量..."

if not exist ".env.cn" (
    call :log_warning ".env.cn 文件不存在，请先配置环境变量"
    exit /b 1
)

findstr "ZHIPU_API_KEY=" .env.cn >nul
if errorlevel 1 (
    findstr "QIANFAN_API_KEY=" .env.cn >nul
    if errorlevel 1 (
        call :log_warning "请在 .env.cn 文件中配置至少一个国内AI服务的API密钥"
    )
)

call :log_success "环境变量设置完成"
exit /b

REM 拉取镜像
:pull_images
call :log_info "拉取Docker镜像 (使用阿里云镜像加速)..."

call :log_info "建议配置Docker镜像加速器（Docker Desktop设置中）"
call :log_info "推荐镜像源："
call :log_info "  https://mirror.ccs.tencentyun.com"
call :log_info "  https://registry.cn-hangzhou.aliyuncs.com"

docker-compose -f docker-compose.cn.yml pull

if errorlevel 1 (
    call :log_error "镜像拉取失败"
    exit /b 1
)

call :log_success "Docker镜像拉取完成"
exit /b

REM 构建镜像
:build_images
call :log_info "构建自定义镜像..."

docker-compose -f docker-compose.cn.yml build

if errorlevel 1 (
    call :log_error "镜像构建失败"
    exit /b 1
)

call :log_success "镜像构建完成"
exit /b

REM 启动服务
:start_services
call :log_info "启动服务..."

docker-compose -f docker-compose.cn.yml up -d

if errorlevel 1 (
    call :log_error "服务启动失败"
    exit /b 1
)

call :log_success "服务启动完成"
exit /b

REM 检查服务状态
:check_services
call :log_info "检查服务状态..."

timeout /t 10 /nobreak >nul

call :log_info "容器状态："
docker-compose -f docker-compose.cn.yml ps

call :log_info "检查服务健康状态..."

REM 检查主服务
curl -f http://localhost:8080/health >nul 2>&1
if errorlevel 1 (
    call :log_error "✗ AI Gateway 主服务未响应"
) else (
    call :log_success "✓ AI Gateway 主服务运行正常"
)

REM 检查Python模型服务
curl -f http://localhost:5000/health >nul 2>&1
if errorlevel 1 (
    call :log_warning "✗ Python 模型服务未响应"
) else (
    call :log_success "✓ Python 模型服务运行正常"
)

REM 检查Redis
docker exec redis-cn-dev redis-cli ping >nul 2>&1
if errorlevel 1 (
    call :log_error "✗ Redis 服务未响应"
) else (
    call :log_success "✓ Redis 服务运行正常"
)

REM 检查PostgreSQL
docker exec postgres-cn-dev pg_isready -U postgres >nul 2>&1
if errorlevel 1 (
    call :log_error "✗ PostgreSQL 服务未响应"
) else (
    call :log_success "✓ PostgreSQL 服务运行正常"
)

exit /b

REM 显示服务信息
:show_services_info
call :log_success "=== 中国大陆开发环境部署完成 ==="
echo.
call :log_info "服务访问地址："
echo   🚀 AI Gateway API:    http://localhost:8080
echo   🐍 Python 模型服务:   http://localhost:5000
echo   📊 Prometheus:       http://localhost:9091
echo   📈 Grafana:          http://localhost:3001 (admin/admin)
echo   🔗 Nginx 代理:       http://localhost:80
echo.
call :log_info "数据库连接："
echo   🗄️  PostgreSQL:       localhost:5432 (postgres/postgres)
echo   🔄 Redis:            localhost:6379
echo.
call :log_info "常用命令："
echo   查看日志:    docker-compose -f docker-compose.cn.yml logs -f [service_name]
echo   停止服务:    docker-compose -f docker-compose.cn.yml down
echo   重启服务:    docker-compose -f docker-compose.cn.yml restart [service_name]
echo   查看状态:    docker-compose -f docker-compose.cn.yml ps
echo.
call :log_info "配置文件："
echo   环境变量:    .env.cn
echo   Docker配置:  docker-compose.cn.yml
echo.
exit /b

REM 主部署流程
:main_deploy
call :log_info "开始部署 AI Gateway 中国大陆开发环境..."
echo.

call :check_dependencies
if errorlevel 1 exit /b 1

call :create_directories
call :setup_env
if errorlevel 1 exit /b 1

call :pull_images
if errorlevel 1 exit /b 1

call :build_images
if errorlevel 1 exit /b 1

call :start_services
if errorlevel 1 exit /b 1

call :check_services
call :show_services_info

call :log_success "部署完成！"
exit /b

REM 主程序入口
if "%ACTION%"=="" goto main_deploy
if "%ACTION%"=="start" goto start_services
if "%ACTION%"=="stop" goto stop_services
if "%ACTION%"=="restart" goto restart_services
if "%ACTION%"=="status" goto check_services
if "%ACTION%"=="logs" goto show_logs
if "%ACTION%"=="clean" goto clean_env
goto show_help

:stop_services
call :log_info "停止服务..."
docker-compose -f docker-compose.cn.yml down
call :log_success "服务已停止"
exit /b

:restart_services
call :log_info "重启服务..."
docker-compose -f docker-compose.cn.yml restart
call :log_success "服务已重启"
exit /b

:show_logs
if "%SERVICE%"=="" (
    docker-compose -f docker-compose.cn.yml logs -f
) else (
    docker-compose -f docker-compose.cn.yml logs -f %SERVICE%
)
exit /b

:clean_env
call :log_info "清理环境..."
docker-compose -f docker-compose.cn.yml down -v --remove-orphans
docker system prune -f
call :log_success "环境清理完成"
exit /b

:show_help
echo 使用方法: %~nx0 [命令] [服务名]
echo.
echo 命令:
echo   start    - 启动服务
echo   stop     - 停止服务
echo   restart  - 重启服务
echo   status   - 检查服务状态
echo   logs     - 查看日志 (可指定服务名)
echo   clean    - 清理环境
echo   (无参数) - 完整部署流程
echo.
echo 示例:
echo   %~nx0                    - 运行完整部署
echo   %~nx0 start              - 启动所有服务
echo   %~nx0 logs go-aigateway  - 查看主服务日志
exit /b
