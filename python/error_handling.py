import logging
import traceback
import functools
from typing import Any, Callable, Dict, Optional
from flask import jsonify, request
from werkzeug.exceptions import BadRequest, Unauthorized, Forbidden, NotFound, InternalServerError
import time

# Configure structured logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s - %(pathname)s:%(lineno)d'
)
logger = logging.getLogger(__name__)

class SecurityError(Exception):
    """Custom exception for security-related errors"""
    def __init__(self, message: str, code: str = "SECURITY_ERROR"):
        self.message = message
        self.code = code
        super().__init__(self.message)

class ValidationError(Exception):
    """Custom exception for validation errors"""
    def __init__(self, message: str, field: str = None):
        self.message = message
        self.field = field
        super().__init__(self.message)

class ResourceError(Exception):
    """Custom exception for resource-related errors"""
    def __init__(self, message: str, resource_type: str = None):
        self.message = message
        self.resource_type = resource_type
        super().__init__(self.message)

def handle_errors(f: Callable) -> Callable:
    """Decorator for comprehensive error handling"""
    @functools.wraps(f)
    def wrapper(*args, **kwargs):
        try:
            return f(*args, **kwargs)
        except ValidationError as e:
            logger.warning(f"Validation error in {f.__name__}: {e.message}")
            return jsonify({
                "error": {
                    "type": "validation_error",
                    "code": "VALIDATION_FAILED",
                    "message": e.message,
                    "field": e.field
                }
            }), 400
        except SecurityError as e:
            logger.warning(f"Security error in {f.__name__}: {e.message}")
            return jsonify({
                "error": {
                    "type": "security_error",
                    "code": e.code,
                    "message": "Security violation detected"
                }
            }), 403
        except ResourceError as e:
            logger.error(f"Resource error in {f.__name__}: {e.message}")
            return jsonify({
                "error": {
                    "type": "resource_error",
                    "code": "RESOURCE_UNAVAILABLE",
                    "message": e.message,
                    "resource_type": e.resource_type
                }
            }), 503
        except BadRequest as e:
            logger.warning(f"Bad request in {f.__name__}: {e.description}")
            return jsonify({
                "error": {
                    "type": "bad_request",
                    "code": "INVALID_REQUEST",
                    "message": e.description or "Invalid request format"
                }
            }), 400
        except Exception as e:
            # Log the full traceback for debugging
            logger.error(f"Unexpected error in {f.__name__}: {str(e)}")
            logger.error(traceback.format_exc())
            
            return jsonify({
                "error": {
                    "type": "internal_error",
                    "code": "INTERNAL_SERVER_ERROR",
                    "message": "An unexpected error occurred"
                }
            }), 500
    return wrapper

def validate_request_data(required_fields: list = None, max_size: int = 1024*1024) -> Callable:
    """Decorator for request validation"""
    def decorator(f: Callable) -> Callable:
        @functools.wraps(f)
        def wrapper(*args, **kwargs):
            # Check content type
            if not request.is_json:
                raise ValidationError("Content-Type must be application/json")
            
            # Check request size
            if request.content_length and request.content_length > max_size:
                raise ValidationError(f"Request too large. Maximum size: {max_size} bytes")
            
            try:
                data = request.get_json()
            except Exception as e:
                raise ValidationError("Invalid JSON format")
            
            if data is None:
                raise ValidationError("Request body is required")
            
            # Validate required fields
            if required_fields:
                missing_fields = []
                for field in required_fields:
                    if field not in data or data[field] is None:
                        missing_fields.append(field)
                
                if missing_fields:
                    raise ValidationError(
                        f"Missing required fields: {', '.join(missing_fields)}"
                    )
            
            # Validate against dangerous patterns
            validate_input_security(data)
            
            return f(*args, **kwargs)
        return wrapper
    return decorator

def validate_input_security(data: Any, max_depth: int = 10, current_depth: int = 0):
    """Validate input for security threats"""
    if current_depth > max_depth:
        raise SecurityError("Input structure too deep")
    
    dangerous_patterns = [
        "__import__", "eval", "exec", "compile", "open", "file",
        "<script", "</script>", "javascript:", "data:",
        "../", "..\\", "/etc/", "c:\\", "cmd.exe", "powershell",
        "rm -rf", "del /", "format c:", "DROP TABLE"
    ]
    
    if isinstance(data, str):
        lower_data = data.lower()
        for pattern in dangerous_patterns:
            if pattern in lower_data:
                raise SecurityError(f"Dangerous pattern detected: {pattern}")
        
        # Check for excessively long strings
        if len(data) > 10000:
            raise SecurityError("Input string too long")
    
    elif isinstance(data, dict):
        for key, value in data.items():
            if not isinstance(key, str):
                raise SecurityError("Dictionary keys must be strings")
            
            # Check key names
            if key.startswith('__') or key in ['constructor', 'prototype']:
                raise SecurityError(f"Dangerous key name: {key}")
            
            validate_input_security(value, max_depth, current_depth + 1)
    
    elif isinstance(data, list):
        if len(data) > 1000:
            raise SecurityError("Array too large")
        
        for item in data:
            validate_input_security(item, max_depth, current_depth + 1)

