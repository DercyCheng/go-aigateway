-- AI Gateway 数据库初始化脚本
-- 中国大陆开发环境版本

-- 设置编码
SET client_encoding = 'UTF8';

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) UNIQUE,
    quota_used BIGINT DEFAULT 0,
    quota_limit BIGINT DEFAULT 1000000,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建API调用记录表
CREATE TABLE IF NOT EXISTS api_calls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    api_endpoint VARCHAR(255) NOT NULL,
    model_name VARCHAR(100),
    provider VARCHAR(50),
    request_tokens INTEGER DEFAULT 0,
    response_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    cost DECIMAL(10, 6) DEFAULT 0,
    duration_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建模型配置表
CREATE TABLE IF NOT EXISTS model_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    api_endpoint VARCHAR(255),
    config_json JSONB,
    is_enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(model_name, provider)
);

-- 创建服务健康状态表
CREATE TABLE IF NOT EXISTS service_health (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    response_time_ms INTEGER,
    error_count INTEGER DEFAULT 0,
    last_check TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB
);

-- 创建配置表
CREATE TABLE IF NOT EXISTS app_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT,
    config_type VARCHAR(20) DEFAULT 'string',
    description TEXT,
    is_sensitive BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_api_calls_user_id ON api_calls(user_id);
CREATE INDEX IF NOT EXISTS idx_api_calls_created_at ON api_calls(created_at);
CREATE INDEX IF NOT EXISTS idx_api_calls_provider ON api_calls(provider);
CREATE INDEX IF NOT EXISTS idx_model_configs_provider ON model_configs(provider);
CREATE INDEX IF NOT EXISTS idx_service_health_service ON service_health(service_name, provider);

-- 插入默认配置
INSERT INTO app_configs (config_key, config_value, config_type, description) VALUES
('default_model', 'glm-4', 'string', '默认AI模型'),
('max_tokens', '4096', 'integer', '最大Token数量'),
('temperature', '0.7', 'float', '默认温度参数'),
('rate_limit_rpm', '60', 'integer', '每分钟请求限制'),
('enable_streaming', 'true', 'boolean', '是否启用流式响应'),
('enable_logging', 'true', 'boolean', '是否启用详细日志'),
('maintenance_mode', 'false', 'boolean', '维护模式开关')
ON CONFLICT (config_key) DO NOTHING;

-- 插入默认模型配置 (中国大陆AI服务)
INSERT INTO model_configs (model_name, provider, api_endpoint, config_json, is_enabled, priority) VALUES
('glm-4', 'zhipu', 'https://open.bigmodel.cn/api/paas/v4/chat/completions', 
 '{"max_tokens": 4096, "temperature": 0.7, "top_p": 0.9}', true, 1),
('glm-3-turbo', 'zhipu', 'https://open.bigmodel.cn/api/paas/v4/chat/completions', 
 '{"max_tokens": 4096, "temperature": 0.7, "top_p": 0.9}', true, 2),
('ERNIE-Bot-4', 'qianfan', 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions_pro', 
 '{"max_tokens": 2048, "temperature": 0.7, "top_p": 0.8}', true, 3),
('ERNIE-Bot-turbo', 'qianfan', 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/eb-instant', 
 '{"max_tokens": 2048, "temperature": 0.7, "top_p": 0.8}', true, 4),
('qwen-turbo', 'dashscope', 'https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation', 
 '{"max_tokens": 2048, "temperature": 0.7, "top_p": 0.8}', true, 5),
('qwen-plus', 'dashscope', 'https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation', 
 '{"max_tokens": 4096, "temperature": 0.7, "top_p": 0.8}', true, 6),
('hunyuan-lite', 'hunyuan', 'https://hunyuan.tencentcloudapi.com/', 
 '{"max_tokens": 2048, "temperature": 0.7, "top_p": 0.8}', true, 7),
('spark-3.5', 'xinghuo', 'https://spark-api.xf-yun.com/v3.5/chat', 
 '{"max_tokens": 4096, "temperature": 0.7, "top_k": 4}', true, 8),
('doubao-lite-4k', 'doubao', 'https://ark.cn-beijing.volces.com/api/v3/chat/completions', 
 '{"max_tokens": 4096, "temperature": 0.7, "top_p": 0.9}', true, 9)
