import unittest
import json
import tempfile
import os
from unittest.mock import patch, MagicMock
import sys

# Add the parent directory to Python path for imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from error_handling import (
    ValidationError, SecurityError, ResourceError,
    validate_input_security, ResourceManager
)

class TestErrorHandling(unittest.TestCase):
    """Test cases for error handling utilities"""
    
    def test_validation_error(self):
        """Test ValidationError creation and properties"""
        error = ValidationError("Test message", "test_field")
        self.assertEqual(error.message, "Test message")
        self.assertEqual(error.field, "test_field")
        self.assertEqual(str(error), "Test message")
    
    def test_security_error(self):
        """Test SecurityError creation and properties"""
        error = SecurityError("Security violation", "TEST_CODE")
        self.assertEqual(error.message, "Security violation")
        self.assertEqual(error.code, "TEST_CODE")
    
    def test_resource_error(self):
        """Test ResourceError creation and properties"""
        error = ResourceError("Resource unavailable", "gpu")
        self.assertEqual(error.message, "Resource unavailable")
        self.assertEqual(error.resource_type, "gpu")

class TestInputSecurity(unittest.TestCase):
    """Test cases for input security validation"""
    
    def test_safe_input(self):
        """Test that safe input passes validation"""
        safe_data = {
            "message": "Hello, world!",
            "number": 42,
            "array": [1, 2, 3],
            "nested": {"key": "value"}
        }
        # Should not raise any exception
        validate_input_security(safe_data)
    
    def test_dangerous_strings(self):
        """Test detection of dangerous string patterns"""
        dangerous_inputs = [
            "__import__('os').system('rm -rf /')",
            "eval('malicious code')",
            "<script>alert('xss')</script>",
            "javascript:void(0)",
            "../../../etc/passwd",
            "DROP TABLE users;",
            "cmd.exe /c dir"
        ]
        
        for dangerous_input in dangerous_inputs:
            with self.assertRaises(SecurityError):
                validate_input_security(dangerous_input)
    
    def test_dangerous_keys(self):
        """Test detection of dangerous dictionary keys"""
        dangerous_data = {
            "__proto__": "malicious",
            "constructor": "bad",
            "prototype": "evil"
        }
        
        for key in dangerous_data:
            with self.assertRaises(SecurityError):
                validate_input_security({key: "value"})
    
    def test_deep_nesting(self):
        """Test protection against deeply nested structures"""
        # Create deeply nested structure
        deep_data = {"level": 0}
        current = deep_data
        for i in range(1, 15):  # Exceed max depth of 10
            current["next"] = {"level": i}
            current = current["next"]
        
        with self.assertRaises(SecurityError):
            validate_input_security(deep_data)
    
    def test_large_array(self):
        """Test protection against large arrays"""
        large_array = list(range(2000))  # Exceed limit of 1000
        
        with self.assertRaises(SecurityError):
            validate_input_security(large_array)
    
    def test_long_string(self):
        """Test protection against excessively long strings"""
        long_string = "a" * 15000  # Exceed limit of 10000
        
        with self.assertRaises(SecurityError):
            validate_input_security(long_string)

