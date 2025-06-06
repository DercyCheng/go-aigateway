import { useState } from 'react'
import { Plus, Edit, Trash2, Download, RefreshCw, AlertTriangle, CheckCircle, Clock } from 'lucide-react'

interface Certificate {
    id: string
    name: string
    domain: string
    issuer: string
    type: 'lets_encrypt' | 'custom' | 'self_signed'
    status: 'valid' | 'expiring' | 'expired' | 'pending'
    issuedAt: string
    expiresAt: string
    autoRenew: boolean
    fingerprint: string
    serialNumber: string
    keySize: number
    createdAt: string
}

const CertificateManagement = () => {
    const [certificates, setCertificates] = useState<Certificate[]>([
        {
            id: '1',
            name: 'API Gateway SSL',
            domain: 'api.aigateway.com',
            issuer: "Let's Encrypt",
            type: 'lets_encrypt',
            status: 'valid',
            issuedAt: '2024-01-15',
            expiresAt: '2024-04-15',
            autoRenew: true,
            fingerprint: 'SHA256:abc123def456...',
            serialNumber: '03:a1:b2:c3:d4:e5:f6',
            keySize: 2048,
            createdAt: '2024-01-15'
        },
        {
            id: '2',
            name: 'Wildcard Certificate',
            domain: '*.aigateway.com',
            issuer: 'DigiCert',
            type: 'custom',
            status: 'expiring',
            issuedAt: '2023-12-01',
            expiresAt: '2024-02-01',
            autoRenew: false,
            fingerprint: 'SHA256:def456ghi789...',
            serialNumber: '04:b2:c3:d4:e5:f6:a1',
            keySize: 4096,
            createdAt: '2023-12-01'
        },
        {
            id: '3',
            name: 'Test Certificate',
            domain: 'test.aigateway.dev',
            issuer: 'Self-Signed',
            type: 'self_signed',
            status: 'valid',
            issuedAt: '2024-01-20',
            expiresAt: '2025-01-20',
            autoRenew: false,
            fingerprint: 'SHA256:ghi789jkl012...',
            serialNumber: '05:c3:d4:e5:f6:a1:b2',
            keySize: 2048,
            createdAt: '2024-01-20'
        }
    ])

    const [showForm, setShowForm] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [formData, setFormData] = useState({
        name: '',
        domain: '',
        type: 'lets_encrypt' as Certificate['type'],
        autoRenew: true
    })

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (editingId) {
            setCertificates(certificates.map(cert =>
                cert.id === editingId
                    ? { ...cert, ...formData }
                    : cert
            ))
            setEditingId(null)
        } else {
            const newCert: Certificate = {
                id: Date.now().toString(),
                ...formData,
                issuer: formData.type === 'lets_encrypt' ? "Let's Encrypt" :
                    formData.type === 'custom' ? 'Custom CA' : 'Self-Signed',
                status: 'pending',
                issuedAt: new Date().toISOString().split('T')[0],
                expiresAt: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
                fingerprint: 'SHA256:' + Math.random().toString(36).substring(2, 15),
                serialNumber: Array.from({ length: 7 }, () => Math.floor(Math.random() * 256).toString(16).padStart(2, '0')).join(':'),
                keySize: 2048,
                createdAt: new Date().toISOString().split('T')[0]
            }
            setCertificates([...certificates, newCert])
        }
        setFormData({
            name: '',
            domain: '',
            type: 'lets_encrypt',
            autoRenew: true
        })
        setShowForm(false)
    }

    const handleEdit = (cert: Certificate) => {
        setFormData({
            name: cert.name,
            domain: cert.domain,
            type: cert.type,
            autoRenew: cert.autoRenew
        })
        setEditingId(cert.id)
        setShowForm(true)
    }

    const handleDelete = (id: string) => {
        setCertificates(certificates.filter(cert => cert.id !== id))
    }

    const renewCertificate = (id: string) => {
        setCertificates(certificates.map(cert =>
            cert.id === id
                ? {
                    ...cert,
                    status: 'pending',
                    issuedAt: new Date().toISOString().split('T')[0],
                    expiresAt: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
                }
                : cert
        ))
    }

    const toggleAutoRenew = (id: string) => {
        setCertificates(certificates.map(cert =>
            cert.id === id
                ? { ...cert, autoRenew: !cert.autoRenew }
                : cert
        ))
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'valid': return 'bg-green-100 text-green-800'
            case 'expiring': return 'bg-yellow-100 text-yellow-800'
            case 'expired': return 'bg-red-100 text-red-800'
            case 'pending': return 'bg-blue-100 text-blue-800'
            default: return 'bg-gray-100 text-gray-800'
        }
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'valid': return <CheckCircle className="h-4 w-4 text-green-600" />
            case 'expiring': return <AlertTriangle className="h-4 w-4 text-yellow-600" />
            case 'expired': return <AlertTriangle className="h-4 w-4 text-red-600" />
            case 'pending': return <Clock className="h-4 w-4 text-blue-600" />
            default: return <AlertTriangle className="h-4 w-4 text-gray-600" />
        }
    }

    const getTypeColor = (type: string) => {
        switch (type) {
            case 'lets_encrypt': return 'bg-blue-100 text-blue-800'
            case 'custom': return 'bg-purple-100 text-purple-800'
            case 'self_signed': return 'bg-orange-100 text-orange-800'
            default: return 'bg-gray-100 text-gray-800'
        }
    }

    const getDaysUntilExpiry = (expiryDate: string) => {
        const expiry = new Date(expiryDate)
        const now = new Date()
        return Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">证书管理</h1>
                    <p className="mt-2 text-sm text-gray-600">管理SSL/TLS证书和自动续期配置</p>
                </div>
                <button
                    onClick={() => setShowForm(true)}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                >
                    <Plus className="h-4 w-4 mr-2" />
                    添加证书
                </button>
            </div>

            {/* Add/Edit Form */}
            {showForm && (
                <div className="bg-white shadow rounded-lg">
                    <div className="px-4 py-5 sm:p-6">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
                            {editingId ? '编辑证书' : '添加证书'}
                        </h3>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">证书名称</label>
                                    <input
                                        type="text"
                                        required
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">域名</label>
                                    <input
                                        type="text"
                                        required
                                        value={formData.domain}
                                        onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
                                        placeholder="example.com 或 *.example.com"
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">证书类型</label>
                                <select
                                    value={formData.type}
                                    onChange={(e) => setFormData({ ...formData, type: e.target.value as Certificate['type'] })}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                >
                                    <option value="lets_encrypt">Let's Encrypt (免费)</option>
                                    <option value="custom">自定义证书</option>
                                    <option value="self_signed">自签名证书</option>
                                </select>
                            </div>
                            <div className="flex items-center">
                                <input
                                    type="checkbox"
                                    checked={formData.autoRenew}
                                    onChange={(e) => setFormData({ ...formData, autoRenew: e.target.checked })}
                                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                                />
                                <label className="ml-2 block text-sm text-gray-900">启用自动续期</label>
                            </div>
                            <div className="flex justify-end space-x-3">
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowForm(false)
                                        setEditingId(null)
                                        setFormData({
                                            name: '',
                                            domain: '',
                                            type: 'lets_encrypt',
                                            autoRenew: true
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

            {/* Certificates List */}
            <div className="grid grid-cols-1 gap-6">
                {certificates.map((cert) => (
                    <div key={cert.id} className="bg-white shadow rounded-lg p-6">
                        <div className="flex items-start justify-between mb-4">
                            <div className="flex items-center space-x-3">
                                {getStatusIcon(cert.status)}
                                <div>
                                    <h3 className="text-lg font-medium text-gray-900">{cert.name}</h3>
                                    <div className="flex items-center space-x-2 mt-1">
                                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(cert.status)}`}>
                                            {cert.status}
                                        </span>
                                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getTypeColor(cert.type)}`}>
                                            {cert.issuer}
                                        </span>
                                        {cert.autoRenew && (
                                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                                                自动续期
                                            </span>
                                        )}
                                    </div>
                                </div>
                            </div>
                            <div className="flex items-center space-x-2">
                                <button
                                    onClick={() => renewCertificate(cert.id)}
                                    className="p-1 text-blue-600 hover:bg-blue-100 rounded-full"
                                    title="手动续期"
                                >
                                    <RefreshCw className="h-4 w-4" />
                                </button>
                                <button
                                    className="p-1 text-green-600 hover:bg-green-100 rounded-full"
                                    title="下载证书"
                                >
                                    <Download className="h-4 w-4" />
                                </button>
                                <button
                                    onClick={() => handleEdit(cert)}
                                    className="p-1 text-gray-600 hover:bg-gray-100 rounded-full"
                                >
                                    <Edit className="h-4 w-4" />
                                </button>
                                <button
                                    onClick={() => handleDelete(cert.id)}
                                    className="p-1 text-red-600 hover:bg-red-100 rounded-full"
                                >
                                    <Trash2 className="h-4 w-4" />
                                </button>
                            </div>
                        </div>

                        {/* Certificate Details */}
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div className="space-y-3">
                                <div>
                                    <label className="text-sm font-medium text-gray-500">域名</label>
                                    <p className="text-sm text-gray-900 font-mono">{cert.domain}</p>
                                </div>
                                <div>
                                    <label className="text-sm font-medium text-gray-500">颁发日期</label>
                                    <p className="text-sm text-gray-900">{cert.issuedAt}</p>
                                </div>
                                <div>
                                    <label className="text-sm font-medium text-gray-500">过期日期</label>
                                    <div className="flex items-center space-x-2">
                                        <p className={`text-sm ${getDaysUntilExpiry(cert.expiresAt) <= 30 ? 'text-red-600' : 'text-gray-900'}`}>
                                            {cert.expiresAt}
                                        </p>
                                        <span className={`text-xs px-2 py-1 rounded-full ${getDaysUntilExpiry(cert.expiresAt) <= 30
                                                ? 'bg-red-100 text-red-800'
                                                : 'bg-gray-100 text-gray-800'
                                            }`}>
                                            {getDaysUntilExpiry(cert.expiresAt)} 天后过期
                                        </span>
                                    </div>
                                </div>
                            </div>
                            <div className="space-y-3">
                                <div>
                                    <label className="text-sm font-medium text-gray-500">序列号</label>
                                    <p className="text-sm text-gray-900 font-mono">{cert.serialNumber}</p>
                                </div>
                                <div>
                                    <label className="text-sm font-medium text-gray-500">密钥长度</label>
                                    <p className="text-sm text-gray-900">{cert.keySize} bits</p>
                                </div>
                                <div>
                                    <label className="text-sm font-medium text-gray-500">指纹</label>
                                    <p className="text-sm text-gray-900 font-mono truncate">{cert.fingerprint}</p>
                                </div>
                            </div>
                        </div>

                        {/* Actions */}
                        <div className="mt-6 pt-4 border-t border-gray-200 flex items-center justify-between">
                            <div className="text-xs text-gray-500">
                                创建于: {cert.createdAt}
                            </div>
                            <div className="flex items-center space-x-2">
                                <button
                                    onClick={() => toggleAutoRenew(cert.id)}
                                    className={`text-xs px-3 py-1 rounded-full ${cert.autoRenew
                                            ? 'bg-green-100 text-green-700 hover:bg-green-200'
                                            : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                                        }`}
                                >
                                    {cert.autoRenew ? '禁用自动续期' : '启用自动续期'}
                                </button>
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}

export default CertificateManagement
