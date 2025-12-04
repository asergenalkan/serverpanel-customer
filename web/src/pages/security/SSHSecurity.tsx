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
} from 'lucide-react';

interface SSHConfig {
  port: number;
  permit_root_login: string;
  password_authentication: string;
  pubkey_authentication: string;
  max_auth_tries: number;
  login_grace_time: number;
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

  useEffect(() => {
    fetchConfig();
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

  const saveConfig = async () => {
    setSaving(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.put('/security/ssh/config', config);
      if (response.data.success) {
        setSuccess('SSH yapılandırması güncellendi. SSH servisi yeniden başlatıldı.');
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
            <Button onClick={saveConfig} disabled={saving}>
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
      </div>
    </Layout>
  );
}
