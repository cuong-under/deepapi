import { useState, useEffect } from 'react'
import MultiUserLogin from './MultiUserLogin'
import MultiUserDashboard from './MultiUserDashboard'
import { useMultiUserAuth } from './useMultiUserAuth'

export default function MultiUserApp({ onMessage }) {
    const { token, user, loading, isAuthenticated, login, logout } = useMultiUserAuth()

    const handleLogin = (newToken, newUser) => {
        login(newToken, newUser)
    }

    const handleLogout = async () => {
        await logout()
        onMessage('success', 'Đã đăng xuất')
    }

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <div className="w-8 h-8 border-4 border-primary/30 border-t-primary rounded-full animate-spin" />
            </div>
        )
    }

    if (!isAuthenticated) {
        return <MultiUserLogin onLogin={handleLogin} onMessage={onMessage} />
    }

    return (
        <MultiUserDashboard
            user={user}
            token={token}
            onLogout={handleLogout}
            onMessage={onMessage}
        />
    )
}
