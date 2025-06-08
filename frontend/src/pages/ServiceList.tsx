import { useState, useEffect } from 'react'
import { Search, Filter, RefreshCw, Activity, Clock, TrendingUp, TrendingDown } from 'lucide-react'
import { apiService } from '../services/api'

interface Service {
    id: string
    name: string
    model: string
    provider: string
    status: 'healthy' | 'degraded' | 'down'
    requests: number
    avgResponseTime: number
    successRate: number
    lastCheck: string
    endpoint: string
    rateLimit: number
    description: string
}

const ServiceList = () => {
    const [services, setServices] = useState<Service[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [searchTerm, setSearchTerm] = useState('')
    const [statusFilter, setStatusFilter] = useState<string>('all')
    const [providerFilter, setProviderFilter] = useState<string>('all')

    // Fetch services from API
    useEffect(() => {
        fetchServices()
    }, [])

    const fetchServices = async () => {
        try {
            setIsLoading(true)
            const response = await apiService.getServices()
            if (response.success && response.data?.services) {
                setServices(response.data.services)
            } else {
                // Fallback to mock data if API fails
                setServices([
                    {
                        id: '1',
                        name: 'GPT-4 Turbo',
                        model: 'gpt-4-turbo-preview',
                        provider: 'OpenAI',
                        status: 'healthy',
                        requests: 1420,
                        avgResponseTime: 850,
                        successRate: 99.2,
                        lastCheck: '2024-01-20 14:30:00',
                        endpoint: '/v1/chat/completions',
                        rateLimit: 10000,
                        description: 'Latest GPT-4 Turbo model with enhanced capabilities'
                    },
                    {
                        id: '2',
                        name: 'Claude-3 Opus',
                        model: 'claude-3-opus-20240229',
                        provider: 'Anthropic',
                        status: 'healthy',
                        requests: 856,
                        avgResponseTime: 1200,
                        successRate: 98.8,
                        lastCheck: '2024-01-20 14:29:30',
                        endpoint: '/v1/messages',
                        rateLimit: 5000,
                        description: 'Most capable Claude model for complex tasks'
                    },
                    {
                        id: '5',
                        name: 'TinyLlama Chat',
                        model: 'TinyLlama-1.1B-Chat-v1.0',
                        provider: 'Local',
                        status: 'healthy',
                        requests: 73,
                        avgResponseTime: 350,
                        successRate: 99.9,
                        lastCheck: '2024-01-20 14:32:00',
                        endpoint: '/local/chat/completions',
                        rateLimit: 9999,
                        description: '本地轻量对话模型，仅1.1B参数，速度快'
                    }
                ])
            }
        } catch (error) {
            console.error('Error fetching services:', error)
            // Show fallback mock data on error
            setServices([])
        } finally {
            setIsLoading(false)
        }
    }

    const filteredServices = services.filter(service => {
        const matchesSearch = service.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            service.model.toLowerCase().includes(searchTerm.toLowerCase()) ||
            service.provider.toLowerCase().includes(searchTerm.toLowerCase())
        const matchesStatus = statusFilter === 'all' || service.status === statusFilter
        const matchesProvider = providerFilter === 'all' || service.provider === providerFilter

        return matchesSearch && matchesStatus && matchesProvider
    })

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'healthy': return 'bg-green-100 text-green-800'
            case 'degraded': return 'bg-yellow-100 text-yellow-800'
            case 'down': return 'bg-red-100 text-red-800'
            default: return 'bg-gray-100 text-gray-800'
        }
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'healthy': return <div className="h-2 w-2 bg-green-500 rounded-full"></div>
            case 'degraded': return <div className="h-2 w-2 bg-yellow-500 rounded-full"></div>
            case 'down': return <div className="h-2 w-2 bg-red-500 rounded-full"></div>
            default: return <div className="h-2 w-2 bg-gray-500 rounded-full"></div>
        }
    }

    const refreshService = async (id: string) => {
        try {
            // Call API to refresh specific service
            await apiService.refreshService(id)
            // Update the service in local state
            setServices(services.map(service =>
                service.id === id
                    ? { ...service, lastCheck: new Date().toLocaleString('zh-CN') }
                    : service
            ))
        } catch (error) {
            console.error('Error refreshing service:', error)
            // Still update the last check time even if API fails
            setServices(services.map(service =>
                service.id === id
                    ? { ...service, lastCheck: new Date().toLocaleString('zh-CN') }
                    : service
            ))
        }
    }

    const providers = [...new Set(services.map(s => s.provider))]

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-bold text-gray-900">服务列表</h1>
                <p className="mt-2 text-sm text-gray-600">监控和管理所有AI服务的运行状态</p>
            </div>

            {/* Filters */}
            <div className="bg-white shadow rounded-lg p-4">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
                        <input
                            type="text"
                            placeholder="搜索服务..."
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            className="pl-10 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>
                    <div>
                        <select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value)}
                            className="block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="all">所有状态</option>
                            <option value="healthy">健康</option>
                            <option value="degraded">降级</option>
                            <option value="down">停机</option>
                        </select>
                    </div>
                    <div>
                        <select
                            value={providerFilter}
                            onChange={(e) => setProviderFilter(e.target.value)}
                            className="block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="all">所有提供商</option>
                            {providers.map(provider => (
                                <option key={provider} value={provider}>{provider}</option>
                            ))}
                        </select>
                    </div>                    <div className="flex justify-end">
                        <button
                            onClick={fetchServices}
                            disabled={isLoading}
                            className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50"
                        >
                            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                            刷新
                        </button>
                    </div>
                </div>
            </div>            {/* Services Grid */}
            {isLoading ? (
                <div className="text-center py-12">
                    <RefreshCw className="h-8 w-8 animate-spin mx-auto text-gray-400" />
                    <p className="mt-2 text-gray-500">加载服务中...</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                    {filteredServices.map((service) => (
                        <div key={service.id} className="bg-white shadow rounded-lg p-6">
                            <div className="flex items-start justify-between">
                                <div className="flex-1">
                                    <div className="flex items-center space-x-3 mb-2">
                                        {getStatusIcon(service.status)}
                                        <h3 className="text-lg font-medium text-gray-900">{service.name}</h3>
                                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(service.status)}`}>
                                            {service.status}
                                        </span>
                                    </div>
                                    <div className="space-y-1 text-sm text-gray-600">
                                        <p><span className="font-medium">模型:</span> {service.model}</p>
                                        <p><span className="font-medium">提供商:</span> {service.provider}</p>
                                        <p><span className="font-medium">端点:</span> {service.endpoint}</p>
                                        <p className="text-xs">{service.description}</p>
                                    </div>
                                </div>
                                <button
                                    onClick={() => refreshService(service.id)}
                                    className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-full"
                                >
                                    <RefreshCw className="h-4 w-4" />
                                </button>
                            </div>

                            {/* Metrics */}
                            <div className="mt-4 grid grid-cols-2 gap-4">
                                <div className="bg-gray-50 rounded-lg p-3">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <p className="text-xs text-gray-500">请求数</p>
                                            <p className="text-lg font-semibold text-gray-900">{service.requests.toLocaleString()}</p>
                                        </div>
                                        <Activity className="h-5 w-5 text-blue-500" />
                                    </div>
                                </div>
                                <div className="bg-gray-50 rounded-lg p-3">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <p className="text-xs text-gray-500">响应时间</p>
                                            <p className="text-lg font-semibold text-gray-900">{service.avgResponseTime}ms</p>
                                        </div>
                                        <Clock className="h-5 w-5 text-green-500" />
                                    </div>
                                </div>
                                <div className="bg-gray-50 rounded-lg p-3">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <p className="text-xs text-gray-500">成功率</p>
                                            <p className="text-lg font-semibold text-gray-900">{service.successRate}%</p>
                                        </div>
                                        {service.successRate >= 99 ?
                                            <TrendingUp className="h-5 w-5 text-green-500" /> :
                                            <TrendingDown className="h-5 w-5 text-red-500" />
                                        }
                                    </div>
                                </div>
                                <div className="bg-gray-50 rounded-lg p-3">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <p className="text-xs text-gray-500">速率限制</p>
                                            <p className="text-lg font-semibold text-gray-900">{service.rateLimit.toLocaleString()}/h</p>
                                        </div>
                                        <Filter className="h-5 w-5 text-purple-500" />
                                    </div>
                                </div>
                            </div>

                            {/* Last Check */}
                            <div className="mt-4 pt-4 border-t border-gray-200">
                                <p className="text-xs text-gray-500">
                                    最后检查: {service.lastCheck}
                                </p>
                            </div>                    </div>
                    ))}
                </div>
            )}

            {!isLoading && filteredServices.length === 0 && (
                <div className="text-center py-12">
                    <p className="text-gray-500">没有找到匹配的服务</p>
                </div>
            )}
        </div>
    )
}

export default ServiceList
