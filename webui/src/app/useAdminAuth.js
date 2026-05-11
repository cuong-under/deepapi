import { useCallback, useEffect, useMemo, useState } from 'react'
import { detectRuntimeEnv } from '../utils/runtimeEnv'

export function useAdminAuth({ isProduction, location, t }) {
    const [message, setMessage] = useState(null)
    const [token, setToken] = useState(null)
    const [currentUser, setCurrentUser] = useState(null)
    const [authChecking, setAuthChecking] = useState(true)

    const isAdminRoute = location.pathname.startsWith('/admin') || isProduction
    const runtimeEnv = useMemo(() => detectRuntimeEnv(), [])
    const isVercel = runtimeEnv.isVercel

    const showMessage = useCallback((type, text) => {
        setMessage({ type, text })
        setTimeout(() => setMessage(null), 5000)
    }, [])

    const handleLogout = useCallback(() => {
        setToken(null)
        setCurrentUser(null)
        localStorage.removeItem('ds2api_token')
        localStorage.removeItem('ds2api_token_expires')
        localStorage.removeItem('ds2api_user')
        sessionStorage.removeItem('ds2api_token')
        sessionStorage.removeItem('ds2api_token_expires')
        sessionStorage.removeItem('ds2api_user')
    }, [])

    const handleLogin = useCallback((newToken, _rememberMe = true, user = null) => {
        setToken(newToken)
        if (user) {
            setCurrentUser(user)
        } else {
            const storedUser = localStorage.getItem('ds2api_user') || sessionStorage.getItem('ds2api_user')
            if (storedUser) {
                try {
                    setCurrentUser(JSON.parse(storedUser))
                } catch {
                    setCurrentUser(null)
                }
            }
        }
    }, [])

    useEffect(() => {
        if (!isAdminRoute) {
            setAuthChecking(false)
            return
        }

        const checkAuth = async () => {
            const storedToken = localStorage.getItem('ds2api_token') || sessionStorage.getItem('ds2api_token')
            const expiresAt = parseInt(localStorage.getItem('ds2api_token_expires') || sessionStorage.getItem('ds2api_token_expires') || '0')
            const storedUser = localStorage.getItem('ds2api_user') || sessionStorage.getItem('ds2api_user')

            if (storedToken && expiresAt > Date.now()) {
                // Check if multi-user mode (has user info stored)
                if (storedUser) {
                    // Multi-user mode: verify with /api/auth/me
                    try {
                        const res = await fetch('/api/auth/me', {
                            headers: { 'Authorization': `Bearer ${storedToken}` }
                        })
                        if (res.ok) {
                            const user = await res.json()
                            setCurrentUser(user)
                            const storage = localStorage.getItem('ds2api_token') ? localStorage : sessionStorage
                            storage.setItem('ds2api_user', JSON.stringify(user))
                            setToken(storedToken)
                        } else {
                            handleLogout()
                        }
                    } catch {
                        // Network error, trust token if not expired
                        setToken(storedToken)
                    }
                } else {
                    // Legacy mode: verify with /admin/verify
                    try {
                        const res = await fetch('/admin/verify', {
                            headers: { 'Authorization': `Bearer ${storedToken}` }
                        })
                        if (res.ok) {
                            setCurrentUser(null)
                            setToken(storedToken)
                        } else {
                            handleLogout()
                        }
                    } catch {
                        setToken(storedToken)
                    }
                }
            }
            setAuthChecking(false)
        }

        checkAuth()
    }, [handleLogout, isAdminRoute, t])

    return {
        token,
        currentUser,
        authChecking,
        message,
        isAdminRoute,
        isVercel,
        showMessage,
        handleLogin,
        handleLogout,
    }
}
