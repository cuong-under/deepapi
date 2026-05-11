import { useState } from 'react'
import { LogOut, User, Key, Globe, Mail, Zap, UserCircle } from 'lucide-react'
import clsx from 'clsx'
import MultiUserAccounts from './MultiUserAccounts'
import MultiUserKeys from './MultiUserKeys'

export default function MultiUserDashboard({ user, token, onLogout, onMessage }) {
    const [activeTab, setActiveTab] = useState('accounts')

    const tabs = [
        { id: 'accounts', label: 'Accounts', icon: User },
        { id: 'keys', label: 'API Keys', icon: Key },
    ]

    return (
        <div className="min-h-screen bg-background text-foreground">
            {/* Header */}
            <div className="border-b border-border bg-card">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16">
                        <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                                <User className="w-4 h-4 text-primary" />
                            </div>
                            <div>
                                <h1 className="text-lg font-semibold">DeepAPI</h1>
                                <p className="text-xs text-muted-foreground">Multi-User Portal</p>
                            </div>
                        </div>

                        <div className="flex items-center gap-4">
                            <div className="hidden lg:flex items-center gap-2">
                                <a
                                    href="https://tm.cuong.tech"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="inline-flex h-9 items-center gap-2 rounded-lg border border-primary/30 bg-primary/10 px-3 text-xs font-medium text-primary hover:bg-primary/20 hover:border-primary/50 transition-all"
                                >
                                    <Mail className="w-3.5 h-3.5" />
                                    Tạo mail nhanh
                                </a>
                                <a
                                    href="https://vibekit.codes"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="inline-flex h-9 items-center gap-2 rounded-lg border border-neon-purple/30 bg-neon-purple/10 px-3 text-xs font-medium text-neon-purple hover:bg-neon-purple/20 hover:border-neon-purple/50 transition-all"
                                >
                                    <Zap className="w-3.5 h-3.5" />
                                    Skill hỗ trợ Vibecode
                                </a>
                                <a
                                    href="https://www.webshare.io/?referral_code=0ah52e2st71d"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="inline-flex h-9 items-center gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 text-xs font-medium text-emerald-400 hover:bg-emerald-500/20 hover:border-emerald-500/50 transition-all"
                                >
                                    <Globe className="w-3.5 h-3.5" />
                                    Lấy 10 proxy lifetime
                                </a>
                            </div>

                            <button
                                onClick={onLogout}
                                className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-secondary/50 rounded-lg transition-colors"
                            >
                                <LogOut className="w-4 h-4" />
                                <span>Đăng xuất</span>
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            {/* Tabs */}
            <div className="border-b border-border bg-card">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex gap-1">
                        {tabs.map(tab => {
                            const Icon = tab.icon
                            return (
                                <button
                                    key={tab.id}
                                    onClick={() => setActiveTab(tab.id)}
                                    className={clsx(
                                        'flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
                                        activeTab === tab.id
                                            ? 'border-primary text-primary'
                                            : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
                                    )}
                                >
                                    <Icon className="w-4 h-4" />
                                    <span>{tab.label}</span>
                                </button>
                            )
                        })}
                    </div>
                </div>
            </div>

            {/* Content */}
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {activeTab === 'accounts' && (
                    <MultiUserAccounts token={token} user={user} onMessage={onMessage} />
                )}
                {activeTab === 'keys' && (
                    <MultiUserKeys token={token} user={user} onMessage={onMessage} />
                )}

                <footer className="relative left-1/2 mt-12 w-screen -translate-x-1/2 border-t border-border/50 pt-8 px-4 sm:px-6 lg:px-8">
                    <div className="relative flex min-h-[92px] flex-col items-center justify-center gap-4 text-center">
                        <div className="inline-block px-4 py-2 rounded-lg border border-primary/30 bg-primary/5 glow-cyan lg:absolute lg:left-1/2 lg:top-0 lg:-translate-x-1/2">
                            <p className="text-sm text-muted-foreground font-medium mb-1">
                                Dự án phi lợi nhuận cho cộng đồng
                            </p>
                            <p className="text-lg font-bold text-primary text-glow-cyan tracking-wide">
                                VIBECODE VIETNAM
                            </p>
                        </div>
                        <div className="inline-flex h-12 max-w-[260px] items-center gap-2 rounded-full border border-primary/30 bg-card py-1 pl-1 pr-3 shadow-sm shadow-primary/10 lg:absolute lg:right-0 lg:top-0">
                            <div className="w-10 h-10 rounded-full bg-primary/10 text-primary flex items-center justify-center border border-primary/20 shrink-0">
                                <UserCircle className="w-5 h-5" />
                            </div>
                            <div className="min-w-0 text-left">
                                <div className="max-w-[170px] truncate text-xs font-semibold text-foreground" title={user.email || user.username}>
                                    {user.email || user.username}
                                </div>
                                <div className="mt-0.5 inline-flex items-center rounded-md border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[9px] font-semibold uppercase text-primary">
                                    {user.role || 'user'}
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="mt-3 flex items-center justify-center gap-3 text-sm text-muted-foreground">
                        <span className="font-semibold text-foreground">Cuongunder</span>
                        <span className="opacity-50">•</span>
                        <a
                            href="https://t.me/tiensinhcc"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-primary hover:text-primary/80 transition-colors font-medium inline-flex items-center gap-1 hover:glow-cyan"
                        >
                            <span>Telegram:</span>
                            <span className="font-bold">@tiensinhcc</span>
                        </a>
                    </div>
                </footer>
            </div>
        </div>
    )
}
