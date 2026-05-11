// Wrapper to handle multi-user vs legacy accounts
import { useState, useEffect } from 'react'

export function useMultiUserAccounts(authFetch) {
    const [isMultiUser, setIsMultiUser] = useState(null)
    const [user, setUser] = useState(null)

    useEffect(() => {
        // Check if multi-user mode
        const storedUser = localStorage.getItem('ds2api_user') || sessionStorage.getItem('ds2api_user')
        if (storedUser) {
            try {
                const userData = JSON.parse(storedUser)
                setUser(userData)
                setIsMultiUser(true)
            } catch (e) {
                setIsMultiUser(false)
            }
        } else {
            setIsMultiUser(false)
        }
    }, [])

    const fetchAccounts = async ({ page = 1, pageSize = 10, query = '' } = {}) => {
        if (isMultiUser === null) return { accounts: [], total: 0 }

        if (isMultiUser) {
            const params = new URLSearchParams({
                page: String(page),
                page_size: String(pageSize),
            })
            if (query.trim()) params.set('q', query.trim())
            const res = await authFetch(`/api/user/accounts?${params.toString()}`)
            if (!res.ok) throw new Error('Failed to fetch accounts')
            const data = await res.json()
            return {
                accounts: data.accounts || [],
                total: data.total || 0,
                page: data.page || page,
                page_size: data.page_size || pageSize,
                total_pages: data.total_pages || 1,
            }
        } else {
            // Legacy mode: fetch from /admin/accounts
            const res = await authFetch('/admin/accounts')
            if (!res.ok) throw new Error('Failed to fetch accounts')
            const data = await res.json()
            return {
                accounts: data.items || [],
                total: data.total || 0
            }
        }
    }

    const fetchKeys = async () => {
        if (isMultiUser === null) return { keys: [], total: 0 }

        if (isMultiUser) {
            // Multi-user mode: fetch from /api/user/keys
            const res = await authFetch('/api/user/keys')
            if (!res.ok) throw new Error('Failed to fetch keys')
            const data = await res.json()
            return {
                keys: data.keys || [],
                total: data.total || 0
            }
        } else {
            // Legacy mode: fetch from config
            const res = await authFetch('/admin/config')
            if (!res.ok) throw new Error('Failed to fetch config')
            const data = await res.json()
            return {
                keys: data.keys || [],
                total: (data.keys || []).length
            }
        }
    }

    const createAccount = async (accountData) => {
        if (isMultiUser) {
            const res = await authFetch('/api/user/accounts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(accountData)
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.error || 'Failed to create account')
            }
            return await res.json()
        } else {
            // Legacy mode: use existing /admin/accounts endpoint
            const res = await authFetch('/admin/accounts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(accountData)
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.detail || 'Failed to create account')
            }
            return await res.json()
        }
    }

    const updateAccount = async (id, accountData) => {
        if (isMultiUser) {
            const res = await authFetch(`/api/user/accounts/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(accountData)
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.error || 'Failed to update account')
            }
            return await res.json()
        } else {
            // Legacy mode
            const res = await authFetch(`/admin/accounts/${encodeURIComponent(id)}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(accountData)
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.detail || 'Failed to update account')
            }
            return await res.json()
        }
    }

    const deleteAccount = async (id) => {
        if (isMultiUser) {
            const res = await authFetch(`/api/user/accounts/${id}`, {
                method: 'DELETE'
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.error || 'Failed to delete account')
            }
            return await res.json()
        } else {
            // Legacy mode
            const res = await authFetch(`/admin/accounts/${encodeURIComponent(id)}`, {
                method: 'DELETE'
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.detail || 'Failed to delete account')
            }
            return await res.json()
        }
    }

    const createKey = async (keyData) => {
        if (isMultiUser) {
            const res = await authFetch('/api/user/keys', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(keyData)
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.error || 'Failed to create key')
            }
            return await res.json()
        } else {
            // Legacy mode: use existing endpoint
            const res = await authFetch('/admin/keys', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(keyData)
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.detail || 'Failed to create key')
            }
            return await res.json()
        }
    }

    const deleteKey = async (id) => {
        if (isMultiUser) {
            const res = await authFetch(`/api/user/keys/${id}`, {
                method: 'DELETE'
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.error || 'Failed to delete key')
            }
            return await res.json()
        } else {
            // Legacy mode
            const res = await authFetch(`/admin/keys/${encodeURIComponent(id)}`, {
                method: 'DELETE'
            })
            if (!res.ok) {
                const error = await res.json()
                throw new Error(error.detail || 'Failed to delete key')
            }
            return await res.json()
        }
    }

    return {
        isMultiUser,
        user,
        fetchAccounts,
        fetchKeys,
        createAccount,
        updateAccount,
        deleteAccount,
        createKey,
        deleteKey
    }
}
