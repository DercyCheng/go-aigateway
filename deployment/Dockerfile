# Go 后端 Dockerfile (开发环境)
FROM golang:1.23-alpine AS builder

# 设置中国镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的工具
RUN apk add --no-cache git gcc musl-dev

# 设置工作目录
WORKDIR /app

# 设置Go代理为中国镜像
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GO111MODULE=on

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 开发环境，使用热重载
FROM golang:1.23-alpine

# 设置中国镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的工具
RUN apk add --no-cache git gcc musl-dev

# 设置Go代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GO111MODULE=on

# 安装 air 用于热重载 (使用兼容 Go 1.23 的版本)
RUN go install github.com/cosmtrek/air@v1.49.0

WORKDIR /app

# 暴露端口
EXPOSE 8080

# 开发环境使用 air 热重载
CMD ["air", "-c", ".air.toml"]
