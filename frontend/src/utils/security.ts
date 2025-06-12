import { useEffect, useState } from 'react';

// Error types
export interface APIError {
    code: string;
    message: string;
    details?: any;
    traceId?: string;
}

export interface ErrorState {
    error: APIError | null;
    isLoading: boolean;
    retryCount: number;
}

// Custom error classes
export class ValidationError extends Error {
    public field?: string;
    public code: string;

    constructor(
        message: string,
        field?: string,
        code: string = 'VALIDATION_ERROR'
    ) {
        super(message);
        this.name = 'ValidationError';
        this.field = field;
        this.code = code;
    }
}

export class NetworkError extends Error {
    public statusCode?: number;
    public code: string;

    constructor(
        message: string,
        statusCode?: number,
        code: string = 'NETWORK_ERROR'
    ) {
        super(message);
        this.name = 'NetworkError';
        this.statusCode = statusCode;
        this.code = code;
    }
}

export class SecurityError extends Error {
    public code: string;

    constructor(
        message: string,
        code: string = 'SECURITY_ERROR'
    ) {
        super(message);
        this.name = 'SecurityError';
        this.code = code;
    }
}

// Input validation utilities
const validateInputImpl = {
    email: (email: string): boolean => {
        if (!email || typeof email !== 'string') return false;
        const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
        return emailRegex.test(email) && email.length <= 254;
    },

    url: (url: string): boolean => {
        if (!url || typeof url !== 'string') return false;
        try {
            const urlObj = new URL(url);
            return ['http:', 'https:'].includes(urlObj.protocol);
        } catch {
            return false;
        }
    },

    apiKey: (key: string): boolean => {
        if (!key || typeof key !== 'string') return false;
        // API key should be at least 16 chars and contain only safe characters
        return key.length >= 16 && key.length <= 128 && /^[a-zA-Z0-9_-]+$/.test(key);
    },

    port: (port: string | number): boolean => {
        const portNum = typeof port === 'string' ? parseInt(port, 10) : port;
        return !isNaN(portNum) && portNum > 0 && portNum <= 65535;
    },

    username: (username: string): boolean => {
        if (!username || typeof username !== 'string') return false;
        // Username: 3-50 chars, alphanumeric + underscore/dash
        return /^[a-zA-Z0-9_-]{3,50}$/.test(username);
    },

    password: (password: string): boolean => {
        if (!password || typeof password !== 'string') return false;
        // Password: at least 8 chars, with complexity requirements
        return password.length >= 8 &&
            password.length <= 128 &&
            /(?=.*[a-z])/.test(password) && // lowercase
            /(?=.*[A-Z])/.test(password) && // uppercase  
            /(?=.*\d)/.test(password); // digit
    },

    sanitizeString: (input: string): string => {
        if (typeof input !== 'string') return '';
        return securityUtilsImpl.sanitizeInput(input);
    },

    validateLength: (input: string, min: number = 0, max: number = 1000): boolean => {
        return typeof input === 'string' && input.length >= min && input.length <= max;
    },    // Validate against common injection patterns with enhanced detection
    isSecureInput: (input: string): boolean => {
        if (typeof input !== 'string') return false;

        const dangerousPatterns = [
            /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
            /javascript\s*:/gi,
            /vbscript\s*:/gi,
            /on\w+\s*=/gi,
            /data\s*:/gi,
            /eval\s*\(/gi,
            /expression\s*\(/gi,
            /import\s*\(/gi,
            /require\s*\(/gi,
            /__proto__/gi,
            /constructor/gi,
            /prototype/gi,
            /function\s*\(/gi,
            /=\s*\/.*\/[gimuy]*/gi, // Regex patterns
            /\${.*}/gi, // Template literals
            /<iframe/gi,
            /<object/gi,
            /<embed/gi,
            /<link/gi,
            /<meta/gi,
            /<style/gi,
            /<!--.*-->/gi,
            /\/\*.*\*\//gi,
        ];

        return !dangerousPatterns.some(pattern => pattern.test(input));
    },

    // Enhanced XSS detection
    hasXSSPattern: (input: string): boolean => {
        if (typeof input !== 'string') return false;

        const xssPatterns = [
            /&lt;script/gi,
            /&gt;/gi,
            /&quot;/gi,
            /&#x27;/gi,
            /&#x2F;/gi,
            /&amp;/gi,
            /%3Cscript/gi,
            /%3C%2Fscript%3E/gi,
            /\+ADw-script/gi,
            /\+ADw-/gi,
        ];

        return xssPatterns.some(pattern => pattern.test(input));
    },

    // SQL injection detection
    hasSQLInjection: (input: string): boolean => {
        if (typeof input !== 'string') return false;

        const sqlPatterns = [
            /(\b(select|insert|update|delete|drop|create|alter|exec|execute|union|declare)\b)/gi,
            /(--|\/\*|\*\/|;|'|"|`)/gi,
            /(\bor\b|\band\b).*?=/gi,
            /1\s*=\s*1/gi,
            /'[^']*'/gi,
            /"\s*or\s*"/gi,
            /'\s*or\s*'/gi,
        ];

        return sqlPatterns.some(pattern => pattern.test(input));
    }
};

export { validateInputImpl as validateInput };

// Secure data handling
const secureStorageImpl = {
    setItem: (key: string, value: any): void => {
        try {
            // Simple encoding for non-sensitive data
            // In production, use proper encryption for sensitive data
            const data = typeof value === 'string' ? value : JSON.stringify(value);

            // Add timestamp and basic integrity check
            const payload = {
                data,
                timestamp: Date.now(),
                checksum: btoa(data).slice(0, 8) // Simple integrity check
            };

            const encoded = btoa(JSON.stringify(payload));
            localStorage.setItem(`sec_${key}`, encoded);
        } catch (error) {
            console.error('Failed to store data securely:', error);
        }
    },

    getItem: (key: string): any => {
        try {
            const encoded = localStorage.getItem(`sec_${key}`);
            if (!encoded) return null;

            const payload = JSON.parse(atob(encoded));

            // Check timestamp - expire after 24 hours
            if (payload.timestamp && (Date.now() - payload.timestamp > 24 * 60 * 60 * 1000)) {
                localStorage.removeItem(`sec_${key}`);
                return null;
            }

            // Verify basic integrity
            const expectedChecksum = btoa(payload.data).slice(0, 8);
            if (payload.checksum !== expectedChecksum) {
                console.warn('Data integrity check failed for key:', key);
                localStorage.removeItem(`sec_${key}`);
                return null;
            }

            try {
                return JSON.parse(payload.data);
            } catch {
                return payload.data;
            }
        } catch (error) {
            console.error('Failed to retrieve data securely:', error);
            return null;
        }
    },

    removeItem: (key: string): void => {
        localStorage.removeItem(`sec_${key}`);
    },

    clear: (): void => {
        const keys = Object.keys(localStorage);
        keys.forEach(key => {
            if (key.startsWith('sec_')) {
                localStorage.removeItem(key);
            }
        });
    }
};

export { secureStorageImpl as secureStorage };

// Error boundary hook
export const useErrorBoundary = () => {
    const [error, setError] = useState<Error | null>(null);

    const resetError = () => setError(null);

    const captureError = (error: Error) => {
        console.error('Error captured:', error);
        setError(error);
    };

    useEffect(() => {
        const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
            console.error('Unhandled promise rejection:', event.reason);
            captureError(new Error(event.reason));
        };

        const handleError = (event: ErrorEvent) => {
            console.error('Global error:', event.error);
            captureError(event.error);
        };

        window.addEventListener('unhandledrejection', handleUnhandledRejection);
        window.addEventListener('error', handleError);

        return () => {
            window.removeEventListener('unhandledrejection', handleUnhandledRejection);
            window.removeEventListener('error', handleError);
        };
    }, []);

    return { error, resetError, captureError };
};

// API error handling hook
export const useApiError = () => {
    const [errorState, setErrorState] = useState<ErrorState>({
        error: null,
        isLoading: false,
        retryCount: 0
    });

    const setError = (error: APIError | Error | string) => {
        let apiError: APIError;

        if (typeof error === 'string') {
            apiError = {
                code: 'UNKNOWN_ERROR',
                message: error
            };
        } else if (error instanceof Error) {
            apiError = {
                code: error.name || 'UNKNOWN_ERROR',
                message: error.message
            };
        } else {
            apiError = error;
        }

        setErrorState(prev => ({
            ...prev,
            error: apiError,
            isLoading: false
        }));
    };

    const clearError = () => {
        setErrorState(prev => ({
            ...prev,
            error: null,
            retryCount: 0
        }));
    };

    const setLoading = (loading: boolean) => {
        setErrorState(prev => ({
            ...prev,
            isLoading: loading
        }));
    };

    const retry = (retryFn: () => Promise<any>, maxRetries: number = 3) => {
        if (errorState.retryCount >= maxRetries) {
            setError('Maximum retry attempts exceeded');
            return;
        }

        setErrorState(prev => ({
            ...prev,
            isLoading: true,
            retryCount: prev.retryCount + 1
        }));

        retryFn()
            .then(() => {
                clearError();
            })
            .catch((error) => {
                setError(error);
            });
    };

    return {
        ...errorState,
        setError,
        clearError,
        setLoading,
        retry
    };
};

// Content Security Policy utilities
const cspUtilsImpl = {
    generateNonce: (): string => {
        const array = new Uint8Array(16);
        crypto.getRandomValues(array);
        return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    },

    sanitizeHTML: (html: string): string => {
        const temp = document.createElement('div');
        temp.textContent = html;
        return temp.innerHTML;
    },

    validateOrigin: (url: string, allowedOrigins: string[]): boolean => {
        try {
            const urlObj = new URL(url);
            return allowedOrigins.some(origin => {
                if (origin === '*') return true;
                if (origin.startsWith('*.')) {
                    const domain = origin.slice(2);
                    return urlObj.hostname.endsWith(domain);
                }
                return urlObj.origin === origin;
            });
        } catch {
            return false;
        }
    }
};

export { cspUtilsImpl as cspUtils };

// Form validation utilities
const formValidationImpl = {
    validateRequired: (value: any, fieldName: string): ValidationError | null => {
        if (value === null || value === undefined || value === '') {
            return new ValidationError(`${fieldName} is required`, fieldName);
        }
        return null;
    },

    validateEmail: (email: string, fieldName: string = 'email'): ValidationError | null => {
        if (!validateInputImpl.email(email)) {
            return new ValidationError('Invalid email format', fieldName);
        }
        return null;
    },

    validateURL: (url: string, fieldName: string = 'url'): ValidationError | null => {
        if (!validateInputImpl.url(url)) {
            return new ValidationError('Invalid URL format', fieldName);
        }
        return null;
    },

    validateLength: (
        value: string,
        min: number,
        max: number,
        fieldName: string
    ): ValidationError | null => {
        if (!validateInputImpl.validateLength(value, min, max)) {
            return new ValidationError(
                `${fieldName} must be between ${min} and ${max} characters`,
                fieldName
            );
        }
        return null;
    },

    validateForm: (
        data: Record<string, any>,
        rules: Record<string, Array<(value: any, fieldName: string) => ValidationError | null>>
    ): ValidationError[] => {
        const errors: ValidationError[] = [];

        Object.entries(rules).forEach(([fieldName, validators]) => {
            const value = data[fieldName];

            validators.forEach(validator => {
                const error = validator(value, fieldName);
                if (error) {
                    errors.push(error);
                }
            });
        });

        return errors;
    }
};

export { formValidationImpl as formValidation };

// Security headers validation
export const validateSecurityHeaders = (response: Response): void => {
    const requiredHeaders = [
        'x-content-type-options',
        'x-frame-options',
        'x-xss-protection'
    ];

    const missingHeaders = requiredHeaders.filter(
        header => !response.headers.get(header)
    );

    if (missingHeaders.length > 0) {
        console.warn('Missing security headers:', missingHeaders);
    }
};

// Audit logging
const auditLogImpl = {
    logSecurityEvent: (event: string, details: Record<string, any> = {}) => {
        const logEntry = {
            timestamp: new Date().toISOString(),
            event,
            details,
            userAgent: navigator.userAgent,
            url: window.location.href
        };

        console.warn('Security Event:', logEntry);

        // In a real application, send this to your logging service
        // fetch('/api/audit/security', {
        //   method: 'POST',
        //   headers: { 'Content-Type': 'application/json' },
        //   body: JSON.stringify(logEntry)
        // });
    },

    logError: (error: Error, context: Record<string, any> = {}) => {
        const logEntry = {
            timestamp: new Date().toISOString(),
            error: {
                name: error.name,
                message: error.message,
                stack: error.stack
            },
            context,
            userAgent: navigator.userAgent,
            url: window.location.href
        };

        console.error('Error logged:', logEntry);

        // In a real application, send this to your error tracking service
    }
};

export { auditLogImpl as auditLog };

// Enhanced input sanitization and security utilities
const securityUtilsImpl = {
    // XSS prevention - sanitize HTML input
    sanitizeHTML: (input: string): string => {
        const div = document.createElement('div');
        div.textContent = input;
        return div.innerHTML;
    },

    // Remove script tags and javascript protocols
    removeScriptTags: (input: string): string => {
        return input
            .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
            .replace(/javascript:/gi, '')
            .replace(/vbscript:/gi, '')
            .replace(/on\w+\s*=/gi, '');
    },

    // Comprehensive input sanitization
    sanitizeInput: (input: string): string => {
        if (typeof input !== 'string') {
            return '';
        }

        return input
            .trim()
            .replace(/[<>]/g, '') // Remove angle brackets
            .replace(/&/g, '&amp;') // Escape ampersands
            .replace(/"/g, '&quot;') // Escape quotes
            .replace(/'/g, '&#x27;') // Escape single quotes
            .replace(/\//g, '&#x2F;') // Escape forward slashes
            .replace(/\0/g, '') // Remove null bytes
            .replace(/[\x00-\x1F\x7F]/g, ''); // Remove control characters
    },

    // Validate and sanitize URL
    sanitizeURL: (url: string): string | null => {
        try {
            const urlObj = new URL(url);
            // Only allow http and https protocols
            if (!['http:', 'https:'].includes(urlObj.protocol)) {
                return null;
            }
            return urlObj.toString();
        } catch {
            return null;
        }
    },

    // Generate secure random strings for CSRF tokens
    generateSecureToken: (length: number = 32): string => {
        const array = new Uint8Array(length);
        crypto.getRandomValues(array);
        return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    },

    // Validate file upload security
    validateFileUpload: (file: File, allowedTypes: string[], maxSize: number): boolean => {
        // Check file type
        if (!allowedTypes.includes(file.type)) {
            return false;
        }

        // Check file size
        if (file.size > maxSize) {
            return false;
        }

        // Check file extension
        const extension = file.name.split('.').pop()?.toLowerCase();
        const allowedExtensions = allowedTypes.map(type => type.split('/')[1]);

        return extension ? allowedExtensions.includes(extension) : false;
    },

    // Rate limiting for client-side
    createRateLimiter: (maxRequests: number, windowMs: number) => {
        const requests: number[] = [];

        return (): boolean => {
            const now = Date.now();
            const windowStart = now - windowMs;

            // Remove old requests
            while (requests.length > 0 && requests[0] < windowStart) {
                requests.shift();
            }

            // Check if under limit
            if (requests.length >= maxRequests) {
                return false;
            }

            // Add current request
            requests.push(now);
            return true;
        };
    }
};

export { securityUtilsImpl as securityUtils };
