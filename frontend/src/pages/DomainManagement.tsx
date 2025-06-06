import { useState } from 'react'
import { Plus, Edit, Trash2, Globe, ExternalLink, AlertTriangle, CheckCircle } from 'lucide-react'

interface Domain {
    id: string
    domain: string
    status: 'active' | 'pending' | 'error'
    sslEnabled: boolean
    certificateExpiry?: string
    provider: string
    records: {
        type: string
        name: string
        value: string
        ttl: number
    }[]
    createdAt: string
    updatedAt: string
}

const DomainManagement = () => {
    const [domains, setDomains] = useState<Domain[]>([
        {
            id: '1',
            domain: 'api.aigateway.com',
            status: 'active',
            sslEnabled: true,
            certificateExpiry: '2024-12-31',
            provider: 'Cloudflare',
            records: [
                { type: 'A', name: '@', value: '192.168.1.100', ttl: 300 },
                { type: 'CNAME', name: 'www', value: 'api.aigateway.com', ttl: 300 }
            ],
            createdAt: '2024-01-15',
            updatedAt: '2024-01-20'
        },
        {
            id: '2',
            domain: 'gateway.example.com',
            status: 'active',
            sslEnabled: true,
            certificateExpiry: '2024-08-15',
            provider: 'Route53',
            records: [
                { type: 'A', name: '@', value: '192.168.1.101', ttl: 600 }
            ],
            createdAt: '2024-01-16',
            updatedAt: '2024-01-19'
        },
        {
            id: '3',
            domain: 'test.aigateway.dev',
            status: 'pending',
            sslEnabled: false,
            provider: 'Manual',
            records: [
                { type: 'A', name: '@', value: '192.168.1.102', ttl: 300 }
            ],
            createdAt: '2024-01-18',
            updatedAt: '2024-01-18'
        }
    ])

    const [showForm, setShowForm] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [formData, setFormData] = useState({
        domain: '',
        provider: 'Cloudflare',
        sslEnabled: true
    })

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (editingId) {
            setDomains(domains.map(domain =>
                domain.id === editingId
                    ? {
                        ...domain,
                        ...formData,
                        updatedAt: new Date().toISOString().split('T')[0]
                    }
                    : domain
            ))
            setEditingId(null)
        } else {
            const newDomain: Domain = {
                id: Date.now().toString(),
                ...formData,
                status: 'pending',
                records: [
                    { type: 'A', name: '@', value: '192.168.1.100', ttl: 300 }
                ],
                createdAt: new Date().toISOString().split('T')[0],
                updatedAt: new Date().toISOString().split('T')[0]
            }
            setDomains([...domains, newDomain])
        }
        setFormData({
            domain: '',
            provider: 'Cloudflare',
            sslEnabled: true
        })
        setShowForm(false)
    }

    const handleEdit = (domain: Domain) => {
        setFormData({
            domain: domain.domain,
            provider: domain.provider,
            sslEnabled: domain.sslEnabled
        })
        setEditingId(domain.id)
        setShowForm(true)
    }

    const handleDelete = (id: string) => {
        setDomains(domains.filter(domain => domain.id !== id))
    }

    const toggleSSL = (id: string) => {
        setDomains(domains.map(domain =>
            domain.id === id
                ? { ...domain, sslEnabled: !domain.sslEnabled, updatedAt: new Date().toISOString().split('T')[0] }
                : domain
        ))
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active': return 'bg-green-100 text-green-800'
            case 'pending': return 'bg-yellow-100 text-yellow-800'
            case 'error': return 'bg-red-100 text-red-800'
            default: return 'bg-gray-100 text-gray-800'
        }
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'active': return <CheckCircle className="h-4 w-4 text-green-600" />
            case 'pending': return <AlertTriangle className="h-4 w-4 text-yellow-600" />
            case 'error': return <AlertTriangle className="h-4 w-4 text-red-600" />
            default: return <Globe className="h-4 w-4 text-gray-600" />
        }
    }

    const isExpiringSoon = (expiryDate?: string) => {
        if (!expiryDate) return false
        const expiry = new Date(expiryDate)
        const now = new Date()
        const daysUntilExpiry = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
        return daysUntilExpiry <= 30
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">域名管理</h1>
                    <p className="mt-2 text-sm text-gray-600">管理域名解析和SSL证书配置</p>
                </div>
                <button
                    onClick={() => setShowForm(true)}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                >
                    <Plus className="h-4 w-4 mr-2" />
                    添加域名
                </button>
            </div>

            {/* Add/Edit Form */}
            {showForm && (
                <div className="bg-white shadow rounded-lg">
                    <div className="px-4 py-5 sm:p-6">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
                            {editingId ? '编辑域名' : '添加域名'}
                        </h3>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700">域名</label>
                                <input
                                    type="text"
                                    required
                                    value={formData.domain}
                                    onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
                                    placeholder="api.example.com"
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">DNS 提供商</label>
                                <select
                                    value={formData.provider}
                                    onChange={(e) => setFormData({ ...formData, provider: e.target.value })}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                >
                                    <option value="Cloudflare">Cloudflare</option>
                                    <option value="Route53">AWS Route53</option>
                                    <option value="DNSPod">DNSPod</option>
                                    <option value="Aliyun">阿里云DNS</option>
                                    <option value="Manual">手动配置</option>
                                </select>
                            </div>
                            <div className="flex items-center">
                                <input
                                    type="checkbox"
                                    checked={formData.sslEnabled}
                                    onChange={(e) => setFormData({ ...formData, sslEnabled: e.target.checked })}
                                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                                />
                                <label className="ml-2 block text-sm text-gray-900">启用 SSL</label>
                            </div>
                            <div className="flex justify-end space-x-3">
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowForm(false)
                                        setEditingId(null)
                                        setFormData({
                                            domain: '',
                                            provider: 'Cloudflare',
                                            sslEnabled: true
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

            {/* Domains List */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                {domains.map((domain) => (
                    <div key={domain.id} className="bg-white shadow rounded-lg p-6">
                        <div className="flex items-start justify-between mb-4">
                            <div className="flex items-center space-x-3">
                                {getStatusIcon(domain.status)}
                                <div>
                                    <h3 className="text-lg font-medium text-gray-900">{domain.domain}</h3>
                                    <div className="flex items-center space-x-2 mt-1">
                                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(domain.status)}`}>
                                            {domain.status}
                                        </span>
                                        <span className="text-xs text-gray-500">{domain.provider}</span>
                                    </div>
                                </div>
                            </div>
                            <div className="flex items-center space-x-2">
                                <a
                                    href={`https://${domain.domain}`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-full"
                                >
                                    <ExternalLink className="h-4 w-4" />
                                </a>
                                <button
                                    onClick={() => handleEdit(domain)}
                                    className="p-1 text-gray-600 hover:bg-gray-100 rounded-full"
                                >
                                    <Edit className="h-4 w-4" />
                                </button>
                                <button
                                    onClick={() => handleDelete(domain.id)}
                                    className="p-1 text-red-600 hover:bg-red-100 rounded-full"
                                >
                                    <Trash2 className="h-4 w-4" />
                                </button>
                            </div>
                        </div>

                        {/* SSL Info */}
                        <div className="mb-4">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <span className="text-sm text-gray-600">SSL 证书:</span>
                                    <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${domain.sslEnabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                                        }`}>
                                        {domain.sslEnabled ? '已启用' : '未启用'}
                                    </span>
                                </div>
                                <button
                                    onClick={() => toggleSSL(domain.id)}
                                    className={`text-xs px-2 py-1 rounded ${domain.sslEnabled
                                            ? 'bg-red-100 text-red-700 hover:bg-red-200'
                                            : 'bg-green-100 text-green-700 hover:bg-green-200'
                                        }`}
                                >
                                    {domain.sslEnabled ? '禁用' : '启用'}
                                </button>
                            </div>
                            {domain.certificateExpiry && (
                                <div className={`mt-2 text-xs ${isExpiringSoon(domain.certificateExpiry) ? 'text-red-600' : 'text-gray-500'}`}>
                                    {isExpiringSoon(domain.certificateExpiry) && <AlertTriangle className="inline h-3 w-3 mr-1" />}
                                    证书过期时间: {domain.certificateExpiry}
                                    {isExpiringSoon(domain.certificateExpiry) && ' (即将过期)'}
                                </div>
                            )}
                        </div>

                        {/* DNS Records */}
                        <div className="border-t border-gray-200 pt-4">
                            <h4 className="text-sm font-medium text-gray-900 mb-2">DNS 记录</h4>
                            <div className="space-y-2">
                                {domain.records.map((record, index) => (
                                    <div key={index} className="bg-gray-50 rounded-md p-3">
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center space-x-3">
                                                <span className="inline-flex items-center px-2 py-1 rounded text-xs font-mono bg-blue-100 text-blue-800">
                                                    {record.type}
                                                </span>
                                                <span className="text-sm text-gray-900">{record.name}</span>
                                            </div>
                                            <span className="text-xs text-gray-500">TTL: {record.ttl}s</span>
                                        </div>
                                        <div className="mt-1 text-sm text-gray-600 font-mono">
                                            {record.value}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>

                        {/* Timestamps */}
                        <div className="mt-4 pt-4 border-t border-gray-200 text-xs text-gray-500">
                            <div className="flex justify-between">
                                <span>创建: {domain.createdAt}</span>
                                <span>更新: {domain.updatedAt}</span>
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}

export default DomainManagement
