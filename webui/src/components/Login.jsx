import { useState, useEffect } from 'react'
import { Key, ArrowRight, ShieldCheck, Lock, Check, User, Mail } from 'lucide-react'
import clsx from 'clsx'
import { useI18n } from '../i18n'
import LanguageToggle from './LanguageToggle'

export default function Login({ onLogin, onMessage }) {
    const { t } = useI18n()
    const [mode, setMode] = useState('checking') // 'checking', 'legacy', 'multiuser'
    const [authMode, setAuthMode] = useState('login') // 'login' or 'register'

    // Legacy admin key mode
    const [adminKey, setAdminKey] = useState('')

    // Multi-user mode
    const [username, setUsername] = useState('')
    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')

    const [loading, setLoading] = useState(false)
    const [remember, setRemember] = useState(true)

    // Check if multi-user mode is enabled
    useEffect(() => {
        const checkMultiUser = async () => {
            try {
                const res = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username: '__check__', password: '__check__' })
                })
                // If endpoint exists (even returns error), multi-user is enabled
                setMode(res.status !== 404 ? 'multiuser' : 'legacy')
            } catch (e) {
                setMode('legacy')
            }
        }
        checkMultiUser()
    }, [])

    // Legacy admin key login
    const handleLegacyLogin = async (e) => {
        e.preventDefault()
        if (!adminKey.trim()) return

        setLoading(true)

        try {
            const res = await fetch('/admin/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ admin_key: adminKey }),
            })

            const data = await res.json()

            if (res.ok && data.success) {
                const storage = remember ? localStorage : sessionStorage
                storage.setItem('ds2api_token', data.token)
                storage.setItem('ds2api_token_expires', Date.now() + data.expires_in * 1000)

                onLogin(data.token)
                if (data.message) {
                    onMessage('warning', data.message)
                }
            } else {
                onMessage('error', data.detail || t('login.signInFailed'))
            }
        } catch (e) {
            onMessage('error', t('login.networkError', { error: e.message }))
        } finally {
            setLoading(false)
        }
    }

    // Multi-user login
    const handleMultiUserLogin = async (e) => {
        e.preventDefault()
        if (!username.trim() || !password.trim()) {
            onMessage('error', 'Username và password không được để trống')
            return
        }

        setLoading(true)

        try {
            console.log('[Login] Sending login request:', { username, password: '***' })
            const res = await fetch('/api/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
            })

            console.log('[Login] Response status:', res.status)
            const data = await res.json()
            console.log('[Login] Response data:', { ...data, token: data.token ? data.token.substring(0, 20) + '...' : 'null' })

            if (res.ok && data.token) {
                const storage = remember ? localStorage : sessionStorage
                storage.setItem('ds2api_token', data.token)
                storage.setItem('ds2api_token_expires', Date.now() + data.expires_in * 1000)
                storage.setItem('ds2api_user', JSON.stringify(data.user))

                onLogin(data.token, remember, data.user)
                onMessage('success', `Chào mừng ${data.user.username}!`)
            } else {
                onMessage('error', data.error || 'Đăng nhập thất bại')
            }
        } catch (e) {
            console.error('[Login] Error:', e)
            onMessage('error', `Lỗi kết nối: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    // Multi-user register
    const handleRegister = async (e) => {
        e.preventDefault()
        if (!username.trim() || !email.trim() || !password.trim()) {
            onMessage('error', 'Vui lòng điền đầy đủ thông tin')
            return
        }

        if (password.length < 8) {
            onMessage('error', 'Password phải có ít nhất 8 ký tự')
            return
        }

        setLoading(true)

        try {
            const res = await fetch('/api/auth/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, email, password }),
            })

            const data = await res.json()

            if (res.ok && data.token) {
                const storage = remember ? localStorage : sessionStorage
                storage.setItem('ds2api_token', data.token)
                storage.setItem('ds2api_token_expires', Date.now() + data.expires_in * 1000)
                storage.setItem('ds2api_user', JSON.stringify(data.user))

                onLogin(data.token, remember, data.user)
                onMessage('success', `Đăng ký thành công! Chào mừng ${data.user.username}!`)
            } else {
                onMessage('error', data.error || 'Đăng ký thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi kết nối: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

    const handleLogin = mode === 'legacy' ? handleLegacyLogin : handleMultiUserLogin

    // Show loading while checking mode
    if (mode === 'checking') {
        return (
            <div className="min-h-screen w-full flex flex-col items-center justify-center p-4 bg-background text-foreground">
                <div className="w-8 h-8 border-4 border-primary/30 border-t-primary rounded-full animate-spin" />
            </div>
        )
    }

    // Legacy admin key mode
    if (mode === 'legacy') {
        return (
            <div className="min-h-screen w-full flex flex-col items-center justify-center p-4 bg-background text-foreground">
                <div className="absolute top-6 right-6">
                    <LanguageToggle />
                </div>

                <div className="w-full max-w-[400px] relative z-10 animate-in fade-in zoom-in-95 duration-200">
                    <div className="w-full bg-card border border-border rounded-xl p-8 shadow-sm">
                        <div className="text-center space-y-2 mb-8 animate-in fade-in slide-in-from-top-4 duration-500">
                            <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-primary/10 text-primary mb-2">
                                <Lock className="w-6 h-6" />
                            </div>
                            <h1 className="text-3xl font-bold tracking-tight text-foreground">{t('login.welcome')}</h1>
                            <p className="text-sm text-muted-foreground/80">{t('login.subtitle')}</p>
                        </div>

                        <form onSubmit={handleLegacyLogin} className="space-y-5 animate-in fade-in slide-in-from-bottom-4 duration-700 delay-150">
                            <div className="space-y-2">
                                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-widest ml-1">{t('login.adminKeyLabel')}</label>
                                <div className="relative group">
                                    <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-muted-foreground group-focus-within:text-primary transition-colors">
                                        <Key className="w-4 h-4" />
                                    </div>
                                    <input
                                        type="password"
                                        className="w-full bg-[#09090b] border border-border rounded-xl pl-10 pr-4 py-3 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all placeholder:text-muted-foreground/30 text-foreground"
                                        placeholder={t('login.adminKeyPlaceholder')}
                                        value={adminKey}
                                        onChange={e => setAdminKey(e.target.value)}
                                        autoFocus
                                    />
                                </div>
                            </div>

                            <div className="flex items-center justify-between px-1">
                                <label className="flex items-center gap-2.5 cursor-pointer group">
                                    <div className="relative flex items-center">
                                        <input
                                            type="checkbox"
                                            className="peer sr-only"
                                            checked={remember}
                                            onChange={e => setRemember(e.target.checked)}
                                        />
                                        <div className="w-[18px] h-[18px] bg-secondary border border-border rounded-md peer-checked:bg-primary peer-checked:border-primary transition-all shadow-sm"></div>
                                        <Check className="absolute inset-0 m-auto w-3 h-3 text-primary-foreground opacity-0 peer-checked:opacity-100 transition-opacity stroke-[3]" />
                                    </div>
                                    <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors">{t('login.rememberSession')}</span>
                                </label>
                            </div>

                            <button
                                type="submit"
                                disabled={loading}
                                className="w-full h-12 flex items-center justify-center gap-2 bg-primary text-primary-foreground rounded-xl hover:bg-primary/90 transition-all font-semibold text-sm shadow-lg shadow-primary/20 hover:shadow-primary/30 disabled:opacity-50 disabled:shadow-none"
                            >
                                {loading ? (
                                    <div className="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
                                ) : (
                                    <div className="flex items-center gap-2">
                                        <span>{t('login.signIn')}</span>
                                        <ArrowRight className="w-4 h-4" />
                                    </div>
                                )}
                            </button>
                        </form>

                        <div className="mt-6 pt-6 border-t border-border flex justify-center">
                            <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground/60 font-medium tracking-wide uppercase">
                                <ShieldCheck className="w-3 h-3" />
                                <span>{t('login.secureConnection')}</span>
                            </div>
                        </div>
                    </div>

                    <div className="mt-8 text-center">
                        <p className="text-[10px] text-muted-foreground/30 font-mono text-center">{t('login.adminPortal')}</p>
                    </div>
                </div>
            </div>
        )
    }

    // Multi-user mode
    return (
        <div className="min-h-screen w-full flex flex-col items-center justify-center p-4 bg-background text-foreground">
            <div className="absolute top-6 right-6">
                <LanguageToggle />
            </div>

            <div className="w-full max-w-[440px] relative z-10 animate-in fade-in zoom-in-95 duration-200">
                <div className="w-full bg-card border border-border rounded-xl p-8 shadow-sm">
                    {/* Header */}
                    <div className="text-center space-y-2 mb-8 animate-in fade-in slide-in-from-top-4 duration-500">
                        <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-primary/10 text-primary mb-2">
                            {authMode === 'login' ? <Lock className="w-6 h-6" /> : <User className="w-6 h-6" />}
                        </div>
                        <h1 className="text-3xl font-bold tracking-tight text-foreground">
                            {authMode === 'login' ? 'Đăng nhập' : 'Đăng ký'}
                        </h1>
                        <p className="text-sm text-muted-foreground/80">
                            {authMode === 'login' ? 'Đăng nhập vào tài khoản của bạn' : 'Tạo tài khoản mới'}
                        </p>
                    </div>

                    {/* Mode Toggle */}
                    <div className="flex gap-2 mb-6 p-1 bg-secondary/50 rounded-lg">
                        <button
                            type="button"
                            onClick={() => setAuthMode('login')}
                            className={clsx(
                                'flex-1 py-2 px-4 rounded-md text-sm font-medium transition-all',
                                authMode === 'login'
                                    ? 'bg-primary text-primary-foreground shadow-sm'
                                    : 'text-muted-foreground hover:text-foreground'
                            )}
                        >
                            Đăng nhập
                        </button>
                        <button
                            type="button"
                            onClick={() => setAuthMode('register')}
                            className={clsx(
                                'flex-1 py-2 px-4 rounded-md text-sm font-medium transition-all',
                                authMode === 'register'
                                    ? 'bg-primary text-primary-foreground shadow-sm'
                                    : 'text-muted-foreground hover:text-foreground'
                            )}
                        >
                            Đăng ký
                        </button>
                    </div>

                    {/* Form */}
                    <form onSubmit={authMode === 'login' ? handleMultiUserLogin : handleRegister} className="space-y-5">
                        {/* Username */}
                        <div className="space-y-2">
                            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-widest ml-1">
                                Username
                            </label>
                            <div className="relative group">
                                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-muted-foreground group-focus-within:text-primary transition-colors">
                                    <User className="w-4 h-4" />
                                </div>
                                <input
                                    type="text"
                                    className="w-full bg-[#09090b] border border-border rounded-xl pl-10 pr-4 py-3 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all placeholder:text-muted-foreground/30 text-foreground"
                                    placeholder="Nhập username"
                                    value={username}
                                    onChange={e => setUsername(e.target.value)}
                                    autoFocus
                                />
                            </div>
                        </div>

                        {/* Email (only for register) */}
                        {authMode === 'register' && (
                            <div className="space-y-2">
                                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-widest ml-1">
                                    Email
                                </label>
                                <div className="relative group">
                                    <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-muted-foreground group-focus-within:text-primary transition-colors">
                                        <Mail className="w-4 h-4" />
                                    </div>
                                    <input
                                        type="email"
                                        className="w-full bg-[#09090b] border border-border rounded-xl pl-10 pr-4 py-3 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all placeholder:text-muted-foreground/30 text-foreground"
                                        placeholder="Nhập email"
                                        value={email}
                                        onChange={e => setEmail(e.target.value)}
                                    />
                                </div>
                            </div>
                        )}

                        {/* Password */}
                        <div className="space-y-2">
                            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-widest ml-1">
                                Password
                            </label>
                            <div className="relative group">
                                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-muted-foreground group-focus-within:text-primary transition-colors">
                                    <Lock className="w-4 h-4" />
                                </div>
                                <input
                                    type="password"
                                    className="w-full bg-[#09090b] border border-border rounded-xl pl-10 pr-4 py-3 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all placeholder:text-muted-foreground/30 text-foreground"
                                    placeholder={authMode === 'register' ? 'Tối thiểu 8 ký tự' : 'Nhập password'}
                                    value={password}
                                    onChange={e => setPassword(e.target.value)}
                                />
                            </div>
                        </div>

                        {/* Remember me (only for login) */}
                        {authMode === 'login' && (
                            <div className="flex items-center justify-between px-1">
                                <label className="flex items-center gap-2.5 cursor-pointer group">
                                    <div className="relative flex items-center">
                                        <input
                                            type="checkbox"
                                            className="peer sr-only"
                                            checked={remember}
                                            onChange={e => setRemember(e.target.checked)}
                                        />
                                        <div className="w-[18px] h-[18px] bg-secondary border border-border rounded-md peer-checked:bg-primary peer-checked:border-primary transition-all shadow-sm"></div>
                                        <Check className="absolute inset-0 m-auto w-3 h-3 text-primary-foreground opacity-0 peer-checked:opacity-100 transition-opacity stroke-[3]" />
                                    </div>
                                    <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors">Ghi nhớ đăng nhập</span>
                                </label>
                            </div>
                        )}

                        {/* Submit Button */}
                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full h-12 flex items-center justify-center gap-2 bg-primary text-primary-foreground rounded-xl hover:bg-primary/90 transition-all font-semibold text-sm shadow-lg shadow-primary/20 hover:shadow-primary/30 disabled:opacity-50 disabled:shadow-none"
                        >
                            {loading ? (
                                <div className="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
                            ) : (
                                <div className="flex items-center gap-2">
                                    <span>{authMode === 'login' ? 'Đăng nhập' : 'Đăng ký'}</span>
                                    <ArrowRight className="w-4 h-4" />
                                </div>
                            )}
                        </button>
                    </form>

                    {/* Footer */}
                    <div className="mt-6 pt-6 border-t border-border flex justify-center">
                        <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground/60 font-medium tracking-wide uppercase">
                            <ShieldCheck className="w-3 h-3" />
                            <span>Multi-User Authentication</span>
                        </div>
                    </div>
                </div>

                <div className="mt-8 text-center">
                    <p className="text-[10px] text-muted-foreground/30 font-mono text-center">
                        DeepAPI Admin Portal
                    </p>
                </div>
            </div>
        </div>
    )
}
