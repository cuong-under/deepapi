import { useState } from 'react'
import { useMultiUserAccounts } from './useMultiUserAccounts'

export function useAccountActions({ apiFetch, t, onMessage, onRefresh, config, fetchAccounts, resolveAccountIdentifier }) {
    const [showAddKey, setShowAddKey] = useState(false)
    const [editingKey, setEditingKey] = useState(null)
    const [showAddAccount, setShowAddAccount] = useState(false)
    const [showEditAccount, setShowEditAccount] = useState(false)
    const [editingAccount, setEditingAccount] = useState(null)
    const [newKey, setNewKey] = useState({ key: '', name: '', remark: '' })
    const [copiedKey, setCopiedKey] = useState(null)
    const [newAccount, setNewAccount] = useState({ name: '', remark: '', email: '', mobile: '', password: '' })
    const [editAccount, setEditAccount] = useState({ name: '', remark: '' })
    const [loading, setLoading] = useState(false)
    const [testing, setTesting] = useState({})
    const [testingAll, setTestingAll] = useState(false)
    const [batchProgress, setBatchProgress] = useState({ current: 0, total: 0, results: [] })
    const [sessionCounts, setSessionCounts] = useState({})
    const [deletingSessions, setDeletingSessions] = useState({})
    const [updatingProxy, setUpdatingProxy] = useState({})

    const {
        isMultiUser,
        fetchAccounts: fetchMultiUserAccounts,
        createAccount: createMultiUserAccount,
        updateAccount: updateMultiUserAccount,
        deleteAccount: deleteMultiUserAccount,
        createKey: createMultiUserKey,
        deleteKey: deleteMultiUserKey
    } = useMultiUserAccounts(apiFetch)

    const openAddKey = () => {
        setEditingKey(null)
        setNewKey({ key: '', name: '', remark: '', createdKey: '' })
        setShowAddKey(true)
    }

    const openEditKey = (item) => {
        if (!item?.key && !item?.id) return
        setEditingKey(item)
        setNewKey({
            key: item.key || '',
            name: item.name || '',
            remark: item.remark || '',
            createdKey: '',
        })
        setShowAddKey(true)
    }

    const closeKeyModal = () => {
        setShowAddKey(false)
        setEditingKey(null)
        setNewKey({ key: '', name: '', remark: '', createdKey: '' })
    }

    const openAddAccount = () => {
        setShowEditAccount(false)
        setEditingAccount(null)
        setEditAccount({ name: '', remark: '' })
        setNewAccount({ name: '', remark: '', email: '', mobile: '', password: '' })
        setShowAddAccount(true)
    }

    const closeAddAccount = () => {
        setShowAddAccount(false)
        setNewAccount({ name: '', remark: '', email: '', mobile: '', password: '' })
    }

    const openEditAccount = (account) => {
        const identifier = resolveAccountIdentifier(account)
        if (!identifier) {
            onMessage('error', t('accountManager.invalidIdentifier'))
            return
        }
        setShowAddAccount(false)
        setEditingAccount({
            identifier,
        })
        setEditAccount({
            name: account?.name || '',
            remark: account?.remark || '',
        })
        setShowEditAccount(true)
    }

    const closeEditAccount = () => {
        setShowEditAccount(false)
        setEditingAccount(null)
        setEditAccount({ name: '', remark: '' })
    }

    const addKey = async () => {
        const isEditing = Boolean(editingKey?.key || editingKey?.id)
        if (!isEditing && !newKey.key.trim() && !isMultiUser) {
            return
        }
        setLoading(true)
        try {
            if (isMultiUser) {
                // Multi-user mode: use database API
                if (isEditing) {
                    // Update key (only name/remark)
                    const res = await apiFetch(`/api/user/keys/${editingKey.id}`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ name: newKey.name, remark: newKey.remark })
                    })
                    if (res.ok) {
                        onMessage('success', t('accountManager.updateKeySuccess'))
                        closeKeyModal()
                        onRefresh()
                    } else {
                        const data = await res.json()
                        onMessage('error', data.error || t('messages.requestFailed'))
                    }
                } else {
                    // Create new key (auto-generated)
                    const result = await createMultiUserKey({ name: newKey.name, remark: newKey.remark })
                    onMessage('success', t('accountManager.addKeySuccess'))
                    setNewKey({
                        key: result.api_key || '',
                        name: result.name || newKey.name,
                        remark: result.remark || newKey.remark,
                        createdKey: result.api_key || '',
                    })
                    onRefresh()
                }
            } else {
                // Legacy mode
                const endpoint = isEditing
                    ? `/admin/keys/${encodeURIComponent(editingKey.key)}`
                    : '/admin/keys'
                const method = isEditing ? 'PUT' : 'POST'
                const payload = isEditing
                    ? { name: newKey.name, remark: newKey.remark }
                    : { key: newKey.key.trim(), name: newKey.name, remark: newKey.remark }
                if (!isEditing && !payload.key) {
                    return
                }
                const res = await apiFetch(endpoint, {
                    method,
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload),
                })
                if (res.ok) {
                    onMessage('success', isEditing ? t('accountManager.updateKeySuccess') : t('accountManager.addKeySuccess'))
                    closeKeyModal()
                    onRefresh()
                } else {
                    const data = await res.json()
                    onMessage('error', data.detail || (isEditing ? t('messages.requestFailed') : t('messages.failedToAdd')))
                }
            }
        } catch (e) {
            onMessage('error', e.message || t('messages.networkError'))
        } finally {
            setLoading(false)
        }
    }

    const deleteKey = async (keyOrId) => {
        if (!confirm(t('accountManager.deleteKeyConfirm'))) return
        try {
            if (isMultiUser) {
                // Multi-user mode: keyOrId is the id
                await deleteMultiUserKey(keyOrId)
                onMessage('success', t('messages.deleted'))
                onRefresh()
            } else {
                // Legacy mode: keyOrId is the key string
                const res = await apiFetch(`/admin/keys/${encodeURIComponent(keyOrId)}`, { method: 'DELETE' })
                if (res.ok) {
                    onMessage('success', t('messages.deleted'))
                    onRefresh()
                } else {
                    onMessage('error', t('messages.deleteFailed'))
                }
            }
        } catch (e) {
            onMessage('error', e.message || t('messages.networkError'))
        }
    }

    const addAccount = async () => {
        if (!newAccount.password || (!newAccount.email && !newAccount.mobile)) {
            onMessage('error', t('accountManager.requiredFields'))
            return
        }
        setLoading(true)
        try {
            if (isMultiUser) {
                // Multi-user mode
                await createMultiUserAccount(newAccount)
                onMessage('success', t('accountManager.addAccountSuccess'))
                closeAddAccount()
                fetchAccounts(1)
                onRefresh()
            } else {
                // Legacy mode
                const res = await apiFetch('/admin/accounts', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(newAccount),
                })
                if (res.ok) {
                    onMessage('success', t('accountManager.addAccountSuccess'))
                    closeAddAccount()
                    fetchAccounts(1)
                    onRefresh()
                } else {
                    const data = await res.json()
                    onMessage('error', data.detail || t('messages.failedToAdd'))
                }
            }
        } catch (e) {
            onMessage('error', e.message || t('messages.networkError'))
        } finally {
            setLoading(false)
        }
    }

    const updateAccount = async () => {
        const identifier = String(editingAccount?.identifier || '').trim()
        if (!identifier) {
            onMessage('error', t('accountManager.invalidIdentifier'))
            return
        }
        setLoading(true)
        try {
            if (isMultiUser) {
                // Multi-user mode: identifier is the id
                await updateMultiUserAccount(identifier, editAccount)
                onMessage('success', t('accountManager.updateAccountSuccess'))
                closeEditAccount()
                fetchAccounts()
                onRefresh()
            } else {
                // Legacy mode
                const res = await apiFetch(`/admin/accounts/${encodeURIComponent(identifier)}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(editAccount),
                })
                if (res.ok) {
                    onMessage('success', t('accountManager.updateAccountSuccess'))
                    closeEditAccount()
                    fetchAccounts()
                    onRefresh()
                } else {
                    const data = await res.json()
                    onMessage('error', data.detail || t('messages.requestFailed'))
                }
            }
        } catch (e) {
            onMessage('error', e.message || t('messages.networkError'))
        } finally {
            setLoading(false)
        }
    }

    const deleteAccount = async (id) => {
        const identifier = String(id || '').trim()
        if (!identifier) {
            onMessage('error', t('accountManager.invalidIdentifier'))
            return
        }
        if (!confirm(t('accountManager.deleteAccountConfirm'))) return
        try {
            if (isMultiUser) {
                // Multi-user mode: id is the database id
                await deleteMultiUserAccount(identifier)
                onMessage('success', t('messages.deleted'))
                fetchAccounts()
                onRefresh()
            } else {
                // Legacy mode: id is the identifier (email/mobile)
                const res = await apiFetch(`/admin/accounts/${encodeURIComponent(identifier)}`, { method: 'DELETE' })
                if (res.ok) {
                    onMessage('success', t('messages.deleted'))
                    fetchAccounts()
                    onRefresh()
                } else {
                    onMessage('error', t('messages.deleteFailed'))
                }
            }
        } catch (e) {
            onMessage('error', e.message || t('messages.networkError'))
        }
    }

    const testAccount = async (identifier) => {
        const accountID = String(identifier || '').trim()
        if (!accountID) {
            onMessage('error', t('accountManager.invalidIdentifier'))
            return
        }
        setTesting(prev => ({ ...prev, [accountID]: true }))
        try {
            if (isMultiUser) {
                // Multi-user mode: call refresh token endpoint
                const res = await apiFetch(`/api/user/accounts/${accountID}/refresh-token`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                })
                const data = await res.json()

                if (data.success) {
                    onMessage('success', data.message || t('accountManager.refreshTokenSuccess'))
                } else {
                    onMessage('error', data.error || t('accountManager.refreshTokenFailed'))
                }
                fetchAccounts()
                onRefresh()
            } else {
                // Legacy mode: call test endpoint
                const res = await apiFetch('/admin/accounts/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ identifier: accountID }),
                })
                const data = await res.json()

                // 更新会话数
                if (data.session_count !== undefined) {
                    setSessionCounts(prev => ({ ...prev, [accountID]: data.session_count }))
                }

                const statusMessage = data.success
                    ? t('apiTester.testSuccess', { account: accountID, time: data.response_time })
                    : `${accountID}: ${data.message}`
                onMessage(data.success ? 'success' : 'error', statusMessage)
                fetchAccounts()
                onRefresh()
            }
        } catch (e) {
            onMessage('error', t('accountManager.testFailed', { error: e.message }))
        } finally {
            setTesting(prev => ({ ...prev, [accountID]: false }))
        }
    }

    const loadAccountsForBatchRefresh = async () => {
        if (isMultiUser) {
            const pageSize = 100
            const firstPage = await fetchMultiUserAccounts({ page: 1, pageSize })
            const total = Number(firstPage.total || firstPage.accounts?.length || 0)
            const allAccounts = [...(firstPage.accounts || [])]
            const totalPages = Math.max(1, Math.ceil(total / pageSize))

            for (let page = 2; page <= totalPages; page++) {
                const data = await fetchMultiUserAccounts({ page, pageSize })
                allAccounts.push(...(data.accounts || []))
            }

            return allAccounts
        }

        const res = await apiFetch('/admin/accounts?page=1&page_size=5000')
        if (!res.ok) throw new Error(t('messages.requestFailed'))
        const data = await res.json()
        return data.items || config.accounts || []
    }

    const testAllAccounts = async () => {
        if (!confirm(t('accountManager.testAllConfirm'))) return

        setTestingAll(true)
        setBatchProgress({ current: 0, total: 0, results: [] })

        let allAccounts = []
        try {
            allAccounts = await loadAccountsForBatchRefresh()
        } catch (e) {
            setTestingAll(false)
            onMessage('error', e.message || t('messages.networkError'))
            return
        }

        if (allAccounts.length === 0) {
            setTestingAll(false)
            return
        }

        setBatchProgress({ current: 0, total: allAccounts.length, results: [] })

        let successCount = 0
        const results = []

        for (let i = 0; i < allAccounts.length; i++) {
            const acc = allAccounts[i]
            const id = resolveAccountIdentifier(acc)
            if (!id) {
                results.push({ id: '-', success: false, message: t('accountManager.invalidIdentifier') })
                setBatchProgress({ current: i + 1, total: allAccounts.length, results: [...results] })
                continue
            }

            try {
                if (isMultiUser) {
                    // Multi-user mode: call refresh token endpoint
                    const res = await apiFetch(`/api/user/accounts/${id}/refresh-token`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                    })
                    const data = await res.json()
                    results.push({ id, success: data.success, message: data.message || data.error })
                    if (data.success) successCount++
                } else {
                    // Legacy mode: call test endpoint
                    const res = await apiFetch('/admin/accounts/test', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ identifier: id }),
                    })
                    const data = await res.json()
                    results.push({ id, success: data.success, message: data.message, time: data.response_time })
                    if (data.success) successCount++
                }
            } catch (e) {
                results.push({ id, success: false, message: e.message })
            }

            setBatchProgress({ current: i + 1, total: allAccounts.length, results: [...results] })
        }

        onMessage('success', t('accountManager.testAllCompleted', { success: successCount, total: allAccounts.length }))
        fetchAccounts()
        onRefresh()
        setTestingAll(false)
        setBatchProgress({ current: 0, total: 0, results: [] })
    }

    const deleteAllSessions = async (identifier) => {
        if (isMultiUser) {
            onMessage('error', t('messages.requestFailed'))
            return
        }
        const accountID = String(identifier || '').trim()
        if (!accountID) {
            onMessage('error', t('accountManager.invalidIdentifier'))
            return
        }
        if (!confirm(t('accountManager.deleteAllSessionsConfirm'))) return
        
        setDeletingSessions(prev => ({ ...prev, [accountID]: true }))
        try {
            const res = await apiFetch('/admin/accounts/sessions/delete-all', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ identifier: accountID }),
            })
            const data = await res.json()
            
            if (data.success) {
                onMessage('success', t('accountManager.deleteAllSessionsSuccess'))
                // 清除会话数显示
                setSessionCounts(prev => {
                    const newCounts = { ...prev }
                    delete newCounts[accountID]
                    return newCounts
                })
            } else {
                onMessage('error', data.message || t('messages.requestFailed'))
            }
        } catch (e) {
            onMessage('error', t('messages.networkError'))
        } finally {
            setDeletingSessions(prev => ({ ...prev, [accountID]: false }))
        }
    }

    const updateAccountProxy = async (identifier, proxyID) => {
        const accountID = String(identifier || '').trim()
        if (!accountID) {
            onMessage('error', t('accountManager.invalidIdentifier'))
            return
        }
        setUpdatingProxy(prev => ({ ...prev, [accountID]: true }))
        try {
            let res
            if (isMultiUser) {
                const existing = config?.accounts?.find(acc => String(acc.id || '') === accountID)
                res = await apiFetch(`/api/user/accounts/${encodeURIComponent(accountID)}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        name: existing?.name || '',
                        remark: existing?.remark || '',
                        email: existing?.email || '',
                        mobile: existing?.mobile || '',
                        proxy_id: proxyID || '',
                    }),
                })
            } else {
                res = await apiFetch(`/admin/accounts/${encodeURIComponent(accountID)}/proxy`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ proxy_id: proxyID || '' }),
                })
            }
            const data = await res.json()
            if (!res.ok) {
                onMessage('error', data.detail || data.error || t('messages.requestFailed'))
                return
            }
            onMessage('success', t('accountManager.proxyUpdateSuccess'))
            fetchAccounts()
            onRefresh()
        } catch (_err) {
            onMessage('error', t('messages.networkError'))
        } finally {
            setUpdatingProxy(prev => ({ ...prev, [accountID]: false }))
        }
    }

    return {
        showAddKey,
        openAddKey,
        openEditKey,
        closeKeyModal,
        editingKey,
        showAddAccount,
        openAddAccount,
        closeAddAccount,
        showEditAccount,
        editingAccount,
        editAccount,
        setEditAccount,
        openEditAccount,
        closeEditAccount,
        newKey,
        setNewKey,
        copiedKey,
        setCopiedKey,
        newAccount,
        setNewAccount,
        loading,
        testing,
        testingAll,
        batchProgress,
        sessionCounts,
        deletingSessions,
        updatingProxy,
        addKey,
        deleteKey,
        addAccount,
        updateAccount,
        deleteAccount,
        testAccount,
        testAllAccounts,
        deleteAllSessions,
        updateAccountProxy,
    }
}
