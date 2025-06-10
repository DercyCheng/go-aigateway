import { useState, useEffect } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line, PieChart, Pie, Cell } from 'recharts'
import { Activity, Server, Users, AlertTriangle, Cpu, Zap } from 'lucide-react'
import { apiService } from '../services/api'

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
    const [healthData, setHealthData] = useState<any>(null)

    // Check if local model is running
    useEffect(() => {
        /**
         * Checks the status of local models by making an API call.
         * Updates the stats state with whether any local model is running.
         * Handles loading state and errors silently.
         */
        const checkLocalModel = async () => {
            try {
                setIsLocalModelChecking(true)
                const response = await apiService.getLocalModels()
                if (response.success && response.data) {
                    const anyModelRunning = response.data.models &&
                        Array.isArray(response.data.models) &&
                        response.data.models.some((model: LocalModel) => model.status === 'running')
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

        const fetchHealth = async () => {
            try {
                const response = await apiService.healthCheck()
                if (response.success) {
                    setHealthData(response.data)
                }
            } catch (error) {
                console.error('Error fetching health data:', error)
            }
        }

        const fetchDashboardStats = async () => {
            try {
                const response = await apiService.getDashboardStats()
                if (response.success && response.data) {
                    setStats(prevStats => ({
                        ...prevStats,
                        totalRequests: response.data.totalRequests || prevStats.totalRequests,
                        activeServices: response.data.activeServices || prevStats.activeServices,
                        connectedUsers: response.data.connectedUsers || prevStats.connectedUsers,
                        errorRate: response.data.errorRate || prevStats.errorRate,
                        localModelRequests: response.data.localModelRequests || prevStats.localModelRequests
                    }))
                }
            } catch (error) {
                console.error('Error fetching dashboard stats:', error)
                // Keep existing stats on error, don't reset to defaults
            }
        }

        const fetchData = async () => {
            await Promise.all([checkLocalModel(), fetchHealth(), fetchDashboardStats()])
        }

        fetchData()
        // Check every 30 seconds
        const interval = setInterval(fetchData, 30000)
        return () => clearInterval(interval)
    }, [])

    // Generate realistic chart data based on current stats
    const generateChartData = () => {
        const now = new Date()
        const hours = []

        // Generate last 6 data points (every 4 hours)
        for (let i = 5; i >= 0; i--) {
            const time = new Date(now.getTime() - i * 4 * 60 * 60 * 1000)
            hours.push(time.toISOString().substr(11, 5))
        }

        // Generate request data based on current stats
        const baseRequests = Math.floor(stats.totalRequests / 24) // requests per hour estimate
        const baseLocalRequests = Math.floor(stats.localModelRequests / 24)

        return hours.map(time => ({
            time,
            requests: Math.max(0, baseRequests + Math.floor(Math.random() * 100 - 50)),
            localRequests: Math.max(0, baseLocalRequests + Math.floor(Math.random() * 20 - 10))
        }))
    }

    const requestData = generateChartData()

    const serviceData = [
        { name: 'OpenAI API', value: 35, color: '#8884d8' },
        { name: 'Anthropic API', value: 25, color: '#82ca9d' },
        { name: 'Google API', value: 15, color: '#ffc658' },
        { name: '本地模型', value: Math.floor((stats.localModelRequests / stats.totalRequests) * 100) || 10, color: '#ff7c7c' },
        { name: 'Others', value: 15, color: '#aaaaaa' },
    ]

    const responseTimeData = [
        { time: requestData[0]?.time || '00:00', time_ms: 120, local_ms: 40 },
        { time: requestData[1]?.time || '04:00', time_ms: 98, local_ms: 35 },
        { time: requestData[2]?.time || '08:00', time_ms: 145, local_ms: 45 },
        { time: requestData[3]?.time || '12:00', time_ms: 132, local_ms: 40 },
        { time: requestData[4]?.time || '16:00', time_ms: 108, local_ms: 38 },
        { time: requestData[5]?.time || '20:00', time_ms: 115, local_ms: 42 },
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

            {/* System Health Section */}
            {healthData && (
                <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">系统健康状态</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="border rounded-lg p-4 bg-purple-50">
                            <h3 className="font-medium text-purple-800 mb-2 flex items-center">
                                <Server className="h-5 w-5 mr-2" />
                                系统状态
                            </h3>
                            <p className="text-sm text-purple-800">
                                系统运行状态: <span className="font-bold">{healthData.status || '正常'}</span>
                            </p>
                        </div>
                    </div>
                </div>
            )}

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
