import { useEffect, useState } from 'react'
import { TrendingUp, TrendingDown, DollarSign, Activity, Zap } from 'lucide-react'
import { useI18n } from '../../i18n'

export default function AnalyticsContainer({ authFetch, onMessage }) {
    const { t } = useI18n()
    const [loading, setLoading] = useState(true)
    const [overview, setOverview] = useState(null)

    useEffect(() => {
        loadOverview()
    }, [])

    const loadOverview = async () => {
        try {
            setLoading(true)
            const res = await authFetch('/admin/analytics/overview')
            if (!res.ok) {
                throw new Error(`HTTP ${res.status}`)
            }
            const data = await res.json()
            setOverview(data)
        } catch (err) {
            onMessage({ type: 'error', text: t('analytics.loadFailed') + ': ' + err.message })
        } finally {
            setLoading(false)
        }
    }

    const formatNumber = (num) => {
        if (num >= 1_000_000) {
            return (num / 1_000_000).toFixed(2) + 'M'
        }
        if (num >= 1_000) {
            return (num / 1_000).toFixed(2) + 'K'
        }
        return num.toLocaleString()
    }

    const formatCost = (cost, currency) => {
        if (currency === 'VND') {
            return (cost * 25000).toLocaleString('vi-VN') + ' ₫'
        }
        return '$' + cost.toFixed(4)
    }

    const calculateChange = (today, yesterday) => {
        if (!yesterday || yesterday === 0) return null
        const change = ((today - yesterday) / yesterday) * 100
        return change.toFixed(1)
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="text-muted-foreground">{t('analytics.loading')}</div>
            </div>
        )
    }

    if (!overview) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="text-muted-foreground">{t('analytics.noData')}</div>
            </div>
        )
    }

    const todayTokens = overview.today?.total_tokens || 0
    const yesterdayTokens = overview.yesterday?.total_tokens || 0
    const tokenChange = calculateChange(todayTokens, yesterdayTokens)

    const todayRequests = overview.today?.request_count || 0
    const yesterdayRequests = overview.yesterday?.request_count || 0
    const requestChange = calculateChange(todayRequests, yesterdayRequests)

    const todayCost = overview.today?.total_cost || 0
    const yesterdayCost = overview.yesterday?.total_cost || 0
    const costChange = calculateChange(todayCost, yesterdayCost)

    return (
        <div className="space-y-6">
            {/* Overview Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {/* Total Tokens Today */}
                <div className="bg-card border border-border rounded-lg p-6 glow-cyan hover:border-primary/50 transition-all">
                    <div className="flex items-center justify-between mb-2">
                        <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider">
                            {t('analytics.totalTokens')}
                        </div>
                        <Zap className="w-5 h-5 text-primary" />
                    </div>
                    <div className="text-3xl font-bold text-foreground mb-1">
                        {formatNumber(todayTokens)}
                    </div>
                    {tokenChange !== null && (
                        <div className={`flex items-center gap-1 text-sm ${parseFloat(tokenChange) >= 0 ? 'text-emerald-500' : 'text-red-500'}`}>
                            {parseFloat(tokenChange) >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
                            <span>{Math.abs(tokenChange)}% {t('analytics.compareYesterday')}</span>
                        </div>
                    )}
                </div>

                {/* Total Requests Today */}
                <div className="bg-card border border-border rounded-lg p-6 glow-cyan hover:border-primary/50 transition-all">
                    <div className="flex items-center justify-between mb-2">
                        <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider">
                            {t('analytics.totalRequests')}
                        </div>
                        <Activity className="w-5 h-5 text-neon-purple" />
                    </div>
                    <div className="text-3xl font-bold text-foreground mb-1">
                        {formatNumber(todayRequests)}
                    </div>
                    {requestChange !== null && (
                        <div className={`flex items-center gap-1 text-sm ${parseFloat(requestChange) >= 0 ? 'text-emerald-500' : 'text-red-500'}`}>
                            {parseFloat(requestChange) >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
                            <span>{Math.abs(requestChange)}% {t('analytics.compareYesterday')}</span>
                        </div>
                    )}
                </div>

                {/* Total Cost Today */}
                <div className="bg-card border border-border rounded-lg p-6 glow-cyan hover:border-primary/50 transition-all">
                    <div className="flex items-center justify-between mb-2">
                        <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider">
                            {t('analytics.totalCost')}
                        </div>
                        <DollarSign className="w-5 h-5 text-neon-green" />
                    </div>
                    <div className="text-3xl font-bold text-foreground mb-1">
                        {formatCost(todayCost, overview.currency)}
                    </div>
                    {costChange !== null && (
                        <div className={`flex items-center gap-1 text-sm ${parseFloat(costChange) >= 0 ? 'text-red-500' : 'text-emerald-500'}`}>
                            {parseFloat(costChange) >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
                            <span>{Math.abs(costChange)}% {t('analytics.compareYesterday')}</span>
                        </div>
                    )}
                </div>

                {/* Success Rate */}
                <div className="bg-card border border-border rounded-lg p-6 glow-cyan hover:border-primary/50 transition-all">
                    <div className="flex items-center justify-between mb-2">
                        <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider">
                            {t('analytics.successRate')}
                        </div>
                        <Activity className="w-5 h-5 text-emerald-500" />
                    </div>
                    <div className="text-3xl font-bold text-foreground mb-1">
                        {todayRequests > 0 ? ((overview.today.success_count / todayRequests) * 100).toFixed(1) : 0}%
                    </div>
                    <div className="text-sm text-muted-foreground">
                        {overview.today.success_count || 0} / {todayRequests} requests
                    </div>
                </div>
            </div>

            {/* Period Stats */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-card border border-border rounded-lg p-6">
                    <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider mb-3">
                        {t('analytics.last7Days')}
                    </div>
                    <div className="space-y-2">
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalTokens')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatNumber(overview.last_7_days?.total_tokens || 0)}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalRequests')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatNumber(overview.last_7_days?.request_count || 0)}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalCost')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatCost(overview.last_7_days?.total_cost || 0, overview.currency)}</span>
                        </div>
                    </div>
                </div>

                <div className="bg-card border border-border rounded-lg p-6">
                    <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider mb-3">
                        {t('analytics.last30Days')}
                    </div>
                    <div className="space-y-2">
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalTokens')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatNumber(overview.last_30_days?.total_tokens || 0)}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalRequests')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatNumber(overview.last_30_days?.request_count || 0)}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalCost')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatCost(overview.last_30_days?.total_cost || 0, overview.currency)}</span>
                        </div>
                    </div>
                </div>

                <div className="bg-card border border-border rounded-lg p-6">
                    <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider mb-3">
                        {t('analytics.yesterday')}
                    </div>
                    <div className="space-y-2">
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalTokens')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatNumber(yesterdayTokens)}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalRequests')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatNumber(yesterdayRequests)}</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-sm text-muted-foreground">{t('analytics.totalCost')}:</span>
                            <span className="text-sm font-semibold text-foreground">{formatCost(yesterdayCost, overview.currency)}</span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Top Lists */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                {/* Top Accounts */}
                <div className="bg-card border border-border rounded-lg p-6">
                    <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider mb-4">
                        {t('analytics.topAccounts')}
                    </div>
                    <div className="space-y-3">
                        {overview.top_accounts && overview.top_accounts.length > 0 ? (
                            overview.top_accounts.slice(0, 5).map((item, idx) => (
                                <div key={idx} className="flex items-center justify-between">
                                    <div className="flex items-center gap-2 flex-1 min-w-0">
                                        <span className="text-xs font-bold text-primary">{idx + 1}</span>
                                        <span className="text-sm text-foreground truncate">{item.account_id || 'Unknown'}</span>
                                    </div>
                                    <span className="text-sm font-semibold text-muted-foreground">{formatNumber(item.total_tokens)}</span>
                                </div>
                            ))
                        ) : (
                            <div className="text-sm text-muted-foreground text-center py-4">{t('analytics.noData')}</div>
                        )}
                    </div>
                </div>

                {/* Top API Keys */}
                <div className="bg-card border border-border rounded-lg p-6">
                    <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider mb-4">
                        {t('analytics.topCallers')}
                    </div>
                    <div className="space-y-3">
                        {overview.top_callers && overview.top_callers.length > 0 ? (
                            overview.top_callers.slice(0, 5).map((item, idx) => (
                                <div key={idx} className="flex items-center justify-between">
                                    <div className="flex items-center gap-2 flex-1 min-w-0">
                                        <span className="text-xs font-bold text-neon-purple">{idx + 1}</span>
                                        <span className="text-sm text-foreground truncate">{item.caller_id || 'Unknown'}</span>
                                    </div>
                                    <span className="text-sm font-semibold text-muted-foreground">{formatNumber(item.total_tokens)}</span>
                                </div>
                            ))
                        ) : (
                            <div className="text-sm text-muted-foreground text-center py-4">{t('analytics.noData')}</div>
                        )}
                    </div>
                </div>

                {/* Top Models */}
                <div className="bg-card border border-border rounded-lg p-6">
                    <div className="text-sm text-muted-foreground font-semibold uppercase tracking-wider mb-4">
                        {t('analytics.topModels')}
                    </div>
                    <div className="space-y-3">
                        {overview.top_models && overview.top_models.length > 0 ? (
                            overview.top_models.slice(0, 5).map((item, idx) => (
                                <div key={idx} className="flex items-center justify-between">
                                    <div className="flex items-center gap-2 flex-1 min-w-0">
                                        <span className="text-xs font-bold text-neon-green">{idx + 1}</span>
                                        <span className="text-sm text-foreground truncate">{item.model || 'Unknown'}</span>
                                    </div>
                                    <span className="text-sm font-semibold text-muted-foreground">{formatNumber(item.total_tokens)}</span>
                                </div>
                            ))
                        ) : (
                            <div className="text-sm text-muted-foreground text-center py-4">{t('analytics.noData')}</div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}
