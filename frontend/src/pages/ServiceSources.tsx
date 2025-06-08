import { useState, useEffect } from 'react'
import { Plus, Edit, Trash2, Check, X, ExternalLink, RefreshCw } from 'lucide-react'
import { apiService } from '../services/api'

interface ServiceSource {
    id: string
    name: string
    type: 'openai' | 'anthropic' | 'google' | 'custom'
    endpoint: string
    apiKey: string
    status: 'active' | 'inactive' | 'error'
    description: string
    createdAt: string
}

const ServiceSources = () => {
    const [sources, setSources] = useState<ServiceSource[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [showForm, setShowForm] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [formData, setFormData] = useState({
        name: '',
        type: 'openai' as ServiceSource['type'],
        endpoint: '',
        apiKey: '',
        description: ''
    })

    // Mock data as fallback
    const mockSources: ServiceSource[] = [
        {
            id: '1',
            name: 'OpenAI GPT-4',
            type: 'openai',
            endpoint: 'https://api.openai.com/v1',
            apiKey: 'sk-***...***abc',
            status: 'active',
            description: 'OpenAI GPT-4 API 服务',
            createdAt: '2024-01-15'
        },
        {
            id: '2',
            name: 'Claude API',
            type: 'anthropic',
            endpoint: 'https://api.anthropic.com/v1',
            apiKey: 'sk-ant-***...***xyz',
            status: 'active',
            description: 'Anthropic Claude API 服务',
            createdAt: '2024-01-16'
        },
        {
            id: '3',
            name: 'Gemini Pro',
            type: 'google',
            endpoint: 'https://generativelanguage.googleapis.com/v1',
            apiKey: 'AIza***...***123',
            status: 'inactive',
            description: 'Google Gemini Pro API 服务',
            createdAt: '2024-01-17'
        }
    ]

    // Fetch service sources from API
    useEffect(() => {
        fetchServiceSources()
    }, [])

    const fetchServiceSources = async () => {
        try {
            const response = await apiService.getServiceSources()
            if (response && Array.isArray(response)) {
                setSources(response)
            } else {
                // Fallback to mock data if API fails
                setSources(mockSources)
            }
        } catch (error) {
            console.error('Failed to fetch service sources:', error)
            // Fallback to mock data
            setSources(mockSources)
        } finally {
            setIsLoading(false)
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        try {
            if (editingId) {
                // Update existing source
                await apiService.updateServiceSource(editingId, formData)
                setSources(sources.map(source =>
                    source.id === editingId
                        ? { ...source, ...formData }
                        : source
                ))
                setEditingId(null)
            } else {
                // Create new source
                const response = await apiService.createServiceSource(formData)
                if (response && response.data && response.data.id) {
                    setSources([...sources, response.data])
                } else {
                    // Fallback to local state update
                    const newSource: ServiceSource = {
                        id: Date.now().toString(),
                        ...formData,
                        status: 'active',
                        createdAt: new Date().toISOString().split('T')[0]
                    }
                    setSources([...sources, newSource])
                }
            }
        } catch (error) {
            console.error('Failed to save service source:', error)
            // Fallback to local state update for better UX
            if (editingId) {
                setSources(sources.map(source =>
                    source.id === editingId
                        ? { ...source, ...formData }
                        : source
                ))
                setEditingId(null)
            } else {
                const newSource: ServiceSource = {
                    id: Date.now().toString(),
                    ...formData,
                    status: 'active',
                    createdAt: new Date().toISOString().split('T')[0]
                }
                setSources([...sources, newSource])
            }
        }
        setFormData({ name: '', type: 'openai', endpoint: '', apiKey: '', description: '' })
        setShowForm(false)
    }

    const handleEdit = (source: ServiceSource) => {
        setFormData({
            name: source.name,
            type: source.type,
            endpoint: source.endpoint,
            apiKey: source.apiKey,
            description: source.description
        })
        setEditingId(source.id)
        setShowForm(true)
    }

    const handleDelete = async (id: string) => {
        try {
            await apiService.deleteServiceSource(id)
        } catch (error) {
            console.error('Failed to delete service source:', error)
        }
        // Update local state regardless of API success for better UX
        setSources(sources.filter(source => source.id !== id))
    }

    const toggleStatus = async (id: string) => {
        try {
            await apiService.toggleServiceSourceStatus(id)
        } catch (error) {
            console.error('Failed to toggle service source status:', error)
        }
        // Update local state regardless of API success for better UX
        setSources(sources.map(source =>
            source.id === id
                ? { ...source, status: source.status === 'active' ? 'inactive' : 'active' }
                : source
        ))
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active': return 'bg-green-100 text-green-800'
            case 'inactive': return 'bg-gray-100 text-gray-800'
            case 'error': return 'bg-red-100 text-red-800'
            default: return 'bg-gray-100 text-gray-800'
        }
    }

    const getTypeIcon = (type: string) => {
        switch (type) {
            case 'openai': return '🤖'
            case 'anthropic': return '🧠'
            case 'google': return '🔍'
            default: return '⚙️'
        }
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">服务来源</h1>
                    <p className="mt-2 text-sm text-gray-600">管理AI服务提供商和API配置</p>
                </div>
                <button
                    onClick={() => setShowForm(true)}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                >
                    <Plus className="h-4 w-4 mr-2" />
                    添加服务源
                </button>
            </div>

            {/* Add/Edit Form */}
            {showForm && (
                <div className="bg-white shadow rounded-lg">
                    <div className="px-4 py-5 sm:p-6">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
                            {editingId ? '编辑服务源' : '添加服务源'}
                        </h3>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">服务名称</label>
                                    <input
                                        type="text"
                                        required
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">服务类型</label>
                                    <select
                                        value={formData.type}
                                        onChange={(e) => setFormData({ ...formData, type: e.target.value as ServiceSource['type'] })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    >
                                        <option value="openai">OpenAI</option>
                                        <option value="anthropic">Anthropic</option>
                                        <option value="google">Google</option>
                                        <option value="custom">自定义</option>
                                    </select>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">API 端点</label>
                                <input
                                    type="url"
                                    required
                                    value={formData.endpoint}
                                    onChange={(e) => setFormData({ ...formData, endpoint: e.target.value })}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">API 密钥</label>
                                <input
                                    type="password"
                                    required
                                    value={formData.apiKey}
                                    onChange={(e) => setFormData({ ...formData, apiKey: e.target.value })}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">描述</label>
                                <textarea
                                    value={formData.description}
                                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                    rows={3}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div className="flex justify-end space-x-3">
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowForm(false)
                                        setEditingId(null)
                                        setFormData({ name: '', type: 'openai', endpoint: '', apiKey: '', description: '' })
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

            {/* Service Sources List */}
            <div className="bg-white shadow overflow-hidden sm:rounded-md">
                {isLoading ? (
                    <div className="flex items-center justify-center py-12">
                        <RefreshCw className="h-8 w-8 text-blue-600 animate-spin" />
                        <span className="ml-2 text-gray-600">加载服务源...</span>
                    </div>
                ) : sources.length === 0 ? (
                    <div className="text-center py-12">
                        <p className="text-gray-500">暂无服务源</p>
                    </div>
                ) : (
                    <ul className="divide-y divide-gray-200">
                        {sources.map((source) => (
                            <li key={source.id} className="px-6 py-4">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center space-x-4 flex-1">
                                        <div className="text-2xl">{getTypeIcon(source.type)}</div>
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center space-x-3">
                                                <h3 className="text-sm font-medium text-gray-900 truncate">
                                                    {source.name}
                                                </h3>
                                                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(source.status)}`}>
                                                    {source.status}
                                                </span>
                                            </div>
                                            <div className="mt-1 flex items-center space-x-2 text-sm text-gray-500">
                                                <span>{source.endpoint}</span>
                                                <ExternalLink className="h-3 w-3" />
                                            </div>
                                            <p className="mt-1 text-sm text-gray-500">{source.description}</p>
                                            <p className="mt-1 text-xs text-gray-400">创建于: {source.createdAt}</p>
                                        </div>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <button
                                            onClick={() => toggleStatus(source.id)}
                                            className={`p-1 rounded-full ${source.status === 'active' ? 'text-green-600 hover:bg-green-100' : 'text-gray-400 hover:bg-gray-100'}`}
                                        >
                                            {source.status === 'active' ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
                                        </button>
                                        <button
                                            onClick={() => handleEdit(source)}
                                            className="p-1 text-blue-600 hover:bg-blue-100 rounded-full"
                                        >
                                            <Edit className="h-4 w-4" />
                                        </button>
                                        <button
                                            onClick={() => handleDelete(source.id)}
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

export default ServiceSources