ON CONFLICT (model_name, provider) DO UPDATE SET
    api_endpoint = EXCLUDED.api_endpoint,
    config_json = EXCLUDED.config_json,
    updated_at = NOW();

-- 创建默认用户 (开发环境)
INSERT INTO users (username, email, password_hash, api_key, quota_limit) VALUES
('admin', 'admin@aigateway.local', '$2a$10$example_hash_for_dev_environment', 'dev-api-key-12345', 10000000),
('developer', 'dev@aigateway.local', '$2a$10$example_hash_for_dev_environment', 'dev-api-key-67890', 1000000)
ON CONFLICT (username) DO NOTHING;

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为相关表创建更新时间触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_model_configs_updated_at BEFORE UPDATE ON model_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_app_configs_updated_at BEFORE UPDATE ON app_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 创建视图：用户使用统计
CREATE OR REPLACE VIEW user_usage_stats AS
SELECT 
    u.id,
    u.username,
    u.email,
    u.quota_used,
    u.quota_limit,
    u.quota_used::float / u.quota_limit::float * 100 as usage_percentage,
    COUNT(ac.id) as total_calls,
    SUM(ac.total_tokens) as total_tokens_used,
    SUM(ac.cost) as total_cost,
    AVG(ac.duration_ms) as avg_duration_ms
FROM users u
LEFT JOIN api_calls ac ON u.id = ac.user_id
GROUP BY u.id, u.username, u.email, u.quota_used, u.quota_limit;

-- 创建视图：模型使用统计
CREATE OR REPLACE VIEW model_usage_stats AS
SELECT 
    model_name,
    provider,
    COUNT(*) as total_calls,
    SUM(request_tokens) as total_request_tokens,
    SUM(response_tokens) as total_response_tokens,
    SUM(total_tokens) as total_tokens,
    AVG(duration_ms) as avg_duration_ms,
    SUM(cost) as total_cost,
    COUNT(CASE WHEN status_code = 200 THEN 1 END) as successful_calls,
    COUNT(CASE WHEN status_code != 200 THEN 1 END) as failed_calls
FROM api_calls
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY model_name, provider
ORDER BY total_calls DESC;

-- 创建视图：今日统计
CREATE OR REPLACE VIEW daily_stats AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_calls,
    SUM(total_tokens) as total_tokens,
    SUM(cost) as total_cost,
    AVG(duration_ms) as avg_duration_ms,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT model_name) as models_used
FROM api_calls
WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- 插入一些示例数据 (仅开发环境)
INSERT INTO api_calls (user_id, api_endpoint, model_name, provider, request_tokens, response_tokens, total_tokens, cost, duration_ms, status_code)
SELECT 
    (SELECT id FROM users WHERE username = 'developer'),
    '/api/v1/chat/completions',
    'glm-4',
    'zhipu',
    100 + (random() * 200)::integer,
    150 + (random() * 300)::integer,
    250 + (random() * 500)::integer,
    (random() * 0.01)::decimal(10,6),
    (500 + random() * 2000)::integer,
    200
FROM generate_series(1, 10);

-- 提交事务
COMMIT;

-- 输出初始化完成信息
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'AI Gateway 数据库初始化完成';
    RAISE NOTICE '========================================';
    RAISE NOTICE '创建的表:';
    RAISE NOTICE '  - users: 用户表';
    RAISE NOTICE '  - api_calls: API调用记录表';
    RAISE NOTICE '  - model_configs: 模型配置表';
    RAISE NOTICE '  - service_health: 服务健康状态表';
    RAISE NOTICE '  - app_configs: 应用配置表';
    RAISE NOTICE '';
    RAISE NOTICE '默认用户:';
    RAISE NOTICE '  - admin (admin@aigateway.local)';
    RAISE NOTICE '  - developer (dev@aigateway.local)';
    RAISE NOTICE '';
    RAISE NOTICE '支持的中国大陆AI服务:';
    RAISE NOTICE '  - 智谱AI (ChatGLM)';
    RAISE NOTICE '  - 百度千帆';
    RAISE NOTICE '  - 阿里云通义千问';
    RAISE NOTICE '  - 腾讯混元';
    RAISE NOTICE '  - 讯飞星火';
    RAISE NOTICE '  - 字节豆包';
    RAISE NOTICE '========================================';
END $$;
