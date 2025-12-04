import React, { useState, useEffect } from 'react';
import { Shield, Mail, AlertTriangle, CheckCircle, Trash2, Plus, RefreshCw, Bug, Filter, Search, FolderSearch, FileWarning, Archive } from 'lucide-react';
import Layout from '../components/Layout';
import LoadingAnimation from '../components/LoadingAnimation';

interface SpamSettings {
  enabled: boolean;
  spam_score: number;
  auto_delete: boolean;
  auto_delete_score: number;
  spam_folder: boolean;
  whitelist: string[];
  blacklist: string[];
}

interface AntivirusStatus {
  clamav_installed: boolean;
  clamav_running: boolean;
  last_update: string;
  virus_db_version: string;
}

interface SpamStats {
  total_scanned: number;
  spam_detected: number;
  viruses_detected: number;
  last_24h_spam: number;
}

const SpamFiltersPage: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [activeTab, setActiveTab] = useState<'spam' | 'antivirus' | 'scanner' | 'quarantine' | 'whitelist' | 'blacklist'>('spam');
  
  const [settings, setSettings] = useState<SpamSettings>({
    enabled: true,
    spam_score: 5,
    auto_delete: false,
    auto_delete_score: 10,
    spam_folder: true,
    whitelist: [],
    blacklist: []
  });
  
  const [antivirusStatus, setAntivirusStatus] = useState<AntivirusStatus>({
    clamav_installed: false,
    clamav_running: false,
    last_update: '',
    virus_db_version: ''
  });
  
  const [stats, setStats] = useState<SpamStats>({
    total_scanned: 0,
    spam_detected: 0,
    viruses_detected: 0,
    last_24h_spam: 0
  });
  
  const [newWhitelist, setNewWhitelist] = useState('');
  const [newBlacklist, setNewBlacklist] = useState('');
  
  // Malware scanning state
  const [scanning, setScanning] = useState(false);
  const [scanPath, setScanPath] = useState('');
  const [scanResults, setScanResults] = useState<any>(null);
  const [quarantinedFiles, setQuarantinedFiles] = useState<any[]>([]);

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/spam/settings', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (response.ok) {
        const data = await response.json();
        setSettings(data.settings || settings);
        setAntivirusStatus(data.antivirus || antivirusStatus);
        setStats(data.stats || stats);
      }
    } catch (error) {
      console.error('Spam ayarları yüklenemedi:', error);
    } finally {
      setLoading(false);
    }
  };

  const saveSettings = async () => {
    setSaving(true);
    setMessage(null);
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/spam/settings', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(settings)
      });
      
      if (response.ok) {
        setMessage({ type: 'success', text: 'Spam ayarları kaydedildi' });
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Ayarlar kaydedilemedi' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    } finally {
      setSaving(false);
    }
  };

  const addToList = (type: 'whitelist' | 'blacklist') => {
    const value = type === 'whitelist' ? newWhitelist.trim() : newBlacklist.trim();
    if (!value) return;
    
    if (type === 'whitelist') {
      if (!settings.whitelist.includes(value)) {
        setSettings({ ...settings, whitelist: [...settings.whitelist, value] });
      }
      setNewWhitelist('');
    } else {
      if (!settings.blacklist.includes(value)) {
        setSettings({ ...settings, blacklist: [...settings.blacklist, value] });
      }
      setNewBlacklist('');
    }
  };

  const removeFromList = (type: 'whitelist' | 'blacklist', value: string) => {
    if (type === 'whitelist') {
      setSettings({ ...settings, whitelist: settings.whitelist.filter(v => v !== value) });
    } else {
      setSettings({ ...settings, blacklist: settings.blacklist.filter(v => v !== value) });
    }
  };

  const updateVirusDb = async () => {
    setMessage(null);
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/spam/update-clamav', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (response.ok) {
        setMessage({ type: 'success', text: 'Virüs veritabanı güncelleniyor...' });
        setTimeout(fetchSettings, 5000);
      } else {
        setMessage({ type: 'error', text: 'Güncelleme başlatılamadı' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const runScan = async (quick = false) => {
    setScanning(true);
    setScanResults(null);
    setMessage(null);
    
    try {
      const token = localStorage.getItem('token');
      const endpoint = quick ? '/api/v1/malware/quick-scan' : '/api/v1/malware/scan';
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ path: scanPath })
      });
      
      if (response.ok) {
        const data = await response.json();
        setScanResults(data.data);
        if (data.data.infected_files > 0 || data.data.infected_count > 0) {
          setMessage({ type: 'error', text: `Tehdit tespit edildi!` });
        } else {
          setMessage({ type: 'success', text: 'Tarama tamamlandı, tehdit bulunamadı' });
        }
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Tarama başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    } finally {
      setScanning(false);
    }
  };

  const fetchQuarantinedFiles = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/malware/quarantine', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (response.ok) {
        const data = await response.json();
        setQuarantinedFiles(data.data || []);
      }
    } catch (error) {
      console.error('Karantina dosyaları yüklenemedi:', error);
    }
  };

  const quarantineFile = async (path: string) => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/malware/quarantine', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ path })
      });
      
      if (response.ok) {
        setMessage({ type: 'success', text: 'Dosya karantinaya alındı' });
        // Remove from scan results
        if (scanResults?.results) {
          setScanResults({
            ...scanResults,
            results: scanResults.results.filter((r: any) => r.path !== path)
          });
        }
        fetchQuarantinedFiles();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Karantina başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const deleteInfectedFile = async (path: string) => {
    if (!confirm('Bu dosyayı kalıcı olarak silmek istediğinizden emin misiniz?')) return;
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/malware/file', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ path })
      });
      
      if (response.ok) {
        setMessage({ type: 'success', text: 'Dosya silindi' });
        // Remove from scan results
        if (scanResults?.results) {
          setScanResults({
            ...scanResults,
            results: scanResults.results.filter((r: any) => r.path !== path)
          });
        }
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Silme başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const deleteFromQuarantine = async (path: string) => {
    if (!confirm('Bu dosyayı karantinadan kalıcı olarak silmek istediğinizden emin misiniz?')) return;
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/malware/file', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ path })
      });
      
      if (response.ok) {
        setMessage({ type: 'success', text: 'Dosya silindi' });
        fetchQuarantinedFiles();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Silme başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  if (loading) {
    return (
      <Layout>
        <LoadingAnimation />
      </Layout>
    );
  }

  return (
    <Layout>
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Shield className="w-8 h-8 text-orange-500" />
          <div>
            <h1 className="text-2xl font-bold">Spam Filtreleri</h1>
            <p className="text-muted-foreground">SpamAssassin ve ClamAV ayarları</p>
          </div>
        </div>
        <button
          onClick={saveSettings}
          disabled={saving}
          className="px-4 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 disabled:opacity-50 flex items-center space-x-2"
        >
          {saving ? (
            <RefreshCw className="w-4 h-4 animate-spin" />
          ) : (
            <CheckCircle className="w-4 h-4" />
          )}
          <span>Kaydet</span>
        </button>
      </div>

      {/* Message */}
      {message && (
        <div className={`p-4 rounded-lg ${message.type === 'success' ? 'bg-green-500/10 text-green-600' : 'bg-destructive/10 text-destructive'}`}>
          {message.text}
        </div>
      )}

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-card rounded-lg border border-border p-4">
          <div className="flex items-center space-x-3">
            <Mail className="w-8 h-8 text-blue-500" />
            <div>
              <p className="text-sm text-muted-foreground">Taranan Mail</p>
              <p className="text-2xl font-bold">{stats.total_scanned}</p>
            </div>
          </div>
        </div>
        <div className="bg-card rounded-lg border border-border p-4">
          <div className="flex items-center space-x-3">
            <AlertTriangle className="w-8 h-8 text-yellow-500" />
            <div>
              <p className="text-sm text-muted-foreground">Spam Tespit</p>
              <p className="text-2xl font-bold">{stats.spam_detected}</p>
            </div>
          </div>
        </div>
        <div className="bg-card rounded-lg border border-border p-4">
          <div className="flex items-center space-x-3">
            <Bug className="w-8 h-8 text-red-500" />
            <div>
              <p className="text-sm text-muted-foreground">Virüs Tespit</p>
              <p className="text-2xl font-bold">{stats.viruses_detected}</p>
            </div>
          </div>
        </div>
        <div className="bg-card rounded-lg border border-border p-4">
          <div className="flex items-center space-x-3">
            <Filter className="w-8 h-8 text-orange-500" />
            <div>
              <p className="text-sm text-muted-foreground">Son 24 Saat</p>
              <p className="text-2xl font-bold">{stats.last_24h_spam}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-card rounded-lg border border-border">
        <div className="border-b border-border">
          <nav className="flex -mb-px">
            {[
              { id: 'spam', label: 'Spam Ayarları', icon: Mail },
              { id: 'antivirus', label: 'Antivirüs', icon: Bug },
              { id: 'scanner', label: 'Malware Tarama', icon: Search },
              { id: 'quarantine', label: 'Karantina', icon: Archive },
              { id: 'whitelist', label: 'Beyaz Liste', icon: CheckCircle },
              { id: 'blacklist', label: 'Kara Liste', icon: AlertTriangle }
            ].map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`flex items-center space-x-2 px-6 py-4 border-b-2 font-medium text-sm ${
                  activeTab === tab.id
                    ? 'border-orange-500 text-orange-500'
                    : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
                }`}
              >
                <tab.icon className="w-4 h-4" />
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {/* Spam Settings Tab */}
          {activeTab === 'spam' && (
            <div className="space-y-6">
              <div className="flex items-center justify-between p-4 bg-muted/50 rounded-lg">
                <div>
                  <h3 className="font-medium">SpamAssassin</h3>
                  <p className="text-sm text-muted-foreground">Spam filtrelemeyi etkinleştir</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.enabled}
                    onChange={(e) => setSettings({ ...settings, enabled: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-muted peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-500"></div>
                </label>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  Spam Eşik Puanı: {settings.spam_score}
                </label>
                <input
                  type="range"
                  min="1"
                  max="10"
                  step="0.5"
                  value={settings.spam_score}
                  onChange={(e) => setSettings({ ...settings, spam_score: parseFloat(e.target.value) })}
                  className="w-full h-2 bg-muted rounded-lg appearance-none cursor-pointer accent-orange-500"
                />
                <div className="flex justify-between text-xs text-muted-foreground mt-1">
                  <span>1 (Çok hassas)</span>
                  <span>10 (Az hassas)</span>
                </div>
              </div>

              <div className="flex items-center justify-between p-4 bg-muted/50 rounded-lg">
                <div>
                  <h3 className="font-medium">Spam Klasörüne Taşı</h3>
                  <p className="text-sm text-muted-foreground">Spam olarak işaretlenen mailleri Junk klasörüne taşı</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.spam_folder}
                    onChange={(e) => setSettings({ ...settings, spam_folder: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-muted peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-500"></div>
                </label>
              </div>

              <div className="flex items-center justify-between p-4 bg-destructive/10 rounded-lg">
                <div>
                  <h3 className="font-medium text-destructive">Otomatik Sil</h3>
                  <p className="text-sm text-destructive/80">Yüksek puanlı spamları otomatik sil (dikkatli kullanın)</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.auto_delete}
                    onChange={(e) => setSettings({ ...settings, auto_delete: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-muted peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-red-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-red-500"></div>
                </label>
              </div>

              {settings.auto_delete && (
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Otomatik Silme Eşiği: {settings.auto_delete_score}
                  </label>
                  <input
                    type="range"
                    min="5"
                    max="15"
                    step="1"
                    value={settings.auto_delete_score}
                    onChange={(e) => setSettings({ ...settings, auto_delete_score: parseInt(e.target.value) })}
                    className="w-full h-2 bg-muted rounded-lg appearance-none cursor-pointer accent-red-500"
                  />
                </div>
              )}
            </div>
          )}

          {/* Antivirus Tab */}
          {activeTab === 'antivirus' && (
            <div className="space-y-6">
              <div className={`p-6 rounded-lg ${antivirusStatus.clamav_running ? 'bg-green-500/10' : 'bg-destructive/10'}`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-4">
                    <Bug className={`w-12 h-12 ${antivirusStatus.clamav_running ? 'text-green-500' : 'text-destructive'}`} />
                    <div>
                      <h3 className="text-lg font-medium">ClamAV Antivirüs</h3>
                      <p className={`text-sm ${antivirusStatus.clamav_running ? 'text-green-600' : 'text-destructive'}`}>
                        {antivirusStatus.clamav_installed 
                          ? (antivirusStatus.clamav_running ? 'Çalışıyor' : 'Durduruldu')
                          : 'Kurulu Değil'}
                      </p>
                    </div>
                  </div>
                  {antivirusStatus.clamav_installed && (
                    <button
                      onClick={updateVirusDb}
                      className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 flex items-center space-x-2"
                    >
                      <RefreshCw className="w-4 h-4" />
                      <span>Veritabanını Güncelle</span>
                    </button>
                  )}
                </div>
              </div>

              {antivirusStatus.clamav_installed && (
                <div className="grid grid-cols-2 gap-4">
                  <div className="p-4 bg-muted/50 rounded-lg">
                    <p className="text-sm text-muted-foreground">Virüs DB Sürümü</p>
                    <p className="text-lg font-medium">{antivirusStatus.virus_db_version || 'Bilinmiyor'}</p>
                  </div>
                  <div className="p-4 bg-muted/50 rounded-lg">
                    <p className="text-sm text-muted-foreground">Son Güncelleme</p>
                    <p className="text-lg font-medium">{antivirusStatus.last_update || 'Bilinmiyor'}</p>
                  </div>
                </div>
              )}

              <div className="p-4 bg-blue-500/10 rounded-lg">
                <h4 className="font-medium text-blue-500 mb-2">ClamAV Özellikleri</h4>
                <ul className="text-sm text-blue-400 space-y-1">
                  <li>• Gelen mail eklerini otomatik tarar</li>
                  <li>• Virüslü mailleri reddeder</li>
                  <li>• Günlük otomatik veritabanı güncellemesi</li>
                  <li>• Ransomware, trojan, malware tespiti</li>
                </ul>
              </div>
            </div>
          )}

          {/* Malware Scanner Tab */}
          {activeTab === 'scanner' && (
            <div className="space-y-6">
              {!antivirusStatus.clamav_installed ? (
                <div className="p-6 bg-yellow-500/10 rounded-lg text-center">
                  <FileWarning className="w-12 h-12 text-yellow-500 mx-auto mb-3" />
                  <h3 className="text-lg font-medium">ClamAV Kurulu Değil</h3>
                  <p className="text-sm text-muted-foreground mt-2">
                    Malware taraması için ClamAV kurulu olmalıdır.
                    Yazılım Yöneticisi'nden kurabilirsiniz.
                  </p>
                </div>
              ) : (
                <>
                  <div className="flex items-center space-x-4">
                    <div className="flex-1">
                      <input
                        type="text"
                        value={scanPath}
                        onChange={(e) => setScanPath(e.target.value)}
                        placeholder="Taranacak dizin yolu (boş bırakırsanız public_html taranacak)"
                        className="w-full px-4 py-2 border border-border rounded-lg bg-background focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                      />
                    </div>
                    <button
                      onClick={() => runScan(true)}
                      disabled={scanning}
                      className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 flex items-center space-x-2"
                    >
                      {scanning ? (
                        <RefreshCw className="w-4 h-4 animate-spin" />
                      ) : (
                        <Search className="w-4 h-4" />
                      )}
                      <span>Hızlı Tara</span>
                    </button>
                    <button
                      onClick={() => runScan(false)}
                      disabled={scanning}
                      className="px-4 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 disabled:opacity-50 flex items-center space-x-2"
                    >
                      {scanning ? (
                        <RefreshCw className="w-4 h-4 animate-spin" />
                      ) : (
                        <FolderSearch className="w-4 h-4" />
                      )}
                      <span>Tam Tarama</span>
                    </button>
                  </div>

                  {scanning && (
                    <div className="p-6 bg-blue-500/10 rounded-lg text-center">
                      <RefreshCw className="w-12 h-12 text-blue-500 mx-auto mb-3 animate-spin" />
                      <h3 className="text-lg font-medium">Tarama Devam Ediyor...</h3>
                      <p className="text-sm text-muted-foreground mt-2">
                        Bu işlem dosya sayısına bağlı olarak birkaç dakika sürebilir.
                      </p>
                    </div>
                  )}

                  {scanResults && !scanning && (
                    <div className="space-y-4">
                      <div className="grid grid-cols-4 gap-4">
                        <div className="p-4 bg-muted/50 rounded-lg text-center">
                          <p className="text-2xl font-bold">{scanResults.total_files || 0}</p>
                          <p className="text-sm text-muted-foreground">Toplam Dosya</p>
                        </div>
                        <div className="p-4 bg-muted/50 rounded-lg text-center">
                          <p className="text-2xl font-bold">{scanResults.scanned_files || 0}</p>
                          <p className="text-sm text-muted-foreground">Taranan</p>
                        </div>
                        <div className={`p-4 rounded-lg text-center ${(scanResults.infected_files || scanResults.infected_count) > 0 ? 'bg-destructive/10' : 'bg-green-500/10'}`}>
                          <p className={`text-2xl font-bold ${(scanResults.infected_files || scanResults.infected_count) > 0 ? 'text-destructive' : 'text-green-500'}`}>
                            {scanResults.infected_files || scanResults.infected_count || 0}
                          </p>
                          <p className="text-sm text-muted-foreground">Tehdit</p>
                        </div>
                        <div className="p-4 bg-muted/50 rounded-lg text-center">
                          <p className="text-2xl font-bold">{scanResults.duration || '0s'}</p>
                          <p className="text-sm text-muted-foreground">Süre</p>
                        </div>
                      </div>

                      {scanResults.results && scanResults.results.length > 0 && (
                        <div className="border border-destructive/20 rounded-lg overflow-hidden">
                          <div className="bg-destructive/10 px-4 py-2 border-b border-destructive/20">
                            <h4 className="font-medium text-destructive">Tespit Edilen Tehditler</h4>
                          </div>
                          <div className="divide-y divide-border">
                            {scanResults.results.map((result: any, index: number) => (
                              <div key={index} className="p-4 flex items-center justify-between">
                                <div className="flex-1 min-w-0">
                                  <p className="font-mono text-sm truncate">{result.path}</p>
                                  <p className="text-sm text-destructive">{result.threat}</p>
                                </div>
                                <div className="flex items-center space-x-2 ml-4">
                                  <button
                                    onClick={() => quarantineFile(result.path)}
                                    className="px-3 py-1 bg-yellow-500 text-white rounded hover:bg-yellow-600 text-sm flex items-center space-x-1"
                                  >
                                    <Archive className="w-3 h-3" />
                                    <span>Karantina</span>
                                  </button>
                                  <button
                                    onClick={() => deleteInfectedFile(result.path)}
                                    className="px-3 py-1 bg-destructive text-white rounded hover:bg-destructive/80 text-sm flex items-center space-x-1"
                                  >
                                    <Trash2 className="w-3 h-3" />
                                    <span>Sil</span>
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {(!scanResults.results || scanResults.results.length === 0) && (
                        <div className="p-6 bg-green-500/10 rounded-lg text-center">
                          <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-3" />
                          <h3 className="text-lg font-medium text-green-600">Temiz!</h3>
                          <p className="text-sm text-muted-foreground mt-2">
                            Tarama tamamlandı, hiçbir tehdit bulunamadı.
                          </p>
                        </div>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>
          )}

          {/* Quarantine Tab */}
          {activeTab === 'quarantine' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  Karantinaya alınan dosyalar burada listelenir. Bu dosyaları geri yükleyebilir veya kalıcı olarak silebilirsiniz.
                </p>
                <button
                  onClick={fetchQuarantinedFiles}
                  className="px-3 py-1 bg-muted text-foreground rounded hover:bg-muted/80 text-sm flex items-center space-x-1"
                >
                  <RefreshCw className="w-3 h-3" />
                  <span>Yenile</span>
                </button>
              </div>

              {quarantinedFiles.length === 0 ? (
                <div className="p-8 text-center text-muted-foreground">
                  <Archive className="w-12 h-12 mx-auto mb-3 opacity-50" />
                  <p>Karantinada dosya yok</p>
                </div>
              ) : (
                <div className="border border-border rounded-lg overflow-hidden">
                  <div className="divide-y divide-border">
                    {quarantinedFiles.map((file, index) => (
                      <div key={index} className="p-4 flex items-center justify-between">
                        <div className="flex-1 min-w-0">
                          <p className="font-mono text-sm truncate">{file.name}</p>
                          <p className="text-xs text-muted-foreground">
                            Karantinaya alındı: {file.quarantined_at} | Boyut: {(file.size / 1024).toFixed(2)} KB
                          </p>
                        </div>
                        <div className="flex items-center space-x-2 ml-4">
                          <button
                            onClick={() => deleteFromQuarantine(file.path)}
                            className="px-3 py-1 bg-destructive text-white rounded hover:bg-destructive/80 text-sm flex items-center space-x-1"
                          >
                            <Trash2 className="w-3 h-3" />
                            <span>Sil</span>
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Whitelist Tab */}
          {activeTab === 'whitelist' && (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  value={newWhitelist}
                  onChange={(e) => setNewWhitelist(e.target.value)}
                  placeholder="email@example.com veya *@domain.com"
                  className="flex-1 px-4 py-2 border border-border rounded-lg bg-background focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  onKeyPress={(e) => e.key === 'Enter' && addToList('whitelist')}
                />
                <button
                  onClick={() => addToList('whitelist')}
                  className="px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 flex items-center space-x-2"
                >
                  <Plus className="w-4 h-4" />
                  <span>Ekle</span>
                </button>
              </div>

              <p className="text-sm text-muted-foreground">
                Beyaz listedeki adreslerden gelen mailler spam kontrolünden geçmez.
              </p>

              <div className="space-y-2">
                {settings.whitelist.length === 0 ? (
                  <p className="text-muted-foreground text-center py-8">Beyaz liste boş</p>
                ) : (
                  settings.whitelist.map((item, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-green-500/10 rounded-lg">
                      <span className="text-green-600">{item}</span>
                      <button
                        onClick={() => removeFromList('whitelist', item)}
                        className="text-red-500 hover:text-red-700"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          {/* Blacklist Tab */}
          {activeTab === 'blacklist' && (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  value={newBlacklist}
                  onChange={(e) => setNewBlacklist(e.target.value)}
                  placeholder="email@example.com veya *@domain.com"
                  className="flex-1 px-4 py-2 border border-border rounded-lg bg-background focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  onKeyPress={(e) => e.key === 'Enter' && addToList('blacklist')}
                />
                <button
                  onClick={() => addToList('blacklist')}
                  className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 flex items-center space-x-2"
                >
                  <Plus className="w-4 h-4" />
                  <span>Ekle</span>
                </button>
              </div>

              <p className="text-sm text-muted-foreground">
                Kara listedeki adreslerden gelen mailler otomatik olarak reddedilir.
              </p>

              <div className="space-y-2">
                {settings.blacklist.length === 0 ? (
                  <p className="text-muted-foreground text-center py-8">Kara liste boş</p>
                ) : (
                  settings.blacklist.map((item, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-destructive/10 rounded-lg">
                      <span className="text-destructive">{item}</span>
                      <button
                        onClick={() => removeFromList('blacklist', item)}
                        className="text-red-500 hover:text-red-700"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  </Layout>
  );
};

export default SpamFiltersPage;
