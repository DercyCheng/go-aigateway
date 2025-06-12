-- AI Gateway 数据库初始化脚本
-- 开发环境数据库表结构

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- 服务配置表
CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    endpoint VARCHAR(500) NOT NULL,
    service_type VARCHAR(50) NOT NULL, -- http, grpc, websocket
    status VARCHAR(20) DEFAULT 'active', -- active, inactive, error
    health_check_url VARCHAR(500),
    timeout_seconds INTEGER DEFAULT 30,
    retry_count INTEGER DEFAULT 3,
    rate_limit INTEGER DEFAULT 1000,
    auth_required BOOLEAN DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 路由配置表
CREATE TABLE IF NOT EXISTS routes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_id UUID REFERENCES services(id) ON DELETE CASCADE,
    path VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL, -- GET, POST, PUT, DELETE, etc.
    upstream_path VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    middleware_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(path, method)
);

-- 认证用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user', -- admin, user, developer
    is_active BOOLEAN DEFAULT true,
    api_key VARCHAR(255) UNIQUE,
    rate_limit INTEGER DEFAULT 1000,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API 密钥表
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    permissions JSONB,
    rate_limit INTEGER DEFAULT 1000,
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 请求日志表
CREATE TABLE IF NOT EXISTS request_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    service_id UUID REFERENCES services(id) ON DELETE SET NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    query_params JSONB,
    request_headers JSONB,
    response_status INTEGER,
    response_time_ms INTEGER,
    request_size INTEGER,
    response_size INTEGER,
    ip_address INET,
    user_agent TEXT,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    is_encrypted BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 证书管理表
CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain VARCHAR(255) NOT NULL,
    certificate_data TEXT NOT NULL,
    private_key_data TEXT NOT NULL,
    ca_bundle TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    auto_renew BOOLEAN DEFAULT true,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(domain)
);

-- 本地模型表
CREATE TABLE IF NOT EXISTS local_models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    model_type VARCHAR(50) NOT NULL, -- llm, embedding, image, etc.
    model_path VARCHAR(500) NOT NULL,
    config_data JSONB,
    status VARCHAR(20) DEFAULT 'inactive', -- active, inactive, loading, error
    resource_requirements JSONB, -- memory, gpu, etc.
    performance_metrics JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
CREATE INDEX IF NOT EXISTS idx_routes_path ON routes(path);
CREATE INDEX IF NOT EXISTS idx_routes_service_id ON routes(service_id);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key);
CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(api_key);
CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_service_id ON request_logs(service_id);
CREATE INDEX IF NOT EXISTS idx_certificates_domain ON certificates(domain);
CREATE INDEX IF NOT EXISTS idx_certificates_expires_at ON certificates(expires_at);
CREATE INDEX IF NOT EXISTS idx_local_models_status ON local_models(status);

-- 插入默认数据
INSERT INTO users (username, email, password_hash, role, api_key) VALUES
('admin', 'admin@localhost', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin', 'dev_admin_key_2024')
ON CONFLICT (username) DO NOTHING;

INSERT INTO system_configs (config_key, config_value, description) VALUES
('rate_limit_global', '{"requests_per_minute": 10000, "burst": 100}', '全局速率限制配置'),
('cache_config', '{"default_ttl": 300, "max_size": "100MB"}', '缓存配置'),
('security_config', '{"cors_enabled": true, "auth_required": false}', '安全配置')
ON CONFLICT (config_key) DO NOTHING;

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为有 updated_at 字段的表创建触发器
CREATE TRIGGER update_services_updated_at BEFORE UPDATE ON services FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_routes_updated_at BEFORE UPDATE ON routes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_system_configs_updated_at BEFORE UPDATE ON system_configs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_certificates_updated_at BEFORE UPDATE ON certificates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_local_models_updated_at BEFORE UPDATE ON local_models FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
