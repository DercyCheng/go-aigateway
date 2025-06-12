# Go AI Gateway 多阶段构建 Dockerfile
# 针对中国内地环境优化

# =================================
# 构建阶段 - 下载依赖和编译
# =================================
FROM golang:1.22-alpine AS builder

# 设置中国内地 Go 代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# 设置Alpine镜像源为阿里云
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的构建工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum（利用Docker缓存机制）
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 构建应用 - 增强安全性
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=$(date +%Y%m%d-%H%M%S) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o ai-gateway .

# =================================
# 开发阶段 - 包含开发工具
# =================================
FROM golang:1.22-alpine AS development

# 设置中国内地 Go 代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GO111MODULE=on

# 设置Alpine镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装开发工具
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    bash \
    curl \
    make \
    gcc \
    musl-dev

# 安装 Air 热重载工具
RUN go install github.com/cosmtrek/air@latest

# 创建非root用户
RUN adduser -D -s /bin/bash gouser

# 设置工作目录
WORKDIR /app

# 改变目录所有者
RUN chown -R gouser:gouser /app

USER gouser

# 设置时区
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 8080

# 开发环境启动命令
CMD ["air", "-c", ".air.toml"]

# =================================
# 生产阶段 - 最小运行时镜像
# =================================
FROM alpine:3.18 AS production

# 设置Alpine镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# 创建非root用户
RUN adduser -D -s /bin/sh appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/ai-gateway .

# 复制配置文件
COPY --from=builder /app/configs/ ./configs/

# 改变文件所有者
RUN chown -R appuser:appuser /app

USER appuser

# 设置时区
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动命令
CMD ["./ai-gateway"]
