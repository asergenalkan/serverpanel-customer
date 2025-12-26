import { useState, useEffect, useRef } from 'react';
import { updatesAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/Card';
import { Download, RefreshCw, CheckCircle, XCircle, Loader2, AlertTriangle, Server } from 'lucide-react';

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
  const [showConfirm, setShowConfirm] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [waitingForServer, setWaitingForServer] = useState(false);
  const [serverCheckCount, setServerCheckCount] = useState(0);
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
    setShowConfirm(false);
    setIsUpdating(true);
    setError(null);
    
    try {
      await updatesAPI.run();
      setShowLogs(true);
      
      // Kısa bir süre sonra sunucu bağlantısı kopacak, bekleme moduna geç
      setTimeout(() => {
        setWaitingForServer(true);
        setServerCheckCount(0);
      }, 2000);
    } catch (err) {
      setError('Güncelleme başlatılamadı');
      setIsUpdating(false);
    }
  };

  const fetchUpdateStatus = async () => {
    try {
      const response = await updatesAPI.getStatus();
      setUpdateStatus(response.data);
    } catch (err) {
      // Bağlantı kopmuş olabilir, bekleme moduna geç
      if (isUpdating) {
        setWaitingForServer(true);
      }
    }
  };

  // Sunucu tekrar çalışıyor mu kontrol et
  const checkServerHealth = async () => {
    try {
      const response = await fetch('/api/v1/health');
      if (response.ok) {
        // Sunucu geri geldi!
        setWaitingForServer(false);
        setIsUpdating(false);
        // Sayfayı yenile
        setTimeout(() => {
          window.location.reload();
        }, 1000);
      }
    } catch (err) {
      // Hala bekliyoruz
      setServerCheckCount(prev => prev + 1);
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

  // Sunucu bekleme modu
  useEffect(() => {
    let interval: ReturnType<typeof setInterval>;
    if (waitingForServer) {
      interval = setInterval(checkServerHealth, 2000);
    }
    return () => clearInterval(interval);
  }, [waitingForServer]);

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

  // Sunucu bekleme ekranı
  if (waitingForServer) {
    return (
      <Card>
        <CardContent className="p-8">
          <div className="flex flex-col items-center justify-center space-y-6 text-center">
            <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <Server className="w-8 h-8 text-primary" />
            </div>
            <div className="space-y-2">
              <h3 className="text-lg font-semibold">Sistem Güncelleniyor</h3>
              <p className="text-sm text-muted-foreground">
                Sunucu yeniden başlatılıyor, lütfen bekleyin...
              </p>
              <p className="text-xs text-muted-foreground">
                Bu işlem 30-60 saniye sürebilir
              </p>
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="w-4 h-4 animate-spin" />
              <span>Bağlantı bekleniyor... ({serverCheckCount} deneme)</span>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      {/* Onay Modalı */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border rounded-lg shadow-lg p-6 max-w-md mx-4 space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-yellow-500/10 flex items-center justify-center">
                <AlertTriangle className="w-5 h-5 text-yellow-500" />
              </div>
              <div>
                <h3 className="font-semibold">Güncelleme Onayı</h3>
                <p className="text-sm text-muted-foreground">
                  Sistemi güncellemek istediğinize emin misiniz?
                </p>
              </div>
            </div>
            
            <div className="p-3 rounded-md bg-muted/50 text-sm space-y-1">
              <p><strong>Dikkat:</strong></p>
              <ul className="list-disc list-inside text-muted-foreground text-xs space-y-1">
                <li>Güncelleme sırasında panel geçici olarak erişilemez olacak</li>
                <li>İşlem 30-60 saniye sürebilir</li>
                <li>Sayfa otomatik olarak yenilenecek</li>
              </ul>
            </div>

            <div className="flex gap-3">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => setShowConfirm(false)}
              >
                İptal
              </Button>
              <Button
                className="flex-1"
                onClick={startUpdate}
              >
                <Download className="w-4 h-4 mr-2" />
                Güncelle
              </Button>
            </div>
          </div>
        </div>
      )}

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
              disabled={checking || isUpdating}
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

          {updateInfo && !isUpdating && (
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
                    onClick={() => setShowConfirm(true)}
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

          {isUpdating && !waitingForServer && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-primary">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="text-sm font-medium">Güncelleme başlatılıyor...</span>
              </div>
            </div>
          )}

          {updateInfo && !isUpdating && (
            <p className="text-xs text-muted-foreground">
              Son kontrol: {formatDate(updateInfo.last_checked)}
            </p>
          )}
        </CardContent>
      </Card>
    </>
  );
}
