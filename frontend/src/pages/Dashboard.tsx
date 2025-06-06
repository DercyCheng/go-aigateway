import { useState } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line, PieChart, Pie, Cell } from 'recharts'
import { Activity, Server, Users, AlertTriangle } from 'lucide-react'

const Dashboard = () => {
    const [stats] = useState({
        totalRequests: 15420,
        activeServices: 12,
        connectedUsers: 234,
        errorRate: 2.3
    })

    const requestData = [
        { time: '00:00', requests: 120 },
        { time: '04:00', requests: 80 },
        { time: '08:00', requests: 350 },
        { time: '12:00', requests: 420 },
        { time: '16:00', requests: 380 },
        { time: '20:00', requests: 250 },
    ]

    const serviceData = [
        { name: 'ChatGPT API', value: 45, color: '#8884d8' },
        { name: 'Claude API', value: 30, color: '#82ca9d' },
        { name: 'Gemini API', value: 15, color: '#ffc658' },
        { name: 'Others', value: 10, color: '#ff7c7c' },
    ]

    const responseTimeData = [
        { time: '00:00', time_ms: 120 },
        { time: '04:00', time_ms: 98 },
        { time: '08:00', time_ms: 145 },
        { time: '12:00', time_ms: 132 },
        { time: '16:00', time_ms: 108 },
        { time: '20:00', time_ms: 115 },
    ]

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-bold text-gray-900">监控面板</h1>
                <p className="mt-2 text-sm text-gray-600">实时监控系统状态和性能指标</p>
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
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
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                {/* Request Volume Chart */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h3 className="text-lg font-medium text-gray-900 mb-4">请求量趋势</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={requestData}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="time" />
                            <YAxis />
                            <Tooltip />
                            <Bar dataKey="requests" fill="#8884d8" />
                        </BarChart>
                    </ResponsiveContainer>
                </div>

                {/* Service Distribution */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h3 className="text-lg font-medium text-gray-900 mb-4">服务分布</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <PieChart>
                            <Pie
                                data={serviceData}
                                cx="50%"
                                cy="50%"
                                labelLine={false}
                                label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                                outerRadius={80}
                                fill="#8884d8"
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

                {/* Response Time Chart */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h3 className="text-lg font-medium text-gray-900 mb-4">响应时间</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={responseTimeData}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="time" />
                            <YAxis />
                            <Tooltip />
                            <Line type="monotone" dataKey="time_ms" stroke="#82ca9d" strokeWidth={2} />
                        </LineChart>
                    </ResponsiveContainer>
                </div>

                {/* Recent Activities */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h3 className="text-lg font-medium text-gray-900 mb-4">最近活动</h3>
                    <div className="space-y-4">
                        <div className="flex items-center space-x-3">
                            <div className="flex-shrink-0">
                                <div className="h-2 w-2 bg-green-500 rounded-full"></div>
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm text-gray-900">新服务上线: ChatGPT-4o</p>
                                <p className="text-xs text-gray-500">2 分钟前</p>
                            </div>
                        </div>
                        <div className="flex items-center space-x-3">
                            <div className="flex-shrink-0">
                                <div className="h-2 w-2 bg-yellow-500 rounded-full"></div>
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm text-gray-900">域名证书即将过期</p>
                                <p className="text-xs text-gray-500">5 分钟前</p>
                            </div>
                        </div>
                        <div className="flex items-center space-x-3">
                            <div className="flex-shrink-0">
                                <div className="h-2 w-2 bg-blue-500 rounded-full"></div>
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm text-gray-900">路由规则更新完成</p>
                                <p className="text-xs text-gray-500">10 分钟前</p>
                            </div>
                        </div>
                        <div className="flex items-center space-x-3">
                            <div className="flex-shrink-0">
                                <div className="h-2 w-2 bg-red-500 rounded-full"></div>
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm text-gray-900">API 调用异常增加</p>
                                <p className="text-xs text-gray-500">15 分钟前</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Dashboard
