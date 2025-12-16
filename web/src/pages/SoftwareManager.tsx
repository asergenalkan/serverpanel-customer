import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import TaskModal from '@/components/TaskModal';
import {
  Package,
  RefreshCw,
  Check,
  X,
  Download,
  Trash2,
  Power,
  PowerOff,
  ChevronDown,
  ChevronRight,
  Search,
  AlertTriangle,
  Box,
  Loader2,
} from 'lucide-react';
import api from '@/lib/api';

interface SoftwarePackage {
  name: string;
  display_name: string;
  description: string;
  version: string;
  installed: boolean;
  active: boolean;
  category: string;
}

interface PHPExtension {
  name: string;
  display_name: string;
  description: string;
  installed: boolean;
  php_versions: string[];
}

interface ApacheModule {
  name: string;
  display_name: string;
  description: string;
  enabled: boolean;
  available: boolean;
}

interface SoftwareOverview {
  php_versions: SoftwarePackage[];
  php_extensions: PHPExtension[];
  apache_modules: ApacheModule[];
  additional_software: SoftwarePackage[];
}

export default function SoftwareManager() {
  const [overview, setOverview] = useState<SoftwareOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState<'php' | 'extensions' | 'apache' | 'software' | 'nodejs'>('php');
  
  // Node.js state
  const [nodejsStatus, setNodejsStatus] = useState<any>(null);
  const [nodejsLoading, setNodejsLoading] = useState(false);
  const [installingNodejs, setInstallingNodejs] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [expandedPHP, setExpandedPHP] = useState<string | null>(null);
  
  // Task Modal state
  const [taskModalOpen, setTaskModalOpen] = useState(false);
  const [currentTaskId, setCurrentTaskId] = useState<string | null>(null);
  const [currentTaskName, setCurrentTaskName] = useState('');

  useEffect(() => {
    fetchOverview();
  }, []);

  useEffect(() => {
    if (activeTab === 'nodejs') {
      fetchNodejsStatus();
    }
  }, [activeTab]);

  const fetchNodejsStatus = async () => {
    setNodejsLoading(true);
    try {
      const response = await api.get('/software/nodejs/status');
      if (response.data.success) {
        setNodejsStatus(response.data.data);
      }
    } catch (err) {
      console.error('Node.js status fetch failed:', err);
    } finally {
      setNodejsLoading(false);
    }
  };

  const installNodejsSupport = async () => {
    if (!confirm('Node.js desteği kurulacak (NVM + PM2). Bu işlem birkaç dakika sürebilir. Devam etmek istiyor musunuz?')) return;
    setInstallingNodejs(true);
    try {
      const response = await api.post('/software/nodejs/install');
      if (response.data.success) {
        alert(response.data.message);
        fetchNodejsStatus();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Node.js kurulumu başarısız');
    } finally {
      setInstallingNodejs(false);
    }
  };

  const uninstallNodejsSupport = async () => {
    if (!confirm('Node.js desteği kaldırılacak (NVM + PM2). Tüm Node.js uygulamaları çalışmayı durduracak. Devam etmek istiyor musunuz?')) return;
    setInstallingNodejs(true);
    try {
      const response = await api.post('/software/nodejs/uninstall');
      if (response.data.success) {
        alert(response.data.message);
        fetchNodejsStatus();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Node.js kaldırma başarısız');
    } finally {
      setInstallingNodejs(false);
    }
  };

  const installNodeVersion = async (version: string) => {
    if (!confirm(`Node.js ${version} kurulacak. Devam etmek istiyor musunuz?`)) return;
    try {
      const response = await api.post('/software/nodejs/version/install', { version });
      if (response.data.success) {
        alert(response.data.message);
        fetchNodejsStatus();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Node.js versiyon kurulumu başarısız');
    }
  };

  const setActiveNodeVersion = async (version: string) => {
    try {
      const response = await api.post('/software/nodejs/version/set-active', { version });
      if (response.data.success) {
        alert(response.data.message);
        fetchNodejsStatus();
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'Node.js versiyon değiştirme başarısız');
    }
  };

  const fetchOverview = async () => {
    setLoading(true);
    try {
      const response = await api.get('/software/overview');
      if (response.data.success) {
        setOverview(response.data.data);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Yazılım bilgileri alınamadı');
    } finally {
      setLoading(false);
    }
  };

  // Start a task with real-time logs
  const startTask = async (type: string, action: string, target: string, phpVersion?: string) => {
    const taskName = `${target} ${action === 'install' ? 'kurulumu' : action === 'uninstall' ? 'kaldırılması' : action === 'enable' ? 'etkinleştirilmesi' : 'devre dışı bırakılması'}`;
    
    try {
      const response = await api.post('/tasks/start', {
        type,
        action,
        target,
        php_version: phpVersion,
      });
      
      if (response.data.success) {
        setCurrentTaskId(response.data.task_id);
        setCurrentTaskName(taskName);
        setTaskModalOpen(true);
      }
    } catch (err: any) {
      alert(err.response?.data?.error || 'İşlem başlatılamadı');
    }
  };

  const handleTaskComplete = (success: boolean) => {
    if (success) {
      fetchOverview();
    }
  };

  const closeTaskModal = () => {
    setTaskModalOpen(false);
    setCurrentTaskId(null);
    setCurrentTaskName('');
  };

  const installPHP = async (version: string) => {
    if (!confirm(`PHP ${version} kurulacak. Bu işlem birkaç dakika sürebilir. Devam etmek istiyor musunuz?`)) return;
    startTask('php', 'install', version);
  };

  const uninstallPHP = async (version: string) => {
    if (!confirm(`PHP ${version} kaldırılacak. Bu işlem geri alınamaz. Devam etmek istiyor musunuz?`)) return;
    startTask('php', 'uninstall', version);
  };

  const installExtension = async (phpVersion: string, extension: string) => {
    startTask('extension', 'install', extension, phpVersion);
  };

  const uninstallExtension = async (phpVersion: string, extension: string) => {
    startTask('extension', 'uninstall', extension, phpVersion);
  };

  const toggleApacheModule = async (module: string, enable: boolean) => {
    startTask('apache', enable ? 'enable' : 'disable', module);
  };

  const installSoftware = async (pkg: string) => {
    if (!confirm(`${pkg} kurulacak. Devam etmek istiyor musunuz?`)) return;
    startTask('software', 'install', pkg);
  };

  const uninstallSoftware = async (pkg: string) => {
    if (!confirm(`${pkg} kaldırılacak. Devam etmek istiyor musunuz?`)) return;
    startTask('software', 'uninstall', pkg);
  };

  const installedPHPVersions = overview?.php_versions.filter(v => v.installed) || [];

  const filteredExtensions = overview?.php_extensions.filter(ext =>
    ext.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    ext.display_name.toLowerCase().includes(searchTerm.toLowerCase())
  ) || [];

  const filteredModules = overview?.apache_modules.filter(mod =>
    mod.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    mod.display_name.toLowerCase().includes(searchTerm.toLowerCase())
  ) || [];

  const filteredSoftware = overview?.additional_software.filter(sw =>
    sw.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    sw.display_name.toLowerCase().includes(searchTerm.toLowerCase())
  ) || [];

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Yazılım Yöneticisi</h1>
            <p className="text-muted-foreground">PHP, Apache modülleri ve ek yazılımları yönetin</p>
          </div>
          <Button onClick={fetchOverview} variant="outline" size="sm" disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Yenile
          </Button>
        </div>

        {error && (
          <div className="bg-destructive/10 text-destructive p-4 rounded-lg flex items-center gap-2">
            <AlertTriangle className="w-5 h-5" />
            {error}
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-2 border-b">
          {[
            { id: 'php', label: 'PHP Sürümleri', count: installedPHPVersions.length },
            { id: 'extensions', label: 'PHP Eklentileri', count: overview?.php_extensions.filter(e => e.installed).length || 0 },
            { id: 'apache', label: 'Apache Modülleri', count: overview?.apache_modules.filter(m => m.enabled).length || 0 },
            { id: 'software', label: 'Ek Yazılımlar', count: overview?.additional_software.filter(s => s.installed).length || 0 },
            { id: 'nodejs', label: 'Node.js', count: nodejsStatus?.node_versions?.length || 0 },
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => { setActiveTab(tab.id as any); setSearchTerm(''); }}
              className={`px-4 py-2 font-medium transition-colors flex items-center gap-2 ${
                activeTab === tab.id
                  ? 'text-primary border-b-2 border-primary'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {tab.label}
              <span className="text-xs bg-muted px-2 py-0.5 rounded-full">{tab.count}</span>
            </button>
          ))}
        </div>

        {/* Search */}
        {activeTab !== 'php' && (
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Ara..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border rounded-lg bg-background"
            />
          </div>
        )}

        {loading ? (
          <LoadingAnimation />
        ) : (
          <>
            {/* PHP Versions Tab */}
            {activeTab === 'php' && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {overview?.php_versions.map(php => (
                  <Card key={php.name} className={php.installed ? 'border-green-500/50' : ''}>
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center justify-between">
                        <span className="flex items-center gap-2">
                          <Package className="w-5 h-5" />
                          {php.display_name}
                        </span>
                        {php.installed ? (
                          <span className="flex items-center gap-1 text-sm text-green-500">
                            <Check className="w-4 h-4" />
                            Kurulu
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-sm text-muted-foreground">
                            <X className="w-4 h-4" />
                            Kurulu Değil
                          </span>
                        )}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        {php.installed && (
                          <>
                            <div className="flex items-center justify-between text-sm">
                              <span className="text-muted-foreground">Sürüm:</span>
                              <span className="font-mono">{php.version}</span>
                            </div>
                            <div className="flex items-center justify-between text-sm">
                              <span className="text-muted-foreground">Durum:</span>
                              {php.active ? (
                                <span className="flex items-center gap-1 text-green-500">
                                  <Power className="w-4 h-4" />
                                  Aktif
                                </span>
                              ) : (
                                <span className="flex items-center gap-1 text-yellow-500">
                                  <PowerOff className="w-4 h-4" />
                                  Pasif
                                </span>
                              )}
                            </div>
                          </>
                        )}
                        <div className="flex gap-2 pt-2">
                          {php.installed ? (
                            <Button
                              variant="destructive"
                              size="sm"
                              className="flex-1"
                              onClick={() => uninstallPHP(php.name.replace('php', ''))}
                              disabled={installedPHPVersions.length <= 1}
                            >
                              <Trash2 className="w-4 h-4 mr-1" />
                              Kaldır
                            </Button>
                          ) : (
                            <Button
                              variant="default"
                              size="sm"
                              className="flex-1"
                              onClick={() => installPHP(php.name.replace('php', ''))}
                            >
                              <Download className="w-4 h-4 mr-1" />
                              Kur
                            </Button>
                          )}
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {/* PHP Extensions Tab */}
            {activeTab === 'extensions' && (
              <Card>
                <CardContent className="pt-6">
                  <div className="space-y-2">
                    {filteredExtensions.map(ext => (
                      <div key={ext.name} className="border rounded-lg">
                        <button
                          onClick={() => setExpandedPHP(expandedPHP === ext.name ? null : ext.name)}
                          className="w-full flex items-center justify-between p-4 hover:bg-muted/50"
                        >
                          <div className="flex items-center gap-3">
                            {ext.installed ? (
                              <Check className="w-5 h-5 text-green-500" />
                            ) : (
                              <X className="w-5 h-5 text-muted-foreground" />
                            )}
                            <div className="text-left">
                              <p className="font-medium">{ext.display_name}</p>
                              <p className="text-sm text-muted-foreground">{ext.description}</p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            {ext.installed && (
                              <span className="text-xs bg-green-500/10 text-green-500 px-2 py-1 rounded">
                                {ext.php_versions.length} PHP
                              </span>
                            )}
                            {expandedPHP === ext.name ? (
                              <ChevronDown className="w-5 h-5" />
                            ) : (
                              <ChevronRight className="w-5 h-5" />
                            )}
                          </div>
                        </button>
                        {expandedPHP === ext.name && (
                          <div className="border-t p-4 bg-muted/30">
                            <p className="text-sm text-muted-foreground mb-3">PHP sürümlerine göre kurulum durumu:</p>
                            <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
                              {installedPHPVersions.map(php => {
                                const version = php.name.replace('php', '');
                                const isInstalled = ext.php_versions.includes(version);
                                
                                return (
                                  <div key={version} className="flex items-center justify-between p-2 border rounded bg-background">
                                    <span className="text-sm font-medium">PHP {version}</span>
                                    <Button
                                      variant={isInstalled ? 'destructive' : 'default'}
                                      size="sm"
                                      onClick={() => isInstalled 
                                        ? uninstallExtension(version, ext.name)
                                        : installExtension(version, ext.name)
                                      }
                                    >
                                      {isInstalled ? (
                                        <Trash2 className="w-3 h-3" />
                                      ) : (
                                        <Download className="w-3 h-3" />
                                      )}
                                    </Button>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Apache Modules Tab */}
            {activeTab === 'apache' && (
              <Card>
                <CardContent className="pt-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {filteredModules.map(mod => (
                      <div
                        key={mod.name}
                        className={`flex items-center justify-between p-4 border rounded-lg ${
                          mod.enabled ? 'border-green-500/50 bg-green-500/5' : ''
                        }`}
                      >
                        <div>
                          <p className="font-medium">{mod.display_name}</p>
                          <p className="text-sm text-muted-foreground">{mod.description}</p>
                        </div>
                        <Button
                          variant={mod.enabled ? 'destructive' : 'default'}
                          size="sm"
                          onClick={() => toggleApacheModule(mod.name, !mod.enabled)}
                          disabled={!mod.available}
                        >
                          {mod.enabled ? (
                            <>
                              <PowerOff className="w-4 h-4 mr-1" />
                              Devre Dışı
                            </>
                          ) : (
                            <>
                              <Power className="w-4 h-4 mr-1" />
                              Etkinleştir
                            </>
                          )}
                        </Button>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Additional Software Tab */}
            {activeTab === 'software' && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {filteredSoftware.map(sw => (
                  <Card key={sw.name} className={sw.installed ? 'border-green-500/50' : ''}>
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center justify-between text-base">
                        <span>{sw.display_name}</span>
                        {sw.installed && sw.active && (
                          <span className="flex items-center gap-1 text-xs text-green-500">
                            <Power className="w-3 h-3" />
                            Aktif
                          </span>
                        )}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-muted-foreground mb-3">{sw.description}</p>
                      {sw.installed && sw.version && (
                        <p className="text-xs text-muted-foreground mb-3">
                          Sürüm: <span className="font-mono">{sw.version}</span>
                        </p>
                      )}
                      {sw.installed ? (
                        <Button
                          variant="destructive"
                          size="sm"
                          className="w-full"
                          onClick={() => uninstallSoftware(sw.name)}
                        >
                          <Trash2 className="w-4 h-4 mr-1" />
                          Kaldır
                        </Button>
                      ) : (
                        <Button
                          variant="default"
                          size="sm"
                          className="w-full"
                          onClick={() => installSoftware(sw.name)}
                        >
                          <Download className="w-4 h-4 mr-1" />
                          Kur
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {/* Node.js Tab */}
            {activeTab === 'nodejs' && (
              <div className="space-y-6">
                {nodejsLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <Loader2 className="w-8 h-8 animate-spin text-primary" />
                  </div>
                ) : !nodejsStatus?.nvm_installed ? (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <Box className="w-5 h-5" />
                        Node.js Desteği Kurulu Değil
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <p className="text-muted-foreground">
                        Node.js desteği aktif değil. Kurulum yapıldığında NVM (Node Version Manager) ve PM2 (Process Manager) kurulacaktır.
                      </p>
                      <ul className="text-sm text-muted-foreground list-disc list-inside space-y-1">
                        <li>Birden fazla Node.js sürümü yönetimi (NVM)</li>
                        <li>Node.js uygulama yönetimi (PM2)</li>
                        <li>Otomatik yeniden başlatma</li>
                        <li>Apache reverse proxy entegrasyonu</li>
                      </ul>
                      <Button 
                        onClick={installNodejsSupport} 
                        disabled={installingNodejs}
                        className="w-full md:w-auto"
                      >
                        {installingNodejs ? (
                          <>
                            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            Kuruluyor...
                          </>
                        ) : (
                          <>
                            <Download className="w-4 h-4 mr-2" />
                            Node.js Desteğini Kur
                          </>
                        )}
                      </Button>
                    </CardContent>
                  </Card>
                ) : (
                  <>
                    {/* Status Card */}
                    <Card className="border-green-500/50">
                      <CardHeader>
                        <CardTitle className="flex items-center justify-between">
                          <span className="flex items-center gap-2">
                            <Box className="w-5 h-5" />
                            Node.js Desteği
                          </span>
                          <span className="flex items-center gap-1 text-sm text-green-500">
                            <Check className="w-4 h-4" />
                            Aktif
                          </span>
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <div className="p-3 bg-muted rounded-lg">
                            <p className="text-xs text-muted-foreground">NVM</p>
                            <p className="font-medium text-green-500">Kurulu</p>
                          </div>
                          <div className="p-3 bg-muted rounded-lg">
                            <p className="text-xs text-muted-foreground">PM2</p>
                            <p className="font-medium text-green-500">{nodejsStatus?.pm2_installed ? 'Kurulu' : 'Kurulu Değil'}</p>
                          </div>
                          <div className="p-3 bg-muted rounded-lg">
                            <p className="text-xs text-muted-foreground">Aktif Sürüm</p>
                            <p className="font-medium font-mono">{nodejsStatus?.active_version || '-'}</p>
                          </div>
                          <div className="p-3 bg-muted rounded-lg">
                            <p className="text-xs text-muted-foreground">Kurulu Sürümler</p>
                            <p className="font-medium">{nodejsStatus?.node_versions?.length || 0}</p>
                          </div>
                        </div>
                        <Button 
                          variant="destructive" 
                          size="sm"
                          onClick={uninstallNodejsSupport}
                          disabled={installingNodejs}
                        >
                          {installingNodejs ? (
                            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                          ) : (
                            <Trash2 className="w-4 h-4 mr-2" />
                          )}
                          Node.js Desteğini Kaldır
                        </Button>
                      </CardContent>
                    </Card>

                    {/* Installed Versions */}
                    <Card>
                      <CardHeader>
                        <CardTitle>Kurulu Node.js Sürümleri</CardTitle>
                      </CardHeader>
                      <CardContent>
                        {nodejsStatus?.node_versions?.length > 0 ? (
                          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                            {nodejsStatus.node_versions.map((v: any) => (
                              <div 
                                key={v.version} 
                                className={`p-3 border rounded-lg ${v.active ? 'border-green-500 bg-green-500/10' : ''}`}
                              >
                                <p className="font-mono font-medium">{v.version}</p>
                                {v.active ? (
                                  <span className="text-xs text-green-500">Aktif</span>
                                ) : (
                                  <Button 
                                    variant="ghost" 
                                    size="sm" 
                                    className="text-xs p-0 h-auto"
                                    onClick={() => setActiveNodeVersion(v.version)}
                                  >
                                    Aktif Yap
                                  </Button>
                                )}
                              </div>
                            ))}
                          </div>
                        ) : (
                          <p className="text-muted-foreground text-sm">Henüz Node.js sürümü kurulmamış.</p>
                        )}
                      </CardContent>
                    </Card>

                    {/* Install New Version */}
                    <Card>
                      <CardHeader>
                        <CardTitle>Yeni Sürüm Kur</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
                          {['18', '20', '22', 'lts'].map(v => (
                            <Button
                              key={v}
                              variant="outline"
                              onClick={() => installNodeVersion(v)}
                            >
                              <Download className="w-4 h-4 mr-2" />
                              {v === 'lts' ? 'LTS' : `Node ${v}`}
                            </Button>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  </>
                )}
              </div>
            )}
          </>
        )}
      </div>

      {/* Task Modal */}
      <TaskModal
        isOpen={taskModalOpen}
        onClose={closeTaskModal}
        taskId={currentTaskId}
        taskName={currentTaskName}
        onComplete={handleTaskComplete}
      />
    </Layout>
  );
}
