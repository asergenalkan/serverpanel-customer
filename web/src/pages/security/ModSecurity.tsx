import React, { useState, useEffect } from 'react';
import { Shield, AlertTriangle, CheckCircle, XCircle, RefreshCw, Trash2, Plus, Activity, FileText, Settings, List, Clock, Globe } from 'lucide-react';
import Layout from '../../components/Layout';
import LoadingAnimation from '../../components/LoadingAnimation';

interface ModSecurityStatus {
  installed: boolean;
  enabled: boolean;
  mode: string;
  crs_installed: boolean;
  crs_version: string;
  rules_count: number;
  audit_log_path: string;
  audit_log_size: number;
}

interface AuditLogEntry {
  timestamp: string;
  client_ip: string;
  request_uri: string;
  rule_id: string;
  rule_message: string;
  action: string;
  severity: string;
}

interface ModSecurityStats {
  total_requests: number;
  blocked_requests: number;
  logged_requests: number;
  top_rules: { rule_id: string; count: number }[];
  top_ips: { ip: string; count: number }[];
}

const ModSecurityPage: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<ModSecurityStatus | null>(null);
  const [stats, setStats] = useState<ModSecurityStats | null>(null);
  const [auditLog, setAuditLog] = useState<AuditLogEntry[]>([]);
  const [whitelist, setWhitelist] = useState<{ ip: string; comment: string }[]>([]);
  const [rules, setRules] = useState<any[]>([]);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'rules' | 'logs' | 'whitelist'>('overview');
  const [newWhitelistIP, setNewWhitelistIP] = useState('');
  const [newWhitelistComment, setNewWhitelistComment] = useState('');

  useEffect(() => {
    fetchStatus();
  }, []);

  useEffect(() => {
    if (activeTab === 'logs') {
      fetchAuditLog();
    } else if (activeTab === 'whitelist') {
      fetchWhitelist();
    } else if (activeTab === 'rules') {
      fetchRules();
    }
  }, [activeTab]);

  const fetchStatus = async () => {
    try {
      const token = localStorage.getItem('token');
      const [statusRes, statsRes] = await Promise.all([
        fetch('/api/v1/security/modsecurity/status', {
          headers: { 'Authorization': `Bearer ${token}` }
        }),
        fetch('/api/v1/security/modsecurity/stats', {
          headers: { 'Authorization': `Bearer ${token}` }
        })
      ]);

      if (statusRes.ok) {
        const data = await statusRes.json();
        setStatus(data.data);
      }

      if (statsRes.ok) {
        const data = await statsRes.json();
        setStats(data.data);
      }
    } catch (error) {
      console.error('ModSecurity durumu alınamadı:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchAuditLog = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/audit-log?limit=100', {
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (response.ok) {
        const data = await response.json();
        setAuditLog(data.data || []);
      }
    } catch (error) {
      console.error('Audit log alınamadı:', error);
    }
  };

  const fetchWhitelist = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/whitelist', {
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (response.ok) {
        const data = await response.json();
        setWhitelist(data.data || []);
      }
    } catch (error) {
      console.error('Whitelist alınamadı:', error);
    }
  };

  const fetchRules = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/rules', {
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (response.ok) {
        const data = await response.json();
        setRules(data.data || []);
      }
    } catch (error) {
      console.error('Kurallar alınamadı:', error);
    }
  };

  const toggleModSecurity = async () => {
    if (!status) return;

    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/toggle', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ enabled: !status.enabled })
      });

      if (response.ok) {
        setMessage({ type: 'success', text: `ModSecurity ${!status.enabled ? 'etkinleştirildi' : 'devre dışı bırakıldı'}` });
        fetchStatus();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'İşlem başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const setMode = async (mode: string) => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/mode', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ mode })
      });

      if (response.ok) {
        const modeText: Record<string, string> = {
          'On': 'Engelleme Modu',
          'DetectionOnly': 'Sadece Tespit Modu',
          'Off': 'Kapalı'
        };
        setMessage({ type: 'success', text: `Mod değiştirildi: ${modeText[mode]}` });
        fetchStatus();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'İşlem başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const addToWhitelist = async () => {
    if (!newWhitelistIP) return;

    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/whitelist', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ ip: newWhitelistIP, comment: newWhitelistComment })
      });

      if (response.ok) {
        setMessage({ type: 'success', text: 'IP whitelist\'e eklendi' });
        setNewWhitelistIP('');
        setNewWhitelistComment('');
        fetchWhitelist();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Ekleme başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const removeFromWhitelist = async (ip: string) => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/whitelist', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ ip })
      });

      if (response.ok) {
        setMessage({ type: 'success', text: 'IP whitelist\'ten kaldırıldı' });
        fetchWhitelist();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Kaldırma başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const clearAuditLog = async () => {
    if (!confirm('Audit log temizlenecek. Emin misiniz?')) return;

    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/security/modsecurity/audit-log', {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (response.ok) {
        setMessage({ type: 'success', text: 'Audit log temizlendi' });
        fetchAuditLog();
        fetchStatus();
      } else {
        const data = await response.json();
        setMessage({ type: 'error', text: data.error || 'Temizleme başarısız' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Bağlantı hatası' });
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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
          <div>
            <h1 className="text-2xl font-bold flex items-center space-x-2">
              <Shield className="w-8 h-8 text-orange-500" />
              <span>ModSecurity WAF</span>
            </h1>
            <p className="text-muted-foreground mt-1">Web Application Firewall yönetimi</p>
          </div>
          <button
            onClick={fetchStatus}
            className="px-4 py-2 bg-muted text-foreground rounded-lg hover:bg-muted/80 flex items-center space-x-2"
          >
            <RefreshCw className="w-4 h-4" />
            <span>Yenile</span>
          </button>
        </div>

        {/* Message */}
        {message && (
          <div className={`p-4 rounded-lg flex items-center space-x-2 ${
            message.type === 'success' ? 'bg-green-500/10 text-green-500' : 'bg-destructive/10 text-destructive'
          }`}>
            {message.type === 'success' ? <CheckCircle className="w-5 h-5" /> : <AlertTriangle className="w-5 h-5" />}
            <span>{message.text}</span>
            <button onClick={() => setMessage(null)} className="ml-auto">
              <XCircle className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Status Cards */}
        {status && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-card rounded-lg border border-border p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Durum</p>
                  <p className={`text-lg font-bold ${status.enabled ? 'text-green-500' : 'text-muted-foreground'}`}>
                    {status.enabled ? 'Aktif' : 'Pasif'}
                  </p>
                </div>
                <div className={`w-12 h-12 rounded-full flex items-center justify-center ${
                  status.enabled ? 'bg-green-500/10' : 'bg-muted'
                }`}>
                  <Shield className={`w-6 h-6 ${status.enabled ? 'text-green-500' : 'text-muted-foreground'}`} />
                </div>
              </div>
            </div>

            <div className="bg-card rounded-lg border border-border p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Mod</p>
                  <p className={`text-lg font-bold ${
                    status.mode === 'On' ? 'text-red-500' : 
                    status.mode === 'DetectionOnly' ? 'text-yellow-500' : 'text-muted-foreground'
                  }`}>
                    {status.mode === 'On' ? 'Engelleme' : 
                     status.mode === 'DetectionOnly' ? 'Tespit' : 'Kapalı'}
                  </p>
                </div>
                <Activity className={`w-6 h-6 ${
                  status.mode === 'On' ? 'text-red-500' : 
                  status.mode === 'DetectionOnly' ? 'text-yellow-500' : 'text-muted-foreground'
                }`} />
              </div>
            </div>

            <div className="bg-card rounded-lg border border-border p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">OWASP CRS</p>
                  <p className="text-lg font-bold">
                    {status.crs_installed ? `v${status.crs_version}` : 'Kurulu Değil'}
                  </p>
                </div>
                <List className={`w-6 h-6 ${status.crs_installed ? 'text-blue-500' : 'text-muted-foreground'}`} />
              </div>
            </div>

            <div className="bg-card rounded-lg border border-border p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Kural Sayısı</p>
                  <p className="text-lg font-bold">{status.rules_count}</p>
                </div>
                <FileText className="w-6 h-6 text-purple-500" />
              </div>
            </div>
          </div>
        )}

        {/* Not Installed Warning */}
        {status && !status.installed && (
          <div className="p-6 bg-yellow-500/10 rounded-lg text-center">
            <AlertTriangle className="w-12 h-12 text-yellow-500 mx-auto mb-3" />
            <h3 className="text-lg font-medium">ModSecurity Kurulu Değil</h3>
            <p className="text-sm text-muted-foreground mt-2">
              ModSecurity kurulumu için Yazılım Yöneticisi'ni kullanın veya sunucuyu yeniden kurun.
            </p>
          </div>
        )}

        {/* Main Content */}
        {status && status.installed && (
          <div className="bg-card rounded-lg border border-border">
            {/* Tabs */}
            <div className="border-b border-border">
              <nav className="flex -mb-px">
                {[
                  { id: 'overview', label: 'Genel Bakış', icon: Settings },
                  { id: 'rules', label: 'Kurallar', icon: List },
                  { id: 'logs', label: 'Audit Log', icon: FileText },
                  { id: 'whitelist', label: 'Whitelist', icon: CheckCircle }
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
              {/* Overview Tab */}
              {activeTab === 'overview' && (
                <div className="space-y-6">
                  {/* Toggle & Mode */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div className="p-4 bg-muted/50 rounded-lg">
                      <h3 className="font-medium mb-4">ModSecurity Durumu</h3>
                      <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">WAF Etkin</span>
                        <button
                          onClick={toggleModSecurity}
                          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                            status.enabled ? 'bg-green-500' : 'bg-muted'
                          }`}
                        >
                          <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                            status.enabled ? 'translate-x-6' : 'translate-x-1'
                          }`} />
                        </button>
                      </div>
                    </div>

                    <div className="p-4 bg-muted/50 rounded-lg">
                      <h3 className="font-medium mb-4">Çalışma Modu</h3>
                      <div className="flex space-x-2">
                        <button
                          onClick={() => setMode('DetectionOnly')}
                          className={`flex-1 px-3 py-2 rounded text-sm ${
                            status.mode === 'DetectionOnly'
                              ? 'bg-yellow-500 text-white'
                              : 'bg-muted text-foreground hover:bg-muted/80'
                          }`}
                        >
                          Sadece Tespit
                        </button>
                        <button
                          onClick={() => setMode('On')}
                          className={`flex-1 px-3 py-2 rounded text-sm ${
                            status.mode === 'On'
                              ? 'bg-red-500 text-white'
                              : 'bg-muted text-foreground hover:bg-muted/80'
                          }`}
                        >
                          Engelleme
                        </button>
                      </div>
                      <p className="text-xs text-muted-foreground mt-2">
                        {status.mode === 'DetectionOnly' 
                          ? 'Saldırılar sadece loglanır, engellenmez.' 
                          : 'Saldırılar tespit edildiğinde engellenir.'}
                      </p>
                    </div>
                  </div>

                  {/* Stats */}
                  {stats && (
                    <div className="space-y-4">
                      <h3 className="font-medium">İstatistikler</h3>
                      <div className="grid grid-cols-3 gap-4">
                        <div className="p-4 bg-blue-500/10 rounded-lg text-center">
                          <p className="text-2xl font-bold text-blue-500">{stats.total_requests}</p>
                          <p className="text-sm text-muted-foreground">Toplam İstek</p>
                        </div>
                        <div className="p-4 bg-red-500/10 rounded-lg text-center">
                          <p className="text-2xl font-bold text-red-500">{stats.blocked_requests}</p>
                          <p className="text-sm text-muted-foreground">Engellenen</p>
                        </div>
                        <div className="p-4 bg-yellow-500/10 rounded-lg text-center">
                          <p className="text-2xl font-bold text-yellow-500">{stats.logged_requests}</p>
                          <p className="text-sm text-muted-foreground">Loglanan</p>
                        </div>
                      </div>

                      {stats.top_rules && stats.top_rules.length > 0 && (
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div className="p-4 bg-muted/50 rounded-lg">
                            <h4 className="font-medium mb-3">En Çok Tetiklenen Kurallar</h4>
                            <div className="space-y-2">
                              {stats.top_rules.slice(0, 5).map((rule, idx) => (
                                <div key={idx} className="flex justify-between text-sm">
                                  <span className="font-mono">Rule #{rule.rule_id}</span>
                                  <span className="text-muted-foreground">{rule.count} kez</span>
                                </div>
                              ))}
                            </div>
                          </div>

                          <div className="p-4 bg-muted/50 rounded-lg">
                            <h4 className="font-medium mb-3">En Çok Engellenen IP'ler</h4>
                            <div className="space-y-2">
                              {stats.top_ips.slice(0, 5).map((item, idx) => (
                                <div key={idx} className="flex justify-between text-sm">
                                  <span className="font-mono">{item.ip}</span>
                                  <span className="text-muted-foreground">{item.count} kez</span>
                                </div>
                              ))}
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Detailed Info Section */}
                  <div className="space-y-4">
                    {/* What is ModSecurity */}
                    <div className="p-4 bg-blue-500/10 rounded-lg border border-blue-500/20">
                      <h4 className="font-medium text-blue-500 mb-3 flex items-center">
                        <Shield className="w-5 h-5 mr-2" />
                        ModSecurity Nedir?
                      </h4>
                      <p className="text-sm text-muted-foreground mb-3">
                        ModSecurity, web uygulamalarınızı koruyan açık kaynaklı bir <strong>Web Application Firewall (WAF)</strong>'dır. 
                        HTTP trafiğini gerçek zamanlı olarak analiz eder ve zararlı istekleri tespit edip engelleyebilir.
                      </p>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
                        <div className="flex items-start space-x-2">
                          <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                          <span><strong>SQL Injection</strong> - Veritabanı saldırılarını engeller</span>
                        </div>
                        <div className="flex items-start space-x-2">
                          <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                          <span><strong>XSS (Cross-Site Scripting)</strong> - Script enjeksiyonlarını önler</span>
                        </div>
                        <div className="flex items-start space-x-2">
                          <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                          <span><strong>CSRF</strong> - Sahte istek saldırılarını engeller</span>
                        </div>
                        <div className="flex items-start space-x-2">
                          <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                          <span><strong>Path Traversal</strong> - Dizin gezinme saldırılarını önler</span>
                        </div>
                        <div className="flex items-start space-x-2">
                          <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                          <span><strong>Remote Code Execution</strong> - Uzaktan kod çalıştırmayı engeller</span>
                        </div>
                        <div className="flex items-start space-x-2">
                          <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                          <span><strong>Bot/Scanner Koruması</strong> - Kötü niyetli botları engeller</span>
                        </div>
                      </div>
                    </div>

                    {/* Working Modes */}
                    <div className="p-4 bg-yellow-500/10 rounded-lg border border-yellow-500/20">
                      <h4 className="font-medium text-yellow-600 mb-3 flex items-center">
                        <AlertTriangle className="w-5 h-5 mr-2" />
                        Çalışma Modları - Önemli!
                      </h4>
                      <div className="space-y-3 text-sm">
                        <div className="p-3 bg-yellow-500/10 rounded border-l-4 border-yellow-500">
                          <p className="font-medium text-yellow-600">🔍 Sadece Tespit (DetectionOnly) - Önerilen Başlangıç</p>
                          <p className="text-muted-foreground mt-1">
                            Saldırılar tespit edilir ve loglanır ama <strong>engellenmez</strong>. 
                            Web siteniz normal çalışmaya devam eder. Önce bu modda çalıştırıp logları inceleyin.
                          </p>
                        </div>
                        <div className="p-3 bg-red-500/10 rounded border-l-4 border-red-500">
                          <p className="font-medium text-red-600">🛡️ Engelleme (On) - Dikkatli Kullanın</p>
                          <p className="text-muted-foreground mt-1">
                            Saldırılar tespit edildiğinde <strong>anında engellenir</strong> (HTTP 403). 
                            Yanlış pozitifler sitenizin çalışmasını engelleyebilir! Önce tespit modunda test edin.
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* Best Practices */}
                    <div className="p-4 bg-green-500/10 rounded-lg border border-green-500/20">
                      <h4 className="font-medium text-green-600 mb-3 flex items-center">
                        <CheckCircle className="w-5 h-5 mr-2" />
                        Önerilen Kullanım
                      </h4>
                      <ol className="text-sm text-muted-foreground space-y-2 list-decimal list-inside">
                        <li><strong>Tespit modunda başlayın</strong> - Önce "Sadece Tespit" modunu aktif edin</li>
                        <li><strong>1-2 hafta logları izleyin</strong> - Audit Log sekmesinden saldırıları ve yanlış pozitifleri inceleyin</li>
                        <li><strong>Yanlış pozitifleri whitelist'e ekleyin</strong> - Güvenilir IP'leri veya uygulamaları muaf tutun</li>
                        <li><strong>Engelleme moduna geçin</strong> - Yanlış pozitifler giderildikten sonra tam korumayı aktif edin</li>
                        <li><strong>Düzenli kontrol edin</strong> - Logları periyodik olarak inceleyin</li>
                      </ol>
                    </div>

                    {/* OWASP CRS Info */}
                    <div className="p-4 bg-purple-500/10 rounded-lg border border-purple-500/20">
                      <h4 className="font-medium text-purple-600 mb-3 flex items-center">
                        <List className="w-5 h-5 mr-2" />
                        OWASP Core Rule Set (CRS)
                      </h4>
                      <p className="text-sm text-muted-foreground mb-2">
                        ModSecurity, <strong>OWASP Core Rule Set</strong> ile birlikte çalışır. Bu kural seti:
                      </p>
                      <ul className="text-sm text-muted-foreground space-y-1">
                        <li>• OWASP Top 10 güvenlik açıklarına karşı koruma sağlar</li>
                        <li>• Dünya genelinde milyonlarca web sitesinde kullanılır</li>
                        <li>• Sürekli güncellenir ve yeni tehditlere karşı korunur</li>
                        <li>• WordPress, Drupal, Joomla gibi popüler CMS'lerle uyumludur</li>
                      </ul>
                    </div>
                  </div>
                </div>
              )}

              {/* Rules Tab */}
              {activeTab === 'rules' && (
                <div className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    OWASP Core Rule Set kuralları. Bu kurallar otomatik olarak yüklenir ve güncellenir.
                  </p>

                  {rules.length === 0 ? (
                    <div className="p-8 text-center text-muted-foreground">
                      <List className="w-12 h-12 mx-auto mb-3 opacity-50" />
                      <p>Kural bulunamadı</p>
                    </div>
                  ) : (
                    <div className="border border-border rounded-lg overflow-hidden">
                      <table className="w-full">
                        <thead className="bg-muted/50">
                          <tr>
                            <th className="px-4 py-3 text-left text-sm font-medium">Kategori</th>
                            <th className="px-4 py-3 text-left text-sm font-medium">Dosya</th>
                            <th className="px-4 py-3 text-left text-sm font-medium">Açıklama</th>
                            <th className="px-4 py-3 text-center text-sm font-medium">Durum</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {rules.map((rule, idx) => (
                            <tr key={idx} className="hover:bg-muted/30">
                              <td className="px-4 py-3 text-sm">
                                <span className={`px-2 py-1 rounded text-xs ${
                                  rule.category === 'REQUEST' ? 'bg-blue-500/10 text-blue-500' :
                                  rule.category === 'RESPONSE' ? 'bg-purple-500/10 text-purple-500' :
                                  'bg-muted text-muted-foreground'
                                }`}>
                                  {rule.category}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-sm font-mono">{rule.file}</td>
                              <td className="px-4 py-3 text-sm text-muted-foreground">{rule.description}</td>
                              <td className="px-4 py-3 text-sm text-center">
                                {rule.enabled ? (
                                  <CheckCircle className="w-4 h-4 text-green-500 mx-auto" />
                                ) : (
                                  <XCircle className="w-4 h-4 text-muted-foreground mx-auto" />
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {/* Logs Tab */}
              {activeTab === 'logs' && (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <p className="text-sm text-muted-foreground">
                      Son engellenen ve loglanan istekler. Log boyutu: {status ? formatBytes(status.audit_log_size) : '0 B'}
                    </p>
                    <div className="flex space-x-2">
                      <button
                        onClick={fetchAuditLog}
                        className="px-3 py-1 bg-muted text-foreground rounded hover:bg-muted/80 text-sm flex items-center space-x-1"
                      >
                        <RefreshCw className="w-3 h-3" />
                        <span>Yenile</span>
                      </button>
                      <button
                        onClick={clearAuditLog}
                        className="px-3 py-1 bg-destructive text-white rounded hover:bg-destructive/80 text-sm flex items-center space-x-1"
                      >
                        <Trash2 className="w-3 h-3" />
                        <span>Temizle</span>
                      </button>
                    </div>
                  </div>

                  {auditLog.length === 0 ? (
                    <div className="p-8 text-center text-muted-foreground">
                      <FileText className="w-12 h-12 mx-auto mb-3 opacity-50" />
                      <p>Henüz log kaydı yok</p>
                    </div>
                  ) : (
                    <div className="border border-border rounded-lg overflow-hidden max-h-[500px] overflow-y-auto">
                      <table className="w-full">
                        <thead className="bg-muted/50 sticky top-0">
                          <tr>
                            <th className="px-4 py-3 text-left text-sm font-medium">Zaman</th>
                            <th className="px-4 py-3 text-left text-sm font-medium">IP</th>
                            <th className="px-4 py-3 text-left text-sm font-medium">URI</th>
                            <th className="px-4 py-3 text-left text-sm font-medium">Kural</th>
                            <th className="px-4 py-3 text-left text-sm font-medium">Mesaj</th>
                            <th className="px-4 py-3 text-center text-sm font-medium">Aksiyon</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {auditLog.map((log, idx) => (
                            <tr key={idx} className="hover:bg-muted/30">
                              <td className="px-4 py-3 text-sm">
                                <div className="flex items-center space-x-1">
                                  <Clock className="w-3 h-3 text-muted-foreground" />
                                  <span>{log.timestamp || '-'}</span>
                                </div>
                              </td>
                              <td className="px-4 py-3 text-sm font-mono">{log.client_ip || '-'}</td>
                              <td className="px-4 py-3 text-sm font-mono truncate max-w-[200px]">{log.request_uri || '-'}</td>
                              <td className="px-4 py-3 text-sm font-mono">{log.rule_id || '-'}</td>
                              <td className="px-4 py-3 text-sm truncate max-w-[200px]">{log.rule_message || '-'}</td>
                              <td className="px-4 py-3 text-sm text-center">
                                <span className={`px-2 py-1 rounded text-xs ${
                                  log.action === 'blocked' ? 'bg-red-500/10 text-red-500' : 'bg-yellow-500/10 text-yellow-500'
                                }`}>
                                  {log.action === 'blocked' ? 'Engellendi' : 'Loglandı'}
                                </span>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {/* Whitelist Tab */}
              {activeTab === 'whitelist' && (
                <div className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    Whitelist'teki IP adresleri ModSecurity kurallarından muaf tutulur.
                  </p>

                  {/* Add Form */}
                  <div className="flex items-center space-x-2">
                    <input
                      type="text"
                      value={newWhitelistIP}
                      onChange={(e) => setNewWhitelistIP(e.target.value)}
                      placeholder="IP adresi (örn: 192.168.1.1)"
                      className="flex-1 px-4 py-2 border border-border rounded-lg bg-background focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                    <input
                      type="text"
                      value={newWhitelistComment}
                      onChange={(e) => setNewWhitelistComment(e.target.value)}
                      placeholder="Açıklama (opsiyonel)"
                      className="flex-1 px-4 py-2 border border-border rounded-lg bg-background focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                    <button
                      onClick={addToWhitelist}
                      className="px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 flex items-center space-x-2"
                    >
                      <Plus className="w-4 h-4" />
                      <span>Ekle</span>
                    </button>
                  </div>

                  {whitelist.length === 0 ? (
                    <div className="p-8 text-center text-muted-foreground">
                      <Globe className="w-12 h-12 mx-auto mb-3 opacity-50" />
                      <p>Whitelist boş</p>
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {whitelist.map((item, idx) => (
                        <div key={idx} className="flex items-center justify-between p-3 bg-green-500/10 rounded-lg">
                          <div>
                            <span className="font-mono text-green-600">{item.ip}</span>
                            {item.comment && (
                              <span className="text-sm text-muted-foreground ml-2">- {item.comment}</span>
                            )}
                          </div>
                          <button
                            onClick={() => removeFromWhitelist(item.ip)}
                            className="text-destructive hover:text-destructive/80"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
};

export default ModSecurityPage;
