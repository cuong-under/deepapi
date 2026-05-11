import { useI18n } from '../../i18n'
import React from 'react'
import { useAccountsData } from './useAccountsData'
import { useAccountActions } from './useAccountActions'
import { useMultiUserAccounts } from './useMultiUserAccounts'
import QueueCards from './QueueCards'
import ApiKeysPanel from './ApiKeysPanel'
import AccountsTable from './AccountsTable'
import AddKeyModal from './AddKeyModal'
import AddAccountModal from './AddAccountModal'
import EditAccountModal from './EditAccountModal'

export default function AccountManagerContainer({ config, onRefresh, onMessage, authFetch }) {
    const { t } = useI18n()
    const apiFetch = authFetch || fetch

    const {
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
        changePageSize,
        resolveAccountIdentifier,
        searchQuery,
        handleSearchChange,
    } = useAccountsData({ apiFetch })

    const { isMultiUser, user, fetchKeys } = useMultiUserAccounts(apiFetch)
    const [apiKeys, setApiKeys] = React.useState([])

    // Fetch keys on mount and when isMultiUser changes
    React.useEffect(() => {
        if (isMultiUser !== null) {
            loadKeys()
        }
    }, [isMultiUser])

    const loadKeys = async () => {
        try {
            const data = await fetchKeys()
            setApiKeys(data.keys || [])
        } catch (e) {
            console.error('Failed to fetch keys:', e)
        }
    }

    // Merge config with fetched keys for multi-user mode
    const configWithKeys = React.useMemo(() => {
        if (isMultiUser) {
            return { ...config, api_keys: apiKeys }
        }
        return config
    }, [config, apiKeys, isMultiUser])

    const {
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
    } = useAccountActions({
        apiFetch,
        t,
        onMessage,
        onRefresh: () => {
            onRefresh()
            loadKeys() // Reload keys after any action
        },
        config,
        fetchAccounts,
        resolveAccountIdentifier,
    })

    return (
        <div className="space-y-6">
            {Boolean(config?.env_source_present) && (
                <div className={`rounded-xl border px-4 py-3 text-sm ${
                    config?.env_writeback_enabled
                        ? (config?.env_backed ? 'border-amber-500/30 bg-amber-500/10 text-amber-600' : 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600')
                        : 'border-amber-500/30 bg-amber-500/10 text-amber-600'
                }`}>
                    <p className="font-medium">
                        {config?.env_writeback_enabled
                            ? (config?.env_backed
                                ? t('accountManager.envModeWritebackPendingTitle')
                                : t('accountManager.envModeWritebackActiveTitle'))
                            : t('accountManager.envModeRiskTitle')}
                    </p>
                    <p className="mt-1 text-xs opacity-90">
                        {config?.env_writeback_enabled
                            ? t('accountManager.envModeWritebackDesc', { path: config?.config_path || 'config.json' })
                            : t('accountManager.envModeRiskDesc')}
                    </p>
                </div>
            )}

            <QueueCards queueStatus={queueStatus} t={t} />

            {isMultiUser && (
                <UserOverviewCards
                    user={user}
                    totalAccounts={totalAccounts}
                    apiKeys={apiKeys}
                    accounts={accounts}
                />
            )}

            <ApiKeysPanel
                t={t}
                config={configWithKeys}
                isMultiUser={isMultiUser}
                keysExpanded={keysExpanded}
                setKeysExpanded={setKeysExpanded}
                onAddKey={openAddKey}
                onEditKey={openEditKey}
                copiedKey={copiedKey}
                setCopiedKey={setCopiedKey}
                onDeleteKey={deleteKey}
            />

            <AccountsTable
                t={t}
                accounts={accounts}
                loadingAccounts={loadingAccounts}
                testing={testing}
                testingAll={testingAll}
                batchProgress={batchProgress}
                sessionCounts={sessionCounts}
                deletingSessions={deletingSessions}
                updatingProxy={updatingProxy}
                totalAccounts={totalAccounts}
                page={page}
                pageSize={pageSize}
                totalPages={totalPages}
                resolveAccountIdentifier={resolveAccountIdentifier}
                proxies={config?.proxies || []}
                onTestAll={testAllAccounts}
                onShowAddAccount={openAddAccount}
                onEditAccount={openEditAccount}
                onTestAccount={testAccount}
                onDeleteAccount={deleteAccount}
                onDeleteAllSessions={deleteAllSessions}
                onUpdateAccountProxy={updateAccountProxy}
                onPrevPage={() => fetchAccounts(page - 1)}
                onNextPage={() => fetchAccounts(page + 1)}
                onPageSizeChange={changePageSize}
                searchQuery={searchQuery}
                onSearchChange={handleSearchChange}
                envBacked={Boolean(config?.env_backed)}
                isMultiUser={isMultiUser}
            />

            <AddKeyModal
                show={showAddKey}
                t={t}
                editingKey={editingKey}
                newKey={newKey}
                setNewKey={setNewKey}
                loading={loading}
                onClose={closeKeyModal}
                onAdd={addKey}
                isMultiUser={isMultiUser}
            />

            <AddAccountModal
                show={showAddAccount}
                t={t}
                newAccount={newAccount}
                setNewAccount={setNewAccount}
                loading={loading}
                onClose={closeAddAccount}
                onAdd={addAccount}
            />

            <EditAccountModal
                show={showEditAccount}
                t={t}
                editingAccount={editingAccount}
                editAccount={editAccount}
                setEditAccount={setEditAccount}
                loading={loading}
                onClose={closeEditAccount}
                onSave={updateAccount}
            />
        </div>
    )
}

function UserOverviewCards({ user, totalAccounts, apiKeys, accounts }) {
    const activeAccounts = accounts.filter((account) => account.last_refreshed_at != null).length
    const displayName = user?.email || user?.username || 'User'

    return (
        <div className="grid gap-3 md:grid-cols-4">
            <OverviewCard label="Đang đăng nhập" value={displayName} compact />
            <OverviewCard label="Tài khoản DeepSeek" value={totalAccounts} />
            <OverviewCard label="API keys" value={apiKeys.length} />
            <OverviewCard label="Đã làm mới trang này" value={activeAccounts} />
        </div>
    )
}

function OverviewCard({ label, value, compact = false }) {
    return (
        <div className="rounded-lg border border-border bg-card p-4">
            <div className="text-xs font-medium uppercase text-muted-foreground">{label}</div>
            <div className={`mt-2 font-bold text-foreground ${compact ? 'truncate text-sm' : 'text-2xl'}`} title={String(value)}>
                {value}
            </div>
        </div>
    )
}
