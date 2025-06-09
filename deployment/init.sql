-- 初始化数据库脚本
-- 注意：数据库已通过 docker-compose.yml 的 POSTGRES_DB 环境变量创建

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建服务配置表
CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    service_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    config JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建路由配置表
CREATE TABLE IF NOT EXISTS routes (
    id SERIAL PRIMARY KEY,
    path VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    service_id INTEGER REFERENCES services(id),
    rate_limit INTEGER DEFAULT 1000,
    auth_required BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建监控数据表
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    service_id INTEGER REFERENCES services(id),
    request_count INTEGER DEFAULT 0,
    response_time_avg DECIMAL(10,2),
    error_count INTEGER DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入示例数据
INSERT INTO services (name, endpoint, service_type, config) VALUES
('OpenAI API', 'https://api.openai.com', 'ai_service', '{"model": "gpt-3.5-turbo", "max_tokens": 1000}'),
('Local Python Model', 'http://python-models:5000', 'local_model', '{"model_type": "transformer", "gpu_enabled": false}');

INSERT INTO routes (path, method, service_id, rate_limit) VALUES
('/chat/completions', 'POST', 1, 100),
('/models/predict', 'POST', 2, 50);
