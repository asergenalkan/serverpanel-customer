import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import {
  Key,
  RefreshCw,
  Save,
  CheckCircle,
  AlertTriangle,
  Info,
  Shield,
  Plus,
  Trash2,
  Download,
  Copy,
  Upload,
  KeyRound,
  X,
} from 'lucide-react';

interface SSHConfig {
  port: number;
  permit_root_login: string;
  password_authentication: string;
  pubkey_authentication: string;
  max_auth_tries: number;
  login_grace_time: number;
}

interface SSHKey {
  id: string;
  name: string;
  fingerprint: string;
  type: string;
  public_key: string;
}

interface GeneratedKey {
  name: string;
  private_key: string;
  public_key: string;
}

export default function SSHSecurity() {
  const [config, setConfig] = useState<SSHConfig>({
    port: 22,
    permit_root_login: 'yes',
    password_authentication: 'yes',
    pubkey_authentication: 'yes',
    max_auth_tries: 6,
    login_grace_time: 120,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [configError, setConfigError] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  
  // SSH Keys state
  const [sshKeys, setSSHKeys] = useState<SSHKey[]>([]);
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [showAddKeyModal, setShowAddKeyModal] = useState(false);
  const [showPrivateKeyModal, setShowPrivateKeyModal] = useState(false);
  const [generatedKey, setGeneratedKey] = useState<GeneratedKey | null>(null);
  const [newKeyName, setNewKeyName] = useState('');
  const [newPublicKey, setNewPublicKey] = useState('');
  const [keyLoading, setKeyLoading] = useState(false);
  const [showPasswordWarning, setShowPasswordWarning] = useState(false);
  const [showRootWarning, setShowRootWarning] = useState(false);
  const [pendingConfig, setPendingConfig] = useState<SSHConfig | null>(null);

  useEffect(() => {
    fetchConfig();
    fetchSSHKeys();
  }, []);

  const fetchConfig = async () => {
    try {
      const response = await api.get('/security/ssh/config');
      if (response.data.success) {
        setConfigError(false);
        setConfig(response.data.data);
      }
    } catch (err: any) {
      setConfigError(true);
      setError(err.response?.data?.error || 'SSH yapılandırması alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const fetchSSHKeys = async () => {
    try {
      const response = await api.get('/security/ssh/keys');
      if (response.data.success) {
        setSSHKeys(response.data.data || []);
      }
    } catch (err) {
      console.error('SSH keys alınamadı:', err);
    }
  };

  const generateKey = async () => {
    setKeyLoading(true);
    setError('');
    try {
      const response = await api.post('/security/ssh/keys/generate', {
        name: newKeyName || 'ServerPanel Key',
      });
      if (response.data.success) {
        setGeneratedKey(response.data.data);
        setShowGenerateModal(false);
        setShowPrivateKeyModal(true);
        setNewKeyName('');
        fetchSSHKeys();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Key oluşturulamadı');
    } finally {
      setKeyLoading(false);
    }
  };

  const addExistingKey = async () => {
    setKeyLoading(true);
    setError('');
    try {
      const response = await api.post('/security/ssh/keys', {
        name: newKeyName,
        public_key: newPublicKey,
      });
      if (response.data.success) {
        setSuccess('SSH key eklendi');
        setShowAddKeyModal(false);
        setNewKeyName('');
        setNewPublicKey('');
        fetchSSHKeys();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Key eklenemedi');
    } finally {
      setKeyLoading(false);
    }
  };

  const deleteKey = async (id: string) => {
    if (!confirm('Bu SSH key\'i silmek istediğinizden emin misiniz?')) return;
    
    try {
      const response = await api.delete(`/security/ssh/keys/${id}`);
      if (response.data.success) {
        setSuccess('SSH key silindi');
        fetchSSHKeys();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Key silinemedi');
    }
  };

  const downloadPrivateKey = () => {
    if (!generatedKey) return;
    const blob = new Blob([generatedKey.private_key], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${generatedKey.name.replace(/\s+/g, '_')}_id_ed25519`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setSuccess('Panoya kopyalandı');
    setTimeout(() => setSuccess(''), 2000);
  };

  const saveConfig = async (forceConfig?: SSHConfig) => {
    const configToSave = forceConfig || config;
    
    // Şifre girişi kapatılıyor ve SSH key yok mu?
    if (configToSave.password_authentication === 'no' && sshKeys.length === 0) {
      setPendingConfig(configToSave);
      setShowPasswordWarning(true);
      return;
    }
    
    // Root girişi tamamen kapatılıyor mu?
    if (configToSave.permit_root_login === 'no') {
      setPendingConfig(configToSave);
      setShowRootWarning(true);
      return;
    }
    
    await doSaveConfig(configToSave);
  };

  const doSaveConfig = async (configToSave: SSHConfig) => {
    setSaving(true);
    setError('');
    setSuccess('');
    setShowPasswordWarning(false);
    setShowRootWarning(false);
    setPendingConfig(null);
    try {
      const response = await api.put('/security/ssh/config', configToSave);
      if (response.data.success) {
        setSuccess('SSH yapılandırması güncellendi. SSH servisi yeniden başlatıldı.');
        setConfig(configToSave);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Yapılandırma kaydedilemedi');
    } finally {
      setSaving(false);
    }
  };

  const getSecurityScore = () => {
    let score = 0;
    if (config.port !== 22) score += 20;
    if (config.permit_root_login === 'no') score += 30;
    if (config.permit_root_login === 'prohibit-password') score += 20;
    if (config.password_authentication === 'no') score += 20;
    if (config.pubkey_authentication === 'yes') score += 10;
    if (config.max_auth_tries <= 3) score += 10;
    if (config.login_grace_time <= 60) score += 10;
    return Math.min(score, 100);
  };

  const getScoreColor = (score: number) => {
    if (score >= 80) return 'text-green-600';
    if (score >= 50) return 'text-yellow-600';
    return 'text-red-600';
  };

  const getScoreLabel = (score: number) => {
    if (score >= 80) return 'Güçlü';
    if (score >= 50) return 'Orta';
    return 'Zayıf';
  };

  if (loading) {
    return (
      <Layout>
        <LoadingAnimation />
      </Layout>
    );
  }

  // Config error state
  if (configError) {
    return (
      <Layout>
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Key className="w-7 h-7" />
              SSH Güvenliği
            </h1>
            <p className="text-muted-foreground">
              SSH sunucu yapılandırması
            </p>
          </div>

          <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-8 text-center">
            <AlertTriangle className="w-16 h-16 text-destructive mx-auto mb-4" />
            <h2 className="text-xl font-semibold mb-2">SSH Yapılandırması Okunamadı</h2>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">
              SSH yapılandırma dosyası (/etc/ssh/sshd_config) okunamadı. 
              Lütfen SSH servisinin kurulu olduğundan emin olun.
            </p>
            <div className="bg-card border border-border rounded-lg p-4 max-w-lg mx-auto">
              <p className="text-sm font-medium mb-2">Kontrol Komutu:</p>
              <code className="block bg-muted p-3 rounded text-sm font-mono text-left">
                systemctl status sshd
              </code>
            </div>
            <Button onClick={fetchConfig} variant="outline" className="mt-4">
              <RefreshCw className="w-4 h-4 mr-2" />
              Tekrar Kontrol Et
            </Button>
          </div>
        </div>
      </Layout>
    );
  }

  const securityScore = getSecurityScore();

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Key className="w-7 h-7" />
              SSH Güvenliği
            </h1>
            <p className="text-muted-foreground">
              SSH sunucu yapılandırması
            </p>
          </div>
          <div className="flex gap-2">
            <Button onClick={fetchConfig} variant="outline" size="sm">
              <RefreshCw className="w-4 h-4 mr-2" />
              Yenile
            </Button>
            <Button onClick={() => saveConfig()} disabled={saving}>
              <Save className="w-4 h-4 mr-2" />
              {saving ? 'Kaydediliyor...' : 'Kaydet'}
            </Button>
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

        {/* Security Score */}
        <div className="bg-card rounded-lg border border-border p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Shield className={`w-12 h-12 ${getScoreColor(securityScore)}`} />
              <div>
                <h2 className="text-lg font-semibold">Güvenlik Puanı</h2>
                <p className="text-sm text-muted-foreground">
                  SSH yapılandırmanızın güvenlik değerlendirmesi
                </p>
              </div>
            </div>
            <div className="text-right">
              <span className={`text-4xl font-bold ${getScoreColor(securityScore)}`}>
                {securityScore}
              </span>
              <span className="text-2xl text-muted-foreground">/100</span>
              <p className={`text-sm ${getScoreColor(securityScore)}`}>
                {getScoreLabel(securityScore)}
              </p>
            </div>
          </div>
          <div className="mt-4 h-2 bg-muted rounded-full overflow-hidden">
            <div
              className={`h-full transition-all ${
                securityScore >= 80
                  ? 'bg-green-500'
                  : securityScore >= 50
                  ? 'bg-yellow-500'
                  : 'bg-red-500'
              }`}
              style={{ width: `${securityScore}%` }}
            />
          </div>
        </div>

        {/* Configuration */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Basic Settings */}
          <div className="bg-card rounded-lg border border-border p-6">
            <h2 className="text-lg font-semibold mb-4">Temel Ayarlar</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">SSH Port</label>
                <input
                  type="number"
                  value={config.port}
                  onChange={(e) =>
                    setConfig({ ...config, port: parseInt(e.target.value) })
                  }
                  min={1}
                  max={65535}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                />
                <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  Varsayılan: 22. Farklı port kullanmak güvenliği artırır.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">Root Girişi</label>
                <select
                  value={config.permit_root_login}
                  onChange={(e) =>
                    setConfig({ ...config, permit_root_login: e.target.value })
                  }
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                >
                  <option value="yes">İzin Ver</option>
                  <option value="prohibit-password">Sadece Key ile</option>
                  <option value="no">Yasakla</option>
                </select>
                <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  Root girişini yasaklamak en güvenli seçenektir.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  Max Deneme Sayısı
                </label>
                <input
                  type="number"
                  value={config.max_auth_tries}
                  onChange={(e) =>
                    setConfig({ ...config, max_auth_tries: parseInt(e.target.value) })
                  }
                  min={1}
                  max={10}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                />
                <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  Bağlantı başına maksimum kimlik doğrulama denemesi.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  Giriş Süresi (saniye)
                </label>
                <input
                  type="number"
                  value={config.login_grace_time}
                  onChange={(e) =>
                    setConfig({ ...config, login_grace_time: parseInt(e.target.value) })
                  }
                  min={10}
                  max={600}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                />
                <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  Kimlik doğrulama için verilen süre.
                </p>
              </div>
            </div>
          </div>

          {/* Authentication Settings */}
          <div className="bg-card rounded-lg border border-border p-6">
            <h2 className="text-lg font-semibold mb-4">Kimlik Doğrulama</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  Şifre ile Giriş
                </label>
                <select
                  value={config.password_authentication}
                  onChange={(e) =>
                    setConfig({ ...config, password_authentication: e.target.value })
                  }
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                >
                  <option value="yes">Etkin</option>
                  <option value="no">Devre Dışı</option>
                </select>
                <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  Devre dışı bırakmak brute-force saldırılarını önler.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  SSH Key ile Giriş
                </label>
                <select
                  value={config.pubkey_authentication}
                  onChange={(e) =>
                    setConfig({ ...config, pubkey_authentication: e.target.value })
                  }
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                >
                  <option value="yes">Etkin</option>
                  <option value="no">Devre Dışı</option>
                </select>
                <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  SSH key kullanımı en güvenli yöntemdir.
                </p>
              </div>
            </div>

            {/* Security Tips */}
            <div className="mt-6 p-4 bg-blue-500/10 rounded-lg">
              <h3 className="font-medium text-blue-600 mb-2">Güvenlik Önerileri</h3>
              <ul className="text-sm text-blue-500 space-y-1">
                {config.port === 22 && (
                  <li>• SSH portunu 22'den farklı bir değere değiştirin</li>
                )}
                {config.permit_root_login === 'yes' && (
                  <li>• Root girişini yasaklayın veya sadece key ile sınırlayın</li>
                )}
                {config.password_authentication === 'yes' && (
                  <li>• Şifre ile girişi devre dışı bırakıp SSH key kullanın</li>
                )}
                {config.max_auth_tries > 3 && (
                  <li>• Max deneme sayısını 3 veya daha aza düşürün</li>
                )}
                {config.login_grace_time > 60 && (
                  <li>• Giriş süresini 60 saniye veya daha aza düşürün</li>
                )}
                {securityScore >= 80 && (
                  <li className="text-green-600">✓ SSH yapılandırmanız güvenli!</li>
                )}
              </ul>
            </div>
          </div>
        </div>

        {/* SSH Keys Section */}
        <div className="bg-card rounded-lg border border-border">
          <div className="p-4 border-b border-border flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold flex items-center gap-2">
                <KeyRound className="w-5 h-5" />
                SSH Key Yönetimi
              </h2>
              <p className="text-sm text-muted-foreground">
                Yetkili SSH key'leri yönetin
              </p>
            </div>
            <div className="flex gap-2">
              <Button onClick={() => setShowAddKeyModal(true)} variant="outline" size="sm">
                <Upload className="w-4 h-4 mr-2" />
                Key Ekle
              </Button>
              <Button onClick={() => setShowGenerateModal(true)} size="sm">
                <Plus className="w-4 h-4 mr-2" />
                Yeni Key Oluştur
              </Button>
            </div>
          </div>
          <div className="divide-y divide-border">
            {sshKeys.map((key) => (
              <div key={key.id} className="p-4 flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Key className="w-4 h-4 text-muted-foreground" />
                    <span className="font-medium">{key.name}</span>
                    <span className="text-xs bg-muted px-2 py-0.5 rounded">{key.type}</span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1 font-mono truncate">
                    {key.fingerprint}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    onClick={() => copyToClipboard(key.public_key)}
                    variant="ghost"
                    size="sm"
                    title="Public key'i kopyala"
                  >
                    <Copy className="w-4 h-4" />
                  </Button>
                  <Button
                    onClick={() => deleteKey(key.id)}
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    title="Sil"
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
            {sshKeys.length === 0 && (
              <div className="p-8 text-center text-muted-foreground">
                <KeyRound className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>Henüz SSH key eklenmemiş</p>
                <p className="text-sm mt-1">
                  Güvenli giriş için bir SSH key oluşturun veya mevcut key'inizi ekleyin.
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Warning */}
        <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-yellow-500 mt-0.5" />
            <div>
              <h3 className="font-medium text-yellow-600">Dikkat</h3>
              <p className="text-sm text-muted-foreground mt-1">
                SSH yapılandırmasını değiştirmeden önce sunucuya alternatif bir erişim
                yönteminiz olduğundan emin olun. Yanlış yapılandırma sunucuya erişiminizi
                engelleyebilir. Şifre ile girişi devre dışı bırakmadan önce SSH key
                kurulumunu tamamlayın.
              </p>
            </div>
          </div>
        </div>

        {/* Generate Key Modal */}
        {showGenerateModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <h3 className="text-lg font-semibold mb-4">Yeni SSH Key Oluştur</h3>
              <p className="text-sm text-muted-foreground mb-4">
                ED25519 algoritması ile güvenli bir key çifti oluşturulacak.
              </p>
              <div>
                <label className="block text-sm font-medium mb-1">Key Adı</label>
                <input
                  type="text"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  placeholder="Örn: Laptop, Ofis PC"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                />
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowGenerateModal(false)}>
                  İptal
                </Button>
                <Button onClick={generateKey} disabled={keyLoading}>
                  {keyLoading ? 'Oluşturuluyor...' : 'Oluştur'}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Add Existing Key Modal */}
        {showAddKeyModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-lg">
              <h3 className="text-lg font-semibold mb-4">Mevcut SSH Key Ekle</h3>
              <p className="text-sm text-muted-foreground mb-4">
                Bilgisayarınızdaki public key'i yapıştırın.
              </p>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Key Adı</label>
                  <input
                    type="text"
                    value={newKeyName}
                    onChange={(e) => setNewKeyName(e.target.value)}
                    placeholder="Örn: MacBook Pro"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Public Key</label>
                  <textarea
                    value={newPublicKey}
                    onChange={(e) => setNewPublicKey(e.target.value)}
                    placeholder="ssh-ed25519 AAAA... veya ssh-rsa AAAA..."
                    rows={4}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Genellikle ~/.ssh/id_ed25519.pub veya ~/.ssh/id_rsa.pub dosyasında bulunur.
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => {
                  setShowAddKeyModal(false);
                  setNewKeyName('');
                  setNewPublicKey('');
                }}>
                  İptal
                </Button>
                <Button onClick={addExistingKey} disabled={keyLoading || !newPublicKey}>
                  {keyLoading ? 'Ekleniyor...' : 'Ekle'}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Private Key Download Modal */}
        {showPrivateKeyModal && generatedKey && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-2xl">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-12 h-12 rounded-full bg-green-500/10 flex items-center justify-center">
                  <CheckCircle className="w-6 h-6 text-green-500" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold">SSH Key Oluşturuldu!</h3>
                  <p className="text-sm text-muted-foreground">{generatedKey.name}</p>
                </div>
              </div>

              <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-4 mb-4">
                <div className="flex items-start gap-2">
                  <AlertTriangle className="w-5 h-5 text-destructive mt-0.5" />
                  <div>
                    <p className="font-medium text-destructive">ÖNEMLİ: Private Key'i Şimdi İndirin!</p>
                    <p className="text-sm text-muted-foreground mt-1">
                      Private key sunucuda SAKLANMIYOR. Bu pencereyi kapattıktan sonra
                      private key'e bir daha erişemezsiniz. Hemen indirin ve güvenli bir yerde saklayın.
                    </p>
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-sm font-medium">Private Key (GIZLI TUTUN!)</label>
                    <div className="flex gap-1">
                      <Button
                        onClick={() => copyToClipboard(generatedKey.private_key)}
                        variant="ghost"
                        size="sm"
                      >
                        <Copy className="w-4 h-4 mr-1" />
                        Kopyala
                      </Button>
                      <Button onClick={downloadPrivateKey} variant="ghost" size="sm">
                        <Download className="w-4 h-4 mr-1" />
                        İndir
                      </Button>
                    </div>
                  </div>
                  <pre className="bg-muted p-3 rounded-lg text-xs font-mono overflow-x-auto max-h-40">
                    {generatedKey.private_key}
                  </pre>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-sm font-medium">Public Key (sunucuya eklendi)</label>
                    <Button
                      onClick={() => copyToClipboard(generatedKey.public_key)}
                      variant="ghost"
                      size="sm"
                    >
                      <Copy className="w-4 h-4 mr-1" />
                      Kopyala
                    </Button>
                  </div>
                  <pre className="bg-muted p-3 rounded-lg text-xs font-mono overflow-x-auto">
                    {generatedKey.public_key}
                  </pre>
                </div>
              </div>

              <div className="bg-blue-500/10 rounded-lg p-4 mt-4">
                <h4 className="font-medium text-blue-600 mb-2">Kullanım</h4>
                <p className="text-sm text-muted-foreground mb-2">
                  İndirdiğiniz private key dosyasını bilgisayarınıza kaydedin ve SSH ile bağlanın:
                </p>
                <code className="block bg-muted p-2 rounded text-sm font-mono">
                  chmod 600 ~/{generatedKey.name.replace(/\s+/g, '_')}_id_ed25519<br />
                  ssh -i ~/{generatedKey.name.replace(/\s+/g, '_')}_id_ed25519 root@sunucu_ip
                </code>
              </div>

              <div className="flex justify-end mt-6">
                <Button onClick={() => {
                  setShowPrivateKeyModal(false);
                  setGeneratedKey(null);
                }}>
                  <X className="w-4 h-4 mr-2" />
                  Kapat (Key'i indirdiğimden eminim)
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Password Authentication Warning Modal */}
        {showPasswordWarning && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center">
                  <AlertTriangle className="w-6 h-6 text-destructive" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-destructive">Tehlikeli İşlem!</h3>
                  <p className="text-sm text-muted-foreground">Şifre ile giriş kapatılıyor</p>
                </div>
              </div>

              <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-4 mb-4">
                <p className="text-sm">
                  <strong>Hiç SSH key'iniz yok!</strong> Şifre ile girişi kapattığınızda sunucuya
                  <strong> erişiminizi tamamen kaybedebilirsiniz.</strong>
                </p>
              </div>

              <p className="text-sm text-muted-foreground mb-4">
                Önce bir SSH key oluşturun, private key'i indirin ve bağlantıyı test edin.
                Sonra şifre ile girişi kapatabilirsiniz.
              </p>

              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => {
                  setShowPasswordWarning(false);
                  setPendingConfig(null);
                }}>
                  İptal
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => pendingConfig && doSaveConfig(pendingConfig)}
                >
                  Yine de Kapat (Riskli!)
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Root Login Warning Modal */}
        {showRootWarning && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-12 h-12 rounded-full bg-yellow-500/10 flex items-center justify-center">
                  <AlertTriangle className="w-6 h-6 text-yellow-500" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-yellow-600">Dikkat!</h3>
                  <p className="text-sm text-muted-foreground">Root girişi kapatılıyor</p>
                </div>
              </div>

              <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4 mb-4">
                <p className="text-sm">
                  Root girişini tamamen kapattığınızda, sunucuya <strong>sudo yetkili başka bir kullanıcı</strong> ile
                  giriş yapmanız gerekecek.
                </p>
              </div>

              <p className="text-sm text-muted-foreground mb-4">
                Sudo yetkili bir kullanıcınız olduğundan ve o kullanıcı ile bağlanabildiğinizden emin misiniz?
              </p>

              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => {
                  setShowRootWarning(false);
                  setPendingConfig(null);
                }}>
                  İptal
                </Button>
                <Button
                  onClick={() => pendingConfig && doSaveConfig(pendingConfig)}
                >
                  Evet, Devam Et
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