class TestResourceManager(unittest.TestCase):
    """Test cases for ResourceManager"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.resource_manager = ResourceManager()
    
    def test_acquire_release_resource(self):
        """Test basic resource acquisition and release"""
        # Initially no active requests
        self.assertEqual(self.resource_manager.active_requests, 0)
        
        # Acquire resource
        self.assertTrue(self.resource_manager.acquire_resource())
        self.assertEqual(self.resource_manager.active_requests, 1)
        
        # Release resource
        self.resource_manager.release_resource()
        self.assertEqual(self.resource_manager.active_requests, 0)
    
    def test_max_concurrent_requests(self):
        """Test max concurrent request limit"""
        # Set low limit for testing
        self.resource_manager.max_concurrent_requests = 2
        
        # Acquire up to limit
        self.assertTrue(self.resource_manager.acquire_resource())
        self.assertTrue(self.resource_manager.acquire_resource())
        
        # Should fail to acquire more
        with self.assertRaises(ResourceError):
            self.resource_manager.acquire_resource()
    
    def test_get_status(self):
        """Test resource status reporting"""
        status = self.resource_manager.get_status()
        
        # Check required fields
        self.assertIn("active_requests", status)
        self.assertIn("max_concurrent_requests", status)
        self.assertIn("cpu_usage_percent", status)
        
        # Check types
        self.assertIsInstance(status["active_requests"], int)
        self.assertIsInstance(status["max_concurrent_requests"], int)
        self.assertIsInstance(status["cpu_usage_percent"], (int, float))
    
    @patch('torch.cuda.is_available')
    @patch('torch.cuda.memory_allocated')
    @patch('torch.cuda.max_memory_allocated')
    def test_gpu_status_with_torch(self, mock_max_mem, mock_alloc_mem, mock_cuda_available):
        """Test GPU status when torch is available"""
        mock_cuda_available.return_value = True
        mock_alloc_mem.return_value = 1000000
        mock_max_mem.return_value = 2000000
        
        status = self.resource_manager.get_status()
        
        self.assertIn("gpu_memory_used", status)
        self.assertIn("gpu_memory_total", status)
        self.assertIn("gpu_memory_percent", status)
        self.assertEqual(status["gpu_memory_percent"], 50.0)

class TestModelServer(unittest.TestCase):
    """Test cases for model server functionality"""
    
    def setUp(self):
        """Set up test Flask app"""
        # Import here to avoid module-level imports during testing
        from server import app
        self.app = app.test_client()
        self.app.testing = True
    
    def test_health_endpoint(self):
        """Test health check endpoint"""
        response = self.app.get('/health')
        self.assertEqual(response.status_code, 200)
        
        data = json.loads(response.data)
        self.assertIn('status', data)
        self.assertIn('timestamp', data)
    
    def test_chat_completions_validation(self):
        """Test input validation for chat completions"""
        # Test missing messages
        response = self.app.post('/v1/chat/completions',
                                json={},
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 400)
        
        # Test invalid messages format
        response = self.app.post('/v1/chat/completions',
                                json={'messages': 'invalid'},
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 400)
        
        # Test empty messages array
        response = self.app.post('/v1/chat/completions',
                                json={'messages': []},
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 400)
    
    def test_invalid_content_type(self):
        """Test rejection of non-JSON content"""
        response = self.app.post('/v1/chat/completions',
                                data='not json',
                                headers={'Content-Type': 'text/plain'})
        self.assertEqual(response.status_code, 400)
    
    @patch('server.model')
    @patch('server.tokenizer')
    def test_model_not_loaded(self, mock_tokenizer, mock_model):
        """Test behavior when model is not loaded"""
        mock_model = None
        mock_tokenizer = None
        
        response = self.app.post('/v1/chat/completions',
                                json={'messages': [{'role': 'user', 'content': 'test'}]},
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 503)

class TestSecurityValidation(unittest.TestCase):
    """Test cases for security validation"""
    
    def setUp(self):
        """Set up test Flask app"""
        from server import app
        self.app = app.test_client()
        self.app.testing = True
    
    def test_xss_protection(self):
        """Test XSS attack protection"""
        malicious_content = "<script>alert('xss')</script>"
        
        response = self.app.post('/v1/chat/completions',
                                json={
                                    'messages': [
                                        {'role': 'user', 'content': malicious_content}
                                    ]
                                },
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 403)
    
    def test_injection_protection(self):
        """Test SQL injection and command injection protection"""
        malicious_inputs = [
            "'; DROP TABLE users; --",
            "__import__('os').system('rm -rf /')",
            "eval('malicious_code')"
        ]
        
        for malicious_input in malicious_inputs:
            response = self.app.post('/v1/chat/completions',
                                    json={
                                        'messages': [
                                            {'role': 'user', 'content': malicious_input}
                                        ]
                                    },
                                    headers={'Content-Type': 'application/json'})
            self.assertEqual(response.status_code, 403)
    
    def test_parameter_validation(self):
        """Test parameter validation"""
        # Test invalid temperature
        response = self.app.post('/v1/chat/completions',
                                json={
                                    'messages': [{'role': 'user', 'content': 'test'}],
                                    'temperature': 5.0  # Invalid: > 2.0
                                },
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 400)
        
        # Test invalid max_tokens
        response = self.app.post('/v1/chat/completions',
                                json={
                                    'messages': [{'role': 'user', 'content': 'test'}],
                                    'max_tokens': 10000  # Invalid: > 4096
                                },
                                headers={'Content-Type': 'application/json'})
        self.assertEqual(response.status_code, 400)

if __name__ == '__main__':
    # Create a test suite
    test_suite = unittest.TestSuite()
    
    # Add test cases
    test_suite.addTest(unittest.makeSuite(TestErrorHandling))
    test_suite.addTest(unittest.makeSuite(TestInputSecurity))
    test_suite.addTest(unittest.makeSuite(TestResourceManager))
    test_suite.addTest(unittest.makeSuite(TestModelServer))
    test_suite.addTest(unittest.makeSuite(TestSecurityValidation))
    
    # Run tests with detailed output
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(test_suite)
    
    # Exit with error code if tests failed
    if not result.wasSuccessful():
        sys.exit(1)
