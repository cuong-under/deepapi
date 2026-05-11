import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit2, Mail, Phone, User, Loader2 } from 'lucide-react'
import clsx from 'clsx'

export default function MultiUserAccounts({ token, user, onMessage }) {
    const [accounts, setAccounts] = useState([])
    const [loading, setLoading] = useState(true)
    const [showAddModal, setShowAddModal] = useState(false)
    const [editingAccount, setEditingAccount] = useState(null)

    const fetchAccounts = async () => {
        try {
            const res = await fetch('/api/user/accounts', {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })

            if (res.ok) {
                const data = await res.json()
                setAccounts(data.accounts || [])
            } else {
                onMessage('error', 'Không thể tải danh sách accounts')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchAccounts()
    }, [token])

    const handleDelete = async (id) => {
        if (!confirm('Bạn có chắc muốn xóa account này?')) return

        try {
            const res = await fetch(`/api/user/accounts/${id}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })

            if (res.ok) {
                onMessage('success', 'Đã xóa account')
                fetchAccounts()
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Xóa thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        }
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
                    <h2 className="text-2xl font-bold">DeepSeek Accounts</h2>
                    <p className="text-sm text-muted-foreground mt-1">
                        Quản lý các tài khoản DeepSeek của bạn
                    </p>
                </div>
                <button
                    onClick={() => setShowAddModal(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors font-medium text-sm"
                >
                    <Plus className="w-4 h-4" />
                    <span>Thêm Account</span>
                </button>
            </div>

            {/* Accounts List */}
            {accounts.length === 0 ? (
                <div className="text-center py-12 bg-card border border-border rounded-xl">
                    <User className="w-12 h-12 mx-auto text-muted-foreground/50 mb-4" />
                    <p className="text-muted-foreground">Chưa có account nào</p>
                    <button
                        onClick={() => setShowAddModal(true)}
                        className="mt-4 text-sm text-primary hover:underline"
                    >
                        Thêm account đầu tiên
                    </button>
                </div>
            ) : (
                <div className="grid gap-4">
                    {accounts.map(account => (
                        <div
                            key={account.id}
                            className="bg-card border border-border rounded-xl p-6 hover:border-primary/50 transition-colors"
                        >
                            <div className="flex items-start justify-between">
                                <div className="flex-1">
                                    <div className="flex items-center gap-3 mb-3">
                                        <h3 className="text-lg font-semibold">
                                            {account.name || 'Unnamed Account'}
                                        </h3>
                                        {user.role === 'admin' && (
                                            <span className="px-2 py-0.5 text-xs bg-secondary text-muted-foreground rounded">
                                                User ID: {account.user_id}
                                            </span>
                                        )}
                                    </div>

                                    {account.remark && (
                                        <p className="text-sm text-muted-foreground mb-3">
                                            {account.remark}
                                        </p>
                                    )}

                                    <div className="flex flex-wrap gap-4 text-sm">
                                        {account.email && (
                                            <div className="flex items-center gap-2 text-muted-foreground">
                                                <Mail className="w-4 h-4" />
                                                <span>{account.email}</span>
                                            </div>
                                        )}
                                        {account.mobile && (
                                            <div className="flex items-center gap-2 text-muted-foreground">
                                                <Phone className="w-4 h-4" />
                                                <span>{account.mobile}</span>
                                            </div>
                                        )}
                                    </div>

                                    <div className="mt-3 text-xs text-muted-foreground">
                                        Tạo lúc: {new Date(account.created_at * 1000).toLocaleString('vi-VN')}
                                    </div>
                                </div>

                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => setEditingAccount(account)}
                                        className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
                                    >
                                        <Edit2 className="w-4 h-4" />
                                    </button>
                                    <button
                                        onClick={() => handleDelete(account.id)}
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
            {(showAddModal || editingAccount) && (
                <AccountModal
                    token={token}
                    account={editingAccount}
                    onClose={() => {
                        setShowAddModal(false)
                        setEditingAccount(null)
                    }}
                    onSuccess={() => {
                        setShowAddModal(false)
                        setEditingAccount(null)
                        fetchAccounts()
                    }}
                    onMessage={onMessage}
                />
            )}
        </div>
    )
}

function AccountModal({ token, account, onClose, onSuccess, onMessage }) {
    const [formData, setFormData] = useState({
        name: account?.name || '',
        remark: account?.remark || '',
        email: account?.email || '',
        mobile: account?.mobile || '',
        password: account?.password || '',
        proxy_id: account?.proxy_id || ''
    })
    const [loading, setLoading] = useState(false)

    const handleSubmit = async (e) => {
        e.preventDefault()

        if (!formData.email && !formData.mobile) {
            onMessage('error', 'Phải có ít nhất email hoặc mobile')
            return
        }

        setLoading(true)

        try {
            const url = account
                ? `/api/user/accounts/${account.id}`
                : '/api/user/accounts'

            const res = await fetch(url, {
                method: account ? 'PUT' : 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(formData)
            })

            if (res.ok) {
                onMessage('success', account ? 'Đã cập nhật account' : 'Đã thêm account')
                onSuccess()
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

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
            <div className="bg-card border border-border rounded-xl p-6 w-full max-w-md">
                <h3 className="text-xl font-bold mb-4">
                    {account ? 'Chỉnh sửa Account' : 'Thêm Account'}
                </h3>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">Tên</label>
                        <input
                            type="text"
                            className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                            value={formData.name}
                            onChange={e => setFormData({ ...formData, name: e.target.value })}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2">Ghi chú</label>
                        <input
                            type="text"
                            className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                            value={formData.remark}
                            onChange={e => setFormData({ ...formData, remark: e.target.value })}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2">Email *</label>
                        <input
                            type="email"
                            className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                            value={formData.email}
                            onChange={e => setFormData({ ...formData, email: e.target.value })}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2">Mobile</label>
                        <input
                            type="text"
                            className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                            value={formData.mobile}
                            onChange={e => setFormData({ ...formData, mobile: e.target.value })}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2">Password DeepSeek</label>
                        <input
                            type="password"
                            className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                            value={formData.password}
                            onChange={e => setFormData({ ...formData, password: e.target.value })}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2">Proxy ID</label>
                        <input
                            type="text"
                            className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
                            value={formData.proxy_id}
                            onChange={e => setFormData({ ...formData, proxy_id: e.target.value })}
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
                            {loading ? 'Đang lưu...' : 'Lưu'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}
