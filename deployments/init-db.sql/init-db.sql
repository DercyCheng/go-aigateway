-- AI Gateway 数据库初始化脚本

-- 创建数据库扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 服务注册表
CREATE TABLE IF NOT EXISTS service_registry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name VARCHAR(255) NOT NULL,
    service_id VARCHAR(255) NOT NULL UNIQUE,
    address VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    health_status VARCHAR(50) DEFAULT 'unknown',
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API 密钥管理
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_id VARCHAR(255) NOT NULL UNIQUE,
    key_hash VARCHAR(512) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    permissions JSONB DEFAULT '{}',
    rate_limit INTEGER DEFAULT 60,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 用户会话管理
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    access_key_id VARCHAR(255),
    permissions JSONB DEFAULT '{}',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 请求日志
CREATE TABLE IF NOT EXISTS request_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id VARCHAR(255) NOT NULL,
    api_key_id VARCHAR(255),
    method VARCHAR(10) NOT NULL,
    path VARCHAR(1000) NOT NULL,
    query_params JSONB DEFAULT '{}',
    headers JSONB DEFAULT '{}',
    request_body TEXT,
    response_status INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    client_ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 云资源缓存
CREATE TABLE IF NOT EXISTS cloud_resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(50) NOT NULL,
    region VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255),
    resource_data JSONB NOT NULL,
    status VARCHAR(50),
    last_sync TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, region, resource_type, resource_id)
);

-- 配置存储
CREATE TABLE IF NOT EXISTS configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(255) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    description TEXT,
    is_encrypted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_service_registry_name ON service_registry(service_name);
CREATE INDEX IF NOT EXISTS idx_service_registry_status ON service_registry(health_status);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_request_logs_api_key ON request_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_cloud_resources_provider_region ON cloud_resources(provider, region);
CREATE INDEX IF NOT EXISTS idx_cloud_resources_sync ON cloud_resources(last_sync);

-- 创建更新时间戳触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为相关表创建更新触发器
CREATE TRIGGER update_service_registry_updated_at BEFORE UPDATE ON service_registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cloud_resources_updated_at BEFORE UPDATE ON cloud_resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_configurations_updated_at BEFORE UPDATE ON configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入一些初始配置
INSERT INTO configurations (key, value, description) VALUES
('rate_limit_default', '60', 'Default rate limit per minute'),
('session_timeout', '3600', 'Default session timeout in seconds'),
('cache_ttl', '300', 'Default cache TTL in seconds'),
('monitoring_enabled', 'true', 'Enable monitoring features'),
('log_level', '"info"', 'Default log level')
ON CONFLICT (key) DO NOTHING;

-- API keys will be managed through the application's local authentication system
-- No default keys are inserted for security reasons
