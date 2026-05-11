import { useState } from 'react'
import { User, Lock, ArrowRight, ShieldCheck, UserPlus, Mail } from 'lucide-react'
import clsx from 'clsx'
import { useI18n } from '../../i18n'
import LanguageToggle from '../../components/LanguageToggle'

export default function MultiUserLogin({ onLogin, onMessage }) {
    const { t } = useI18n()
    const [mode, setMode] = useState('login') // 'login' or 'register'
    const [username, setUsername] = useState('')
    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')
    const [loading, setLoading] = useState(false)

    const handleLogin = async (e) => {
        e.preventDefault()
        if (!username.trim() || !password.trim()) {
            onMessage('error', 'Username và password không được để trống')
            return
        }

        setLoading(true)

        try {
            const res = await fetch('/api/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
            })

            const data = await res.json()

            if (res.ok && data.token) {
                localStorage.setItem('ds2api_multiuser_token', data.token)
                localStorage.setItem('ds2api_multiuser_user', JSON.stringify(data.user))
                localStorage.setItem('ds2api_multiuser_expires', Date.now() + data.expires_in * 1000)

                onLogin(data.token, data.user)
                onMessage('success', `Chào mừng ${data.user.username}!`)
            } else {
                onMessage('error', data.error || 'Đăng nhập thất bại')
            }
        } catch (e) {
            onMessage('error', `Lỗi kết nối: ${e.message}`)
        } finally {
            setLoading(false)
        }
    }

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
                localStorage.setItem('ds2api_multiuser_token', data.token)
                localStorage.setItem('ds2api_multiuser_user', JSON.stringify(data.user))
                localStorage.setItem('ds2api_multiuser_expires', Date.now() + data.expires_in * 1000)

                onLogin(data.token, data.user)
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
                            {mode === 'login' ? <Lock className="w-6 h-6" /> : <UserPlus className="w-6 h-6" />}
                        </div>
                        <h1 className="text-3xl font-bold tracking-tight text-foreground">
                            {mode === 'login' ? 'Đăng nhập' : 'Đăng ký'}
                        </h1>
                        <p className="text-sm text-muted-foreground/80">
                            {mode === 'login' ? 'Đăng nhập vào tài khoản của bạn' : 'Tạo tài khoản mới'}
                        </p>
                    </div>

                    {/* Mode Toggle */}
                    <div className="flex gap-2 mb-6 p-1 bg-secondary/50 rounded-lg">
                        <button
                            type="button"
                            onClick={() => setMode('login')}
                            className={clsx(
                                'flex-1 py-2 px-4 rounded-md text-sm font-medium transition-all',
                                mode === 'login'
                                    ? 'bg-primary text-primary-foreground shadow-sm'
                                    : 'text-muted-foreground hover:text-foreground'
                            )}
                        >
                            Đăng nhập
                        </button>
                        <button
                            type="button"
                            onClick={() => setMode('register')}
                            className={clsx(
                                'flex-1 py-2 px-4 rounded-md text-sm font-medium transition-all',
                                mode === 'register'
                                    ? 'bg-primary text-primary-foreground shadow-sm'
                                    : 'text-muted-foreground hover:text-foreground'
                            )}
                        >
                            Đăng ký
                        </button>
                    </div>

                    {/* Form */}
                    <form onSubmit={mode === 'login' ? handleLogin : handleRegister} className="space-y-5">
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
                        {mode === 'register' && (
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
                                    placeholder={mode === 'register' ? 'Tối thiểu 8 ký tự' : 'Nhập password'}
                                    value={password}
                                    onChange={e => setPassword(e.target.value)}
                                />
                            </div>
                        </div>

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
                                    <span>{mode === 'login' ? 'Đăng nhập' : 'Đăng ký'}</span>
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
                        DeepAPI Multi-User Portal
                    </p>
                </div>
            </div>
        </div>
    )
}
