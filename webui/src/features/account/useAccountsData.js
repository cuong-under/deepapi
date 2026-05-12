import { useEffect, useState } from 'react'
import { useMultiUserAccounts } from './useMultiUserAccounts'

export function useAccountsData({ apiFetch }) {
    const [queueStatus, setQueueStatus] = useState(null)
    const [keysExpanded, setKeysExpanded] = useState(false)

    const [accounts, setAccounts] = useState([])
    const [page, setPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
    const [totalPages, setTotalPages] = useState(1)
    const [totalAccounts, setTotalAccounts] = useState(0)
    const [loadingAccounts, setLoadingAccounts] = useState(false)
    const [accountOverrides, setAccountOverrides] = useState({})

    const { isMultiUser, fetchAccounts: fetchMultiUserAccounts } = useMultiUserAccounts(apiFetch)

    const resolveAccountIdentifier = (acc) => {
        if (!acc || typeof acc !== 'object') return ''
        // Multi-user mode uses id, legacy uses identifier
        if (isMultiUser) {
            return String(acc.id || '')
        }
        return String(acc.identifier || acc.email || acc.mobile || '').trim()
    }

    const [searchQuery, setSearchQuery] = useState('')

    const fetchAccounts = async (targetPage = page, targetPageSize = pageSize, targetQuery = searchQuery) => {
        if (isMultiUser === null) return // Wait for multi-user check

        setLoadingAccounts(true)
        try {
            if (isMultiUser) {
                const data = await fetchMultiUserAccounts({
                    page: targetPage,
                    pageSize: targetPageSize,
                    query: targetQuery,
                })

                setAccounts(applyAccountOverrides(data.accounts || [], accountOverrides, true))
                setTotalPages(data.total_pages || 1)
                setTotalAccounts(data.total || 0)
                setPage(data.page || targetPage)
            } else {
                // Legacy mode: fetch from config
                let url = `/admin/accounts?page=${targetPage}&page_size=${targetPageSize}`
                if (targetQuery.trim()) url += `&q=${encodeURIComponent(targetQuery.trim())}`
                const res = await apiFetch(url)
                if (res.ok) {
                    const data = await res.json()
                    setAccounts(applyAccountOverrides(data.items || [], accountOverrides, false))
                    setTotalPages(data.total_pages || 1)
                    setTotalAccounts(data.total || 0)
                    setPage(data.page || 1)
                }
            }
        } catch (e) {
            console.error('Failed to fetch accounts:', e)
        } finally {
            setLoadingAccounts(false)
        }
    }

    const changePageSize = (newSize) => {
        setPageSize(newSize)
        fetchAccounts(1, newSize)
    }

    const handleSearchChange = (query) => {
        setSearchQuery(query)
        fetchAccounts(1, pageSize, query)
    }

    const updateAccountInList = (identifier, patch) => {
        const targetID = String(identifier || '').trim()
        if (!targetID) return
        setAccounts(prev => prev.map(acc => {
            const id = isMultiUser
                ? String(acc.id || '')
                : String(acc.identifier || acc.email || acc.mobile || '').trim()
            return id === targetID ? { ...acc, ...patch } : acc
        }))
        setAccountOverrides(prev => ({
            ...prev,
            [targetID]: { ...(prev[targetID] || {}), ...patch },
        }))
    }

    const fetchQueueStatus = async () => {
        try {
            const res = await apiFetch('/admin/queue/status')
            if (res.ok) {
                const data = await res.json()
                setQueueStatus(data)
            }
        } catch (e) {
            console.error('Failed to fetch queue status:', e)
        }
    }

    useEffect(() => {
        if (isMultiUser !== null) {
            fetchAccounts()
            if (!isMultiUser) {
                // Only fetch queue status in legacy mode
                fetchQueueStatus()
                const interval = setInterval(fetchQueueStatus, 5000)
                return () => clearInterval(interval)
            }
        }
    }, [isMultiUser])

    return {
        queueStatus,
        keysExpanded,
        setKeysExpanded,
        accounts,
        page,
        pageSize,
        totalPages,
        totalAccounts,
        loadingAccounts,
        fetchAccounts,
        updateAccountInList,
        changePageSize,
        resolveAccountIdentifier,
        searchQuery,
        handleSearchChange,
    }
}

function applyAccountOverrides(accounts, overrides, isMultiUser) {
    if (!overrides || Object.keys(overrides).length === 0) return accounts
    return accounts.map(acc => {
        const id = isMultiUser
            ? String(acc.id || '')
            : String(acc.identifier || acc.email || acc.mobile || '').trim()
        return overrides[id] ? { ...acc, ...overrides[id] } : acc
    })
}
