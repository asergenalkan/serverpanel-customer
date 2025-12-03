import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import { Terminal, RefreshCw, Search } from 'lucide-react';

interface SimpleProcess {
  pid: number;
  name: string;
  file: string;
  cwd: string;
  command: string;
}

export default function RunningProcesses() {
  const [processes, setProcesses] = useState<SimpleProcess[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    fetchProcesses();
  }, []);

  const fetchProcesses = async () => {
    try {
      const response = await api.get('/system/running-processes');
      if (response.data.success) {
        setProcesses(response.data.data || []);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'İşlem listesi alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const filteredProcesses = processes.filter((p) => {
    if (!searchTerm) return true;
    const term = searchTerm.toLowerCase();
    return (
      p.name.toLowerCase().includes(term) ||
      p.file.toLowerCase().includes(term) ||
      p.command.toLowerCase().includes(term) ||
      p.pid.toString().includes(term)
    );
  });

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
              <Terminal className="w-7 h-7" />
              Geçerli Çalışma İşlemlerini Göster
            </h1>
            <p className="text-muted-foreground">
              Sistemde çalışan tüm işlemleri görüntüleyin
            </p>
          </div>
          <Button onClick={fetchProcesses} variant="outline" size="sm">
            <RefreshCw className="w-4 h-4 mr-2" />
            Yenile
          </Button>
        </div>

        {error && (
          <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        {/* Search */}
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="İşlem ara (PID, isim, dosya, komut)..."
            className="w-full pl-10 pr-4 py-2 rounded-lg border border-border bg-background"
          />
        </div>

        {/* Process Table */}
        <div className="bg-card rounded-lg border border-border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium w-20">PID</th>
                  <th className="px-4 py-3 text-left text-sm font-medium w-40">Name</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">File</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Current Directory</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Command Line</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filteredProcesses.map((process) => (
                  <tr key={process.pid} className="hover:bg-muted/30">
                    <td className="px-4 py-2 text-sm font-mono">{process.pid}</td>
                    <td className="px-4 py-2 text-sm">
                      <span className="text-muted-foreground">(</span>
                      {process.name}
                      <span className="text-muted-foreground">)</span>
                    </td>
                    <td className="px-4 py-2 text-sm font-mono text-xs">
                      {process.file || '-'}
                    </td>
                    <td className="px-4 py-2 text-sm font-mono text-xs">
                      {process.cwd || '/'}
                    </td>
                    <td className="px-4 py-2 text-sm">
                      <div className="max-w-lg truncate font-mono text-xs" title={process.command}>
                        {process.command || '-'}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="px-4 py-2 bg-muted/50 text-sm text-muted-foreground">
            Toplam {filteredProcesses.length} işlem
            {searchTerm && ` (${processes.length} işlemden filtrelendi)`}
          </div>
        </div>
      </div>
    </Layout>
  );
}
