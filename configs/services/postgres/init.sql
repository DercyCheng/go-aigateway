-- AI Gateway 数据库初始化脚本
-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建 API 密钥表
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    permissions TEXT[],
    is_active BOOLEAN DEFAULT true,
    rate_limit INTEGER DEFAULT 1000,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE
);

-- 创建模型配置表
CREATE TABLE IF NOT EXISTS model_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model_type VARCHAR(50) NOT NULL,
    endpoint_url TEXT,
    config_data JSONB,
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建请求日志表
CREATE TABLE IF NOT EXISTS request_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    model_name VARCHAR(100),
    provider VARCHAR(50),
    request_method VARCHAR(10),
    request_path TEXT,
    request_size INTEGER,
    response_size INTEGER,
    response_status INTEGER,
    processing_time_ms INTEGER,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_model_configs_name ON model_configs(name);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要的表添加更新触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_model_configs_updated_at BEFORE UPDATE ON model_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入默认管理员用户 (密码: admin123，请在生产环境中修改)
INSERT INTO users (username, email, password_hash) VALUES 
('admin', 'admin@aigateway.local', '$2a$10$N9qo8uLOickgx2ZMRZoMye5FjJu5UvW7.g.hU7.0cC9jm5.1G8/a2')
ON CONFLICT (username) DO NOTHING;

-- 插入默认 API 密钥
INSERT INTO api_keys (user_id, key_name, key_hash, permissions) 
SELECT u.id, 'Default Admin Key', 'sk-admin-default-key-hash', ARRAY['admin', 'read', 'write']
FROM users u WHERE u.username = 'admin'
ON CONFLICT DO NOTHING;

-- 创建视图：用户统计
CREATE OR REPLACE VIEW user_stats AS
SELECT 
    u.id,
    u.username,
    u.email,
    COUNT(DISTINCT ak.id) as api_key_count,
    COUNT(DISTINCT rl.id) as request_count,
    MAX(rl.created_at) as last_request_at
FROM users u
LEFT JOIN api_keys ak ON u.id = ak.user_id AND ak.is_active = true
LEFT JOIN request_logs rl ON u.id = rl.user_id
GROUP BY u.id, u.username, u.email;

-- 创建视图：模型使用统计
CREATE OR REPLACE VIEW model_usage_stats AS
SELECT 
    model_name,
    provider,
    COUNT(*) as request_count,
    AVG(processing_time_ms) as avg_processing_time,
    SUM(request_size) as total_request_size,
    SUM(response_size) as total_response_size,
    COUNT(CASE WHEN response_status >= 400 THEN 1 END) as error_count
FROM request_logs
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY model_name, provider;
