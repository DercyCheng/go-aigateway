# Multi-stage Dockerfile for Go Backend
# 针对中国内地环境优化

# ============================================
# 构建阶段
# ============================================
FROM golang:1.22-alpine AS builder

# 设置阿里云镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /build

# 设置Go模块代理 (中国内地优化)
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV CGO_ENABLED=0
ENV GOOS=linux

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN go build -ldflags="-w -s" -o aigateway main.go

# ============================================
# 开发阶段 (包含开发工具)
# ============================================
FROM golang:1.22-alpine AS development

# 设置阿里云镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装开发工具
RUN apk add --no-cache \
    git \
    curl \
    bash \
    vim \
    htop \
    ca-certificates \
    tzdata

# 安装开发工具
RUN go install github.com/cosmtrek/air@latest
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 设置工作目录
WORKDIR /app

# 设置Go模块代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 复制配置文件
COPY .air.toml .

# 暴露端口
EXPOSE 8080

# 开发模式下使用热重载
CMD ["air", "-c", ".air.toml"]

# ============================================
# 生产阶段 (最小镜像)
# ============================================
FROM alpine:3.18 AS production

# 设置阿里云镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 -S aigateway && \
    adduser -u 1001 -S aigateway -G aigateway

# 设置时区
ENV TZ=Asia/Shanghai

# 创建应用目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/aigateway .

# 创建必要的目录
RUN mkdir -p logs config ssl data && \
    chown -R aigateway:aigateway /app

# 切换到非root用户
USER aigateway

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动应用
CMD ["./aigateway"]
