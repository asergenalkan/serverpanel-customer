import { useEffect, useState } from 'react';
import { phpAPI, domainsAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import Layout from '@/components/Layout';
import {
  Code,
  Settings,
  Globe,
  RefreshCw,
  Save,
  AlertCircle,
  CheckCircle,
  Info,
} from 'lucide-react';

interface PHPVersion {
  version: string;
  path: string;
  is_default: boolean;
  is_active: boolean;
}

interface Domain {
  id: number;
  name: string;
  php_version?: string;
}

interface PHPSettings {
  domain_id: number;
  domain: string;
  php_version: string;
  memory_limit: string;
  max_execution_time: number;
  max_input_time: number;
  post_max_size: string;
  upload_max_filesize: string;
  max_file_uploads: number;
  display_errors: boolean;
  error_reporting: string;
}

export default function PHPSettingsPage() {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [phpVersions, setPHPVersions] = useState<PHPVersion[]>([]);
  const [selectedDomain, setSelectedDomain] = useState<number | null>(null);
  const [settings, setSettings] = useState<PHPSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    if (selectedDomain) {
      fetchDomainSettings(selectedDomain);
    }
  }, [selectedDomain]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [domainsRes, versionsRes] = await Promise.all([
        domainsAPI.list(),
        phpAPI.getVersions(),
      ]);

      if (domainsRes.data.success) {
        setDomains(domainsRes.data.data || []);
        if (domainsRes.data.data?.length > 0) {
          setSelectedDomain(domainsRes.data.data[0].id);
        }
      }

      if (versionsRes.data.success) {
        setPHPVersions(versionsRes.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch data:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchDomainSettings = async (domainId: number) => {
    try {
      const response = await phpAPI.getDomainSettings(domainId);
      if (response.data.success) {
        setSettings(response.data.data);
      }
    } catch (error) {
      console.error('Failed to fetch domain settings:', error);
    }
  };

  const handleVersionChange = async (version: string) => {
    if (!selectedDomain) return;

    setSaving(true);
    setMessage(null);

    try {
      const response = await phpAPI.updateVersion(selectedDomain, version);
      if (response.data.success) {
        setMessage({ type: 'success', text: `PHP versiyonu ${version} olarak güncellendi` });
        setSettings(prev => prev ? { ...prev, php_version: version } : null);
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Güncelleme başarısız' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Güncelleme başarısız' });
    } finally {
      setSaving(false);
    }
  };

  const handleSettingsSave = async () => {
    if (!selectedDomain || !settings) return;

    setSaving(true);
    setMessage(null);

    try {
      const response = await phpAPI.updateSettings(selectedDomain, {
        memory_limit: settings.memory_limit,
        max_execution_time: settings.max_execution_time,
        max_input_time: settings.max_input_time,
        post_max_size: settings.post_max_size,
        upload_max_filesize: settings.upload_max_filesize,
        max_file_uploads: settings.max_file_uploads,
        display_errors: settings.display_errors,
        error_reporting: settings.error_reporting,
      });

      if (response.data.success) {
        setMessage({ type: 'success', text: 'PHP ayarları başarıyla güncellendi' });
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Güncelleme başarısız' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Güncelleme başarısız' });
    } finally {
      setSaving(false);
    }
  };

  const memoryOptions = ['64M', '128M', '256M', '512M', '1G', '2G'];
  const uploadSizeOptions = ['8M', '16M', '32M', '64M', '128M', '256M', '512M'];
  const timeOptions = [30, 60, 120, 300, 600, 900, 1800];

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Code className="w-6 h-6 text-purple-600" />
              PHP Ayarları
            </h1>
            <p className="text-muted-foreground text-sm">
              Domain bazlı PHP versiyonu ve INI ayarlarını yönetin
            </p>
          </div>
          <Button variant="outline" onClick={fetchData} disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Yenile
          </Button>
        </div>

        {/* Message */}
        {message && (
          <div className={`p-4 rounded-lg flex items-center gap-2 border text-white ${
            message.type === 'success' 
              ? 'bg-emerald-600 border-emerald-700 dark:bg-emerald-600 dark:border-emerald-700' 
              : 'bg-rose-600 border-rose-700 dark:bg-rose-600 dark:border-rose-700'
          }`}>
            {message.type === 'success' ? (
              <CheckCircle className="w-5 h-5" />
            ) : (
              <AlertCircle className="w-5 h-5" />
            )}
            {message.text}
          </div>
        )}

        {/* Domain Selector */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Globe className="w-5 h-5" />
              Domain Seçin
            </CardTitle>
          </CardHeader>
          <CardContent>
            {domains.length === 0 ? (
              <p className="text-muted-foreground">Henüz domain eklenmemiş.</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {domains.map((domain) => (
                  <Button
                    key={domain.id}
                    variant={selectedDomain === domain.id ? 'default' : 'outline'}
                    onClick={() => setSelectedDomain(domain.id)}
                  >
                    {domain.name}
                  </Button>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {selectedDomain && settings && (
          <>
            {/* PHP Version Selector */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Code className="w-5 h-5" />
                  PHP Versiyonu
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex flex-wrap gap-3">
                    {phpVersions.map((version) => (
                      <button
                        key={version.version}
                        onClick={() => handleVersionChange(version.version)}
                        disabled={saving}
                        className={`px-4 py-3 rounded-lg border-2 transition-all ${
                          settings.php_version === version.version
                            ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                            : 'border-gray-200 dark:border-gray-700 hover:border-purple-300'
                        }`}
                      >
                        <div className="font-semibold">PHP {version.version}</div>
                        <div className="text-xs text-muted-foreground mt-1">
                          {version.is_default && <span className="text-purple-600">Varsayılan</span>}
                          {version.is_active && <span className="text-green-600 ml-2">Aktif</span>}
                        </div>
                      </button>
                    ))}
                  </div>
                  <div className="flex items-start gap-2 text-sm text-muted-foreground bg-primary/10 p-3 rounded-lg">
                    <Info className="w-4 h-4 mt-0.5 text-primary" />
                    <span>
                      PHP versiyonunu değiştirmek Apache ve PHP-FPM yapılandırmasını günceller.
                      Değişiklik birkaç saniye sürebilir.
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* PHP INI Settings */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Settings className="w-5 h-5" />
                  PHP INI Ayarları
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {/* Memory Limit */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      memory_limit
                    </label>
                    <select
                      value={settings.memory_limit}
                      onChange={(e) => setSettings({ ...settings, memory_limit: e.target.value })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    >
                      {memoryOptions.map((opt) => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      PHP için maksimum bellek kullanımı
                    </p>
                  </div>

                  {/* Max Execution Time */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      max_execution_time
                    </label>
                    <select
                      value={settings.max_execution_time}
                      onChange={(e) => setSettings({ ...settings, max_execution_time: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    >
                      {timeOptions.map((opt) => (
                        <option key={opt} value={opt}>{opt} saniye</option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      Script çalışma süresi limiti
                    </p>
                  </div>

                  {/* Max Input Time */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      max_input_time
                    </label>
                    <select
                      value={settings.max_input_time}
                      onChange={(e) => setSettings({ ...settings, max_input_time: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    >
                      {timeOptions.map((opt) => (
                        <option key={opt} value={opt}>{opt} saniye</option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      Input verisi parse süresi limiti
                    </p>
                  </div>

                  {/* Post Max Size */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      post_max_size
                    </label>
                    <select
                      value={settings.post_max_size}
                      onChange={(e) => setSettings({ ...settings, post_max_size: e.target.value })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    >
                      {uploadSizeOptions.map((opt) => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      POST verisi maksimum boyutu
                    </p>
                  </div>

                  {/* Upload Max Filesize */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      upload_max_filesize
                    </label>
                    <select
                      value={settings.upload_max_filesize}
                      onChange={(e) => setSettings({ ...settings, upload_max_filesize: e.target.value })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    >
                      {uploadSizeOptions.map((opt) => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      Dosya yükleme maksimum boyutu
                    </p>
                  </div>

                  {/* Max File Uploads */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      max_file_uploads
                    </label>
                    <input
                      type="number"
                      value={settings.max_file_uploads}
                      onChange={(e) => setSettings({ ...settings, max_file_uploads: parseInt(e.target.value) || 20 })}
                      min={1}
                      max={100}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Aynı anda yüklenebilecek dosya sayısı
                    </p>
                  </div>

                  {/* Display Errors */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      display_errors
                    </label>
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => setSettings({ ...settings, display_errors: false })}
                        className={`px-4 py-2 rounded-lg border ${
                          !settings.display_errors
                            ? 'bg-primary/10 border-primary text-primary'
                            : 'border-border'
                        }`}
                      >
                        Kapalı (Önerilen)
                      </button>
                      <button
                        onClick={() => setSettings({ ...settings, display_errors: true })}
                        className={`px-4 py-2 rounded-lg border ${
                          settings.display_errors
                            ? 'bg-yellow-100 border-yellow-500 text-yellow-700 dark:bg-yellow-900/30'
                            : 'border-gray-300 dark:border-gray-600'
                        }`}
                      >
                        Açık
                      </button>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      Hataları ekranda göster (geliştirme için)
                    </p>
                  </div>

                  {/* Error Reporting */}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      error_reporting
                    </label>
                    <select
                      value={settings.error_reporting}
                      onChange={(e) => setSettings({ ...settings, error_reporting: e.target.value })}
                      className="w-full px-3 py-2 border rounded-lg bg-background"
                    >
                      <option value="E_ALL & ~E_DEPRECATED & ~E_STRICT">Standart (Önerilen)</option>
                      <option value="E_ALL">Tüm Hatalar</option>
                      <option value="E_ALL & ~E_NOTICE">Notice Hariç</option>
                      <option value="0">Kapalı</option>
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      Hangi hataların raporlanacağı
                    </p>
                  </div>
                </div>

                <div className="mt-6 flex justify-end">
                  <Button onClick={handleSettingsSave} disabled={saving}>
                    <Save className="w-4 h-4 mr-2" />
                    {saving ? 'Kaydediliyor...' : 'Ayarları Kaydet'}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </Layout>
  );
}
