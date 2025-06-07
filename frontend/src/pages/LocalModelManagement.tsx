import { useState, useEffect } from 'react';
import { RefreshCw, Play, Square, Sliders, Download, Zap } from 'lucide-react';

interface LocalModel {
    id: string;
    name: string;
    type: 'chat' | 'completion' | 'embedding';
    size: 'small' | 'medium' | 'large';
    status: 'running' | 'stopped' | 'loading';
    description: string;
}

const LocalModelManagement = () => {
    const [models, setModels] = useState<LocalModel[]>([
        {
            id: 'tiny-llama',
            name: 'TinyLlama Chat',
            type: 'chat',
            size: 'small',
            status: 'stopped',
            description: '轻量对话模型，仅1.1B参数，适合本地部署'
        },
        {
            id: 'phi-2',
            name: 'Phi-2',
            type: 'completion',
            size: 'small',
            status: 'stopped',
            description: '微软研发的高性能小模型，性能优秀'
        },
        {
            id: 'miniLM',
            name: 'MiniLM Embeddings',
            type: 'embedding',
            size: 'small',
            status: 'stopped',
            description: '文本向量嵌入模型，适合本地部署'
        },
        {
            id: 'mistral-7b',
            name: 'Mistral-7B',
            type: 'chat',
            size: 'large',
            status: 'stopped',
            description: '高性能开源大模型，7B参数'
        }
    ]);

    const [selectedModel, setSelectedModel] = useState<LocalModel | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [settings, setSettings] = useState({
        maxTokens: 1024,
        temperature: 0.7,
    });
    // Fetch models
    useEffect(() => {
        fetchModels();
    }, []);

    const fetchModels = async () => {
        try {
            setIsLoading(true);

            // Real API call to the backend
            const response = await fetch('/api/local/models');
            if (response.ok) {
                const data = await response.json();
                if (data.models && Array.isArray(data.models)) {
                    setModels(data.models);
                }
            } else {
                console.error('Error fetching models:', response.statusText);
            }

            setIsLoading(false);
        } catch (error) {
            console.error('Error fetching models:', error);
            setIsLoading(false);
        }
    };
    const toggleModelStatus = async (modelId: string) => {
        try {
            const model = models.find(m => m.id === modelId);
            if (!model) return;

            // Set loading state
            setModels(models.map(m => {
                if (m.id === modelId) {
                    return { ...m, status: 'loading' };
                }
                return m;
            }));

            // Call API to start or stop the model
            const action = model.status === 'running' ? 'stop' : 'start';
            const response = await fetch(`/api/local/models/${modelId}/${action}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                // Success - wait a bit then fetch updated status
                setTimeout(() => fetchModels(), 2000);
            } else {
                console.error(`Error ${action}ing model:`, response.statusText);
                // Revert to original status
                setModels(models.map(m => {
                    if (m.id === modelId) {
                        return model;
                    }
                    return m;
                }));
            }
        } catch (error) {
            console.error(`Error toggling model status:`, error);
        }
    };
    const downloadModel = async (modelId: string) => {
        try {
            const model = models.find(m => m.id === modelId);
            if (!model) return;

            // Set model to loading state
            setModels(models.map(m => {
                if (m.id === modelId) {
                    return { ...m, status: 'loading' };
                }
                return m;
            }));

            // Call API to download the model
            const response = await fetch(`/api/local/models/${modelId}/download`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                // Poll for status updates every 3 seconds
                const statusInterval = setInterval(async () => {
                    try {
                        const statusResponse = await fetch(`/api/local/models/${modelId}/status`);
                        if (statusResponse.ok) {
                            const statusData = await statusResponse.json();

                            // If status is no longer loading, clear interval and update models
                            if (statusData.status !== 'loading') {
                                clearInterval(statusInterval);
                                fetchModels();
                            }
                        }
                    } catch (error) {
                        console.error('Error checking model status:', error);
                    }
                }, 3000);

                // Set a timeout to stop polling after 5 minutes
                setTimeout(() => {
                    clearInterval(statusInterval);
                    fetchModels();
                }, 5 * 60 * 1000);
            } else {
                console.error(`Error downloading model:`, response.statusText);
                // Revert to original status
                setModels(models.map(m => {
                    if (m.id === modelId) {
                        return model;
                    }
                    return m;
                }));
            }
        } catch (error) {
            console.error(`Error downloading model:`, error);
        }
    };

    const openSettings = (model: LocalModel) => {
        setSelectedModel(model);
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'running': return 'bg-green-100 text-green-800';
            case 'stopped': return 'bg-gray-100 text-gray-800';
            case 'loading': return 'bg-blue-100 text-blue-800';
            default: return 'bg-gray-100 text-gray-800';
        }
    };

    return (
        <div className="container mx-auto py-6 px-4">
            <div className="flex justify-between items-center mb-6">
                <h1 className="text-2xl font-bold">本地模型管理</h1>
                <button
                    onClick={fetchModels}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-50 text-blue-700 rounded-md hover:bg-blue-100"
                    disabled={isLoading}
                >
                    <RefreshCw size={16} className={isLoading ? "animate-spin" : ""} />
                    刷新模型
                </button>
            </div>

            <div className="bg-white rounded-lg shadow overflow-hidden">
                <div className="px-6 py-4 border-b">
                    <h2 className="text-lg font-medium">可用模型</h2>
                </div>
                <div className="divide-y">
                    {models.map(model => (
                        <div key={model.id} className="p-6 flex items-center justify-between">
                            <div className="flex-1">
                                <div className="flex items-center gap-3">
                                    <h3 className="text-lg font-medium">{model.name}</h3>
                                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(model.status)}`}>
                                        {model.status === 'running' ? '运行中' : model.status === 'loading' ? '处理中...' : '已停止'}
                                    </span>
                                    <span className="px-2 py-1 bg-purple-100 text-purple-800 rounded-full text-xs font-medium">
                                        {model.type === 'chat' ? '对话' : model.type === 'completion' ? '补全' : '向量嵌入'}
                                    </span>
                                    <span className="px-2 py-1 bg-indigo-100 text-indigo-800 rounded-full text-xs font-medium">
                                        {model.size === 'small' ? '小型' : model.size === 'medium' ? '中型' : '大型'}
                                    </span>
                                </div>
                                <p className="text-gray-600 mt-1">{model.description}</p>
                            </div>
                            <div className="flex gap-2">
                                <button
                                    onClick={() => downloadModel(model.id)}
                                    className="p-2 text-gray-600 hover:text-blue-600 hover:bg-blue-50 rounded-md"
                                    disabled={model.status === 'loading' || model.status === 'running'}
                                    title="下载模型"
                                >
                                    <Download size={20} />
                                </button>
                                <button
                                    onClick={() => toggleModelStatus(model.id)}
                                    className={`p-2 rounded-md ${model.status === 'running'
                                            ? 'text-red-600 hover:bg-red-50'
                                            : 'text-green-600 hover:bg-green-50'
                                        }`}
                                    disabled={model.status === 'loading'}
                                    title={model.status === 'running' ? '停止模型' : '启动模型'}
                                >
                                    {model.status === 'running' ? <Square size={20} /> : <Play size={20} />}
                                </button>
                                <button
                                    onClick={() => openSettings(model)}
                                    className="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-md"
                                    disabled={model.status === 'loading'}
                                    title="模型设置"
                                >
                                    <Sliders size={20} />
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            <div className="mt-8 bg-white rounded-lg shadow">
                <div className="px-6 py-4 border-b">
                    <h2 className="text-lg font-medium">使用本地模型</h2>
                </div>
                <div className="p-6">
                    <p className="text-gray-600 mb-4">
                        本地模型提供与第三方 API 兼容的接口，您可以通过以下方式使用：
                    </p>
                    <div className="bg-gray-50 p-4 rounded-md mb-4">
                        <h3 className="font-medium mb-2 flex items-center gap-2">
                            <Zap size={16} className="text-yellow-500" />
                            对话接口
                        </h3>
                        <code className="block text-sm bg-gray-100 p-2 rounded">
                            POST /local/chat/completions
                        </code>
                    </div>
                    <div className="bg-gray-50 p-4 rounded-md mb-4">
                        <h3 className="font-medium mb-2 flex items-center gap-2">
                            <Zap size={16} className="text-yellow-500" />
                            补全接口
                        </h3>
                        <code className="block text-sm bg-gray-100 p-2 rounded">
                            POST /local/completions
                        </code>
                    </div>
                    <div className="bg-gray-50 p-4 rounded-md">
                        <h3 className="font-medium mb-2 flex items-center gap-2">
                            <Zap size={16} className="text-yellow-500" />
                            向量嵌入接口
                        </h3>
                        <code className="block text-sm bg-gray-100 p-2 rounded">
                            POST /local/embeddings
                        </code>
                    </div>
                </div>
            </div>

            {selectedModel && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4">
                    <div className="bg-white rounded-lg shadow-lg w-full max-w-md">
                        <div className="px-6 py-4 border-b flex justify-between items-center">
                            <h3 className="text-lg font-medium">模型设置: {selectedModel.name}</h3>
                            <button
                                onClick={() => setSelectedModel(null)}
                                className="text-gray-500 hover:text-gray-700"
                            >
                                ✕
                            </button>
                        </div>
                        <div className="p-6">
                            <div className="mb-4">
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    最大令牌数
                                </label>
                                <input
                                    type="number"
                                    value={settings.maxTokens}
                                    onChange={e => setSettings({ ...settings, maxTokens: parseInt(e.target.value) })}
                                    className="w-full border border-gray-300 rounded-md px-3 py-2"
                                    min="1"
                                    max="4096"
                                />
                                <p className="mt-1 text-sm text-gray-500">
                                    生成文本的最大长度限制
                                </p>
                            </div>
                            <div className="mb-4">
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    温度 ({settings.temperature})
                                </label>
                                <input
                                    type="range"
                                    value={settings.temperature}
                                    onChange={e => setSettings({ ...settings, temperature: parseFloat(e.target.value) })}
                                    className="w-full"
                                    min="0"
                                    max="2"
                                    step="0.1"
                                />
                                <p className="mt-1 text-sm text-gray-500">
                                    较低的值使输出更确定，较高的值使输出更随机
                                </p>
                            </div>
                            <div className="flex justify-end gap-2 mt-6">
                                <button
                                    onClick={() => setSelectedModel(null)}
                                    className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                                >
                                    取消
                                </button>
                                <button onClick={() => {
                                    // Save the settings to the server
                                    if (selectedModel) {
                                        fetch(`/api/local/models/${selectedModel.id}/settings`, {
                                            method: 'POST',
                                            headers: {
                                                'Content-Type': 'application/x-www-form-urlencoded',
                                            },
                                            body: new URLSearchParams({
                                                maxTokens: settings.maxTokens.toString(),
                                                temperature: settings.temperature.toString()
                                            })
                                        })
                                            .then(response => {
                                                if (response.ok) {
                                                    alert('设置已保存');
                                                } else {
                                                    alert('保存设置失败');
                                                }
                                                setSelectedModel(null);
                                            })
                                            .catch(error => {
                                                console.error('Error saving settings:', error);
                                                alert('保存设置失败');
                                                setSelectedModel(null);
                                            });
                                    }
                                }}
                                    className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                                >
                                    保存设置
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default LocalModelManagement;
