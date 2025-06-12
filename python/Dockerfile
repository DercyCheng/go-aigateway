# Python 模型服务 Dockerfile
# 针对中国内地环境和 HuggingFace 模型优化

# =================================
# 基础镜像
# =================================
FROM python:3.11-slim as base

# 设置环境变量
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PIP_NO_CACHE_DIR=1
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

# 设置中国内地 PyPI 镜像
ENV PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple/
ENV PIP_TRUSTED_HOST=pypi.tuna.tsinghua.edu.cn

# 设置 HuggingFace 镜像（中国内地加速）
ENV HF_ENDPOINT=https://hf-mirror.com
ENV HF_HOME=/app/.cache/huggingface

# 更新系统包管理器源为中科大镜像
RUN sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/debian.sources

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# =================================
# 开发阶段
# =================================
FROM base AS development

# 安装开发工具
RUN apt-get update && apt-get install -y \
    vim \
    htop \
    && rm -rf /var/lib/apt/lists/*

# 创建应用用户
RUN useradd --create-home --shell /bin/bash appuser

# 设置工作目录
WORKDIR /app

# 安装 Python 依赖
COPY python/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 安装开发依赖
RUN pip install --no-cache-dir \
    jupyter \
    notebook \
    ipython \
    pytest \
    pytest-cov \
    black \
    flake8

# 复制应用代码
COPY python/ .

# 更改目录所有者
RUN chown -R appuser:appuser /app

USER appuser

# 创建缓存目录
RUN mkdir -p /app/.cache/huggingface

# 暴露端口
EXPOSE 5000 8888

# 开发环境启动命令（支持热重载）
CMD ["python", "-u", "server.py"]

# =================================
# 生产阶段
# =================================
FROM base AS production

# 创建应用用户
RUN useradd --create-home --shell /bin/bash appuser

# 设置工作目录
WORKDIR /app

# 复制并安装依赖
COPY python/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制应用代码
COPY python/ .

# 创建缓存目录
RUN mkdir -p /app/.cache/huggingface

# 更改目录所有者
RUN chown -R appuser:appuser /app

USER appuser

# 暴露端口
EXPOSE 5000

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:5000/health || exit 1

# 生产环境启动命令
CMD ["python", "-u", "server.py"]
