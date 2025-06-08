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
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    },

    url: (url: string): boolean => {
        try {
            new URL(url);
            return true;
        } catch {
            return false;
        }
    },

    apiKey: (key: string): boolean => {
        // Basic API key validation
        return key.length >= 10 && /^[a-zA-Z0-9_-]+$/.test(key);
    },

    port: (port: string | number): boolean => {
        const portNum = typeof port === 'string' ? parseInt(port, 10) : port;
        return !isNaN(portNum) && portNum > 0 && portNum <= 65535;
    },

    sanitizeString: (input: string): string => {
        // Remove potentially dangerous characters
        return input
            .replace(/[<>\"']/g, '') // Remove HTML/script injection chars
            .replace(/\0/g, '') // Remove null bytes
            .trim();
    },

    validateLength: (input: string, min: number = 0, max: number = 1000): boolean => {
        return input.length >= min && input.length <= max;
    }
};

export { validateInputImpl as validateInput };

// Secure data handling
const secureStorageImpl = {
    setItem: (key: string, value: any): void => {
        try {
            // Encrypt sensitive data before storing (in a real app, use proper encryption)
            const data = typeof value === 'string' ? value : JSON.stringify(value);
            const encoded = btoa(data); // Base64 encoding (not secure, just obfuscation)
            localStorage.setItem(`sec_${key}`, encoded);
        } catch (error) {
            console.error('Failed to store data securely:', error);
        }
    },

    getItem: (key: string): any => {
        try {
            const encoded = localStorage.getItem(`sec_${key}`);
            if (!encoded) return null;

            const decoded = atob(encoded);
            try {
                return JSON.parse(decoded);
            } catch {
                return decoded;
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

// Rate limiting hook
export const useRateLimit = (maxRequests: number = 10, windowMs: number = 60000) => {
    const [requests, setRequests] = useState<number[]>([]);

    const canMakeRequest = (): boolean => {
        const now = Date.now();
        const windowStart = now - windowMs;

        // Filter out old requests
        const recentRequests = requests.filter(timestamp => timestamp > windowStart);

        return recentRequests.length < maxRequests;
    };

    const recordRequest = (): boolean => {
        if (!canMakeRequest()) {
            return false;
        }

        const now = Date.now();
        setRequests(prev => {
            const windowStart = now - windowMs;
            const recentRequests = prev.filter(timestamp => timestamp > windowStart);
            return [...recentRequests, now];
        });

        return true;
    };

    const getRemainingRequests = (): number => {
        const now = Date.now();
        const windowStart = now - windowMs;
        const recentRequests = requests.filter(timestamp => timestamp > windowStart);
        return Math.max(0, maxRequests - recentRequests.length);
    };

    const getTimeUntilReset = (): number => {
        if (requests.length === 0) return 0;

        const oldestRequest = Math.min(...requests);
        const timeUntilReset = (oldestRequest + windowMs) - Date.now();

        return Math.max(0, timeUntilReset);
    };

    return {
        canMakeRequest,
        recordRequest,
        getRemainingRequests,
        getTimeUntilReset
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
