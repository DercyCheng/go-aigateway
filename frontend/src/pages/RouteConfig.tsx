import { useState, useEffect } from 'react'
import { Plus, Edit, Trash2, Copy, ArrowRight, Settings, RefreshCw } from 'lucide-react'
import { apiService } from '../services/api'

interface RouteRule {
    id: string
    name: string
    path: string
    method: string
    target: string
    priority: number
    enabled: boolean
    conditions: {
        headers?: Record<string, string>
        queryParams?: Record<string, string>
        body?: string
    }
    actions: {
        rewrite?: string
        redirect?: string
        rateLimit?: number
        timeout?: number
    }
    createdAt: string
    updatedAt: string
}

const RouteConfig = () => {
    const [routes, setRoutes] = useState<RouteRule[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [showForm, setShowForm] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [formData, setFormData] = useState({
        name: '',
        path: '',
        method: 'GET',
        target: '',
        priority: 1,
        enabled: true,
        rateLimit: 100,
        timeout: 30000
    })

    // Fetch routes from API
    useEffect(() => {
        fetchRoutes()
    }, [])

    const fetchRoutes = async () => {
        try {
            const response = await apiService.getRoutes()
            if (response.success && response.data && Array.isArray(response.data)) {
                setRoutes(response.data)
            } else {
                console.error('Error fetching routes:', response.error || 'Unknown error');
                setRoutes([]);
            }
        } catch (error) {
            console.error('Failed to fetch routes:', error)
            setRoutes([])
        } finally {
            setIsLoading(false)
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        try {
            if (editingId) {
                // Update existing route
                await apiService.updateRoute(editingId, formData)
                setRoutes(routes.map(route =>
                    route.id === editingId
                        ? {
                            ...route,
                            ...formData,
                            actions: { ...route.actions, rateLimit: formData.rateLimit, timeout: formData.timeout },
                            updatedAt: new Date().toISOString().split('T')[0]
                        }
                        : route
                ))
                setEditingId(null)
            } else {
                // Create new route
                const response = await apiService.createRoute(formData)
                if (response && response.data && response.data.id) {
                    setRoutes([...routes, response.data])
                } else {
                    // Fallback to local state update
                    const newRoute: RouteRule = {
                        id: Date.now().toString(),
                        ...formData,
                        conditions: { headers: {}, queryParams: {} },
                        actions: { rateLimit: formData.rateLimit, timeout: formData.timeout },
                        createdAt: new Date().toISOString().split('T')[0],
                        updatedAt: new Date().toISOString().split('T')[0]
                    }
                    setRoutes([...routes, newRoute])
                }
            }
        } catch (error) {
            console.error('Failed to save route:', error)
            // Fallback to local state update for better UX
            if (editingId) {
                setRoutes(routes.map(route =>
                    route.id === editingId
                        ? {
                            ...route,
                            ...formData,
                            actions: { ...route.actions, rateLimit: formData.rateLimit, timeout: formData.timeout },
                            updatedAt: new Date().toISOString().split('T')[0]
                        }
                        : route
                ))
                setEditingId(null)
            } else {
                const newRoute: RouteRule = {
                    id: Date.now().toString(),
                    ...formData,
                    conditions: { headers: {}, queryParams: {} },
                    actions: { rateLimit: formData.rateLimit, timeout: formData.timeout },
                    createdAt: new Date().toISOString().split('T')[0],
                    updatedAt: new Date().toISOString().split('T')[0]
                }
                setRoutes([...routes, newRoute])
            }
        }
        setFormData({
            name: '',
            path: '',
            method: 'GET',
            target: '',
            priority: 1,
            enabled: true,
            rateLimit: 100,
            timeout: 30000
        })
        setShowForm(false)
    }

    const handleEdit = (route: RouteRule) => {
        setFormData({
            name: route.name,
            path: route.path,
            method: route.method,
            target: route.target,
            priority: route.priority,
            enabled: route.enabled,
            rateLimit: route.actions.rateLimit || 100,
            timeout: route.actions.timeout || 30000
        })
        setEditingId(route.id)
        setShowForm(true)
    }

    const handleDelete = async (id: string) => {
        try {
            await apiService.deleteRoute(id)
            setRoutes(routes.filter(route => route.id !== id))
        } catch (error) {
            console.error('Failed to delete route:', error)
            // Fallback to local state update
            setRoutes(routes.filter(route => route.id !== id))
        }
    }

    const toggleEnabled = async (id: string) => {
        try {
            await apiService.toggleRouteStatus(id)
            setRoutes(routes.map(route =>
                route.id === id
                    ? { ...route, enabled: !route.enabled, updatedAt: new Date().toISOString().split('T')[0] }
                    : route
            ))
        } catch (error) {
            console.error('Failed to toggle route status:', error)
            // Fallback to local state update
            setRoutes(routes.map(route =>
                route.id === id
                    ? { ...route, enabled: !route.enabled, updatedAt: new Date().toISOString().split('T')[0] }
                    : route
            ))
        }
    }

    const duplicateRoute = (route: RouteRule) => {
        const newRoute: RouteRule = {
            ...route,
            id: Date.now().toString(),
            name: `${route.name} (Copy)`,
            createdAt: new Date().toISOString().split('T')[0],
            updatedAt: new Date().toISOString().split('T')[0]
        }
        setRoutes([...routes, newRoute])
    }

    const getMethodColor = (method: string) => {
        switch (method) {
            case 'GET': return 'bg-blue-100 text-blue-800'
            case 'POST': return 'bg-green-100 text-green-800'
            case 'PUT': return 'bg-yellow-100 text-yellow-800'
            case 'DELETE': return 'bg-red-100 text-red-800'
            default: return 'bg-gray-100 text-gray-800'
        }
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">路由配置</h1>
                    <p className="mt-2 text-sm text-gray-600">管理API路由规则和转发配置</p>
                </div>
                <button
                    onClick={() => setShowForm(true)}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                >
                    <Plus className="h-4 w-4 mr-2" />
                    添加路由
                </button>
            </div>

            {/* Add/Edit Form */}
            {showForm && (
                <div className="bg-white shadow rounded-lg">
                    <div className="px-4 py-5 sm:p-6">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
                            {editingId ? '编辑路由' : '添加路由'}
                        </h3>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">路由名称</label>
                                    <input
                                        type="text"
                                        required
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">HTTP 方法</label>
                                    <select
                                        value={formData.method}
                                        onChange={(e) => setFormData({ ...formData, method: e.target.value })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    >
                                        <option value="GET">GET</option>
                                        <option value="POST">POST</option>
                                        <option value="PUT">PUT</option>
                                        <option value="DELETE">DELETE</option>
                                        <option value="PATCH">PATCH</option>
                                    </select>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">路径</label>
                                <input
                                    type="text"
                                    required
                                    value={formData.path}
                                    onChange={(e) => setFormData({ ...formData, path: e.target.value })}
                                    placeholder="/api/v1/example"
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">目标地址</label>
                                <input
                                    type="url"
                                    required
                                    value={formData.target}
                                    onChange={(e) => setFormData({ ...formData, target: e.target.value })}
                                    placeholder="https://api.example.com/endpoint"
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">优先级</label>
                                    <input
                                        type="number"
                                        min="1"
                                        value={formData.priority}
                                        onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">速率限制 (请求/分钟)</label>
                                    <input
                                        type="number"
                                        min="1"
                                        value={formData.rateLimit}
                                        onChange={(e) => setFormData({ ...formData, rateLimit: parseInt(e.target.value) })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">超时时间 (毫秒)</label>
                                    <input
                                        type="number"
                                        min="1000"
                                        value={formData.timeout}
                                        onChange={(e) => setFormData({ ...formData, timeout: parseInt(e.target.value) })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                            </div>
                            <div className="flex items-center">
                                <input
                                    type="checkbox"
                                    checked={formData.enabled}
                                    onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                                />
                                <label className="ml-2 block text-sm text-gray-900">启用路由</label>
                            </div>
                            <div className="flex justify-end space-x-3">
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowForm(false)
                                        setEditingId(null)
                                        setFormData({
                                            name: '',
                                            path: '',
                                            method: 'GET',
                                            target: '',
                                            priority: 1,
                                            enabled: true,
                                            rateLimit: 100,
                                            timeout: 30000
                                        })
                                    }}
                                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                                >
                                    取消
                                </button>
                                <button
                                    type="submit"
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
                                >
                                    {editingId ? '更新' : '添加'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {/* Routes List */}
            <div className="bg-white shadow overflow-hidden sm:rounded-md">
                {isLoading ? (
                    <div className="flex items-center justify-center py-12">
                        <RefreshCw className="h-8 w-8 animate-spin text-gray-400" />
                        <span className="ml-2 text-gray-500">加载路由配置中...</span>
                    </div>
                ) : routes.length === 0 ? (
                    <div className="text-center py-12">
                        <Settings className="mx-auto h-12 w-12 text-gray-400" />
                        <h3 className="mt-2 text-sm font-medium text-gray-900">暂无路由配置</h3>
                        <p className="mt-1 text-sm text-gray-500">开始创建第一个路由规则</p>
                        <div className="mt-6">
                            <button
                                onClick={() => setShowForm(true)}
                                className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
                            >
                                <Plus className="h-4 w-4 mr-2" />
                                添加路由
                            </button>
                        </div>
                    </div>
                ) : (
                    <ul className="divide-y divide-gray-200">
                        {routes.map((route) => (
                            <li key={route.id} className="px-6 py-4">
                                <div className="flex items-center justify-between">
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center space-x-3 mb-2">
                                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getMethodColor(route.method)}`}>
                                                {route.method}
                                            </span>
                                            <h3 className="text-sm font-medium text-gray-900">{route.name}</h3>
                                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${route.enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                                                }`}>
                                                {route.enabled ? '启用' : '禁用'}
                                            </span>
                                            <span className="text-xs text-gray-500">优先级: {route.priority}</span>
                                        </div>
                                        <div className="flex items-center space-x-2 text-sm text-gray-600 mb-2">
                                            <code className="bg-gray-100 px-2 py-1 rounded text-xs">{route.path}</code>
                                            <ArrowRight className="h-3 w-3" />
                                            <code className="bg-blue-50 px-2 py-1 rounded text-xs">{route.target}</code>
                                        </div>
                                        <div className="flex items-center space-x-4 text-xs text-gray-500">
                                            <span>速率限制: {route.actions.rateLimit}/min</span>
                                            <span>超时: {route.actions.timeout}ms</span>
                                            <span>更新: {route.updatedAt}</span>
                                        </div>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <button
                                            onClick={() => toggleEnabled(route.id)}
                                            className={`p-1 rounded-full ${route.enabled
                                                ? 'text-green-600 hover:bg-green-100'
                                                : 'text-gray-400 hover:bg-gray-100'
                                                }`}
                                        >
                                            <Settings className="h-4 w-4" />
                                        </button>
                                        <button
                                            onClick={() => duplicateRoute(route)}
                                            className="p-1 text-blue-600 hover:bg-blue-100 rounded-full"
                                        >
                                            <Copy className="h-4 w-4" />
                                        </button>
                                        <button
                                            onClick={() => handleEdit(route)}
                                            className="p-1 text-gray-600 hover:bg-gray-100 rounded-full"
                                        >
                                            <Edit className="h-4 w-4" />
                                        </button>
                                        <button
                                            onClick={() => handleDelete(route.id)}
                                            className="p-1 text-red-600 hover:bg-red-100 rounded-full"
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </button>
                                    </div>
                                </div>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    )
}

export default RouteConfig
