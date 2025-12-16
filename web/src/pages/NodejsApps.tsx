import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import {
  Box,
  Plus,
  Play,
  Square,
  RotateCcw,
  Trash2,
  Settings,
  FileText,
  RefreshCw,
  ExternalLink,
  Loader2,
  AlertTriangle,
  Check,
  X,
  FolderOpen,
} from 'lucide-react';
import api from '@/lib/api';

interface NodejsApp {
  id: number;
  user_id: number;
  domain_id: number | null;
  name: string;
  app_root: string;
  startup_file: string;
  node_version: string;
  port: number;
  app_url: string;
  mode: string;
  environment: string;
  auto_restart: boolean;
  status: string;
  pm2_id: number | null;
  created_at: string;
  domain_name: string;
  username: string;
}

interface Domain {
  id: number;
  name: string;
}

export default function NodejsApps() {
  const [apps, setApps] = useState<NodejsApp[]>([]);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [loading, setLoading] = useState(true);
  const [nodejsEnabled, setNodejsEnabled] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showLogsModal, setShowLogsModal] = useState(false);
  const [showEnvModal, setShowEnvModal] = useState(false);
  const [selectedApp, setSelectedApp] = useState<NodejsApp | null>(null);
  const [logs, setLogs] = useState('');
  const [envVars, setEnvVars] = useState<{key: string, value: string}[]>([]);
  const [actionLoading, setActionLoading] = useState<number | null>(null);

  const [newApp, setNewApp] = useState({
    name: '',
    domain_id: null as number | null,
    app_root: '',
    startup_file: 'app.js',
    node_version: 'lts',
    app_url: '',
    mode: 'production',
  });

  useEffect(() => {
    checkNodejsSupport();
  }, []);

  const checkNodejsSupport = async () => {
    try {
      const response = await api.get('/settings/server');
      if (response.data.success) {
        setNodejsEnabled(response.data.data.nodejs_enabled);
        if (response.data.data.nodejs_enabled) {
          fetchApps();
          fetchDomains();
        }
      }
    } catch (err) {
      console.error('Failed to check Node.js support:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchApps = async () => {
    try {
      const response = await api.get('/nodejs/apps');
      if (response.data.success) {
        setApps(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch apps:', err);
    }
  };

  const fetchDomains = async () => {
    try {
      const response = await api.get('/domains');
      if (response.data.success) {
        setDomains(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch domains:', err);
    }
  };

  const createApp = async () => {
    if (!newApp.name || !newApp.app_root) {
      alert('Uygulama adı ve dizini gereklidir');
      return;
    }

    try {
      const response = await api.post('/nodejs/apps', newApp);
      if (response.data.success) {
        alert('Uygulama oluşturuldu');
        setShowCreateModal(false);
        setNewApp({
          name: '',
          domain_id: null,
          app_root: '',
          startup_file: 'app.js',
          node_version: 'lts',
          app_url: '',
          mode: 'production',
        });
        fetchApps();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Uygulama oluşturulamadı');
    }
  };

  const startApp = async (app: NodejsApp) => {
    setActionLoading(app.id);
    try {
      const response = await api.post(`/nodejs/apps/${app.id}/start`);
      if (response.data.success) {
        fetchApps();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Uygulama başlatılamadı');
    } finally {
      setActionLoading(null);
    }
  };

  const stopApp = async (app: NodejsApp) => {
    setActionLoading(app.id);
    try {
      const response = await api.post(`/nodejs/apps/${app.id}/stop`);
      if (response.data.success) {
        fetchApps();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Uygulama durdurulamadı');
    } finally {
      setActionLoading(null);
    }
  };

  const restartApp = async (app: NodejsApp) => {
    setActionLoading(app.id);
    try {
      const response = await api.post(`/nodejs/apps/${app.id}/restart`);
      if (response.data.success) {
        fetchApps();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Uygulama yeniden başlatılamadı');
    } finally {
      setActionLoading(null);
    }
  };

  const deleteApp = async (app: NodejsApp) => {
    if (!confirm(`"${app.name}" uygulamasını silmek istediğinize emin misiniz?`)) return;
    
    setActionLoading(app.id);
    try {
      const response = await api.delete(`/nodejs/apps/${app.id}`);
      if (response.data.success) {
        fetchApps();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Uygulama silinemedi');
    } finally {
      setActionLoading(null);
    }
  };

  const viewLogs = async (app: NodejsApp) => {
    setSelectedApp(app);
    setShowLogsModal(true);
    try {
      const response = await api.get(`/nodejs/apps/${app.id}/logs?lines=100`);
      if (response.data.success) {
        setLogs(response.data.data || 'Log bulunamadı');
      }
    } catch (err) {
      setLogs('Loglar alınamadı');
    }
  };

  const viewEnv = async (app: NodejsApp) => {
    setSelectedApp(app);
    setShowEnvModal(true);
    try {
      const response = await api.get(`/nodejs/apps/${app.id}/env`);
      if (response.data.success) {
        setEnvVars(response.data.data || []);
      }
    } catch (err) {
      setEnvVars([]);
    }
  };

  const saveEnv = async () => {
    if (!selectedApp) return;
    
    const envObj: Record<string, string> = {};
    envVars.forEach(v => {
      if (v.key) envObj[v.key] = v.value;
    });

    try {
      const response = await api.put(`/nodejs/apps/${selectedApp.id}/env`, { environment: envObj });
      if (response.data.success) {
        alert('Ortam değişkenleri kaydedildi');
        setShowEnvModal(false);
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Ortam değişkenleri kaydedilemedi');
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'online':
      case 'running':
        return <span className="flex items-center gap-1 text-green-500"><Check className="w-4 h-4" /> Çalışıyor</span>;
      case 'stopped':
        return <span className="flex items-center gap-1 text-gray-500"><Square className="w-4 h-4" /> Durduruldu</span>;
      case 'errored':
        return <span className="flex items-center gap-1 text-red-500"><X className="w-4 h-4" /> Hata</span>;
      default:
        return <span className="text-gray-400">{status}</span>;
    }
  };

  if (loading) {
    return (
      <Layout>
        <LoadingAnimation />
      </Layout>
    );
  }

  if (!nodejsEnabled) {
    return (
      <Layout>
        <div className="space-y-6">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-amber-500/10 rounded-lg">
              <AlertTriangle className="w-6 h-6 text-amber-500" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">Node.js Uygulamaları</h1>
              <p className="text-sm text-muted-foreground">Node.js desteği aktif değil</p>
            </div>
          </div>

          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <Box className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-medium mb-2">Node.js Desteği Kapalı</h3>
                <p className="text-muted-foreground mb-4">
                  Node.js uygulamaları oluşturmak için önce Node.js desteğinin etkinleştirilmesi gerekiyor.
                </p>
                <p className="text-sm text-muted-foreground">
                  Admin panelinden <strong>Sunucu Ayarları</strong> veya <strong>Yazılım Yöneticisi</strong> sayfasından Node.js desteğini etkinleştirebilirsiniz.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-500/10 rounded-lg">
              <Box className="w-6 h-6 text-green-500" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">Node.js Uygulamaları</h1>
              <p className="text-sm text-muted-foreground">PM2 ile Node.js uygulama yönetimi</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button onClick={fetchApps} variant="outline" size="sm">
              <RefreshCw className="w-4 h-4 mr-2" />
              Yenile
            </Button>
            <Button onClick={() => setShowCreateModal(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Yeni Uygulama
            </Button>
          </div>
        </div>

        {/* Apps List */}
        {apps.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <Box className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-medium mb-2">Henüz Uygulama Yok</h3>
                <p className="text-muted-foreground mb-4">
                  İlk Node.js uygulamanızı oluşturun.
                </p>
                <Button onClick={() => setShowCreateModal(true)}>
                  <Plus className="w-4 h-4 mr-2" />
                  Uygulama Oluştur
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {apps.map(app => (
              <Card key={app.id} className={app.status === 'online' || app.status === 'running' ? 'border-green-500/50' : ''}>
                <CardHeader className="pb-2">
                  <CardTitle className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Box className="w-5 h-5" />
                      <span>{app.name}</span>
                      {getStatusBadge(app.status)}
                    </div>
                    <div className="flex items-center gap-2">
                      {actionLoading === app.id ? (
                        <Loader2 className="w-5 h-5 animate-spin" />
                      ) : (
                        <>
                          {app.status === 'online' || app.status === 'running' ? (
                            <>
                              <Button variant="outline" size="sm" onClick={() => restartApp(app)} title="Yeniden Başlat">
                                <RotateCcw className="w-4 h-4" />
                              </Button>
                              <Button variant="outline" size="sm" onClick={() => stopApp(app)} title="Durdur">
                                <Square className="w-4 h-4" />
                              </Button>
                            </>
                          ) : (
                            <Button variant="default" size="sm" onClick={() => startApp(app)} title="Başlat">
                              <Play className="w-4 h-4" />
                            </Button>
                          )}
                          <Button variant="outline" size="sm" onClick={() => viewLogs(app)} title="Loglar">
                            <FileText className="w-4 h-4" />
                          </Button>
                          <Button variant="outline" size="sm" onClick={() => viewEnv(app)} title="Ortam Değişkenleri">
                            <Settings className="w-4 h-4" />
                          </Button>
                          <Button variant="destructive" size="sm" onClick={() => deleteApp(app)} title="Sil">
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </>
                      )}
                    </div>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <p className="text-muted-foreground">Dizin</p>
                      <p className="font-mono text-xs truncate" title={app.app_root}>{app.app_root}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Başlangıç Dosyası</p>
                      <p className="font-mono">{app.startup_file}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Port</p>
                      <p className="font-mono">{app.port}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Mode</p>
                      <p>{app.mode}</p>
                    </div>
                    {app.app_url && (
                      <div className="col-span-2">
                        <p className="text-muted-foreground">URL</p>
                        <a href={`http://${app.app_url}`} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline flex items-center gap-1">
                          {app.app_url} <ExternalLink className="w-3 h-3" />
                        </a>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background rounded-lg p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <h2 className="text-xl font-bold mb-4">Yeni Node.js Uygulaması</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Uygulama Adı *</label>
                <input
                  type="text"
                  value={newApp.name}
                  onChange={(e) => setNewApp({ ...newApp, name: e.target.value })}
                  placeholder="my-app"
                  className="w-full p-2 border rounded-lg bg-background"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Uygulama Dizini *</label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={newApp.app_root}
                    onChange={(e) => setNewApp({ ...newApp, app_root: e.target.value })}
                    placeholder="/home/user/myapp"
                    className="flex-1 p-2 border rounded-lg bg-background"
                  />
                  <Button variant="outline" size="sm" className="shrink-0">
                    <FolderOpen className="w-4 h-4" />
                  </Button>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Başlangıç Dosyası</label>
                <input
                  type="text"
                  value={newApp.startup_file}
                  onChange={(e) => setNewApp({ ...newApp, startup_file: e.target.value })}
                  placeholder="app.js"
                  className="w-full p-2 border rounded-lg bg-background"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Node.js Sürümü</label>
                <select
                  value={newApp.node_version}
                  onChange={(e) => setNewApp({ ...newApp, node_version: e.target.value })}
                  className="w-full p-2 border rounded-lg bg-background"
                >
                  <option value="lts">LTS (Önerilen)</option>
                  <option value="18">Node 18</option>
                  <option value="20">Node 20</option>
                  <option value="22">Node 22</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Domain (İsteğe Bağlı)</label>
                <select
                  value={newApp.domain_id || ''}
                  onChange={(e) => setNewApp({ ...newApp, domain_id: e.target.value ? parseInt(e.target.value) : null })}
                  className="w-full p-2 border rounded-lg bg-background"
                >
                  <option value="">Domain Seçin</option>
                  {domains.map(d => (
                    <option key={d.id} value={d.id}>{d.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Uygulama URL Yolu</label>
                <input
                  type="text"
                  value={newApp.app_url}
                  onChange={(e) => setNewApp({ ...newApp, app_url: e.target.value })}
                  placeholder="example.com/api"
                  className="w-full p-2 border rounded-lg bg-background"
                />
                <p className="text-xs text-muted-foreground mt-1">Domain seçiliyse, bu yol için proxy yapılandırması oluşturulur</p>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Mode</label>
                <select
                  value={newApp.mode}
                  onChange={(e) => setNewApp({ ...newApp, mode: e.target.value })}
                  className="w-full p-2 border rounded-lg bg-background"
                >
                  <option value="production">Production</option>
                  <option value="development">Development</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <Button variant="outline" onClick={() => setShowCreateModal(false)}>İptal</Button>
              <Button onClick={createApp}>Oluştur</Button>
            </div>
          </div>
        </div>
      )}

      {/* Logs Modal */}
      {showLogsModal && selectedApp && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background rounded-lg p-6 w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold">{selectedApp.name} - Loglar</h2>
              <Button variant="ghost" size="sm" onClick={() => setShowLogsModal(false)}>
                <X className="w-5 h-5" />
              </Button>
            </div>
            <div className="flex-1 overflow-auto bg-[#1a1b26] rounded-lg p-4">
              <pre className="text-sm font-mono text-gray-300 whitespace-pre-wrap">{logs}</pre>
            </div>
          </div>
        </div>
      )}

      {/* Environment Variables Modal */}
      {showEnvModal && selectedApp && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background rounded-lg p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold">{selectedApp.name} - Ortam Değişkenleri</h2>
              <Button variant="ghost" size="sm" onClick={() => setShowEnvModal(false)}>
                <X className="w-5 h-5" />
              </Button>
            </div>
            <div className="space-y-3">
              {envVars.map((env, index) => (
                <div key={index} className="flex gap-2">
                  <input
                    type="text"
                    value={env.key}
                    onChange={(e) => {
                      const newVars = [...envVars];
                      newVars[index].key = e.target.value;
                      setEnvVars(newVars);
                    }}
                    placeholder="KEY"
                    className="flex-1 p-2 border rounded-lg bg-background font-mono text-sm"
                  />
                  <input
                    type="text"
                    value={env.value}
                    onChange={(e) => {
                      const newVars = [...envVars];
                      newVars[index].value = e.target.value;
                      setEnvVars(newVars);
                    }}
                    placeholder="value"
                    className="flex-1 p-2 border rounded-lg bg-background font-mono text-sm"
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEnvVars(envVars.filter((_, i) => i !== index))}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                onClick={() => setEnvVars([...envVars, { key: '', value: '' }])}
                className="w-full"
              >
                <Plus className="w-4 h-4 mr-2" />
                Değişken Ekle
              </Button>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <Button variant="outline" onClick={() => setShowEnvModal(false)}>İptal</Button>
              <Button onClick={saveEnv}>Kaydet</Button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}