def rate_limit(max_requests: int = 60, window: int = 60) -> Callable:
    """Simple rate limiting decorator"""
    request_counts = {}
    
    def decorator(f: Callable) -> Callable:
        @functools.wraps(f)
        def wrapper(*args, **kwargs):
            client_ip = request.environ.get('HTTP_X_FORWARDED_FOR', request.environ.get('REMOTE_ADDR'))
            current_time = time.time()
            
            # Clean old entries
            cutoff = current_time - window
            request_counts[client_ip] = [
                timestamp for timestamp in request_counts.get(client_ip, [])
                if timestamp > cutoff
            ]
            
            # Check rate limit
            if len(request_counts.get(client_ip, [])) >= max_requests:
                logger.warning(f"Rate limit exceeded for IP: {client_ip}")
                return jsonify({
                    "error": {
                        "type": "rate_limit_error",
                        "code": "RATE_LIMIT_EXCEEDED",
                        "message": f"Rate limit exceeded: {max_requests} requests per {window} seconds"
                    }
                }), 429
            
            # Record this request
            if client_ip not in request_counts:
                request_counts[client_ip] = []
            request_counts[client_ip].append(current_time)
            
            return f(*args, **kwargs)
        return wrapper
    return decorator

def require_api_key(f: Callable) -> Callable:
    """API key authentication decorator"""
    @functools.wraps(f)
    def wrapper(*args, **kwargs):
        auth_header = request.headers.get('Authorization', '')
        
        if not auth_header.startswith('Bearer '):
            raise Unauthorized("Missing or invalid Authorization header")
        
        api_key = auth_header[7:]  # Remove 'Bearer ' prefix
        
        # Validate API key format
        if len(api_key) < 10:
            raise Unauthorized("Invalid API key format")
        
        # Here you would validate against your API key store
        # For now, we'll just check it's not empty and has minimum length
        
        return f(*args, **kwargs)
    return wrapper

def log_request(f: Callable) -> Callable:
    """Request logging decorator"""
    @functools.wraps(f)
    def wrapper(*args, **kwargs):
        start_time = time.time()
        client_ip = request.environ.get('HTTP_X_FORWARDED_FOR', request.environ.get('REMOTE_ADDR'))
        
        logger.info(f"Request started: {request.method} {request.path} from {client_ip}")
        
        try:
            result = f(*args, **kwargs)
            duration = time.time() - start_time
            logger.info(f"Request completed: {request.method} {request.path} in {duration:.3f}s")
            return result
        except Exception as e:
            duration = time.time() - start_time
            logger.error(f"Request failed: {request.method} {request.path} in {duration:.3f}s - {str(e)}")
            raise
    return wrapper

class ResourceManager:
    """Manages computational resources"""
    def __init__(self):
        self.active_requests = 0
        self.max_concurrent_requests = 10
        self.gpu_memory_threshold = 0.9
    
    def acquire_resource(self) -> bool:
        """Attempt to acquire resources for processing"""
        if self.active_requests >= self.max_concurrent_requests:
            raise ResourceError("Too many concurrent requests", "compute")
        
        # Check GPU memory if available
        try:
            import torch
            if torch.cuda.is_available():
                memory_used = torch.cuda.memory_allocated() / torch.cuda.max_memory_allocated()
                if memory_used > self.gpu_memory_threshold:
                    raise ResourceError("GPU memory threshold exceeded", "gpu_memory")
        except ImportError:
            pass  # torch not available
        
        self.active_requests += 1
        return True
    
    def release_resource(self):
        """Release acquired resources"""
        if self.active_requests > 0:
            self.active_requests -= 1
    
    def get_status(self) -> Dict[str, Any]:
        """Get current resource status"""
        status = {
            "active_requests": self.active_requests,
            "max_concurrent_requests": self.max_concurrent_requests,
            "cpu_usage_percent": self._get_cpu_usage(),
        }
        
        try:
            import torch
            if torch.cuda.is_available():
                status["gpu_memory_used"] = torch.cuda.memory_allocated()
                status["gpu_memory_total"] = torch.cuda.max_memory_allocated()
                status["gpu_memory_percent"] = (
                    status["gpu_memory_used"] / status["gpu_memory_total"] * 100
                    if status["gpu_memory_total"] > 0 else 0
                )
        except ImportError:
            status["gpu_available"] = False
        
        return status
    
    def _get_cpu_usage(self) -> float:
        """Get current CPU usage percentage"""
        try:
            import psutil
            return psutil.cpu_percent(interval=1)
        except ImportError:
            return 0.0

# Global resource manager instance
resource_manager = ResourceManager()

def with_resource_management(f: Callable) -> Callable:
    """Decorator for resource management"""
    @functools.wraps(f)
    def wrapper(*args, **kwargs):
        resource_manager.acquire_resource()
        try:
            return f(*args, **kwargs)
        finally:
            resource_manager.release_resource()
    return wrapper
