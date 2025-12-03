import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import { HardDrive, RefreshCw } from 'lucide-react';

interface DiskInfo {
  device: string;
  size: string;
  used: string;
  available: string;
  percent_used: number;
  mount_point: string;
}

interface IOStats {
  device: string;
  trans_per_sec: string;
  blocks_read_per_sec: string;
  blocks_write_per_sec: string;
  total_blocks_read: string;
  total_blocks_write: string;
}

export default function DiskUsage() {
  const [disks, setDisks] = useState<DiskInfo[]>([]);
  const [ioStats, setIoStats] = useState<IOStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchDiskUsage();
  }, []);

  const fetchDiskUsage = async () => {
    try {
      const response = await api.get('/system/disk-usage');
      if (response.data.success) {
        setDisks(response.data.data?.disks || []);
        setIoStats(response.data.data?.io_stats || []);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Disk bilgisi alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const getUsageColor = (percent: number) => {
    if (percent >= 90) return 'bg-red-500';
    if (percent >= 75) return 'bg-orange-500';
    if (percent >= 50) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  const getDiskIcon = (mountPoint: string) => {
    if (mountPoint === '/') return '🔴';
    if (mountPoint === '/boot') return '🔵';
    if (mountPoint.includes('tmp')) return '⚪';
    return '🟢';
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
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <HardDrive className="w-7 h-7" />
              Geçerli Disk Kullanımı
            </h1>
            <p className="text-muted-foreground">
              Disk alanı ve I/O istatistiklerini görüntüleyin
            </p>
          </div>
          <Button onClick={fetchDiskUsage} variant="outline" size="sm">
            <RefreshCw className="w-4 h-4 mr-2" />
            Yenile
          </Button>
        </div>

        {error && (
          <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        {/* Current Disk Usage */}
        <div className="bg-card rounded-lg border border-border p-6">
          <h2 className="text-lg font-semibold mb-4">Disk Kullanım Bilgisi</h2>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium">Cihaz</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Boyut</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Kullanılan</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Kullanılabilir</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Kullanım %</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Bağlama Noktası</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {disks.map((disk, index) => (
                  <tr key={index} className="hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <span className="text-xl">{getDiskIcon(disk.mount_point)}</span>
                        <span className="font-mono text-sm">{disk.device}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm">{disk.size}</td>
                    <td className="px-4 py-3 text-sm">{disk.used}</td>
                    <td className="px-4 py-3 text-sm">{disk.available}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <div className="w-24 bg-muted rounded-full h-3">
                          <div
                            className={`h-3 rounded-full ${getUsageColor(disk.percent_used)}`}
                            style={{ width: `${disk.percent_used}%` }}
                          />
                        </div>
                        <span className={`text-sm font-medium ${
                          disk.percent_used >= 90 ? 'text-red-500' :
                          disk.percent_used >= 75 ? 'text-orange-500' : ''
                        }`}>
                          {disk.percent_used}%
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm font-mono">{disk.mount_point}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* IO Statistics */}
        {ioStats.length > 0 && (
          <div className="bg-card rounded-lg border border-border p-6">
            <h2 className="text-lg font-semibold mb-4">I/O İstatistikleri</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-muted/50">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-medium">Cihaz</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Trans./Sec</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Blocks Read/sec</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Blocks Written/Sec</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Total Blocks Read</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Total Blocks Written</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {ioStats.map((stat, index) => (
                    <tr key={index} className="hover:bg-muted/30">
                      <td className="px-4 py-3 font-mono text-sm">{stat.device}</td>
                      <td className="px-4 py-3 text-sm">{stat.trans_per_sec}</td>
                      <td className="px-4 py-3 text-sm">
                        <div className="flex items-center gap-2">
                          <div className="w-12 bg-blue-500 h-2 rounded" 
                               style={{ width: `${Math.min(parseFloat(stat.blocks_read_per_sec) * 2, 48)}px` }} />
                          {stat.blocks_read_per_sec}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <div className="flex items-center gap-2">
                          <div className="w-12 bg-green-500 h-2 rounded" 
                               style={{ width: `${Math.min(parseFloat(stat.blocks_write_per_sec) * 2, 48)}px` }} />
                          {stat.blocks_write_per_sec}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm">{stat.total_blocks_read}</td>
                      <td className="px-4 py-3 text-sm">{stat.total_blocks_write}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Legend */}
        <div className="bg-card rounded-lg border border-border p-4">
          <h3 className="text-sm font-medium mb-2">Renk Açıklaması</h3>
          <div className="flex flex-wrap gap-4 text-sm">
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-green-500" />
              <span>Normal (&lt;50%)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-yellow-500" />
              <span>Dikkat (50-75%)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-orange-500" />
              <span>Uyarı (75-90%)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-red-500" />
              <span>Kritik (&gt;90%)</span>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
