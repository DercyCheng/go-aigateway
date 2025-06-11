#!/bin/bash

# 数据库初始化脚本
set -e

# 创建数据库
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- 创建扩展
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
    
    -- 创建用户表
    CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        username VARCHAR(255) UNIQUE NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL,
        role VARCHAR(50) DEFAULT 'user',
        api_key VARCHAR(255) UNIQUE,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    -- 创建服务表
    CREATE TABLE IF NOT EXISTS services (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        name VARCHAR(255) UNIQUE NOT NULL,
        endpoint VARCHAR(255) NOT NULL,
        protocol VARCHAR(50) DEFAULT 'http',
        health_check_path VARCHAR(255) DEFAULT '/health',
        enabled BOOLEAN DEFAULT true,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    -- 创建请求日志表
    CREATE TABLE IF NOT EXISTS request_logs (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        user_id UUID REFERENCES users(id),
        service_name VARCHAR(255),
        method VARCHAR(10),
        path VARCHAR(255),
        status_code INTEGER,
        response_time_ms INTEGER,
        request_size INTEGER,
        response_size INTEGER,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    -- 创建模型表
    CREATE TABLE IF NOT EXISTS models (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        name VARCHAR(255) UNIQUE NOT NULL,
        type VARCHAR(50) NOT NULL,
        version VARCHAR(50) DEFAULT '1.0.0',
        path VARCHAR(255),
        config JSONB,
        enabled BOOLEAN DEFAULT true,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    -- 创建API密钥表
    CREATE TABLE IF NOT EXISTS api_keys (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        user_id UUID REFERENCES users(id),
        key_hash VARCHAR(255) UNIQUE NOT NULL,
        name VARCHAR(255),
        permissions JSONB,
        last_used_at TIMESTAMP WITH TIME ZONE,
        expires_at TIMESTAMP WITH TIME ZONE,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    -- 创建索引
    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    CREATE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key);
    CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
    CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
    CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
    CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
    
    -- 插入示例数据
    INSERT INTO users (username, email, password_hash, role, api_key) 
    VALUES ('admin', 'admin@example.com', 'hashed_password_here', 'admin', 'ak_dev_admin_key_12345')
    ON CONFLICT (username) DO NOTHING;
    
    INSERT INTO services (name, endpoint, protocol) 
    VALUES 
        ('python-models', 'http://python-models:5000', 'http'),
        ('external-api', 'https://api.example.com', 'https')
    ON CONFLICT (name) DO NOTHING;
EOSQL

echo "数据库初始化完成"
