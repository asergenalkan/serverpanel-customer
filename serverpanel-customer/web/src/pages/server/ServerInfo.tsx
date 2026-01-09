import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Cpu, HardDrive, MemoryStick, Clock, Server, Wifi, Activity } from 'lucide-react';
import Layout from '@/components/Layout';
import api from '@/lib/api';

interface ServerInfo {
  hostname: string;
  os: string;
  kernel: string;
  uptime: string;
  load_average: string;
  cpu: {
    model: string;
    cores: number;
    usage: number;
  };
  memory: {
    total: number;
    used: number;
    free: number;
    usage: number;
  };
  disk: {
    total: number;
    used: number;
    free: number;
    usage: number;
  };
  network: {
    ip: string;
    interfaces: string[];
  };
}

export default function ServerInfoPage() {
  const [info, setInfo] = useState<ServerInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchServerInfo();
    const interval = setInterval(fetchServerInfo, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchServerInfo = async () => {
    try {
      const response = await api.get('/server/info');
      if (response.data.success) {
        setInfo(response.data.data);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Sunucu bilgileri alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getUsageColor = (usage: number) => {
    if (usage < 50) return 'bg-green-500';
    if (usage < 80) return 'bg-yellow-500';
    return 'bg-red-500';
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-destructive/10 text-destructive p-4 rounded-lg">
        {error}
      </div>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Sunucu Bilgileri</h1>
          <p className="text-muted-foreground">Sunucu durumu ve kaynak kullanımı</p>
        </div>

        {/* System Info */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-blue-500/10 rounded-lg">
                  <Server className="w-6 h-6 text-blue-500" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Hostname</p>
                  <p className="font-semibold">{info?.hostname || '-'}</p>
                </div>
              </div>
            </CardContent>
          </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-purple-500/10 rounded-lg">
                <Activity className="w-6 h-6 text-purple-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">İşletim Sistemi</p>
                <p className="font-semibold">{info?.os || '-'}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-green-500/10 rounded-lg">
                <Clock className="w-6 h-6 text-green-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Uptime</p>
                <p className="font-semibold">{info?.uptime || '-'}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-orange-500/10 rounded-lg">
                <Wifi className="w-6 h-6 text-orange-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">IP Adresi</p>
                <p className="font-semibold">{info?.network?.ip || '-'}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Resource Usage */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* CPU */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Cpu className="w-5 h-5" />
              CPU
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground">{info?.cpu?.model}</p>
                <p className="text-sm text-muted-foreground">{info?.cpu?.cores} Çekirdek</p>
              </div>
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm">Kullanım</span>
                  <span className="text-sm font-medium">{info?.cpu?.usage?.toFixed(1)}%</span>
                </div>
                <div className="h-3 bg-muted rounded-full overflow-hidden">
                  <div
                    className={`h-full ${getUsageColor(info?.cpu?.usage || 0)} transition-all`}
                    style={{ width: `${info?.cpu?.usage || 0}%` }}
                  />
                </div>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Load Average: {info?.load_average}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Memory */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MemoryStick className="w-5 h-5" />
              Bellek (RAM)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-2 text-center">
                <div>
                  <p className="text-lg font-bold">{formatBytes(info?.memory?.total || 0)}</p>
                  <p className="text-xs text-muted-foreground">Toplam</p>
                </div>
                <div>
                  <p className="text-lg font-bold text-blue-500">{formatBytes(info?.memory?.used || 0)}</p>
                  <p className="text-xs text-muted-foreground">Kullanılan</p>
                </div>
                <div>
                  <p className="text-lg font-bold text-green-500">{formatBytes(info?.memory?.free || 0)}</p>
                  <p className="text-xs text-muted-foreground">Boş</p>
                </div>
              </div>
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm">Kullanım</span>
                  <span className="text-sm font-medium">{info?.memory?.usage?.toFixed(1)}%</span>
                </div>
                <div className="h-3 bg-muted rounded-full overflow-hidden">
                  <div
                    className={`h-full ${getUsageColor(info?.memory?.usage || 0)} transition-all`}
                    style={{ width: `${info?.memory?.usage || 0}%` }}
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Disk */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <HardDrive className="w-5 h-5" />
              Disk
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-2 text-center">
                <div>
                  <p className="text-lg font-bold">{formatBytes(info?.disk?.total || 0)}</p>
                  <p className="text-xs text-muted-foreground">Toplam</p>
                </div>
                <div>
                  <p className="text-lg font-bold text-blue-500">{formatBytes(info?.disk?.used || 0)}</p>
                  <p className="text-xs text-muted-foreground">Kullanılan</p>
                </div>
                <div>
                  <p className="text-lg font-bold text-green-500">{formatBytes(info?.disk?.free || 0)}</p>
                  <p className="text-xs text-muted-foreground">Boş</p>
                </div>
              </div>
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm">Kullanım</span>
                  <span className="text-sm font-medium">{info?.disk?.usage?.toFixed(1)}%</span>
                </div>
                <div className="h-3 bg-muted rounded-full overflow-hidden">
                  <div
                    className={`h-full ${getUsageColor(info?.disk?.usage || 0)} transition-all`}
                    style={{ width: `${info?.disk?.usage || 0}%` }}
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Additional Info */}
      <Card>
        <CardHeader>
          <CardTitle>Sistem Detayları</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Kernel</span>
                <span className="font-mono text-sm">{info?.kernel}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Load Average</span>
                <span className="font-mono text-sm">{info?.load_average}</span>
              </div>
            </div>
            <div className="space-y-2">
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Network Interfaces</span>
                <span className="font-mono text-sm">{info?.network?.interfaces?.join(', ')}</span>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
    </Layout>
  );
}
