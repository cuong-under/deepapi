import { useEffect, useState } from 'react'
import { Users, Plus, Edit2, Trash2, Key, Shield, User as UserIcon } from 'lucide-react'

export default function UserManagementContainer({ onMessage, authFetch }) {
    const [users, setUsers] = useState([])
    const [loading, setLoading] = useState(false)
    const [showAddUser, setShowAddUser] = useState(false)
    const [editingUser, setEditingUser] = useState(null)
    const [showChangePassword, setShowChangePassword] = useState(null)
    const [newUser, setNewUser] = useState({ username: '', email: '', password: '', role: 'user' })
    const [editUser, setEditUser] = useState({ username: '', email: '', role: '' })
    const [newPassword, setNewPassword] = useState('')

    const apiFetch = authFetch || fetch

    const fetchUsers = async () => {
        setLoading(true)
        try {
            const res = await apiFetch('/api/admin/users')
            if (res.ok) {
                const data = await res.json()
                setUsers(data.users || [])
            } else {
                onMessage('error', 'Không thể tải danh sách users')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchUsers()
    }, [])

    const handleAddUser = async (e) => {
        e.preventDefault()
        if (!newUser.username || !newUser.email || !newUser.password) {
            onMessage('error', 'Vui lòng điền đầy đủ thông tin')
            return
        }
        setLoading(true)
        try {
            const res = await apiFetch('/api/admin/users', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newUser)
            })
            if (res.ok) {
                onMessage('success', 'Tạo user thành công')
                setShowAddUser(false)
                setNewUser({ username: '', email: '', password: '', role: 'user' })
                fetchUsers()
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Tạo user thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const handleUpdateUser = async (e) => {
        e.preventDefault()
        if (!editUser.username || !editUser.email) {
            onMessage('error', 'Username và email không được để trống')
            return
        }
        setLoading(true)
        try {
            const res = await apiFetch(`/api/admin/users/${editingUser.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(editUser)
            })
            if (res.ok) {
                onMessage('success', 'Cập nhật user thành công')
                setEditingUser(null)
                setEditUser({ username: '', email: '', role: '' })
                fetchUsers()
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Cập nhật thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const handleChangePassword = async (e) => {
        e.preventDefault()
        if (newPassword.length < 8) {
            onMessage('error', 'Password phải có ít nhất 8 ký tự')
            return
        }
        setLoading(true)
        try {
            const res = await apiFetch(`/api/admin/users/${showChangePassword.id}/password`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password: newPassword })
            })
            if (res.ok) {
                onMessage('success', 'Đổi password thành công')
                setShowChangePassword(null)
                setNewPassword('')
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Đổi password thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const handleDeleteUser = async (user) => {
        if (!confirm(`Xóa user "${user.username}"? Tất cả accounts và keys của user này cũng sẽ bị xóa.`)) return
        setLoading(true)
        try {
            const res = await apiFetch(`/api/admin/users/${user.id}`, {
                method: 'DELETE'
            })
            if (res.ok) {
                onMessage('success', 'Xóa user thành công')
                fetchUsers()
            } else {
                const data = await res.json()
                onMessage('error', data.error || 'Xóa user thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-foreground">Quản lý Users</h2>
                    <p className="text-sm text-muted-foreground mt-1">Quản lý tài khoản người dùng hệ thống</p>
                </div>
                <button
                    onClick={() => setShowAddUser(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                >
                    <Plus className="w-4 h-4" />
                    Thêm User
                </button>
            </div>

            {/* Users Table */}
            <div className="bg-card rounded-lg border border-border overflow-hidden">
                <table className="w-full">
                    <thead className="bg-muted/50">
                        <tr>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Username</th>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Email</th>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Role</th>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Accounts</th>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Keys</th>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Sessions</th>
                            <th className="text-left px-4 py-3 text-sm font-semibold text-foreground">Created</th>
                            <th className="text-right px-4 py-3 text-sm font-semibold text-foreground">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {users.map((user) => (
                            <tr key={user.id} className="border-t border-border hover:bg-muted/30">
                                <td className="px-4 py-3">
                                    <div className="flex items-center gap-2">
                                        {user.role === 'admin' ? (
                                            <Shield className="w-4 h-4 text-amber-500" />
                                        ) : (
                                            <UserIcon className="w-4 h-4 text-muted-foreground" />
                                        )}
                                        <span className="font-medium text-foreground">{user.username}</span>
                                    </div>
                                </td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">{user.email}</td>
                                <td className="px-4 py-3">
                                    <span className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${
                                        user.role === 'admin'
                                            ? 'bg-amber-500/10 text-amber-500'
                                            : 'bg-blue-500/10 text-blue-500'
                                    }`}>
                                        {user.role}
                                    </span>
                                </td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">{user.accounts_count}</td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">{user.keys_count}</td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">{user.sessions_count}</td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">
                                    {new Date(user.created_at).toLocaleDateString('vi-VN')}
                                </td>
                                <td className="px-4 py-3">
                                    <div className="flex items-center justify-end gap-2">
                                        <button
                                            onClick={() => {
                                                setEditingUser(user)
                                                setEditUser({ username: user.username, email: user.email, role: user.role })
                                            }}
                                            className="p-1.5 hover:bg-muted rounded transition-colors"
                                            title="Sửa"
                                        >
                                            <Edit2 className="w-4 h-4 text-muted-foreground" />
                                        </button>
                                        <button
                                            onClick={() => setShowChangePassword(user)}
                                            className="p-1.5 hover:bg-muted rounded transition-colors"
                                            title="Đổi password"
                                        >
                                            <Key className="w-4 h-4 text-muted-foreground" />
                                        </button>
                                        <button
                                            onClick={() => handleDeleteUser(user)}
                                            className="p-1.5 hover:bg-destructive/10 rounded transition-colors"
                                            title="Xóa"
                                        >
                                            <Trash2 className="w-4 h-4 text-destructive" />
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
                {users.length === 0 && !loading && (
                    <div className="text-center py-12 text-muted-foreground">
                        <Users className="w-12 h-12 mx-auto mb-3 opacity-50" />
                        <p>Chưa có user nào</p>
                    </div>
                )}
            </div>

            {/* Add User Modal */}
            {showAddUser && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                    <div className="bg-card rounded-lg p-6 w-full max-w-md border border-border">
                        <h3 className="text-lg font-semibold mb-4">Thêm User Mới</h3>
                        <form onSubmit={handleAddUser} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-1">Username</label>
                                <input
                                    type="text"
                                    value={newUser.username}
                                    onChange={(e) => setNewUser({ ...newUser, username: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1">Email</label>
                                <input
                                    type="email"
                                    value={newUser.email}
                                    onChange={(e) => setNewUser({ ...newUser, email: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1">Password</label>
                                <input
                                    type="password"
                                    value={newUser.password}
                                    onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                    required
                                    minLength={8}
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1">Role</label>
                                <select
                                    value={newUser.role}
                                    onChange={(e) => setNewUser({ ...newUser, role: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                >
                                    <option value="user">User</option>
                                    <option value="admin">Admin</option>
                                </select>
                            </div>
                            <div className="flex gap-2 pt-2">
                                <button
                                    type="submit"
                                    disabled={loading}
                                    className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50"
                                >
                                    Tạo User
                                </button>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowAddUser(false)
                                        setNewUser({ username: '', email: '', password: '', role: 'user' })
                                    }}
                                    className="px-4 py-2 bg-muted text-foreground rounded-lg hover:bg-muted/80"
                                >
                                    Hủy
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {/* Edit User Modal */}
            {editingUser && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                    <div className="bg-card rounded-lg p-6 w-full max-w-md border border-border">
                        <h3 className="text-lg font-semibold mb-4">Sửa User</h3>
                        <form onSubmit={handleUpdateUser} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-1">Username</label>
                                <input
                                    type="text"
                                    value={editUser.username}
                                    onChange={(e) => setEditUser({ ...editUser, username: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1">Email</label>
                                <input
                                    type="email"
                                    value={editUser.email}
                                    onChange={(e) => setEditUser({ ...editUser, email: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1">Role</label>
                                <select
                                    value={editUser.role}
                                    onChange={(e) => setEditUser({ ...editUser, role: e.target.value })}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                >
                                    <option value="user">User</option>
                                    <option value="admin">Admin</option>
                                </select>
                            </div>
                            <div className="flex gap-2 pt-2">
                                <button
                                    type="submit"
                                    disabled={loading}
                                    className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50"
                                >
                                    Cập nhật
                                </button>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setEditingUser(null)
                                        setEditUser({ username: '', email: '', role: '' })
                                    }}
                                    className="px-4 py-2 bg-muted text-foreground rounded-lg hover:bg-muted/80"
                                >
                                    Hủy
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {/* Change Password Modal */}
            {showChangePassword && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                    <div className="bg-card rounded-lg p-6 w-full max-w-md border border-border">
                        <h3 className="text-lg font-semibold mb-4">Đổi Password - {showChangePassword.username}</h3>
                        <form onSubmit={handleChangePassword} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-1">Password Mới</label>
                                <input
                                    type="password"
                                    value={newPassword}
                                    onChange={(e) => setNewPassword(e.target.value)}
                                    className="w-full px-3 py-2 bg-background border border-border rounded-lg"
                                    required
                                    minLength={8}
                                    placeholder="Tối thiểu 8 ký tự"
                                />
                            </div>
                            <div className="flex gap-2 pt-2">
                                <button
                                    type="submit"
                                    disabled={loading}
                                    className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50"
                                >
                                    Đổi Password
                                </button>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowChangePassword(null)
                                        setNewPassword('')
                                    }}
                                    className="px-4 py-2 bg-muted text-foreground rounded-lg hover:bg-muted/80"
                                >
                                    Hủy
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    )
}
