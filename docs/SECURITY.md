# AI Gateway Security Guide

## Security Improvements Implemented

This document outlines the critical security issues that have been addressed in the AI Gateway project.

## 🔒 Issues Fixed

### 1. CORS Configuration Security
**Issue**: Wildcard origins (`*`) with credentials enabled creates security vulnerabilities
**Fix**: 
- Implemented configurable allowed origins
- Removed wildcard usage in production
- Added proper origin validation
- Fixed credentials header conflicts

**Configuration**:
```env
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### 2. Hardcoded API Keys Removal
**Issue**: Hardcoded API keys in database initialization and test files
**Fix**:
- Removed all hardcoded API keys from `init-db.sql`
- Updated test files to use environment variables
- Updated Kubernetes secrets configuration
- Enhanced `.env.example` with secure defaults

**Before**:
```sql
INSERT INTO api_keys (key_id, key_hash, name, description, permissions) VALUES
('dev-key-001', 'sha256-hash-placeholder', 'Development Key', 'Default development API key', '{"admin": true, "read": true, "write": true}')
```

**After**:
```sql
-- API keys will be managed through the application's local authentication system
-- No default keys are inserted for security reasons
```

### 3. Local Authentication System
**Issue**: RAM authentication dependency on external cloud services
**Fix**: 
- Implemented complete local authentication system
- JWT-based token authentication
- Role-based permission system
- API key management with granular permissions
- Session management with automatic cleanup

## 🛡️ New Security Features

### Local Authentication System
- **JWT Tokens**: Secure, stateless authentication
- **API Key Management**: Generate, revoke, and manage API keys
- **Role-Based Access Control**: Admin, API user roles with specific permissions
- **Session Management**: Automatic cleanup of expired sessions
- **Rate Limiting**: Per-user and global rate limiting

### Authentication Methods
1. **JWT Bearer Tokens**: For user sessions
2. **API Keys**: For service-to-service communication

### Default Users
- **Admin User**: `admin` / `admin123` (change in production!)
- **API User**: `apiuser` / `api123` (change in production!)

## 🔧 Configuration

### Environment Variables
```env
# Local Authentication & Security Configuration
LOCAL_AUTH_ENABLED=true
JWT_SECRET=your-super-secure-jwt-secret-at-least-32-chars-long
JWT_EXPIRATION=24h
REQUIRE_HTTPS=false
API_KEY_PREFIX=gw_

# CORS Configuration (Security Enhanced)
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Requested-With
```

### API Endpoints

#### Authentication
- `POST /auth/login` - User login
- `POST /auth/refresh` - Token refresh

#### API Key Management (Admin only)
- `POST /admin/api-keys` - Create API key
- `GET /admin/api-keys` - List API keys
- `DELETE /admin/api-keys/:id` - Delete API key
- `PUT /admin/api-keys/:id` - Update API key

#### Protected API Endpoints
- `POST /v1/chat/completions` - Chat completions (requires API permission)
- `POST /v1/completions` - Text completions (requires API permission)
- `GET /v1/models` - List models (requires API permission)

## 🔐 Usage Examples

### 1. Login and Get Token
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

### 2. Create API Key (Admin)
```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "permissions": {
      "api": true,
      "read": true
    },
    "rate_limit": 100
  }'
```

### 3. Use API Key for Requests
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "X-API-Key: gw_your_generated_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen-plus",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### 4. Use JWT Token for Requests
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen-plus",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 🚨 Security Best Practices

### For Production Deployment:

1. **Change Default Passwords**: 
   - Update default admin/apiuser passwords
   - Use strong, unique passwords

2. **Secure JWT Secret**:
   - Use a cryptographically secure random string (at least 32 characters)
   - Store in secure environment variables

3. **HTTPS Only**:
   ```env
   REQUIRE_HTTPS=true
   TLS_ENABLED=true
   TLS_CERT_FILE=/path/to/cert.pem
   TLS_KEY_FILE=/path/to/key.pem
   ```

4. **Restrict CORS Origins**:
   - Only allow specific frontend domains
   - Never use wildcard (`*`) in production

5. **API Key Management**:
   - Regularly rotate API keys
   - Set appropriate expiration times
   - Use least-privilege permissions

6. **Rate Limiting**:
   - Configure appropriate rate limits per user/key
   - Monitor for suspicious activity

## 🔍 Security Monitoring

The system includes built-in security monitoring:
- Failed authentication attempts logging
- API key usage tracking
- Rate limit violation alerts
- Session expiration cleanup

## 📝 Migration from Old System

If upgrading from the previous RAM-based authentication:

1. Update environment variables (remove RAM_* configs)
2. Deploy new authentication system
3. Create new API keys using the admin interface
4. Update client applications to use new authentication methods
5. Test thoroughly before switching traffic

## 🛠️ Development

For development environment:
- Default users are pre-created for testing
- CORS allows localhost origins
- JWT secrets can be simple for testing
- HTTPS requirement is disabled by default

**Note**: Always use secure configurations in production!
