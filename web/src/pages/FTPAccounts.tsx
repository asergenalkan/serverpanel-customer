import { useEffect, useState } from 'react';
import { ftpAPI, filesAPI } from '@/lib/api';
import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import Layout from '@/components/Layout';
import {
  Upload,
  Plus,
  Trash2,
  RefreshCw,
  Eye,
  EyeOff,
  Settings,
  Power,
  Copy,
  CheckCircle,
  AlertCircle,
  Server,
  Key,
  Folder,
  HardDrive,
  FolderOpen,
} from 'lucide-react';

interface FTPAccount {
  id: number;
  user_id: number;
  username: string;
  home_directory: string;
  quota_mb: number;
  upload_bandwidth: number;
  download_bandwidth: number;
  active: boolean;
  created_at: string;
  owner_username?: string;
}

interface FTPSettings {
  tls_encryption: string;
  tls_cipher_suite: string;
  allow_anonymous_logins: boolean;
  allow_anonymous_uploads: boolean;
  max_idle_time: number;
  max_connections: number;
  max_connections_per_ip: number;
  allow_root_login: boolean;
  passive_port_min: number;
  passive_port_max: number;
}

export default function FTPAccountsPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  
  const [accounts, setAccounts] = useState<FTPAccount[]>([]);
  const [settings, setSettings] = useState<FTPSettings | null>(null);
  const [serverStatus, setServerStatus] = useState<string>('unknown');
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [showPassword, setShowPassword] = useState<number | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [directories, setDirectories] = useState<string[]>([]);
  const [loadingDirs, setLoadingDirs] = useState(false);
  
  // Form state
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    directory: 'public_html', // Relative path from home
    quota_mb: 0,
  });

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [accountsRes] = await Promise.all([
        ftpAPI.list(),
      ]);

      if (accountsRes.data.success) {
        setAccounts(accountsRes.data.data || []);
      }

      if (isAdmin) {
        const [settingsRes, statusRes] = await Promise.all([
          ftpAPI.getSettings(),
          ftpAPI.getStatus(),
        ]);
        
        if (settingsRes.data.success) {
          setSettings(settingsRes.data.data);
        }
        if (statusRes.data.success) {
          setServerStatus(statusRes.data.data.status);
        }
      }
    } catch (error) {
      console.error('Failed to fetch FTP data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!formData.username || !formData.password) {
      setMessage({ type: 'error', text: 'Kullanıcı adı ve şifre gerekli' });
      return;
    }

    try {
      // directory alanını backend'e gönder (backend tam path'e çevirecek)
      const response = await ftpAPI.create({
        username: formData.username,
        password: formData.password,
        home_directory: formData.directory, // Relative path gönder
        quota_mb: formData.quota_mb,
      });
      if (response.data.success) {
        setMessage({ type: 'success', text: 'FTP hesabı oluşturuldu' });
        setShowCreateModal(false);
        setFormData({ username: '', password: '', directory: 'public_html', quota_mb: 0 });
        fetchData();
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Oluşturma başarısız' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Oluşturma başarısız' });
    }
  };

  const fetchDirectories = async () => {
    setLoadingDirs(true);
    try {
      const response = await filesAPI.list('/');
      if (response.data.success && response.data.data) {
        const dirs = response.data.data
          .filter((item: any) => item.is_dir)
          .map((item: any) => item.name);
        setDirectories(dirs);
      }
    } catch (error) {
      console.error('Failed to fetch directories:', error);
    } finally {
      setLoadingDirs(false);
    }
  };

  const openCreateModal = () => {
    setFormData({ username: '', password: '', directory: 'public_html', quota_mb: 0 });
    fetchDirectories();
    setShowCreateModal(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Bu FTP hesabını silmek istediğinize emin misiniz?')) return;

    try {
      const response = await ftpAPI.delete(id);
      if (response.data.success) {
        setMessage({ type: 'success', text: 'FTP hesabı silindi' });
        fetchData();
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Silme başarısız' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Silme başarısız' });
    }
  };

  const handleToggle = async (id: number) => {
    try {
      const response = await ftpAPI.toggle(id);
      if (response.data.success) {
        setMessage({ type: 'success', text: response.data.message });
        fetchData();
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'İşlem başarısız' });
    }
  };

  const handleSaveSettings = async () => {
    if (!settings) return;

    try {
      const response = await ftpAPI.updateSettings(settings);
      if (response.data.success) {
        setMessage({ type: 'success', text: 'FTP ayarları kaydedildi' });
        setShowSettingsModal(false);
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Kaydetme başarısız' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Kaydetme başarısız' });
    }
  };

  const handleRestartServer = async () => {
    if (!confirm('FTP sunucusunu yeniden başlatmak istediğinize emin misiniz?')) return;

    try {
      const response = await ftpAPI.restart();
      if (response.data.success) {
        setMessage({ type: 'success', text: 'FTP sunucusu yeniden başlatıldı' });
        fetchData();
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Yeniden başlatma başarısız' });
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setMessage({ type: 'success', text: 'Panoya kopyalandı' });
  };

  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';
    for (let i = 0; i < 16; i++) {
      password += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setFormData({ ...formData, password });
  };

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Upload className="w-6 h-6 text-blue-600" />
              FTP Hesapları
            </h1>
            <p className="text-muted-foreground text-sm">
              FTP hesaplarınızı yönetin
            </p>
          </div>
          <div className="flex gap-2">
            {isAdmin && (
              <Button variant="outline" onClick={() => setShowSettingsModal(true)}>
                <Settings className="w-4 h-4 mr-2" />
                Sunucu Ayarları
              </Button>
            )}
            <Button variant="outline" onClick={fetchData} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Yenile
            </Button>
            <Button onClick={openCreateModal}>
              <Plus className="w-4 h-4 mr-2" />
              Yeni FTP Hesabı
            </Button>
          </div>
        </div>

        {/* Message */}
        {message && (
          <div className={`p-4 rounded-lg flex items-center gap-2 ${
            message.type === 'success' 
              ? 'bg-green-50 text-green-800 dark:bg-green-900/20 dark:text-green-400' 
              : 'bg-red-50 text-red-800 dark:bg-red-900/20 dark:text-red-400'
          }`}>
            {message.type === 'success' ? <CheckCircle className="w-5 h-5" /> : <AlertCircle className="w-5 h-5" />}
            {message.text}
            <button onClick={() => setMessage(null)} className="ml-auto">×</button>
          </div>
        )}

        {/* Admin: Server Status */}
        {isAdmin && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="w-5 h-5" />
                FTP Sunucu Durumu
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-3 h-3 rounded-full ${
                    serverStatus === 'active' ? 'bg-green-500' : 
                    serverStatus === 'development' ? 'bg-yellow-500' : 'bg-red-500'
                  }`} />
                  <span className="font-medium">
                    {serverStatus === 'active' ? 'Çalışıyor' : 
                     serverStatus === 'development' ? 'Geliştirme Modu' : 'Durdu'}
                  </span>
                  <span className="text-sm text-muted-foreground">Pure-FTPd</span>
                </div>
                <Button variant="outline" size="sm" onClick={handleRestartServer}>
                  <Power className="w-4 h-4 mr-2" />
                  Yeniden Başlat
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* FTP Accounts List */}
        <Card>
          <CardHeader>
            <CardTitle>FTP Hesapları ({accounts.length})</CardTitle>
          </CardHeader>
          <CardContent>
            {accounts.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <Upload className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>Henüz FTP hesabı yok</p>
                <Button className="mt-4" onClick={openCreateModal}>
                  <Plus className="w-4 h-4 mr-2" />
                  İlk FTP Hesabını Oluştur
                </Button>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-3 px-4">Kullanıcı Adı</th>
                      <th className="text-left py-3 px-4">Dizin</th>
                      <th className="text-left py-3 px-4">Kota</th>
                      {isAdmin && <th className="text-left py-3 px-4">Sahip</th>}
                      <th className="text-left py-3 px-4">Durum</th>
                      <th className="text-right py-3 px-4">İşlemler</th>
                    </tr>
                  </thead>
                  <tbody>
                    {accounts.map((account) => (
                      <tr key={account.id} className="border-b hover:bg-muted/50">
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <Key className="w-4 h-4 text-blue-500" />
                            <span className="font-mono">{account.username}</span>
                            <button onClick={() => copyToClipboard(account.username)} className="text-muted-foreground hover:text-foreground">
                              <Copy className="w-3 h-3" />
                            </button>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <Folder className="w-4 h-4 text-yellow-500" />
                            <span className="font-mono text-sm">{account.home_directory}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <HardDrive className="w-4 h-4 text-gray-500" />
                            {account.quota_mb > 0 ? `${account.quota_mb} MB` : 'Sınırsız'}
                          </div>
                        </td>
                        {isAdmin && (
                          <td className="py-3 px-4 text-sm text-muted-foreground">
                            {account.owner_username}
                          </td>
                        )}
                        <td className="py-3 px-4">
                          <span className={`px-2 py-1 rounded-full text-xs ${
                            account.active 
                              ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' 
                              : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                          }`}>
                            {account.active ? 'Aktif' : 'Pasif'}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleToggle(account.id)}
                              title={account.active ? 'Devre Dışı Bırak' : 'Etkinleştir'}
                            >
                              {account.active ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(account.id)}
                              className="text-red-600 hover:text-red-700"
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Connection Info */}
        <Card>
          <CardHeader>
            <CardTitle>Bağlantı Bilgileri</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="p-4 bg-muted rounded-lg">
                <div className="text-sm text-muted-foreground mb-1">FTP Sunucu</div>
                <div className="font-mono">{window.location.hostname}</div>
              </div>
              <div className="p-4 bg-muted rounded-lg">
                <div className="text-sm text-muted-foreground mb-1">Port</div>
                <div className="font-mono">21</div>
              </div>
              <div className="p-4 bg-muted rounded-lg">
                <div className="text-sm text-muted-foreground mb-1">Protokol</div>
                <div className="font-mono">FTP / FTPS</div>
              </div>
              <div className="p-4 bg-muted rounded-lg">
                <div className="text-sm text-muted-foreground mb-1">Passive Ports</div>
                <div className="font-mono">30000-31000</div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Create Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-background rounded-lg p-6 w-full max-w-md">
              <h2 className="text-xl font-bold mb-4">Yeni FTP Hesabı</h2>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Kullanıcı Adı</label>
                  <input
                    type="text"
                    value={formData.username}
                    onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                    placeholder="ftpuser"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Sistem kullanıcı adınız prefix olarak eklenecek
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Şifre</label>
                  <div className="flex gap-2">
                    <input
                      type={showPassword === -1 ? 'text' : 'password'}
                      value={formData.password}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                      className="flex-1 px-3 py-2 border rounded-lg bg-background"
                      placeholder="••••••••"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setShowPassword(showPassword === -1 ? null : -1)}
                    >
                      {showPassword === -1 ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                    <Button type="button" variant="outline" onClick={generatePassword}>
                      Oluştur
                    </Button>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">
                    <FolderOpen className="w-4 h-4 inline mr-1" />
                    Erişim Dizini
                  </label>
                  <select
                    value={formData.directory}
                    onChange={(e) => setFormData({ ...formData, directory: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                    disabled={loadingDirs}
                  >
                    <option value="public_html">public_html (Web Kök Dizini)</option>
                    {directories.filter(d => d !== 'public_html').map((dir) => (
                      <option key={dir} value={dir}>{dir}</option>
                    ))}
                  </select>
                  <p className="text-xs text-muted-foreground mt-1">
                    FTP kullanıcısı sadece bu dizine erişebilir
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Kota (MB)</label>
                  <input
                    type="number"
                    value={formData.quota_mb}
                    onChange={(e) => setFormData({ ...formData, quota_mb: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                    placeholder="0 = Sınırsız"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowCreateModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleCreate}>
                  Oluştur
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Settings Modal (Admin Only) */}
        {showSettingsModal && settings && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-y-auto">
            <div className="bg-background rounded-lg p-6 w-full max-w-2xl my-8">
              <h2 className="text-xl font-bold mb-4">FTP Sunucu Ayarları</h2>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">TLS Şifreleme</label>
                  <select
                    value={settings.tls_encryption}
                    onChange={(e) => setSettings({ ...settings, tls_encryption: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                  >
                    <option value="disabled">Kapalı</option>
                    <option value="optional">Opsiyonel (Önerilen)</option>
                    <option value="required">Zorunlu</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">TLS Cipher Suite</label>
                  <input
                    type="text"
                    value={settings.tls_cipher_suite}
                    onChange={(e) => setSettings({ ...settings, tls_cipher_suite: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Maks. Bekleme Süresi (dk)</label>
                  <input
                    type="number"
                    value={settings.max_idle_time}
                    onChange={(e) => setSettings({ ...settings, max_idle_time: parseInt(e.target.value) })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Maks. Bağlantı</label>
                  <input
                    type="number"
                    value={settings.max_connections}
                    onChange={(e) => setSettings({ ...settings, max_connections: parseInt(e.target.value) })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">IP Başına Maks. Bağlantı</label>
                  <input
                    type="number"
                    value={settings.max_connections_per_ip}
                    onChange={(e) => setSettings({ ...settings, max_connections_per_ip: parseInt(e.target.value) })}
                    className="w-full px-3 py-2 border rounded-lg bg-background"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Passive Port Aralığı</label>
                  <div className="flex gap-2">
                    <input
                      type="number"
                      value={settings.passive_port_min}
                      onChange={(e) => setSettings({ ...settings, passive_port_min: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                      placeholder="Min"
                    />
                    <input
                      type="number"
                      value={settings.passive_port_max}
                      onChange={(e) => setSettings({ ...settings, passive_port_max: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                      placeholder="Max"
                    />
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="allow_anonymous"
                    checked={settings.allow_anonymous_logins}
                    onChange={(e) => setSettings({ ...settings, allow_anonymous_logins: e.target.checked })}
                    className="rounded"
                  />
                  <label htmlFor="allow_anonymous" className="text-sm">Anonim Girişe İzin Ver</label>
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="allow_root"
                    checked={settings.allow_root_login}
                    onChange={(e) => setSettings({ ...settings, allow_root_login: e.target.checked })}
                    className="rounded"
                  />
                  <label htmlFor="allow_root" className="text-sm">Root Girişine İzin Ver</label>
                </div>
              </div>

              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowSettingsModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleSaveSettings}>
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
