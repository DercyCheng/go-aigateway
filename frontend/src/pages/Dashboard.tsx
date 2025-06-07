import { useState, useEffect } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line, PieChart, Pie, Cell } from 'recharts'
import { Activity, Server, Users, AlertTriangle, Cpu, Zap } from 'lucide-react'

interface LocalModel {
    id: string;
    status: string;
}

const Dashboard = () => {
    const [stats, setStats] = useState({
        totalRequests: 15420,
        activeServices: 12,
        connectedUsers: 234,
        errorRate: 2.3,
        localModelActive: false,
        localModelRequests: 115
    })

    const [isLocalModelChecking, setIsLocalModelChecking] = useState(false)

    // Check if local model is running
    useEffect(() => {
        const checkLocalModel = async () => {
            try {
                setIsLocalModelChecking(true)
                const response = await fetch('/api/local/models')
                if (response.ok) {
                    const data = await response.json()
                    const anyModelRunning = data.models &&
                        Array.isArray(data.models) &&
                        data.models.some((model: LocalModel) => model.status === 'running')
                    setStats(prevStats => ({
                        ...prevStats,
                        localModelActive: anyModelRunning
                    }))
                }
            } catch (error) {
                console.error('Error checking local model:', error)
            } finally {
                setIsLocalModelChecking(false)
            }
        }

        checkLocalModel()
        // Check every 30 seconds
        const interval = setInterval(checkLocalModel, 30000)
        return () => clearInterval(interval)
    }, [])

    const requestData = [
        { time: '00:00', requests: 120, localRequests: 10 },
        { time: '04:00', requests: 80, localRequests: 8 },
        { time: '08:00', requests: 350, localRequests: 22 },
        { time: '12:00', requests: 420, localRequests: 30 },
        { time: '16:00', requests: 380, localRequests: 25 },
        { time: '20:00', requests: 250, localRequests: 20 },
    ]

    const serviceData = [
        { name: 'ChatGPT API', value: 45, color: '#8884d8' },
        { name: 'Claude API', value: 30, color: '#82ca9d' },
        { name: 'Gemini API', value: 15, color: '#ffc658' },
        { name: '本地模型', value: 5, color: '#ff7c7c' },
        { name: 'Others', value: 5, color: '#aaaaaa' },
    ]

    const responseTimeData = [
        { time: '00:00', time_ms: 120, local_ms: 40 },
        { time: '04:00', time_ms: 98, local_ms: 35 },
        { time: '08:00', time_ms: 145, local_ms: 45 },
        { time: '12:00', time_ms: 132, local_ms: 40 },
        { time: '16:00', time_ms: 108, local_ms: 38 },
        { time: '20:00', time_ms: 115, local_ms: 42 },
    ]

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-bold text-gray-900">监控面板</h1>
                <p className="mt-2 text-sm text-gray-600">实时监控系统状态和性能指标</p>
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-6">
                <div className="bg-white overflow-hidden shadow rounded-lg">
                    <div className="p-5">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <Activity className="h-6 w-6 text-blue-600" />
                            </div>
                            <div className="ml-5 w-0 flex-1">
                                <dl>
                                    <dt className="text-sm font-medium text-gray-500 truncate">总请求数</dt>
                                    <dd className="text-lg font-medium text-gray-900">{stats.totalRequests.toLocaleString()}</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="bg-white overflow-hidden shadow rounded-lg">
                    <div className="p-5">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <Server className="h-6 w-6 text-green-600" />
                            </div>
                            <div className="ml-5 w-0 flex-1">
                                <dl>
                                    <dt className="text-sm font-medium text-gray-500 truncate">活跃服务</dt>
                                    <dd className="text-lg font-medium text-gray-900">{stats.activeServices}</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="bg-white overflow-hidden shadow rounded-lg">
                    <div className="p-5">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <Users className="h-6 w-6 text-purple-600" />
                            </div>
                            <div className="ml-5 w-0 flex-1">
                                <dl>
                                    <dt className="text-sm font-medium text-gray-500 truncate">在线用户</dt>
                                    <dd className="text-lg font-medium text-gray-900">{stats.connectedUsers}</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="bg-white overflow-hidden shadow rounded-lg">
                    <div className="p-5">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <AlertTriangle className="h-6 w-6 text-red-600" />
                            </div>
                            <div className="ml-5 w-0 flex-1">
                                <dl>
                                    <dt className="text-sm font-medium text-gray-500 truncate">错误率</dt>
                                    <dd className="text-lg font-medium text-gray-900">{stats.errorRate}%</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="bg-white overflow-hidden shadow rounded-lg">
                    <div className="p-5">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <Cpu className="h-6 w-6 text-indigo-600" />
                            </div>
                            <div className="ml-5 w-0 flex-1">
                                <dl>
                                    <dt className="text-sm font-medium text-gray-500 truncate">本地模型状态</dt>
                                    <dd className={`text-lg font-medium ${stats.localModelActive ? 'text-green-600' : 'text-gray-400'}`}>
                                        {isLocalModelChecking ? '检查中...' : (stats.localModelActive ? '运行中' : '已停止')}
                                    </dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="bg-white overflow-hidden shadow rounded-lg">
                    <div className="p-5">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <Zap className="h-6 w-6 text-yellow-500" />
                            </div>
                            <div className="ml-5 w-0 flex-1">
                                <dl>
                                    <dt className="text-sm font-medium text-gray-500 truncate">本地模型请求</dt>
                                    <dd className="text-lg font-medium text-gray-900">{stats.localModelRequests}</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
                {/* Requests Chart */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">请求趋势</h2>
                    <div className="h-72">
                        <ResponsiveContainer width="100%" height="100%">
                            <BarChart data={requestData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="time" />
                                <YAxis />
                                <Tooltip />
                                <Bar dataKey="requests" name="所有请求" fill="#8884d8" />
                                <Bar dataKey="localRequests" name="本地模型请求" fill="#82ca9d" />
                            </BarChart>
                        </ResponsiveContainer>
                    </div>
                </div>

                {/* Service Usage Chart */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">服务使用分布</h2>
                    <div className="h-72">
                        <ResponsiveContainer width="100%" height="100%">
                            <PieChart>
                                <Pie
                                    data={serviceData}
                                    cx="50%"
                                    cy="50%"
                                    labelLine={true}
                                    label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                                    outerRadius={80}
                                    dataKey="value"
                                >
                                    {serviceData.map((entry, index) => (
                                        <Cell key={`cell-${index}`} fill={entry.color} />
                                    ))}
                                </Pie>
                                <Tooltip />
                            </PieChart>
                        </ResponsiveContainer>
                    </div>
                </div>

                {/* Response Time Chart */}
                <div className="bg-white shadow rounded-lg p-6 lg:col-span-2">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">响应时间 (ms)</h2>
                    <div className="h-72">
                        <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={responseTimeData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="time" />
                                <YAxis />
                                <Tooltip />
                                <Line type="monotone" dataKey="time_ms" name="平均响应时间" stroke="#8884d8" activeDot={{ r: 8 }} />
                                <Line type="monotone" dataKey="local_ms" name="本地模型响应时间" stroke="#82ca9d" />
                            </LineChart>
                        </ResponsiveContainer>
                    </div>
                </div>
            </div>

            {/* Local Model Info Section */}
            {stats.localModelActive && (
                <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">本地模型信息</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="border rounded-lg p-4 bg-blue-50">
                            <h3 className="font-medium text-blue-800 mb-2 flex items-center">
                                <Cpu className="h-5 w-5 mr-2" />
                                本地模型性能
                            </h3>
                            <p className="text-sm text-blue-800">
                                本地部署的AI模型提供更低的延迟和更高的隐私保护。当前响应时间比云服务快约{' '}
                                <span className="font-bold">65%</span>。
                            </p>
                        </div>
                        <div className="border rounded-lg p-4 bg-green-50">
                            <h3 className="font-medium text-green-800 mb-2 flex items-center">
                                <Zap className="h-5 w-5 mr-2" />
                                使用统计
                            </h3>
                            <p className="text-sm text-green-800">
                                本地模型已处理 <span className="font-bold">{stats.localModelRequests}</span> 个请求，
                                占总请求量的 <span className="font-bold">{((stats.localModelRequests / stats.totalRequests) * 100).toFixed(1)}%</span>。
                            </p>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default Dashboard
