# React 前端应用 Dockerfile
# 针对中国内地环境优化

# =================================
# 基础 Node.js 镜像
# =================================
FROM node:18-alpine AS base

# 设置 npm 镜像源为 npmmirror
RUN npm config set registry https://registry.npmmirror.com/

# 设置Alpine镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要工具
RUN apk add --no-cache git

# =================================
# 依赖安装阶段
# =================================
FROM base AS deps

WORKDIR /app

# 复制 package.json 和 package-lock.json（如果存在）
COPY frontend/package*.json ./

# 安装依赖
RUN npm ci --only=production && npm cache clean --force

# =================================
# 开发阶段
# =================================
FROM base AS development

WORKDIR /app

# 复制 package 文件
COPY frontend/package*.json ./

# 安装所有依赖（包括开发依赖）
RUN npm install

# 复制应用代码
COPY frontend/ .

# 创建非root用户
RUN adduser -D -s /bin/sh nodeuser && \
    chown -R nodeuser:nodeuser /app

USER nodeuser

# 暴露端口
EXPOSE 3000

# 开发环境启动命令（热重载）
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

# =================================
# 构建阶段
# =================================
FROM base AS builder

WORKDIR /app

# 复制依赖
COPY --from=deps /app/node_modules ./node_modules

# 复制应用代码
COPY frontend/ .

# 构建应用
RUN npm run build

# =================================
# Nginx 生产阶段
# =================================
FROM nginx:alpine AS production

# 设置Alpine镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要工具
RUN apk add --no-cache curl

# 复制构建结果到 nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf

# 创建 nginx 用户和目录
RUN mkdir -p /var/cache/nginx && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/ || exit 1

# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]
