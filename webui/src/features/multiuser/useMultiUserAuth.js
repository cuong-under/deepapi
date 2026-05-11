import { useState, useEffect, useCallback } from 'react'

export function useMultiUserAuth() {
    const [token, setToken] = useState(null)
    const [user, setUser] = useState(null)
    const [loading, setLoading] = useState(true)

    // Check if token exists and is valid
    useEffect(() => {
        const storedToken = localStorage.getItem('ds2api_multiuser_token')
        const storedUser = localStorage.getItem('ds2api_multiuser_user')
        const expiresAt = localStorage.getItem('ds2api_multiuser_expires')

        if (storedToken && storedUser && expiresAt) {
            const now = Date.now()
            if (now < parseInt(expiresAt)) {
                setToken(storedToken)
                setUser(JSON.parse(storedUser))
            } else {
                // Token expired, clear storage
                localStorage.removeItem('ds2api_multiuser_token')
                localStorage.removeItem('ds2api_multiuser_user')
                localStorage.removeItem('ds2api_multiuser_expires')
            }
        }
        setLoading(false)
    }, [])

    const login = useCallback((newToken, newUser) => {
        setToken(newToken)
        setUser(newUser)
    }, [])

    const logout = useCallback(async () => {
        try {
            // Call logout API
            await fetch('/api/auth/logout', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })
        } catch (e) {
            console.error('Logout error:', e)
        } finally {
            // Clear local storage
            localStorage.removeItem('ds2api_multiuser_token')
            localStorage.removeItem('ds2api_multiuser_user')
            localStorage.removeItem('ds2api_multiuser_expires')
            setToken(null)
            setUser(null)
        }
    }, [token])

    const refreshToken = useCallback(async () => {
        if (!token) return false

        try {
            const res = await fetch('/api/auth/refresh', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })

            if (res.ok) {
                const data = await res.json()
                localStorage.setItem('ds2api_multiuser_token', data.token)
                localStorage.setItem('ds2api_multiuser_expires', Date.now() + data.expires_in * 1000)
                setToken(data.token)
                return true
            }
            return false
        } catch (e) {
            console.error('Refresh token error:', e)
            return false
        }
    }, [token])

    const isAuthenticated = !!token && !!user
    const isAdmin = user?.role === 'admin'

    return {
        token,
        user,
        loading,
        isAuthenticated,
        isAdmin,
        login,
        logout,
        refreshToken
    }
}
