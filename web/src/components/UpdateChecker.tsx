import { useState, useEffect, useRef } from 'react';
import { updatesAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/Card';
import { Download, RefreshCw, CheckCircle, XCircle, Loader2, AlertTriangle } from 'lucide-react';

interface UpdateInfo {
  current_version: string;
  latest_commit: string;
  local_commit: string;
  has_update: boolean;
  commit_message: string;
  commit_author: string;
  commit_date: string;
  last_checked: string;
}

interface UpdateStatus {
  is_running: boolean;
  progress: string;
  logs: string[];
  started_at: string;
  completed_at: string;
  success: boolean;
  error_message: string;
}

export function UpdateChecker() {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showLogs, setShowLogs] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);

  const checkForUpdates = async () => {
    setChecking(true);
    setError(null);
    try {
      const response = await updatesAPI.check();
      setUpdateInfo(response.data);
    } catch (err) {
      setError('Güncelleme kontrolü başarısız');
    } finally {
      setChecking(false);
    }
  };

  const startUpdate = async () => {
    try {
      await updatesAPI.run();
      setShowLogs(true);
    } catch (err) {
      setError('Güncelleme başlatılamadı');
    }
  };

  const fetchUpdateStatus = async () => {
    try {
      const response = await updatesAPI.getStatus();
      setUpdateStatus(response.data);
    } catch (err) {
      // Ignore errors
    }
  };

  useEffect(() => {
    checkForUpdates();
  }, []);

  useEffect(() => {
    let interval: ReturnType<typeof setInterval>;
    if (updateStatus?.is_running || showLogs) {
      interval = setInterval(fetchUpdateStatus, 1000);
    }
    return () => clearInterval(interval);
  }, [updateStatus?.is_running, showLogs]);

  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [updateStatus?.logs]);

  const formatDate = (dateStr: string) => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleString('tr-TR');
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <Download className="w-5 h-5" />
            Sistem Güncellemeleri
          </CardTitle>
          <Button
            variant="ghost"
            size="sm"
            onClick={checkForUpdates}
            disabled={checking || updateStatus?.is_running}
          >
            <RefreshCw className={`w-4 h-4 ${checking ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="p-3 rounded-md border border-destructive/40 bg-destructive/5 text-destructive text-sm flex items-center gap-2">
            <XCircle className="w-4 h-4" />
            {error}
          </div>
        )}

        {updateInfo && !updateStatus?.is_running && (
          <div className="space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Mevcut Versiyon:</span>
              <span className="font-mono">{updateInfo.local_commit || 'Bilinmiyor'}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Son Versiyon:</span>
              <span className="font-mono">{updateInfo.latest_commit}</span>
            </div>

            {updateInfo.has_update ? (
              <div className="p-3 rounded-md border border-yellow-500/40 bg-yellow-500/5 space-y-2">
                <div className="flex items-center gap-2 text-yellow-600 dark:text-yellow-500">
                  <AlertTriangle className="w-4 h-4" />
                  <span className="font-medium text-sm">Yeni güncelleme mevcut!</span>
                </div>
                <p className="text-xs text-muted-foreground line-clamp-2">
                  {updateInfo.commit_message}
                </p>
                <p className="text-xs text-muted-foreground">
                  {updateInfo.commit_author} • {formatDate(updateInfo.commit_date)}
                </p>
                <Button
                  onClick={startUpdate}
                  className="w-full mt-2"
                  size="sm"
                >
                  <Download className="w-4 h-4 mr-2" />
                  Güncelle
                </Button>
              </div>
            ) : (
              <div className="p-3 rounded-md border border-green-500/40 bg-green-500/5 flex items-center gap-2 text-green-600 dark:text-green-500">
                <CheckCircle className="w-4 h-4" />
                <span className="text-sm">Sistem güncel</span>
              </div>
            )}
          </div>
        )}

        {updateStatus?.is_running && (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-primary">
              <Loader2 className="w-4 h-4 animate-spin" />
              <span className="text-sm font-medium">Güncelleme devam ediyor...</span>
            </div>
            <div className="text-xs text-muted-foreground">
              {updateStatus.progress}
            </div>
          </div>
        )}

        {(showLogs || updateStatus?.is_running) && updateStatus?.logs && (
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium">Günlük</span>
              {!updateStatus.is_running && (
                <Button variant="ghost" size="sm" onClick={() => setShowLogs(false)}>
                  Kapat
                </Button>
              )}
            </div>
            <div className="bg-muted/50 rounded-md p-3 max-h-48 overflow-y-auto font-mono text-xs space-y-1">
              {updateStatus.logs.map((log, i) => (
                <div key={i} className={log.includes('[ERROR]') ? 'text-destructive' : ''}>
                  {log}
                </div>
              ))}
              <div ref={logsEndRef} />
            </div>
          </div>
        )}

        {updateStatus && !updateStatus.is_running && updateStatus.completed_at && (
          <div className={`p-3 rounded-md border ${
            updateStatus.success 
              ? 'border-green-500/40 bg-green-500/5 text-green-600 dark:text-green-500'
              : 'border-destructive/40 bg-destructive/5 text-destructive'
          }`}>
            <div className="flex items-center gap-2">
              {updateStatus.success ? (
                <>
                  <CheckCircle className="w-4 h-4" />
                  <span className="text-sm">Güncelleme tamamlandı! Sayfa yenilenecek...</span>
                </>
              ) : (
                <>
                  <XCircle className="w-4 h-4" />
                  <span className="text-sm">{updateStatus.error_message || 'Güncelleme başarısız'}</span>
                </>
              )}
            </div>
          </div>
        )}

        {updateInfo && (
          <p className="text-xs text-muted-foreground">
            Son kontrol: {formatDate(updateInfo.last_checked)}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
