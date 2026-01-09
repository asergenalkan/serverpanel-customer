import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import {
  ShieldCheck,
  RefreshCw,
  Play,
  Square,
  Ban,
  CheckCircle,
  AlertTriangle,
  Plus,
  X,
  Settings,
  Users,
} from 'lucide-react';

interface JailInfo {
  name: string;
  enabled: boolean;
  currently_banned: number;
  total_banned: number;
  banned_ips: string[];
  filter: string;
  max_retry: number;
  ban_time: number;
  find_time: number;
}

interface Fail2banStatus {
  installed: boolean;
  running: boolean;
  version: string;
  jails: JailInfo[];
  total_banned: number;
}

export default function Fail2ban() {
  const navigate = useNavigate();
  const [status, setStatus] = useState<Fail2banStatus | null>(null);
  const [notInstalled, setNotInstalled] = useState(false);
  const [whitelist, setWhitelist] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showBanModal, setShowBanModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [showWhitelistModal, setShowWhitelistModal] = useState(false);
  const [selectedJail, setSelectedJail] = useState<JailInfo | null>(null);
  const [banForm, setBanForm] = useState({ jail: '', ip: '' });
  const [settingsForm, setSettingsForm] = useState({
    jail: '',
    max_retry: 5,
    ban_time: 3600,
    find_time: 600,
  });
  const [newWhitelistIP, setNewWhitelistIP] = useState('');

  useEffect(() => {
    fetchStatus();
    fetchWhitelist();
  }, []);

  const fetchStatus = async () => {
    try {
      const response = await api.get('/security/fail2ban/status');
      if (response.data.success) {
        const data = response.data.data;
        if (!data.version && !data.running && (!data.jails || data.jails.length === 0)) {
          setNotInstalled(true);
        } else {
          setNotInstalled(false);
          setStatus(data);
        }
      }
    } catch (err: any) {
      setNotInstalled(true);
    } finally {
      setLoading(false);
    }
  };

  const fetchWhitelist = async () => {
    try {
      const response = await api.get('/security/fail2ban/whitelist');
      if (response.data.success) {
        setWhitelist(response.data.data || []);
      }
    } catch (err) {
      console.error('Whitelist alınamadı:', err);
    }
  };

  const toggleService = async (action: string) => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.post('/security/fail2ban/toggle', { action });
      if (response.data.success) {
        setSuccess(response.data.message);
        setTimeout(fetchStatus, 1000);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'İşlem başarısız');
    } finally {
      setActionLoading(false);
    }
  };

  const banIP = async () => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.post('/security/fail2ban/ban', banForm);
      if (response.data.success) {
        setSuccess(response.data.message);
        setShowBanModal(false);
        setBanForm({ jail: '', ip: '' });
        fetchStatus();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'IP engellenemedi');
    } finally {
      setActionLoading(false);
    }
  };

  const unbanIP = async (jail: string, ip: string) => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.post('/security/fail2ban/unban', { jail, ip });
      if (response.data.success) {
        setSuccess(response.data.message);
        fetchStatus();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'IP engeli kaldırılamadı');
    } finally {
      setActionLoading(false);
    }
  };

  const updateJailSettings = async () => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.put('/security/fail2ban/jail', settingsForm);
      if (response.data.success) {
        setSuccess(response.data.message);
        setShowSettingsModal(false);
        fetchStatus();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Ayarlar güncellenemedi');
    } finally {
      setActionLoading(false);
    }
  };

  const addToWhitelist = () => {
    if (newWhitelistIP && !whitelist.includes(newWhitelistIP)) {
      setWhitelist([...whitelist, newWhitelistIP]);
      setNewWhitelistIP('');
    }
  };

  const removeFromWhitelist = (ip: string) => {
    setWhitelist(whitelist.filter((i) => i !== ip));
  };

  const saveWhitelist = async () => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.put('/security/fail2ban/whitelist', { ips: whitelist });
      if (response.data.success) {
        setSuccess('Whitelist güncellendi');
        setShowWhitelistModal(false);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Whitelist güncellenemedi');
    } finally {
      setActionLoading(false);
    }
  };

  const formatTime = (seconds: number) => {
    if (seconds >= 86400) return `${Math.floor(seconds / 86400)} gün`;
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)} saat`;
    if (seconds >= 60) return `${Math.floor(seconds / 60)} dakika`;
    return `${seconds} saniye`;
  };

  if (loading) {
    return (
      <Layout>
        <LoadingAnimation />
      </Layout>
    );
  }

  // Not installed state
  if (notInstalled) {
    return (
      <Layout>
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <ShieldCheck className="w-7 h-7" />
              Fail2ban Yönetimi
            </h1>
            <p className="text-muted-foreground">
              Brute-force saldırılarına karşı koruma
            </p>
          </div>

          <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-8 text-center">
            <AlertTriangle className="w-16 h-16 text-yellow-500 mx-auto mb-4" />
            <h2 className="text-xl font-semibold mb-2">Fail2ban Kurulu Değil</h2>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">
              Fail2ban, brute-force saldırılarına karşı sunucunuzu koruyan bir güvenlik aracıdır.
              Yazılım Yöneticisi'nden tek tıkla kurabilirsiniz.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
              <Button onClick={() => navigate('/software')}>
                Yazılım Yöneticisini Aç
              </Button>
              <Button onClick={fetchStatus} variant="outline">
                <RefreshCw className="w-4 h-4 mr-2" />
                Tekrar Kontrol Et
              </Button>
            </div>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <ShieldCheck className="w-7 h-7" />
              Fail2ban Yönetimi
            </h1>
            <p className="text-muted-foreground">
              Brute-force saldırılarına karşı koruma
            </p>
          </div>
          <div className="flex gap-2">
            <Button onClick={fetchStatus} variant="outline" size="sm">
              <RefreshCw className="w-4 h-4 mr-2" />
              Yenile
            </Button>
            {status?.running ? (
              <Button
                onClick={() => toggleService('stop')}
                variant="destructive"
                size="sm"
                disabled={actionLoading}
              >
                <Square className="w-4 h-4 mr-2" />
                Durdur
              </Button>
            ) : (
              <Button
                onClick={() => toggleService('start')}
                variant="default"
                size="sm"
                disabled={actionLoading}
              >
                <Play className="w-4 h-4 mr-2" />
                Başlat
              </Button>
            )}
          </div>
        </div>

        {/* Messages */}
        {error && (
          <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg flex items-center gap-2">
            <AlertTriangle className="w-4 h-4" />
            {error}
          </div>
        )}
        {success && (
          <div className="bg-green-500/10 text-green-600 px-4 py-3 rounded-lg flex items-center gap-2">
            <CheckCircle className="w-4 h-4" />
            {success}
          </div>
        )}

        {/* Status Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Durum</span>
              {status?.running ? (
                <span className="flex items-center gap-1 text-green-600">
                  <CheckCircle className="w-4 h-4" />
                  Çalışıyor
                </span>
              ) : (
                <span className="flex items-center gap-1 text-destructive">
                  <Square className="w-4 h-4" />
                  Durdu
                </span>
              )}
            </div>
          </div>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Versiyon</span>
              <span className="font-mono text-sm">{status?.version || '-'}</span>
            </div>
          </div>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Aktif Jail</span>
              <span className="text-xl font-bold">{status?.jails?.length || 0}</span>
            </div>
          </div>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Engelli IP</span>
              <span className="text-xl font-bold text-destructive">
                {status?.total_banned || 0}
              </span>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-2">
          <Button
            onClick={() => {
              setBanForm({ jail: status?.jails?.[0]?.name || '', ip: '' });
              setShowBanModal(true);
            }}
            variant="outline"
            disabled={!status?.running}
          >
            <Ban className="w-4 h-4 mr-2" />
            IP Engelle
          </Button>
          <Button onClick={() => setShowWhitelistModal(true)} variant="outline">
            <Users className="w-4 h-4 mr-2" />
            Whitelist ({whitelist.length})
          </Button>
        </div>

        {/* Jails */}
        <div className="bg-card rounded-lg border border-border">
          <div className="p-4 border-b border-border">
            <h2 className="text-lg font-semibold">Jail Listesi</h2>
          </div>
          <div className="divide-y divide-border">
            {status?.jails?.map((jail) => (
              <div key={jail.name} className="p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div
                      className={`w-3 h-3 rounded-full ${
                        jail.enabled ? 'bg-green-500' : 'bg-gray-400'
                      }`}
                    />
                    <h3 className="font-medium">{jail.name}</h3>
                    <span className="text-xs bg-muted px-2 py-0.5 rounded">
                      {jail.currently_banned} engelli
                    </span>
                  </div>
                  <Button
                    onClick={() => {
                      setSelectedJail(jail);
                      setSettingsForm({
                        jail: jail.name,
                        max_retry: jail.max_retry,
                        ban_time: jail.ban_time,
                        find_time: jail.find_time,
                      });
                      setShowSettingsModal(true);
                    }}
                    variant="ghost"
                    size="sm"
                  >
                    <Settings className="w-4 h-4" />
                  </Button>
                </div>

                <div className="grid grid-cols-3 gap-4 text-sm mb-3">
                  <div>
                    <span className="text-muted-foreground">Max Deneme:</span>{' '}
                    <span className="font-medium">{jail.max_retry}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Ban Süresi:</span>{' '}
                    <span className="font-medium">{formatTime(jail.ban_time)}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Arama Süresi:</span>{' '}
                    <span className="font-medium">{formatTime(jail.find_time)}</span>
                  </div>
                </div>

                {jail.banned_ips && jail.banned_ips.length > 0 && (
                  <div className="mt-3">
                    <p className="text-sm text-muted-foreground mb-2">Engelli IP'ler:</p>
                    <div className="flex flex-wrap gap-2">
                      {jail.banned_ips.map((ip) => (
                        <span
                          key={ip}
                          className="inline-flex items-center gap-1 bg-destructive/10 text-destructive px-2 py-1 rounded text-sm"
                        >
                          {ip}
                          <button
                            onClick={() => unbanIP(jail.name, ip)}
                            className="hover:bg-destructive/20 rounded p-0.5"
                            disabled={actionLoading}
                          >
                            <X className="w-3 h-3" />
                          </button>
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}

            {(!status?.jails || status.jails.length === 0) && (
              <div className="p-8 text-center text-muted-foreground">
                Aktif jail bulunamadı
              </div>
            )}
          </div>
        </div>

        {/* Ban Modal */}
        {showBanModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <h3 className="text-lg font-semibold mb-4">IP Engelle</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Jail</label>
                  <select
                    value={banForm.jail}
                    onChange={(e) => setBanForm({ ...banForm, jail: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  >
                    {status?.jails?.map((jail) => (
                      <option key={jail.name} value={jail.name}>
                        {jail.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">IP Adresi</label>
                  <input
                    type="text"
                    value={banForm.ip}
                    onChange={(e) => setBanForm({ ...banForm, ip: e.target.value })}
                    placeholder="192.168.1.100"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowBanModal(false)}>
                  İptal
                </Button>
                <Button onClick={banIP} disabled={actionLoading || !banForm.ip}>
                  Engelle
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Settings Modal */}
        {showSettingsModal && selectedJail && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <h3 className="text-lg font-semibold mb-4">
                {selectedJail.name} Ayarları
              </h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">
                    Max Deneme Sayısı
                  </label>
                  <input
                    type="number"
                    value={settingsForm.max_retry}
                    onChange={(e) =>
                      setSettingsForm({
                        ...settingsForm,
                        max_retry: parseInt(e.target.value),
                      })
                    }
                    min={1}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">
                    Ban Süresi (saniye)
                  </label>
                  <input
                    type="number"
                    value={settingsForm.ban_time}
                    onChange={(e) =>
                      setSettingsForm({
                        ...settingsForm,
                        ban_time: parseInt(e.target.value),
                      })
                    }
                    min={60}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatTime(settingsForm.ban_time)}
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">
                    Arama Süresi (saniye)
                  </label>
                  <input
                    type="number"
                    value={settingsForm.find_time}
                    onChange={(e) =>
                      setSettingsForm({
                        ...settingsForm,
                        find_time: parseInt(e.target.value),
                      })
                    }
                    min={60}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatTime(settingsForm.find_time)}
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowSettingsModal(false)}>
                  İptal
                </Button>
                <Button onClick={updateJailSettings} disabled={actionLoading}>
                  Kaydet
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Whitelist Modal */}
        {showWhitelistModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <h3 className="text-lg font-semibold mb-4">Whitelist Yönetimi</h3>
              <p className="text-sm text-muted-foreground mb-4">
                Bu IP adresleri hiçbir zaman engellenmez.
              </p>
              <div className="flex gap-2 mb-4">
                <input
                  type="text"
                  value={newWhitelistIP}
                  onChange={(e) => setNewWhitelistIP(e.target.value)}
                  placeholder="IP adresi veya CIDR (örn: 192.168.1.0/24)"
                  className="flex-1 px-3 py-2 rounded-lg border border-border bg-background"
                  onKeyPress={(e) => e.key === 'Enter' && addToWhitelist()}
                />
                <Button onClick={addToWhitelist} size="sm">
                  <Plus className="w-4 h-4" />
                </Button>
              </div>
              <div className="space-y-2 max-h-60 overflow-y-auto">
                {whitelist.map((ip) => (
                  <div
                    key={ip}
                    className="flex items-center justify-between p-2 bg-muted/50 rounded"
                  >
                    <span className="font-mono text-sm">{ip}</span>
                    <button
                      onClick={() => removeFromWhitelist(ip)}
                      className="text-destructive hover:bg-destructive/10 rounded p-1"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
                {whitelist.length === 0 && (
                  <p className="text-center text-muted-foreground py-4">
                    Whitelist boş
                  </p>
                )}
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowWhitelistModal(false)}>
                  İptal
                </Button>
                <Button onClick={saveWhitelist} disabled={actionLoading}>
                  Kaydet
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
