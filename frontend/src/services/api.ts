// API service layer for communicating with the Go backend

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface ApiResponse<T = any> {
    data?: T;
    error?: {
        code: string;
        message: string;
        details?: any;
    };
    success: boolean;
}

class ApiService {
    private async request<T = any>(
        endpoint: string,
        options: RequestInit = {}
    ): Promise<ApiResponse<T>> {
        try {
            const url = `${API_BASE_URL}${endpoint}`;
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers,
                },
                ...options,
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                return {
                    success: false,
                    error: {
                        code: errorData.error?.code || 'HTTP_ERROR',
                        message: errorData.error?.message || `HTTP ${response.status}`,
                        details: errorData.error?.details,
                    },
                };
            }

            const data = await response.json();
            return {
                success: true,
                data,
            };
        } catch (error) {
            return {
                success: false,
                error: {
                    code: 'NETWORK_ERROR',
                    message: error instanceof Error ? error.message : 'Network error occurred',
                },
            };
        }
    }

    // Health check
    async healthCheck() {
        return this.request('/health');
    }

    // Local models API
    async getLocalModels() {
        return this.request('/api/local/models');
    }

    async startLocalModel(modelId: string) {
        return this.request(`/api/local/models/${modelId}/start`, {
            method: 'POST',
        });
    }

    async stopLocalModel(modelId: string) {
        return this.request(`/api/local/models/${modelId}/stop`, {
            method: 'POST',
        });
    }

    async downloadModel(modelId: string) {
        return this.request(`/api/local/models/${modelId}/download`, {
            method: 'POST',
        });
    }

    async getModelStatus(modelId: string) {
        return this.request(`/api/local/models/${modelId}/status`);
    }

    async updateLocalModelSettings(modelId: string, settings: any) {
        return this.request(`/api/local/models/${modelId}/settings`, {
            method: 'PUT',
            body: JSON.stringify(settings),
        });
    }

    // Chat completions
    async chatCompletions(payload: any) {
        return this.request('/v1/chat/completions', {
            method: 'POST',
            body: JSON.stringify(payload),
        });
    }

    async localChatCompletions(payload: any) {
        return this.request('/local/chat/completions', {
            method: 'POST',
            body: JSON.stringify(payload),
        });
    }

    // Models endpoint
    async getModels() {
        return this.request('/v1/models');
    }

    async getLocalModelsList() {
        return this.request('/local/models');
    }

    // Authentication - updated to use standardized paths
    async login(username: string, password: string) {
        return this.request('/api/v1/auth/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });
    }

    async createApiKey(name: string, permissions: any = {}) {
        return this.request('/api/v1/admin/api-keys', {
            method: 'POST',
            body: JSON.stringify({ name, permissions }),
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('auth_token')}`,
            },
        });
    }

    async getApiKeys() {
        return this.request('/api/v1/admin/api-keys', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('auth_token')}`,
            },
        });
    }

    // Monitoring
    async getMetrics() {
        return this.request('/api/v1/monitoring/metrics');
    }

    async getDashboardStats() {
        return this.request('/api/v1/monitoring/dashboard/stats');
    }

    async getSystemStatus() {
        return this.request('/api/v1/monitoring/system/status');
    }

    // Service Management
    async getServices() {
        return this.request('/api/v1/monitoring/services');
    }

    async getServiceHealth(serviceId: string) {
        return this.request(`/api/v1/monitoring/services/${serviceId}/health`);
    }

    async refreshService(serviceId: string) {
        return this.request(`/api/v1/monitoring/services/${serviceId}/refresh`, {
            method: 'POST',
        });
    }

    // Service Sources Management
    async getServiceSources() {
        return this.request('/api/v1/service-sources');
    }

    async createServiceSource(source: any) {
        return this.request('/api/v1/service-sources', {
            method: 'POST',
            body: JSON.stringify(source),
        });
    }

    async updateServiceSource(sourceId: string, source: any) {
        return this.request(`/api/v1/service-sources/${sourceId}`, {
            method: 'PUT',
            body: JSON.stringify(source),
        });
    }

    async deleteServiceSource(sourceId: string) {
        return this.request(`/api/v1/service-sources/${sourceId}`, {
            method: 'DELETE',
        });
    }

    async toggleServiceSourceStatus(sourceId: string) {
        return this.request(`/api/v1/service-sources/${sourceId}/toggle`, {
            method: 'POST',
        });
    }

    // Route management methods
    async getRoutes() {
        return this.request('/api/v1/routes');
    }

    async createRoute(route: any) {
        return this.request('/api/v1/routes', {
            method: 'POST',
            body: JSON.stringify(route),
        });
    }

    async updateRoute(routeId: string, route: any) {
        return this.request(`/api/v1/routes/${routeId}`, {
            method: 'PUT',
            body: JSON.stringify(route),
        });
    }

    async deleteRoute(routeId: string) {
        return this.request(`/api/v1/routes/${routeId}`, {
            method: 'DELETE',
        });
    }

    async toggleRouteStatus(routeId: string) {
        return this.request(`/api/v1/routes/${routeId}/toggle`, {
            method: 'POST',
        });
    }

    // Certificate management methods
    async getCertificates() {
        return this.request('/api/v1/certificates');
    }

    async createCertificate(certificate: any) {
        return this.request('/api/v1/certificates', {
            method: 'POST',
            body: JSON.stringify(certificate),
        });
    }

    async updateCertificate(certificateId: string, certificate: any) {
        return this.request(`/api/v1/certificates/${certificateId}`, {
            method: 'PUT',
            body: JSON.stringify(certificate),
        });
    }

    async deleteCertificate(certificateId: string) {
        return this.request(`/api/v1/certificates/${certificateId}`, {
            method: 'DELETE',
        });
    }

    async renewCertificate(certificateId: string) {
        return this.request(`/api/v1/certificates/${certificateId}/renew`, {
            method: 'POST',
        });
    }

    async toggleCertificateAutoRenew(certificateId: string) {
        return this.request(`/api/v1/certificates/${certificateId}/auto-renew`, {
            method: 'POST',
        });
    }

    // Domain management methods
    async getDomains() {
        return this.request('/api/v1/domains');
    }

    async createDomain(domain: any) {
        return this.request('/api/v1/domains', {
            method: 'POST',
            body: JSON.stringify(domain),
        });
    }

    async updateDomain(domainId: string, domain: any) {
        return this.request(`/api/v1/domains/${domainId}`, {
            method: 'PUT',
            body: JSON.stringify(domain),
        });
    }

    async deleteDomain(domainId: string) {
        return this.request(`/api/v1/domains/${domainId}`, {
            method: 'DELETE',
        });
    }

    async toggleDomainSSL(domainId: string) {
        return this.request(`/api/v1/domains/${domainId}/ssl`, {
            method: 'POST',
        });
    }

    async renewDomainCertificate(domainId: string) {
        return this.request(`/api/v1/domains/${domainId}/renew-certificate`, {
            method: 'POST',
        });
    }

    // Cloud services management (standardized paths)
    async getCloudServices() {
        return this.request('/api/v1/cloud/services');
    }

    async getCloudServiceHealth(serviceName: string) {
        return this.request(`/api/v1/cloud/services/${serviceName}/health`);
    }

    async scaleCloudService(serviceName: string, replicas: number) {
        return this.request(`/api/v1/cloud/services/${serviceName}/scale`, {
            method: 'POST',
            body: JSON.stringify({ replicas }),
        });
    }

    async getCloudServiceMetrics(serviceName: string, start?: string, end?: string) {
        const params = new URLSearchParams();
        if (start) params.append('start', start);
        if (end) params.append('end', end);

        return this.request(`/api/v1/cloud/services/${serviceName}/metrics?${params.toString()}`);
    }

    async getCloudServiceLogs(serviceName: string, start?: string, end?: string) {
        const params = new URLSearchParams();
        if (start) params.append('start', start);
        if (end) params.append('end', end);

        return this.request(`/api/v1/cloud/services/${serviceName}/logs?${params.toString()}`);
    }

    async updateCloudServiceConfig(serviceName: string, config: any) {
        return this.request(`/api/v1/cloud/services/${serviceName}/config`, {
            method: 'PUT',
            body: JSON.stringify(config),
        });
    }
}

export const apiService = new ApiService();
export type { ApiResponse };
