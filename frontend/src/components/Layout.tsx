import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
    BarChart3,
    Database,
    List,
    Route,
    Globe,
    Shield,
    Menu,
    X,
    Cpu
} from 'lucide-react'

interface LayoutProps {
    children: React.ReactNode
}

const Layout = ({ children }: LayoutProps) => {
    const [sidebarOpen, setSidebarOpen] = useState(false)
    const location = useLocation()

    const navigation = [
        { name: '监控面板', href: '/dashboard', icon: BarChart3, current: location.pathname === '/dashboard' || location.pathname === '/' },
        { name: '服务来源', href: '/service-sources', icon: Database, current: location.pathname === '/service-sources' },
        { name: '服务列表', href: '/service-list', icon: List, current: location.pathname === '/service-list' },
        { name: '路由配置', href: '/route-config', icon: Route, current: location.pathname === '/route-config' },
        { name: '域名管理', href: '/domain-management', icon: Globe, current: location.pathname === '/domain-management' },
        { name: '证书管理', href: '/certificate-management', icon: Shield, current: location.pathname === '/certificate-management' },
        { name: '本地模型', href: '/local-models', icon: Cpu, current: location.pathname === '/local-models' },
    ]

    return (
        <div className="min-h-screen bg-gray-50">
            {/* Mobile sidebar */}
            <div className={`fixed inset-0 z-40 lg:hidden ${sidebarOpen ? '' : 'hidden'}`}>
                <div className="fixed inset-0 bg-gray-600 bg-opacity-75" onClick={() => setSidebarOpen(false)}></div>
                <div className="relative flex w-full max-w-xs flex-1 flex-col bg-white">
                    <div className="absolute top-0 right-0 -mr-12 pt-2">
                        <button
                            type="button"
                            className="ml-1 flex h-10 w-10 items-center justify-center rounded-full focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
                            onClick={() => setSidebarOpen(false)}
                        >
                            <X className="h-6 w-6 text-white" />
                        </button>
                    </div>
                    <div className="h-0 flex-1 overflow-y-auto pt-5 pb-4">
                        <div className="flex flex-shrink-0 items-center px-4">
                            <h1 className="text-xl font-bold text-gray-900">AI Gateway</h1>
                        </div>
                        <nav className="mt-8 space-y-1 px-2">
                            {navigation.map((item) => (
                                <Link
                                    key={item.name}
                                    to={item.href}
                                    className={`group flex items-center px-2 py-2 text-base font-medium rounded-md ${item.current
                                        ? 'bg-blue-100 text-blue-900'
                                        : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                                        }`}
                                >
                                    <item.icon className="mr-4 h-6 w-6" />
                                    {item.name}
                                </Link>
                            ))}
                        </nav>
                    </div>
                </div>
            </div>

            {/* Desktop sidebar */}
            <div className="hidden lg:fixed lg:inset-y-0 lg:flex lg:w-64 lg:flex-col">
                <div className="flex min-h-0 flex-1 flex-col bg-white border-r border-gray-200">
                    <div className="flex h-16 flex-shrink-0 items-center px-4 border-b border-gray-200">
                        <h1 className="text-xl font-bold text-gray-900">AI Gateway</h1>
                    </div>
                    <div className="flex flex-1 flex-col overflow-y-auto pt-8">
                        <nav className="flex-1 space-y-1 px-2 pb-4">
                            {navigation.map((item) => (
                                <Link
                                    key={item.name}
                                    to={item.href}
                                    className={`group flex items-center px-2 py-2 text-sm font-medium rounded-md ${item.current
                                        ? 'bg-blue-100 text-blue-900'
                                        : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                                        }`}
                                >
                                    <item.icon className="mr-3 h-5 w-5" />
                                    {item.name}
                                </Link>
                            ))}
                        </nav>
                    </div>
                </div>
            </div>

            {/* Main content */}
            <div className="lg:pl-64">
                {/* Mobile header */}
                <div className="flex h-16 items-center gap-x-4 border-b border-gray-200 bg-white px-4 shadow-sm lg:hidden">
                    <button
                        type="button"
                        className="-m-2.5 p-2.5 text-gray-700 lg:hidden"
                        onClick={() => setSidebarOpen(true)}
                    >
                        <Menu className="h-6 w-6" />
                    </button>
                    <div className="h-6 w-px bg-gray-200" />
                    <h1 className="text-lg font-semibold text-gray-900">AI Gateway</h1>
                </div>

                {/* Page content */}
                <main className="py-10">
                    <div className="px-4 sm:px-6 lg:px-8">
                        {children}
                    </div>
                </main>
            </div>
        </div>
    )
}

export default Layout
