import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import Layout from '@/components/Layout';
import { 
  Server, 
  RefreshCw, 
  Check, 
  Code, 
  Puzzle, 
  Package,
  Info
} from 'lucide-react';
import api from '@/lib/api';

interface PHPVersion {
  name: string;
  display_name: string;
  version: string;
  installed: boolean;
  active: boolean;
}

interface Feature {
  name: string;
  display_name: string;
  description: string;
  version?: string;
}

interface ServerFeaturesData {
  php_versions: PHPVersion[];
  php_extensions: Feature[];
  apache_modules: Feature[];
  additional_software: Feature[];
}

export default function ServerFeatures() {
  const [features, setFeatures] = useState<ServerFeaturesData | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'php' | 'extensions' | 'modules' | 'software'>('php');

  useEffect(() => {
    fetchFeatures();
  }, []);

  const fetchFeatures = async () => {
    setLoading(true);
    try {
      const response = await api.get('/server/features');
      if (response.data.success) {
        setFeatures(response.data.data);
      }
    } catch (err) {
      console.error('Failed to fetch features:', err);
    } finally {
      setLoading(false);
    }
  };

  const installedPHP = features?.php_versions.filter(v => v.installed) || [];

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Sunucu Özellikleri</h1>
            <p className="text-muted-foreground">Sunucuda mevcut olan yazılımlar ve özellikler</p>
          </div>
          <Button onClick={fetchFeatures} variant="outline" size="sm" disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Yenile
          </Button>
        </div>

        {/* Info Banner */}
        <div className="bg-blue-500/10 text-blue-600 dark:text-blue-400 p-4 rounded-lg flex items-start gap-3">
          <Info className="w-5 h-5 mt-0.5" />
          <div className="text-sm">
            <p className="font-medium">Bu sayfa sadece bilgi amaçlıdır</p>
            <p>Sunucuda kurulu olan yazılımları ve özellikleri gösterir. PHP sürümünüzü değiştirmek için PHP Ayarları sayfasını kullanın.</p>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 border-b overflow-x-auto">
          {[
            { id: 'php', label: 'PHP Sürümleri', icon: Code, count: installedPHP.length },
            { id: 'extensions', label: 'PHP Eklentileri', icon: Puzzle, count: features?.php_extensions.length || 0 },
            { id: 'modules', label: 'Apache Modülleri', icon: Server, count: features?.apache_modules.length || 0 },
            { id: 'software', label: 'Ek Yazılımlar', icon: Package, count: features?.additional_software.length || 0 },
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`px-4 py-2 font-medium transition-colors flex items-center gap-2 whitespace-nowrap ${
                activeTab === tab.id
                  ? 'text-primary border-b-2 border-primary'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
              <span className="text-xs bg-muted px-2 py-0.5 rounded-full">{tab.count}</span>
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : (
          <>
            {/* PHP Versions */}
            {activeTab === 'php' && (
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                {features?.php_versions.map(php => (
                  <Card key={php.name} className={php.installed ? 'border-green-500/50' : 'opacity-50'}>
                    <CardContent className="pt-6 text-center">
                      <Code className={`w-8 h-8 mx-auto mb-2 ${php.installed ? 'text-primary' : 'text-muted-foreground'}`} />
                      <p className="font-bold">{php.display_name}</p>
                      {php.installed ? (
                        <>
                          <p className="text-xs text-muted-foreground">{php.version}</p>
                          <div className="flex items-center justify-center gap-1 mt-2 text-green-500 text-sm">
                            <Check className="w-4 h-4" />
                            {php.active ? 'Aktif' : 'Kurulu'}
                          </div>
                        </>
                      ) : (
                        <p className="text-xs text-muted-foreground mt-2">Kurulu Değil</p>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {/* PHP Extensions */}
            {activeTab === 'extensions' && (
              <Card>
                <CardHeader>
                  <CardTitle>Kurulu PHP Eklentileri</CardTitle>
                </CardHeader>
                <CardContent>
                  {features?.php_extensions.length === 0 ? (
                    <p className="text-muted-foreground text-center py-8">Kurulu eklenti bulunamadı</p>
                  ) : (
                    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                      {features?.php_extensions.map(ext => (
                        <div key={ext.name} className="flex items-center gap-2 p-3 border rounded-lg">
                          <Check className="w-4 h-4 text-green-500 flex-shrink-0" />
                          <div className="min-w-0">
                            <p className="font-medium truncate">{ext.display_name}</p>
                            <p className="text-xs text-muted-foreground truncate">{ext.description}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Apache Modules */}
            {activeTab === 'modules' && (
              <Card>
                <CardHeader>
                  <CardTitle>Aktif Apache Modülleri</CardTitle>
                </CardHeader>
                <CardContent>
                  {features?.apache_modules.length === 0 ? (
                    <p className="text-muted-foreground text-center py-8">Aktif modül bulunamadı</p>
                  ) : (
                    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                      {features?.apache_modules.map(mod => (
                        <div key={mod.name} className="flex items-center gap-2 p-3 border rounded-lg">
                          <Check className="w-4 h-4 text-green-500 flex-shrink-0" />
                          <div className="min-w-0">
                            <p className="font-medium truncate">{mod.display_name}</p>
                            <p className="text-xs text-muted-foreground truncate">{mod.description}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Additional Software */}
            {activeTab === 'software' && (
              <Card>
                <CardHeader>
                  <CardTitle>Kurulu Yazılımlar</CardTitle>
                </CardHeader>
                <CardContent>
                  {features?.additional_software.length === 0 ? (
                    <p className="text-muted-foreground text-center py-8">Kurulu yazılım bulunamadı</p>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {features?.additional_software.map(sw => (
                        <div key={sw.name} className="flex items-start gap-3 p-4 border rounded-lg">
                          <Package className="w-5 h-5 text-primary flex-shrink-0 mt-0.5" />
                          <div>
                            <p className="font-medium">{sw.name}</p>
                            <p className="text-sm text-muted-foreground">{sw.description}</p>
                            {sw.version && (
                              <p className="text-xs text-muted-foreground mt-1">
                                Sürüm: <span className="font-mono">{sw.version}</span>
                              </p>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            )}
          </>
        )}
      </div>
    </Layout>
  );
}
