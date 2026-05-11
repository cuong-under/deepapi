import { Check, Copy, X } from 'lucide-react'
import { useState } from 'react'
import { v4 as uuidv4 } from 'uuid'

import { maskSecret } from '../../utils/maskSecret'

export default function AddKeyModal({ show, t, editingKey, newKey, setNewKey, loading, onClose, onAdd, isMultiUser = false }) {
    const [copied, setCopied] = useState(false)

    if (!show) {
        return null
    }

    const isEditing = Boolean(editingKey?.key || editingKey?.id)
    const createdKey = newKey.createdKey || ''
    const displayKey = isEditing
        ? (editingKey?.key ? maskSecret(editingKey.key) : (editingKey?.api_key_preview || newKey.key))
        : newKey.key

    const copyCreatedKey = async () => {
        if (!createdKey) return
        await navigator.clipboard.writeText(createdKey)
        setCopied(true)
        setTimeout(() => setCopied(false), 1800)
    }

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4 animate-in fade-in">
            <div className="bg-card w-full max-w-md rounded-xl border border-border shadow-2xl overflow-hidden animate-in zoom-in-95">
                <div className="p-4 border-b border-border flex justify-between items-center">
                    <h3 className="font-semibold">{isEditing ? t('accountManager.modalEditKeyTitle') : t('accountManager.modalAddKeyTitle')}</h3>
                    <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
                        <X className="w-5 h-5" />
                    </button>
                </div>
                <div className="p-6 space-y-4">
                    {createdKey ? (
                        <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3">
                            <div className="text-sm font-semibold text-emerald-500">{t('accountManager.generatedKeyTitle')}</div>
                            <p className="mt-1 text-xs text-muted-foreground">{t('accountManager.generatedKeyHint')}</p>
                            <div className="mt-3 flex gap-2">
                                <code className="min-w-0 flex-1 overflow-x-auto rounded-md border border-border bg-background px-3 py-2 text-xs">{createdKey}</code>
                                <button
                                    type="button"
                                    onClick={copyCreatedKey}
                                    className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border hover:bg-muted"
                                    title={t('accountManager.copyKeyTitle')}
                                >
                                    {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div>
                        <label className="block text-sm font-medium mb-1.5">{isEditing ? t('accountManager.keyLabel') : t('accountManager.newKeyLabel')}</label>
                            {isMultiUser && !isEditing ? (
                                <div className="rounded-lg border border-border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                                    {t('accountManager.autoGenerateKeyHint')}
                                </div>
                            ) : (
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        className={isEditing ? "input-field bg-muted/30 flex-1 cursor-not-allowed" : "input-field bg-[#09090b] flex-1"}
                                        placeholder={isEditing ? t('accountManager.keyReadonlyPlaceholder') : t('accountManager.newKeyPlaceholder')}
                                        value={displayKey}
                                        onChange={e => setNewKey({ ...newKey, key: e.target.value })}
                                        autoFocus={!isEditing}
                                        readOnly={isEditing}
                                    />
                                    {!isEditing && (
                                        <button
                                            type="button"
                                            onClick={() => setNewKey({ ...newKey, key: 'sk-' + uuidv4().replace(/-/g, '') })}
                                            className="px-3 py-2 bg-secondary text-secondary-foreground rounded-lg hover:bg-secondary/80 transition-colors text-sm font-medium border border-border whitespace-nowrap"
                                        >
                                            {t('accountManager.generate')}
                                        </button>
                                    )}
                                </div>
                            )}
                        <p className="text-xs text-muted-foreground mt-1.5">
                            {isEditing ? t('accountManager.keyReadonlyHint') : (isMultiUser ? t('accountManager.keyOneTimeHint') : t('accountManager.generateHint'))}
                        </p>
                        </div>
                    )}
                    <div>
                        <label className="block text-sm font-medium mb-1.5">{t('accountManager.nameOptional')}</label>
                        <input
                            type="text"
                            className="input-field"
                            placeholder={t('accountManager.namePlaceholder')}
                            value={newKey.name}
                            onChange={e => setNewKey({ ...newKey, name: e.target.value })}
                            autoFocus={isEditing}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1.5">{t('accountManager.remarkOptional')}</label>
                        <input
                            type="text"
                            className="input-field"
                            placeholder={t('accountManager.remarkPlaceholder')}
                            value={newKey.remark}
                            onChange={e => setNewKey({ ...newKey, remark: e.target.value })}
                        />
                    </div>
                    <div className="flex justify-end gap-2 pt-2">
                        <button onClick={onClose} className="px-4 py-2 rounded-lg border border-border hover:bg-secondary transition-colors text-sm font-medium">{t('actions.cancel')}</button>
                        {!createdKey && (
                            <button onClick={onAdd} disabled={loading} className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors text-sm font-medium disabled:opacity-50">
                                {loading
                                    ? (isEditing ? t('accountManager.editKeyLoading') : t('accountManager.addKeyLoading'))
                                    : (isEditing ? t('accountManager.editKeyAction') : t('accountManager.addKeyAction'))}
                            </button>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}
