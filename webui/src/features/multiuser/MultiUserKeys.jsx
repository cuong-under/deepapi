import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit2, Key, Copy, Loader2, Eye, EyeOff } from 'lucide-react'
import clsx from 'clsx'

export default function MultiUserKeys({ token, user, onMessage }) {
    const [keys, setKeys] = useState([])
    const [loading, setLoading] = useState(true)
    const [showAddModal, setShowAddModal] = useState(false)
    const [editingKey, setEditingKey] = useState(null)
    const [visibleKeys, setVisibleKeys] = useState({})

    const fetchKeys = async () => {
        try {
            const res = await fetch('/api/user/keys', {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })

            if (res.ok) {
                const data = await res.json()
                setKeys(data.keys || [])
            } else {
                onMessage('error', 'Không thể tải danh sách API keys')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchKeys()
    }, [token])

    const handleDelete = async (id) => {
        if (!confirm('Bạn có chắc muốn xóa API key này?')) return

        try {
            const res = await fetch(`/api/user/keys/${id}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })

            if (res.ok) {
                onMessage('success', 'Đã xóa API key')
                fetchKeys()
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Xóa thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        }
    }

    const copyToClipboard = (text) => {
        navigator.clipboard.writeText(text)
        onMessage('success', 'Đã copy API key')
    }

    const toggleKeyVisibility = (id) => {
        setVisibleKeys(prev => ({
            ...prev,
            [id]: !prev[id]
        }))
    }

    const maskKey = (key) => {
        if (!key) return ''
        const prefix = key.substring(0, 8)
        const suffix = key.substring(key.length - 4)
        return `${prefix}${'*'.repeat(20)}${suffix}`
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 className="w-6 h-6 animate-spin text-primary" />
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold">API Keys</h2>
                    <p className="text-sm text-muted-foreground mt-1">
                        Quản lý các API keys để truy cập DeepAPI
                    </p>
                </div>
                <button
                    onClick={() => setShowAddModal(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors font-medium text-sm"
                >
                    <Plus className="w-4 h-4" />
                    <span>Tạo API Key</span>
                </button>
            </div>

            {/* Keys List */}
            {keys.length === 0 ? (
                <div className="text-center py-12 bg-card border border-border rounded-xl">
                    <Key className="w-12 h-12 mx-auto text-muted-foreground/50 mb-4" />
                    <p className="text-muted-foreground">Chưa có API key nào</p>
                    <button
                        onClick={() => setShowAddModal(true)}
                        className="mt-4 text-sm text-primary hover:underline"
                    >
                        Tạo API key đầu tiên
                    </button>
                </div>
            ) : (
                <div className="grid gap-4">
                    {keys.map(key => (
                        <div
                            key={key.id}
                            className="bg-card border border-border rounded-xl p-6 hover:border-primary/50 transition-colors"
                        >
                            <div className="flex items-start justify-between">
                                <div className="flex-1">
                                    <div className="flex items-center gap-3 mb-3">
                                        <h3 className="text-lg font-semibold">
                                            {key.name || 'Unnamed Key'}
                                        </h3>
                                        {user.role === 'admin' && (
                                            <span className="px-2 py-0.5 text-xs bg-secondary text-muted-foreground rounded">
                                                User ID: {key.user_id}
                                            </span>
                                        )}
                                    </div>

                                    {key.remark && (
                                        <p className="text-sm text-muted-foreground mb-3">
                                            {key.remark}
                                        </p>
                                    )}

                                    <div className="flex items-center gap-2 mb-3">
                                        <code className="flex-1 px-3 py-2 bg-background border border-border rounded-lg text-sm font-mono">
                                            {visibleKeys[key.id] ? key.api_key : maskKey(key.api_key)}
                                        </code>
                                        <button
                                            onClick={() => toggleKeyVisibility(key.id)}
                                            className="p-2 text-muted-foreground hover:text-foreground hover:bg-secondary/50 rounded-lg transition-colors"
                                            title={visibleKeys[key.id] ? 'Ẩn key' : 'Hiện key'}
                                        >
                                            {visibleKeys[key.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                        </button>
                                        <button
                                            onClick={() => copyToClipboard(key.api_key)}
                                            className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
                                            title="Copy key"
                                        >
                                            <Copy className="w-4 h-4" />
                                        </button>
                                    </div>

                                    <div className="text-xs text-muted-foreground">
                                        Tạo lúc: {new Date(key.created_at * 1000).toLocaleString('vi-VN')}
                                    </div>
                                </div>

                                <div className="flex items-center gap-2 ml-4">
                                    <button
                                        onClick={() => setEditingKey(key)}
                                        className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
                                    >
                                        <Edit2 className="w-4 h-4" />
                                    </button>
                                    <button
                                        onClick={() => handleDelete(key.id)}
                                        className="p-2 text-muted-foreground hover:text-red-500 hover:bg-red-500/10 rounded-lg transition-colors"
                                    >
                                        <Trash2 className="w-4 h-4" />
                                    </button>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* Add/Edit Modal */}
            {(showAddModal || editingKey) && (
                <KeyModal
                    token={token}
                    apiKey={editingKey}
                    onClose={() => {
                        setShowAddModal(false)
                        setEditingKey(null)
                    }}
                    onSuccess={() => {
                        setShowAddModal(false)
                        setEditingKey(null)
                        fetchKeys()
                    }}
                    onMessage={onMessage}
                />
            )}
        </div>
    )
}

function KeyModal({ token, apiKey, onClose, onSuccess, onMessage }) {
    const [formData, setFormData] = useState({
        name: apiKey?.name || '',
        remark: apiKey?.remark || ''
    })
    const [loading, setLoading] = useState(false)
    const [newKey, setNewKey] = useState(null)

    const handleSubmit = async (e) => {
        e.preventDefault()

        setLoading(true)

        try {
            const url = apiKey
                ? `/api/user/keys/${apiKey.id}`
                : '/api/user/keys'

            const res = await fetch(url, {
                method: apiKey ? 'PUT' : 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(formData)
            })

            if (res.ok) {
                const data = await res.json()
                if (!apiKey && data.api_key) {
                    // Show new key
                    setNewKey(data.api_key)
                    onMessage('success', 'Đã tạo API key mới')
                } else {
                    onMessage('success', 'Đã cập nhật API key')
                    onSuccess()
                }
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Thao tác thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const copyNewKey = () => {
        navigator.clipboard.writeText(newKey)
        onMessage('success', 'Đã copy API key')
    }

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
            <div className="bg-card border border-border rounded-xl p-6 w-full max-w-md">
                <h3 className="text-xl font-bold mb-4">
                    {apiKey ? 'Chỉnh sửa API Key' : 'Tạo API Key'}
                </h3>

                {newKey ? (
                    <div className="space-y-4">
                        <div className="p-4 bg-green-500/10 border border-green-500/20 rounded-lg">
                            <p className="text-sm text-green-600 dark:text-green-400 font-medium mb-2">
                                ✓ API Key đã được tạo thành công!
                            </p>
                            <p className="text-xs text-muted-foreground mb-3">
                                Hãy copy và lưu key này. Bạn sẽ không thể xem lại key này sau khi đóng cửa sổ.
                            </p>
                            <div className="flex items-center gap-2">
                                <code className="flex-1 px-3 py-2 bg-background border border-border rounded-lg text-xs font-mono break-all">
                                    {newKey}
                                </code>
                                <button
                                    onClick={copyNewKey}
                                    className="p-2 text-primary hover:bg-primary/10 rounded-lg transition-colors"
                                >
                                    <Copy className="w-4 h-4" />
                                </button>
                            </div>
                        </div>

                        <button
                            onClick={onSuccess}
                            className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                        >
                            Đóng
                        </button>
                    </div>
                ) : (
                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium mb-2">Tên *</label>
                            <input
                                type="text"
                                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                                value={formData.name}
                                onChange={e => setFormData({ ...formData, name: e.target.value })}
                                required
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium mb-2">Ghi chú</label>
                            <textarea
                                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                                rows={3}
                                value={formData.remark}
                                onChange={e => setFormData({ ...formData, remark: e.target.value })}
                            />
                        </div>

                        <div className="flex gap-3 pt-4">
                            <button
                                type="button"
                                onClick={onClose}
                                className="flex-1 px-4 py-2 border border-border rounded-lg hover:bg-secondary/50 transition-colors"
                            >
                                Hủy
                            </button>
                            <button
                                type="submit"
                                disabled={loading}
                                className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                            >
                                {loading ? 'Đang lưu...' : apiKey ? 'Cập nhật' : 'Tạo'}
                            </button>
                        </div>
                    </form>
                )}
            </div>
        </div>
    )
}
