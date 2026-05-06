import { Suspense, lazy, useCallback, useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
    LayoutDashboard,
    Upload,
    Cloud,
    Settings as SettingsIcon,
    LogOut,
    Menu,
    X,
    Server,
    Users,
    Globe,
    History,
    Loader2,
    BarChart3,
    RefreshCw
} from 'lucide-react'
import clsx from 'clsx'

import LanguageToggle from '../components/LanguageToggle'
import { useI18n } from '../i18n'
import logoImage from '/ds2api-favicon.svg'

const AccountManagerContainer = lazy(() => import('../features/account/AccountManagerContainer'))
const ApiTesterContainer = lazy(() => import('../features/apiTester/ApiTesterContainer'))
const ChatHistoryContainer = lazy(() => import('../features/chatHistory/ChatHistoryContainer'))
const BatchImport = lazy(() => import('../components/BatchImport'))
const VercelSyncContainer = lazy(() => import('../features/vercel/VercelSyncContainer'))
const SettingsContainer = lazy(() => import('../features/settings/SettingsContainer'))
const ProxyManagerContainer = lazy(() => import('../features/proxy/ProxyManagerContainer'))
const AnalyticsContainer = lazy(() => import('../features/analytics/AnalyticsContainer'))
const UpdateContainer = lazy(() => import('../features/update/UpdateContainer'))

function TabLoadingFallback({ label }) {
    return (
        <div className="min-h-[320px] rounded-lg border border-border bg-card flex items-center justify-center">
            <div className="flex items-center gap-3 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>{label}</span>
            </div>
        </div>
    )
}

