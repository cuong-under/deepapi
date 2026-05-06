import React, { useState, useEffect } from 'react';
import { RefreshCw, Download, AlertTriangle, CheckCircle, Clock, Package, ArrowLeft } from 'lucide-react';
import { useI18n } from '../../i18n';

export default function UpdateContainer({ authFetch }) {
  const { t } = useI18n();
  const [updateInfo, setUpdateInfo] = useState(null);
  const [isChecking, setIsChecking] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [updateStatus, setUpdateStatus] = useState(null);
  const [backups, setBackups] = useState([]);
  const [error, setError] = useState(null);

  // Check for updates on mount
  useEffect(() => {
    checkForUpdates();
    loadBackups();
  }, []);

  // Poll status when updating
  useEffect(() => {
    if (isUpdating) {
      const interval = setInterval(() => {
        fetchUpdateStatus();
      }, 2000);
      return () => clearInterval(interval);
    }
  }, [isUpdating]);

  const checkForUpdates = async () => {
    setIsChecking(true);
    setError(null);
    try {
      const res = await authFetch('/admin/update/check');
      if (!res.ok) throw new Error('Failed to check for updates');
      const data = await res.json();
      setUpdateInfo(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setIsChecking(false);
    }
  };

  const installUpdate = async () => {
    if (!confirm(t('update.confirm_install'))) return;

    setIsUpdating(true);
    setError(null);
    try {
      const res = await authFetch('/admin/update/install', {
        method: 'POST'
      });
      if (!res.ok) throw new Error('Failed to start update');
      // Status will be polled automatically
    } catch (err) {
      setError(err.message);
      setIsUpdating(false);
    }
  };

  const fetchUpdateStatus = async () => {
    try {
      const res = await authFetch('/admin/update/status');
      if (!res.ok) throw new Error('Failed to fetch status');
      const data = await res.json();
      setUpdateStatus(data);

      if (data.stage === 'complete' || data.stage === 'failed') {
        setIsUpdating(false);
      }
    } catch (err) {
      console.error('Failed to fetch update status:', err);
    }
  };

  const loadBackups = async () => {
    try {
      const res = await authFetch('/admin/update/backups');
      if (!res.ok) throw new Error('Failed to load backups');
      const data = await res.json();
      setBackups(data.backups || []);
    } catch (err) {
      console.error('Failed to load backups:', err);
    }
  };

  const rollbackToBackup = async (backupName) => {
    if (!confirm(t('update.confirm_rollback'))) return;

    try {
      const res = await authFetch('/admin/update/rollback', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ backup_name: backupName })
      });
      if (!res.ok) throw new Error('Failed to rollback');
      alert(t('update.rollback_success'));
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-cyan-400 glow-cyan tracking-wider">
            {t('update.title')}
          </h1>
          <p className="text-sm text-cyan-400/60 mt-1">
            {t('update.subtitle')}
          </p>
        </div>
        <button
          onClick={checkForUpdates}
          disabled={isChecking || isUpdating}
          className="btn-cyber flex items-center gap-2"
        >
          <RefreshCw className={`w-4 h-4 ${isChecking ? 'animate-spin' : ''}`} />
          {t('update.check_updates')}
        </button>
      </div>

      {/* Error Alert */}
      {error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4 flex items-start gap-3">
          <AlertTriangle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-red-400 font-medium">{t('update.error')}</p>
            <p className="text-red-400/80 text-sm mt-1">{error}</p>
          </div>
        </div>
      )}

      {/* Update Available Card */}
      {updateInfo && updateInfo.update_available && (
        <div className="cyber-card p-6 border-2 border-cyan-500/50 glow-cyan">
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-3">
                <Download className="w-6 h-6 text-cyan-400" />
                <h2 className="text-xl font-bold text-cyan-400">
                  {t('update.new_version_available')}
                </h2>
              </div>

              <div className="space-y-2 mb-4">
                <div className="flex items-center gap-2 text-sm">
                  <span className="text-cyan-400/60">{t('update.current_version')}:</span>
                  <span className="text-cyan-400 font-mono">{updateInfo.current_version}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <span className="text-cyan-400/60">{t('update.latest_version')}:</span>
                  <span className="text-cyan-400 font-mono font-bold">{updateInfo.latest_version}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <span className="text-cyan-400/60">{t('update.published')}:</span>
                  <span className="text-cyan-400">{new Date(updateInfo.published_at).toLocaleDateString('vi-VN')}</span>
                </div>
              </div>

              {updateInfo.release_notes && (
                <div className="bg-black/30 rounded p-3 mb-4">
                  <p className="text-xs text-cyan-400/60 mb-2">{t('update.release_notes')}:</p>
                  <div className="text-sm text-cyan-400/80 whitespace-pre-wrap max-h-40 overflow-y-auto">
                    {updateInfo.release_notes}
                  </div>
                </div>
              )}
            </div>
          </div>

          <button
            onClick={installUpdate}
            disabled={isUpdating}
            className="btn-cyber w-full flex items-center justify-center gap-2 bg-cyan-500/20 hover:bg-cyan-500/30"
          >
            <Download className="w-4 h-4" />
            {t('update.install_now')}
          </button>
        </div>
      )}

      {/* Up to Date Card */}
      {updateInfo && !updateInfo.update_available && (
        <div className="cyber-card p-6 border border-green-500/30">
          <div className="flex items-center gap-3">
            <CheckCircle className="w-6 h-6 text-green-400" />
            <div>
              <h2 className="text-lg font-bold text-green-400">{t('update.up_to_date')}</h2>
              <p className="text-sm text-green-400/60">
                {t('update.current_version')}: {updateInfo.current_version}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Update Progress */}
      {isUpdating && updateStatus && (
        <div className="cyber-card p-6 border-2 border-cyan-500/50 glow-cyan">
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <RefreshCw className="w-6 h-6 text-cyan-400 animate-spin" />
              <h2 className="text-xl font-bold text-cyan-400">{t('update.updating')}</h2>
            </div>

            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-cyan-400/60">{updateStatus.message}</span>
                <span className="text-cyan-400 font-mono">{updateStatus.progress}%</span>
              </div>
              <div className="h-2 bg-black/50 rounded-full overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 transition-all duration-500 glow-cyan"
                  style={{ width: `${updateStatus.progress}%` }}
                />
              </div>
            </div>

            {/* Stage indicator */}
            <div className="bg-black/30 rounded p-3">
              <div className="flex items-center gap-2 text-sm">
                {updateStatus.stage === 'checking' && (
                  <>
                    <RefreshCw className="w-4 h-4 text-cyan-400 animate-spin" />
                    <span className="text-cyan-400">{t('update.stage_checking')}</span>
                  </>
                )}
                {updateStatus.stage === 'downloading' && (
                  <>
                    <Download className="w-4 h-4 text-cyan-400 animate-pulse" />
                    <span className="text-cyan-400">{t('update.stage_downloading')}</span>
                  </>
                )}
                {updateStatus.stage === 'backing_up' && (
                  <>
                    <Package className="w-4 h-4 text-cyan-400 animate-pulse" />
                    <span className="text-cyan-400">{t('update.stage_backing_up')}</span>
                  </>
                )}
                {updateStatus.stage === 'installing' && (
                  <>
                    <RefreshCw className="w-4 h-4 text-cyan-400 animate-spin" />
                    <span className="text-cyan-400">{t('update.stage_installing')}</span>
                  </>
                )}
                {updateStatus.stage === 'verifying' && (
                  <>
                    <CheckCircle className="w-4 h-4 text-cyan-400 animate-pulse" />
                    <span className="text-cyan-400">{t('update.stage_verifying')}</span>
                  </>
                )}
                {updateStatus.stage === 'complete' && (
                  <>
                    <CheckCircle className="w-4 h-4 text-green-400" />
                    <span className="text-green-400">{t('update.stage_complete')}</span>
                  </>
                )}
              </div>
            </div>

            {updateStatus.error && (
              <div className="bg-red-500/10 border border-red-500/30 rounded p-3">
                <p className="text-red-400 text-sm">{updateStatus.error}</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Success Message - Restart Required */}
      {updateStatus && updateStatus.stage === 'complete' && !isUpdating && (
        <div className="bg-green-500/10 border-2 border-green-500/30 rounded-lg p-6 glow-green">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-6 h-6 text-green-400 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-green-400 font-bold text-lg mb-2">{t('update.success_title')}</p>
              <p className="text-green-400/80 text-sm mb-4">{t('update.success_message')}</p>
              <div className="bg-black/30 rounded p-3 border border-green-500/20">
                <p className="text-xs text-green-400/60 mb-2">{t('update.restart_instructions')}:</p>
                <ol className="list-decimal list-inside space-y-1 text-sm text-green-400">
                  <li>{t('update.restart_step1')}</li>
                  <li>{t('update.restart_step2')}</li>
                  <li>{t('update.restart_step3')}</li>
                </ol>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Backups List */}
      <div className="cyber-card p-6">
        <div className="flex items-center gap-3 mb-4">
          <Package className="w-5 h-5 text-cyan-400" />
          <h2 className="text-lg font-bold text-cyan-400">{t('update.backups')}</h2>
          <span className="text-sm text-cyan-400/60">({backups.length})</span>
        </div>

        {backups.length === 0 ? (
          <p className="text-cyan-400/60 text-sm">{t('update.no_backups')}</p>
        ) : (
          <div className="space-y-2">
            {backups.map((backup) => (
              <div
                key={backup.name}
                className="bg-black/30 rounded p-3 flex items-center justify-between hover:bg-black/50 transition-colors"
              >
                <div className="flex-1">
                  <p className="text-cyan-400 font-mono text-sm">{backup.name}</p>
                  <div className="flex items-center gap-4 mt-1">
                    <span className="text-xs text-cyan-400/60">
                      <Clock className="w-3 h-3 inline mr-1" />
                      {new Date(backup.created_at).toLocaleString('vi-VN')}
                    </span>
                    {backup.binary_exists && (
                      <span className="text-xs text-green-400/60">
                        Binary: {(backup.binary_size / 1024 / 1024).toFixed(1)} MB
                      </span>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => rollbackToBackup(backup.name)}
                  className="btn-cyber text-xs px-3 py-1 flex items-center gap-1"
                >
                  <ArrowLeft className="w-3 h-3" />
                  {t('update.rollback')}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Warning */}
      <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <AlertTriangle className="w-5 h-5 text-yellow-400 flex-shrink-0 mt-0.5" />
          <div className="text-sm text-yellow-400/80">
            <p className="font-medium text-yellow-400 mb-1">{t('update.warning_title')}</p>
            <ul className="list-disc list-inside space-y-1">
              <li>{t('update.warning_1')}</li>
              <li>{t('update.warning_2')}</li>
              <li>{t('update.warning_3')}</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
