import { useEffect, useMemo, useState } from 'react'
import {
    ChevronLeft,
    ChevronRight,
    Edit2,
    Key,
    Loader2,
    Plus,
    RefreshCw,
    Search,
    Shield,
    Trash2,
    User as UserIcon,
    Users,
    X,
} from 'lucide-react'

const PAGE_SIZE = 25

const emptyNewUser = { username: '', email: '', password: '', role: 'user' }
const emptyEditUser = { username: '', email: '', role: '' }

export default function UserManagementContainer({ onMessage, authFetch, currentUser }) {
    const [users, setUsers] = useState([])
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const [query, setQuery] = useState('')
    const [roleFilter, setRoleFilter] = useState('')
    const [loading, setLoading] = useState(false)
    const [showAddUser, setShowAddUser] = useState(false)
    const [editingUser, setEditingUser] = useState(null)
    const [showChangePassword, setShowChangePassword] = useState(null)
    const [deleteTarget, setDeleteTarget] = useState(null)
    const [newUser, setNewUser] = useState(emptyNewUser)
    const [editUser, setEditUser] = useState(emptyEditUser)
    const [newPassword, setNewPassword] = useState('')

    const apiFetch = authFetch || fetch
    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
    const pageStats = useMemo(() => ({
        admins: users.filter((user) => user.role === 'admin').length,
        regular: users.filter((user) => user.role === 'user').length,
        sessions: users.reduce((sum, user) => sum + (user.sessions_count || 0), 0),
    }), [users])

    const fetchUsers = async () => {
        setLoading(true)
        try {
            const params = new URLSearchParams({
                page: String(page),
                page_size: String(PAGE_SIZE),
            })
            if (query.trim()) params.set('q', query.trim())
            if (roleFilter) params.set('role', roleFilter)

            const res = await apiFetch(`/api/admin/users?${params.toString()}`)
            const data = await res.json().catch(() => ({}))
            if (res.ok) {
                setUsers(data.users || [])
                setTotal(data.total || 0)
            } else {
                onMessage('error', data.error || 'Không thể tải danh sách người dùng')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchUsers()
    }, [page, query, roleFilter])

    const resetFilters = () => {
        setQuery('')
        setRoleFilter('')
        setPage(1)
    }

    const handleAddUser = async (e) => {
        e.preventDefault()
        setLoading(true)
        try {
            const res = await apiFetch('/api/admin/users', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newUser),
            })
            const data = await res.json().catch(() => ({}))
            if (res.ok) {
                onMessage('success', 'Tạo người dùng thành công')
                setShowAddUser(false)
                setNewUser(emptyNewUser)
                setPage(1)
                fetchUsers()
            } else {
                onMessage('error', data.error || 'Tạo người dùng thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const handleUpdateUser = async (e) => {
        e.preventDefault()
        setLoading(true)
        try {
            const res = await apiFetch(`/api/admin/users/${editingUser.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(editUser),
            })
            const data = await res.json().catch(() => ({}))
            if (res.ok) {
                onMessage('success', 'Cập nhật người dùng thành công')
                setEditingUser(null)
                setEditUser(emptyEditUser)
                fetchUsers()
            } else {
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
        setLoading(true)
        try {
            const res = await apiFetch(`/api/admin/users/${showChangePassword.id}/password`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password: newPassword }),
            })
            const data = await res.json().catch(() => ({}))
            if (res.ok) {
                onMessage('success', 'Đổi mật khẩu thành công')
                setShowChangePassword(null)
                setNewPassword('')
                fetchUsers()
            } else {
                onMessage('error', data.error || 'Đổi mật khẩu thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const handleDeleteUser = async () => {
        if (!deleteTarget) return
        setLoading(true)
        try {
            const res = await apiFetch(`/api/admin/users/${deleteTarget.id}`, { method: 'DELETE' })
            const data = await res.json().catch(() => ({}))
            if (res.ok) {
                onMessage('success', 'Xóa người dùng thành công')
                setDeleteTarget(null)
                fetchUsers()
            } else {
                onMessage('error', data.error || 'Xóa người dùng thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const openEdit = (user) => {
        setEditingUser(user)
        setEditUser({ username: user.username, email: user.email, role: user.role })
    }

    const isCurrentUser = (user) => currentUser?.id === user.id
    const canDelete = (user) => !isCurrentUser(user)

    return (
        <div className="space-y-5">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-foreground">Quản lý người dùng</h2>
                    <p className="mt-1 text-sm text-muted-foreground">Tạo tài khoản, phân quyền, đặt lại mật khẩu và kiểm soát phiên đăng nhập.</p>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={fetchUsers}
                        disabled={loading}
                        className="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-border bg-card hover:bg-muted disabled:opacity-50"
                        title="Tải lại"
                    >
                        <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                    </button>
                    <button
                        onClick={() => setShowAddUser(true)}
                        className="inline-flex h-10 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90"
                    >
                        <Plus className="h-4 w-4" />
                        Thêm người dùng
                    </button>
                </div>
            </div>

            <div className="grid gap-3 md:grid-cols-4">
                <StatCard label="Tổng phù hợp" value={total} />
                <StatCard label="Admin trang này" value={pageStats.admins} />
                <StatCard label="User trang này" value={pageStats.regular} />
                <StatCard label="Session trang này" value={pageStats.sessions} />
            </div>

            <div className="flex flex-col gap-3 rounded-lg border border-border bg-card p-3 lg:flex-row lg:items-center">
                <div className="relative flex-1">
                    <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                    <input
                        value={query}
                        onChange={(e) => {
                            setQuery(e.target.value)
                            setPage(1)
                        }}
                        placeholder="Tìm username hoặc email"
                        className="h-10 w-full rounded-lg border border-border bg-background pl-9 pr-3 text-sm outline-none focus:border-primary"
                    />
                </div>
                <select
                    value={roleFilter}
                    onChange={(e) => {
                        setRoleFilter(e.target.value)
                        setPage(1)
                    }}
                    className="h-10 rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
                >
                    <option value="">Tất cả role</option>
                    <option value="admin">Admin</option>
                    <option value="user">User</option>
                </select>
                {(query || roleFilter) && (
                    <button
                        onClick={resetFilters}
                        className="inline-flex h-10 items-center gap-2 rounded-lg border border-border px-3 text-sm hover:bg-muted"
                    >
                        <X className="h-4 w-4" />
                        Xóa lọc
                    </button>
                )}
            </div>

            <div className="overflow-hidden rounded-lg border border-border bg-card">
                <div className="overflow-x-auto">
                    <table className="w-full min-w-[920px]">
                        <thead className="bg-muted/50">
                            <tr>
                                <TableHead>Người dùng</TableHead>
                                <TableHead>Email</TableHead>
                                <TableHead>Role</TableHead>
                                <TableHead>Tài khoản</TableHead>
                                <TableHead>API keys</TableHead>
                                <TableHead>Sessions</TableHead>
                                <TableHead>Ngày tạo</TableHead>
                                <TableHead align="right">Thao tác</TableHead>
                            </tr>
                        </thead>
                        <tbody>
                            {users.map((user) => (
                                <tr key={user.id} className="border-t border-border hover:bg-muted/30">
                                    <td className="px-4 py-3">
                                        <div className="flex items-center gap-2">
                                            {user.role === 'admin' ? (
                                                <Shield className="h-4 w-4 text-amber-500" />
                                            ) : (
                                                <UserIcon className="h-4 w-4 text-muted-foreground" />
                                            )}
                                            <div className="min-w-0">
                                                <div className="flex items-center gap-2">
                                                    <span className="font-medium text-foreground">{user.username}</span>
                                                    {isCurrentUser(user) && (
                                                        <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-semibold text-primary">Bạn</span>
                                                    )}
                                                </div>
                                                <div className="text-xs text-muted-foreground">ID {user.id}</div>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-4 py-3 text-sm text-muted-foreground">{user.email}</td>
                                    <td className="px-4 py-3"><RoleBadge role={user.role} /></td>
                                    <td className="px-4 py-3 text-sm text-muted-foreground">{user.accounts_count}</td>
                                    <td className="px-4 py-3 text-sm text-muted-foreground">{user.keys_count}</td>
                                    <td className="px-4 py-3 text-sm text-muted-foreground">{user.sessions_count}</td>
                                    <td className="px-4 py-3 text-sm text-muted-foreground">{formatDate(user.created_at)}</td>
                                    <td className="px-4 py-3">
                                        <div className="flex items-center justify-end gap-2">
                                            <IconButton title="Sửa" onClick={() => openEdit(user)}>
                                                <Edit2 className="h-4 w-4" />
                                            </IconButton>
                                            <IconButton title="Đổi mật khẩu" onClick={() => setShowChangePassword(user)}>
                                                <Key className="h-4 w-4" />
                                            </IconButton>
                                            <IconButton
                                                title={canDelete(user) ? 'Xóa' : 'Không thể xóa chính mình'}
                                                onClick={() => canDelete(user) && setDeleteTarget(user)}
                                                disabled={!canDelete(user)}
                                                danger
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </IconButton>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                {users.length === 0 && !loading && (
                    <div className="py-12 text-center text-muted-foreground">
                        <Users className="mx-auto mb-3 h-12 w-12 opacity-50" />
                        <p>Không có người dùng phù hợp</p>
                    </div>
                )}

                {loading && users.length === 0 && (
                    <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        Đang tải người dùng
                    </div>
                )}

                <div className="flex items-center justify-between border-t border-border px-4 py-3 text-sm text-muted-foreground">
                    <span>Trang {page} / {totalPages}</span>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => setPage((value) => Math.max(1, value - 1))}
                            disabled={page <= 1 || loading}
                            className="inline-flex h-8 w-8 items-center justify-center rounded border border-border hover:bg-muted disabled:opacity-40"
                            title="Trang trước"
                        >
                            <ChevronLeft className="h-4 w-4" />
                        </button>
                        <button
                            onClick={() => setPage((value) => Math.min(totalPages, value + 1))}
                            disabled={page >= totalPages || loading}
                            className="inline-flex h-8 w-8 items-center justify-center rounded border border-border hover:bg-muted disabled:opacity-40"
                            title="Trang sau"
                        >
                            <ChevronRight className="h-4 w-4" />
                        </button>
                    </div>
                </div>
            </div>

            {showAddUser && (
                <UserFormModal
                    title="Thêm người dùng"
                    submitLabel="Tạo người dùng"
                    loading={loading}
                    user={newUser}
                    setUser={setNewUser}
                    onSubmit={handleAddUser}
                    onClose={() => {
                        setShowAddUser(false)
                        setNewUser(emptyNewUser)
                    }}
                    includePassword
                />
            )}

            {editingUser && (
                <UserFormModal
                    title={`Sửa ${editingUser.username}`}
                    submitLabel="Cập nhật"
                    loading={loading}
                    user={editUser}
                    setUser={setEditUser}
                    currentUser={currentUser}
                    editingUser={editingUser}
                    onSubmit={handleUpdateUser}
                    onClose={() => {
                        setEditingUser(null)
                        setEditUser(emptyEditUser)
                    }}
                />
            )}

            {showChangePassword && (
                <PasswordModal
                    user={showChangePassword}
                    password={newPassword}
                    setPassword={setNewPassword}
                    loading={loading}
                    onSubmit={handleChangePassword}
                    onClose={() => {
                        setShowChangePassword(null)
                        setNewPassword('')
                    }}
                />
            )}

            {deleteTarget && (
                <ConfirmDeleteModal
                    user={deleteTarget}
                    loading={loading}
                    onConfirm={handleDeleteUser}
                    onClose={() => setDeleteTarget(null)}
                />
            )}
        </div>
    )
}

function StatCard({ label, value }) {
    return (
        <div className="rounded-lg border border-border bg-card p-4">
            <div className="text-xs font-medium uppercase text-muted-foreground">{label}</div>
            <div className="mt-2 text-2xl font-bold text-foreground">{value}</div>
        </div>
    )
}

function TableHead({ children, align = 'left' }) {
    const alignClass = align === 'right' ? 'text-right' : 'text-left'
    return (
        <th className={`px-4 py-3 ${alignClass} text-sm font-semibold text-foreground`}>
            {children}
        </th>
    )
}

function RoleBadge({ role }) {
    return (
        <span className={`inline-flex items-center rounded px-2 py-1 text-xs font-medium ${
            role === 'admin'
                ? 'bg-amber-500/10 text-amber-500'
                : 'bg-blue-500/10 text-blue-500'
        }`}>
            {role}
        </span>
    )
}

function IconButton({ children, title, onClick, disabled, danger }) {
    return (
        <button
            onClick={onClick}
            disabled={disabled}
            className={`inline-flex h-8 w-8 items-center justify-center rounded transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
                danger ? 'text-destructive hover:bg-destructive/10' : 'text-muted-foreground hover:bg-muted'
            }`}
            title={title}
        >
            {children}
        </button>
    )
}

function UserFormModal({
    title,
    submitLabel,
    loading,
    user,
    setUser,
    onSubmit,
    onClose,
    includePassword = false,
    currentUser,
    editingUser,
}) {
    const editingSelf = currentUser?.id === editingUser?.id
    return (
        <Modal title={title} onClose={onClose}>
            <form onSubmit={onSubmit} className="space-y-4">
                <Field label="Username">
                    <input
                        type="text"
                        value={user.username}
                        onChange={(e) => setUser({ ...user, username: e.target.value })}
                        className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
                        required
                        minLength={3}
                        maxLength={20}
                    />
                </Field>
                <Field label="Email">
                    <input
                        type="email"
                        value={user.email}
                        onChange={(e) => setUser({ ...user, email: e.target.value })}
                        className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
                        required
                    />
                </Field>
                {includePassword && (
                    <Field label="Mật khẩu">
                        <input
                            type="password"
                            value={user.password}
                            onChange={(e) => setUser({ ...user, password: e.target.value })}
                            className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
                            required
                            minLength={8}
                            placeholder="Tối thiểu 8 ký tự"
                        />
                    </Field>
                )}
                <Field label="Role">
                    <select
                        value={user.role}
                        onChange={(e) => setUser({ ...user, role: e.target.value })}
                        className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary disabled:opacity-60"
                        disabled={editingSelf}
                    >
                        <option value="user">User</option>
                        <option value="admin">Admin</option>
                    </select>
                    {editingSelf && (
                        <p className="mt-1 text-xs text-muted-foreground">Không tự hạ quyền tài khoản đang đăng nhập.</p>
                    )}
                </Field>
                <ModalActions loading={loading} submitLabel={submitLabel} onClose={onClose} />
            </form>
        </Modal>
    )
}

function PasswordModal({ user, password, setPassword, loading, onSubmit, onClose }) {
    return (
        <Modal title={`Đổi mật khẩu - ${user.username}`} onClose={onClose}>
            <form onSubmit={onSubmit} className="space-y-4">
                <Field label="Mật khẩu mới">
                    <input
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
                        required
                        minLength={8}
                        placeholder="Tối thiểu 8 ký tự"
                    />
                </Field>
                <p className="text-xs text-muted-foreground">Sau khi đổi mật khẩu, toàn bộ phiên đăng nhập của người dùng này sẽ bị thu hồi.</p>
                <ModalActions loading={loading} submitLabel="Đổi mật khẩu" onClose={onClose} />
            </form>
        </Modal>
    )
}

function ConfirmDeleteModal({ user, loading, onConfirm, onClose }) {
    return (
        <Modal title="Xác nhận xóa người dùng" onClose={onClose}>
            <div className="space-y-4">
                <p className="text-sm text-muted-foreground">
                    Xóa <span className="font-semibold text-foreground">{user.username}</span> sẽ xóa luôn tài khoản DeepSeek, API keys và sessions thuộc user này.
                </p>
                <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                    Hành động này không thể hoàn tác từ giao diện.
                </div>
                <div className="flex gap-2 pt-2">
                    <button
                        type="button"
                        onClick={onConfirm}
                        disabled={loading}
                        className="flex-1 rounded-lg bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
                    >
                        {loading ? 'Đang xóa...' : 'Xóa người dùng'}
                    </button>
                    <button
                        type="button"
                        onClick={onClose}
                        className="rounded-lg bg-muted px-4 py-2 text-sm text-foreground hover:bg-muted/80"
                    >
                        Hủy
                    </button>
                </div>
            </div>
        </Modal>
    )
}

function Field({ label, children }) {
    return (
        <label className="block">
            <span className="mb-1 block text-sm font-medium text-foreground">{label}</span>
            {children}
        </label>
    )
}

function Modal({ title, children, onClose }) {
    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
            <div className="w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-xl">
                <div className="mb-4 flex items-center justify-between gap-3">
                    <h3 className="text-lg font-semibold text-foreground">{title}</h3>
                    <button onClick={onClose} className="rounded p-1 text-muted-foreground hover:bg-muted" title="Đóng">
                        <X className="h-4 w-4" />
                    </button>
                </div>
                {children}
            </div>
        </div>
    )
}

function ModalActions({ loading, submitLabel, onClose }) {
    return (
        <div className="flex gap-2 pt-2">
            <button
                type="submit"
                disabled={loading}
                className="flex-1 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
                {loading ? 'Đang xử lý...' : submitLabel}
            </button>
            <button
                type="button"
                onClick={onClose}
                className="rounded-lg bg-muted px-4 py-2 text-sm text-foreground hover:bg-muted/80"
            >
                Hủy
            </button>
        </div>
    )
}

function formatDate(value) {
    if (!value) return '-'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleDateString('vi-VN')
}
