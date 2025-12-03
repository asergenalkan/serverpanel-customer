import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import Layout from '@/components/Layout';
import { Settings, Save, RefreshCw, Check, Info } from 'lucide-react';
import api from '@/lib/api';

interface ServerSettingsData {
  multiphp_enabled: boolean;
  default_php_version: string;
  allowed_php_versions: string[];
  domain_based_php: boolean;
}

interface PHPVersion {
  version: string;
  installed: boolean;
  active: boolean;
}

export default function ServerSettings() {
  const [settings, setSettings] = useState<ServerSettingsData>({
    multiphp_enabled: true,
    default_php_version: '8.1',
    allowed_php_versions: ['7.4', '8.0', '8.1', '8.2', '8.3'],
    domain_based_php: true,
  });
  const [installedPHP, setInstalledPHP] = useState<PHPVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [settingsRes, softwareRes] = await Promise.all([
        api.get('/settings/server'),
        api.get('/software/overview'),
      ]);

      if (settingsRes.data.success) {
        setSettings(settingsRes.data.data);
      }

      if (softwareRes.data.success) {
        const phpVersions = softwareRes.data.data.php_versions.map((v: any) => ({
          version: v.name.replace('php', ''),
          installed: v.installed,
          active: v.active,
        }));
        setInstalledPHP(phpVersions);
      }
    } catch (err) {
      console.error('Failed to fetch settings:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setMessage('');
    try {
      const response = await api.put('/settings/server', settings);
      if (response.data.success) {
        setMessage('Ayarlar başarıyla kaydedildi');
        setTimeout(() => setMessage(''), 3000);
      }
    } catch (err: any) {
      setMessage(err.response?.data?.error || 'Ayarlar kaydedilemedi');
    } finally {
      setSaving(false);
    }
  };

  const togglePHPVersion = (version: string) => {
    const newAllowed = settings.allowed_php_versions.includes(version)
      ? settings.allowed_php_versions.filter(v => v !== version)
      : [...settings.allowed_php_versions, version];
    
    // Ensure at least one version is allowed
    if (newAllowed.length === 0) return;

    // If removing default version, set new default
    let newDefault = settings.default_php_version;
    if (!newAllowed.includes(newDefault)) {
      newDefault = newAllowed[0];
    }

    setSettings({
      ...settings,
      allowed_php_versions: newAllowed,
      default_php_version: newDefault,
    });
  };

  if (loading) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-64 gap-3 text-muted-foreground">
          <div className="h-10 w-10 rounded-full border-2 border-muted border-t-2 border-t-primary animate-spin"></div>
          <p className="text-sm">Yükleniyor...</p>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Sunucu Ayarları</h1>
            <p className="text-muted-foreground">MultiPHP ve sunucu yapılandırması</p>
          </div>
          <div className="flex items-center gap-2">
            <Button onClick={fetchData} variant="outline" size="sm">
              <RefreshCw className="w-4 h-4 mr-2" />
              Yenile
            </Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving ? (
                <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Save className="w-4 h-4 mr-2" />
              )}
              Kaydet
            </Button>
          </div>
        </div>

        {message && (
          <div className={`p-4 rounded-lg ${message.includes('başarı') ? 'bg-green-500/10 text-green-500' : 'bg-destructive/10 text-destructive'}`}>
            {message}
          </div>
        )}

        {/* MultiPHP Settings */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="w-5 h-5" />
              MultiPHP Ayarları
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Enable/Disable MultiPHP */}
            <div className="flex items-center justify-between p-4 border rounded-lg">
              <div>
                <p className="font-medium">MultiPHP Aktif</p>
                <p className="text-sm text-muted-foreground">
                  Kullanıcıların farklı PHP sürümleri seçmesine izin ver
                </p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.multiphp_enabled}
                  onChange={(e) => setSettings({ ...settings, multiphp_enabled: e.target.checked })}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
              </label>
            </div>

            {/* Domain-based PHP */}
            <div className="flex items-center justify-between p-4 border rounded-lg">
              <div>
                <p className="font-medium">Domain Bazlı PHP</p>
                <p className="text-sm text-muted-foreground">
                  Her domain için ayrı PHP sürümü seçilebilsin
                </p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.domain_based_php}
                  onChange={(e) => setSettings({ ...settings, domain_based_php: e.target.checked })}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
              </label>
            </div>

            {/* Default PHP Version */}
            <div className="p-4 border rounded-lg">
              <p className="font-medium mb-2">Varsayılan PHP Sürümü</p>
              <p className="text-sm text-muted-foreground mb-3">
                Yeni hesaplar için varsayılan PHP sürümü
              </p>
              <select
                value={settings.default_php_version}
                onChange={(e) => setSettings({ ...settings, default_php_version: e.target.value })}
                className="w-full md:w-48 p-2 border rounded-lg bg-background"
              >
                {settings.allowed_php_versions.map(v => (
                  <option key={v} value={v}>PHP {v}</option>
                ))}
              </select>
            </div>

            {/* Allowed PHP Versions */}
            <div className="p-4 border rounded-lg">
              <p className="font-medium mb-2">İzin Verilen PHP Sürümleri</p>
              <p className="text-sm text-muted-foreground mb-3">
                Kullanıcıların seçebileceği PHP sürümleri
              </p>
              <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
                {installedPHP.map(php => (
                  <button
                    key={php.version}
                    onClick={() => php.installed && togglePHPVersion(php.version)}
                    disabled={!php.installed}
                    className={`p-3 border rounded-lg text-center transition-colors ${
                      !php.installed
                        ? 'opacity-50 cursor-not-allowed bg-muted'
                        : settings.allowed_php_versions.includes(php.version)
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'hover:border-primary/50'
                    }`}
                  >
                    <p className="font-medium">PHP {php.version}</p>
                    <p className="text-xs text-muted-foreground">
                      {!php.installed ? 'Kurulu Değil' : php.active ? 'Aktif' : 'Pasif'}
                    </p>
                    {settings.allowed_php_versions.includes(php.version) && php.installed && (
                      <Check className="w-4 h-4 mx-auto mt-1 text-primary" />
                    )}
                  </button>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Info Card */}
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-start gap-3">
              <Info className="w-5 h-5 text-blue-500 mt-0.5" />
              <div className="text-sm text-muted-foreground">
                <p className="font-medium text-foreground mb-1">Nasıl Çalışır?</p>
                <ul className="list-disc list-inside space-y-1">
                  <li><strong>MultiPHP Aktif:</strong> Kullanıcılar PHP Ayarları sayfasından PHP sürümü seçebilir</li>
                  <li><strong>Domain Bazlı PHP:</strong> Her domain için farklı PHP sürümü kullanılabilir</li>
                  <li><strong>İzin Verilen Sürümler:</strong> Sadece işaretli sürümler kullanıcılara sunulur</li>
                  <li>Kurulu olmayan PHP sürümleri seçilemez. Önce Yazılım Yöneticisi'nden kurun.</li>
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
