import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Activity, RefreshCw } from 'lucide-react';
import Layout from '@/components/Layout';
import api from '@/lib/api';

interface Process {
  user: string;
  domain: string;
  cpu_percent: number;
  command: string;
  pid: number;
  memory: string;
}

export default function TopProcessesPage() {
  const [processes, setProcesses] = useState<Process[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);

  useEffect(() => {
    fetchProcesses();
    let interval: ReturnType<typeof setInterval>;
    if (autoRefresh) {
      interval = setInterval(fetchProcesses, 3000);
    }
    return () => clearInterval(interval);
  }, [autoRefresh]);

  const fetchProcesses = async () => {
    try {
      const response = await api.get('/server/processes');
      if (response.data.success) {
        setProcesses(response.data.data || []);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Process bilgileri alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const getCpuColor = (cpu: number) => {
    if (cpu > 80) return 'text-red-500 font-bold';
    if (cpu > 50) return 'text-orange-500';
    if (cpu > 20) return 'text-yellow-500';
    return '';
  };

  return (
    <Layout>
      <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Top Processes</h1>
          <p className="text-muted-foreground">En çok kaynak kullanan işlemler</p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant={autoRefresh ? 'default' : 'outline'}
            size="sm"
            onClick={() => setAutoRefresh(!autoRefresh)}
          >
            {autoRefresh ? 'Otomatik Yenileme: Açık' : 'Otomatik Yenileme: Kapalı'}
          </Button>
          <Button onClick={fetchProcesses} variant="outline" size="sm">
            <RefreshCw className="w-4 h-4 mr-2" />
            Yenile
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="w-5 h-5" />
            Aktif İşlemler
            {autoRefresh && (
              <span className="ml-2 text-xs text-muted-foreground font-normal">
                (Her 3 saniyede güncellenir)
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : error ? (
            <div className="text-center text-muted-foreground py-8">{error}</div>
          ) : processes.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              Aktif işlem bulunamadı.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-3 px-4 font-medium">Kullanıcı</th>
                    <th className="text-left py-3 px-4 font-medium">Domain</th>
                    <th className="text-right py-3 px-4 font-medium">% CPU</th>
                    <th className="text-right py-3 px-4 font-medium">Bellek</th>
                    <th className="text-left py-3 px-4 font-medium">İşlem</th>
                  </tr>
                </thead>
                <tbody>
                  {processes.map((proc, index) => (
                    <tr key={index} className="border-b hover:bg-muted/50">
                      <td className="py-3 px-4 font-mono text-sm">{proc.user}</td>
                      <td className="py-3 px-4 text-blue-500">{proc.domain || '-'}</td>
                      <td className={`py-3 px-4 text-right font-mono ${getCpuColor(proc.cpu_percent)}`}>
                        {proc.cpu_percent.toFixed(1)}
                      </td>
                      <td className="py-3 px-4 text-right font-mono text-sm">{proc.memory}</td>
                      <td className="py-3 px-4 text-sm max-w-md truncate" title={proc.command}>
                        <code className="text-xs bg-muted px-2 py-1 rounded">
                          {proc.command.length > 80 ? proc.command.substring(0, 80) + '...' : proc.command}
                        </code>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Legend */}
      <Card>
        <CardContent className="py-4">
          <div className="flex items-center gap-6 text-sm">
            <span className="text-muted-foreground">CPU Kullanımı:</span>
            <span className="text-green-500">● 0-20% Normal</span>
            <span className="text-yellow-500">● 20-50% Orta</span>
            <span className="text-orange-500">● 50-80% Yüksek</span>
            <span className="text-red-500">● 80%+ Kritik</span>
          </div>
        </CardContent>
      </Card>
    </div>
    </Layout>
  );
}