export default function DashboardShell({ token, onLogout, config, fetchConfig, showMessage, message, onForceLogout, isVercel }) {
    const { t } = useI18n()
    const location = useLocation()
    const navigate = useNavigate()
    const [sidebarOpen, setSidebarOpen] = useState(false)

    const navItems = [
        { id: 'accounts', label: t('nav.accounts.label'), icon: Users, description: t('nav.accounts.desc') },
        { id: 'proxies', label: t('nav.proxies.label'), icon: Globe, description: t('nav.proxies.desc') },
        { id: 'analytics', label: t('nav.analytics.label'), icon: BarChart3, description: t('nav.analytics.desc') },
        { id: 'test', label: t('nav.test.label'), icon: Server, description: t('nav.test.desc') },
        { id: 'history', label: t('nav.history.label'), icon: History, description: t('nav.history.desc') },
        { id: 'import', label: t('nav.import.label'), icon: Upload, description: t('nav.import.desc') },
        // { id: 'vercel', label: t('nav.vercel.label'), icon: Cloud, description: t('nav.vercel.desc') },
        { id: 'update', label: t('nav.update.label'), icon: RefreshCw, description: t('nav.update.desc') },
        { id: 'settings', label: t('nav.settings.label'), icon: SettingsIcon, description: t('nav.settings.desc') },
    ]

    const tabIds = new Set(navItems.map(item => item.id))
    const pathSegments = location.pathname.replace(/^\/+|\/+$/g, '').split('/').filter(Boolean)
    const routeSegments = pathSegments[0] === 'admin' ? pathSegments.slice(1) : pathSegments
    const pathTab = routeSegments[0] || ''
    const activeTab = tabIds.has(pathTab) ? pathTab : 'accounts'
    const adminBasePath = pathSegments[0] === 'admin' ? '/admin' : ''
    const activeNavItem = navItems.find(n => n.id === activeTab)

    const navigateToTab = useCallback((tabID) => {
        const nextPath = tabID === 'accounts'
            ? `${adminBasePath || ''}/`
            : `${adminBasePath}/${tabID}`
        navigate(nextPath)
        setSidebarOpen(false)
    }, [adminBasePath, navigate])

    const authFetch = useCallback(async (url, options = {}) => {
        const headers = {
            ...options.headers,
            'Authorization': `Bearer ${token}`
        }
        const res = await fetch(url, { ...options, headers })

        if (res.status === 401) {
            onLogout()
            throw new Error(t('auth.expired'))
        }
        return res
    }, [onLogout, t, token])


    const [versionInfo, setVersionInfo] = useState(null)

    useEffect(() => {
        let disposed = false
        async function loadVersion() {
            try {
                const res = await authFetch('/admin/version')
                const data = await res.json()
                if (!disposed) {
                    setVersionInfo(data)
                }
            } catch (_err) {
                if (!disposed) {
                    setVersionInfo(null)
                }
            }
        }
        loadVersion()
        return () => {
            disposed = true
        }
    }, [authFetch])
    const renderTab = () => {
        switch (activeTab) {
            case 'accounts':
                return <AccountManagerContainer config={config} onRefresh={fetchConfig} onMessage={showMessage} authFetch={authFetch} />
            case 'proxies':
                return <ProxyManagerContainer config={config} onRefresh={fetchConfig} onMessage={showMessage} authFetch={authFetch} />
            case 'analytics':
                return <AnalyticsContainer onMessage={showMessage} authFetch={authFetch} />
            case 'test':
                return <ApiTesterContainer config={config} onMessage={showMessage} authFetch={authFetch} />
            case 'history':
                return <ChatHistoryContainer onMessage={showMessage} authFetch={authFetch} />
            case 'import':
                return <BatchImport onRefresh={fetchConfig} onMessage={showMessage} authFetch={authFetch} />
            case 'vercel':
                return <VercelSyncContainer onMessage={showMessage} authFetch={authFetch} isVercel={isVercel} config={config} />
            case 'update':
                return <UpdateContainer authFetch={authFetch} />
            case 'settings':
                return <SettingsContainer onRefresh={fetchConfig} onMessage={showMessage} authFetch={authFetch} onForceLogout={onForceLogout} isVercel={isVercel} />
            default:
                return null
        }
    }

    return (
        <div className="flex h-screen bg-background overflow-hidden text-foreground">
            {sidebarOpen && (
                <div
                    className="fixed inset-0 bg-background/80 backdrop-blur-sm z-40 lg:hidden"
                    onClick={() => setSidebarOpen(false)}
                />
            )}

            <aside className={clsx(
                "fixed lg:static inset-y-0 left-0 z-50 w-64 bg-card border-r border-border transition-transform duration-300 ease-in-out lg:transform-none flex flex-col shadow-2xl lg:shadow-none",
                sidebarOpen ? "translate-x-0" : "-translate-x-full"
            )}>
                <div className="p-6">
                    <div className="flex items-center gap-2.5 font-bold text-xl text-foreground tracking-tight">
                        <img
                            src={logoImage}
                            alt="DeepAPI Logo"
                            className="w-8 h-8 rounded-lg shadow-lg shadow-primary/20"
                        />
                        <span>DeepAPI</span>
                    </div>
                    <div className="flex items-center justify-between mt-2">
                        <p className="text-[10px] text-muted-foreground font-semibold tracking-[0.1em] uppercase opacity-60 px-1">{t('sidebar.onlineAdminConsole')}</p>
                        <LanguageToggle />
                    </div>
                </div>

                <nav className="flex-1 px-3 space-y-1 overflow-y-auto pt-2">
                    {navItems.map((item) => {
                        const Icon = item.icon
                        const isActive = activeTab === item.id
                        return (
                            <button
                                key={item.id}
                                onClick={() => {
                                    navigateToTab(item.id)
                                }}
                                className={clsx(
                                    "w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 group border",
                                    isActive
                                        ? "bg-secondary text-primary border-border shadow-sm"
                                        : "text-muted-foreground border-transparent hover:bg-secondary/80 hover:text-foreground"
                                )}
                            >
                                <Icon className={clsx("w-4 h-4 transition-colors", isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground")} />
                                <span className="flex-1 text-left">{item.label}</span>
                                {isActive && <div className="w-1.5 h-1.5 rounded-full bg-primary" />}
                            </button>
                        )
                    })}
                </nav>

                <div className="p-4 border-t border-border bg-card">
                    <div className="space-y-4">
                        <div className="flex items-center justify-between text-sm px-1">
                            <span className="text-muted-foreground font-semibold text-[10px] uppercase tracking-wider">{t('sidebar.systemStatus')}</span>
                            <span className="flex items-center gap-1.5 text-[10px] font-bold text-emerald-500 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                                {t('sidebar.statusOnline')}
                            </span>
                        </div>
                        <div className="grid grid-cols-2 gap-2">
                            <div className="bg-background rounded-lg p-3 border border-border shadow-sm">
                                <div className="text-[9px] text-muted-foreground font-bold uppercase tracking-wider mb-0.5 opacity-70">{t('sidebar.accounts')}</div>
                                <div className="text-lg font-bold text-foreground leading-tight">{config.accounts?.length || 0}</div>
                            </div>
                            <div className="bg-background rounded-lg p-3 border border-border shadow-sm">
                                <div className="text-[9px] text-muted-foreground font-bold uppercase tracking-wider mb-0.5 opacity-70">{t('sidebar.keys')}</div>
                                <div className="text-lg font-bold text-foreground">{config.keys?.length || 0}</div>
                            </div>
                        </div>
                        <div className="bg-background rounded-lg p-3 border border-border shadow-sm">
                            <div className="text-[9px] text-muted-foreground font-bold uppercase tracking-wider mb-1 opacity-70">{t('sidebar.version')}</div>
                            <div className="text-xs font-semibold text-foreground">{versionInfo?.current_tag || '-'}</div>
                            {versionInfo?.has_update && (
                                <a
                                    className="inline-flex mt-1 text-[10px] text-amber-500 hover:text-amber-400"
                                    href={versionInfo?.release_url || 'https://github.com/CJackHwang/ds2api/releases/latest'}
                                    target="_blank"
                                    rel="noreferrer"
                                >
                                    {t('sidebar.updateAvailable', { latest: versionInfo.latest_tag || '' })}
                                </a>
                            )}
                        </div>
                        <a
                            href="https://tm.cuong.tech"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="w-full h-10 flex items-center justify-center gap-2 rounded-lg border border-primary/30 bg-primary/10 text-xs font-medium text-primary hover:bg-primary/20 hover:border-primary/50 transition-all glow-cyan"
                        >
                            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                            </svg>
                            Tạo mail nhanh
                        </a>
                        <a
                            href="https://vibekit.codes"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="w-full h-10 flex items-center justify-center gap-2 rounded-lg border border-neon-purple/30 bg-neon-purple/10 text-xs font-medium text-neon-purple hover:bg-neon-purple/20 hover:border-neon-purple/50 transition-all glow-pink"
                        >
                            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                            </svg>
                            Skill hỗ trợ Vibecode
                        </a>
                        <button
                            onClick={onLogout}
                            className="w-full h-10 flex items-center justify-center gap-2 rounded-lg border border-border text-xs font-medium text-muted-foreground hover:bg-destructive/10 hover:text-destructive hover:border-destructive/20 transition-all"
                        >
                            <LogOut className="w-3.5 h-3.5" />
                            {t('sidebar.signOut')}
                        </button>
                    </div>
                </div>
            </aside>

            <main className="flex-1 flex flex-col min-w-0 overflow-hidden relative">
                <header className="lg:hidden h-14 flex items-center justify-between px-4 border-b border-border bg-card">
                    <div className="flex items-center gap-2">
                        <img
                            src={logoImage}
                            alt="DeepAPI Logo"
                            className="w-6 h-6 rounded"
                        />
                        <span className="font-semibold text-sm">DeepAPI</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <LanguageToggle />
                        <button
                            onClick={() => setSidebarOpen(true)}
                            className="p-2 -mr-2 text-muted-foreground hover:text-foreground"
                        >
                            <Menu className="w-5 h-5" />
                        </button>
                    </div>
                </header>

                <div className="flex-1 overflow-auto bg-background p-4 lg:p-10">
                    <div className="max-w-6xl mx-auto space-y-4 lg:space-y-6">
                        <div className="hidden lg:block mb-8">
                            <h1 className="text-3xl font-bold tracking-tight mb-2">
                                {activeNavItem?.label}
                            </h1>
                            <p className="text-muted-foreground">
                                {activeNavItem?.description}
                            </p>
                        </div>

                        {message && (
                            <div className={clsx(
                                "p-4 rounded-lg border flex items-center gap-3 animate-in fade-in slide-in-from-top-2",
                                message.type === 'error' ? "bg-destructive/10 border-destructive/20 text-destructive" :
                                    "bg-emerald-500/10 border-emerald-500/20 text-emerald-500"
                            )}>
                                {message.type === 'error' ? <X className="w-5 h-5" /> : <div className="w-5 h-5 rounded-full border-2 border-emerald-500 flex items-center justify-center text-[10px]">✓</div>}
                                {message.text}
                            </div>
                        )}

                        <div className="animate-in fade-in duration-500">
                            <Suspense fallback={<TabLoadingFallback label={activeNavItem?.label || 'DeepAPI'} />}>
                                {renderTab()}
                            </Suspense>
                        </div>

                        {/* Footer - Community Info */}
                        <footer className="mt-12 pt-8 border-t border-border/50">
                            <div className="text-center space-y-3">
                                <div className="inline-block px-4 py-2 rounded-lg border border-primary/30 bg-primary/5 glow-cyan">
                                    <p className="text-sm text-muted-foreground font-medium mb-1">
                                        Dự án phi lợi nhuận cho cộng đồng
                                    </p>
                                    <p className="text-lg font-bold text-primary text-glow-cyan tracking-wide">
                                        VIBECODE VIETNAM
                                    </p>
                                </div>
                                <div className="flex items-center justify-center gap-3 text-sm text-muted-foreground">
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
                            </div>
                        </footer>
                    </div>
                </div>
            </main>
        </div>
    )
}
