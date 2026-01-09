import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import {
  Activity,
  RefreshCw,
  Skull,
  Search,
  AlertTriangle,
} from 'lucide-react';

interface Process {
  pid: number;
  name: string;
  user: string;
  priority: number;
  cpu_percent: number;
  mem_percent: number;
  command: string;
  file: string;
  cwd: string;
}

export default function ProcessManager() {
  const [processes, setProcesses] = useState<Process[]>([]);
  const [users, setUsers] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [searchTerm, setSearchTerm] = useState('');
  const [showKillModal, setShowKillModal] = useState(false);
  const [killTarget, setKillTarget] = useState<{ type: 'process' | 'user'; pid?: number; username?: string } | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    fetchProcesses();
    fetchUsers();
    let interval: ReturnType<typeof setInterval>;
    if (autoRefresh) {
      interval = setInterval(fetchProcesses, 3000);
    }
    return () => clearInterval(interval);
  }, [autoRefresh]);

  const fetchProcesses = async () => {
    try {
      const response = await api.get('/system/process-manager');
      if (response.data.success) {
        setProcesses(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch processes:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchUsers = async () => {
    try {
      const response = await api.get('/system/users');
      if (response.data.success) {
        setUsers(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch users:', err);
    }
  };

  const handleKillProcess = async (pid: number) => {
    setKillTarget({ type: 'process', pid });
    setShowKillModal(true);
  };

  const handleKillUserProcesses = () => {
    if (!selectedUser) return;
    setKillTarget({ type: 'user', username: selectedUser });
    setShowKillModal(true);
  };

  const confirmKill = async () => {
    if (!killTarget) return;

    setError('');
    setSuccess('');

    try {
      if (killTarget.type === 'process') {
        await api.post('/system/kill-process', { pid: killTarget.pid });
        setSuccess(`İşlem ${killTarget.pid} sonlandırıldı`);
      } else {
        await api.post('/system/kill-user-processes', { username: killTarget.username });
        setSuccess(`${killTarget.username} kullanıcısının tüm işlemleri sonlandırıldı`);
      }
      fetchProcesses();
    } catch (err: any) {
      setError(err.response?.data?.error || 'İşlem başarısız');
    } finally {
      setShowKillModal(false);
      setKillTarget(null);
    }
  };

  const getCpuColor = (cpu: number) => {
    if (cpu > 80) return 'text-red-500 font-bold';
    if (cpu > 50) return 'text-orange-500';
    if (cpu > 20) return 'text-yellow-500';
    return '';
  };

  const getMemColor = (mem: number) => {
    if (mem > 50) return 'text-red-500 font-bold';
    if (mem > 25) return 'text-orange-500';
    if (mem > 10) return 'text-yellow-500';
    return '';
  };

  const filteredProcesses = processes.filter((p) => {
    const matchesUser = !selectedUser || p.user === selectedUser;
    const matchesSearch = !searchTerm || 
      p.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      p.command.toLowerCase().includes(searchTerm.toLowerCase()) ||
      p.user.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesUser && matchesSearch;
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
              <Activity className="w-7 h-7" />
              İşlem Yöneticisi
            </h1>
            <p className="text-muted-foreground">
              Çalışan işlemleri görüntüleyin ve yönetin
            </p>
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

        {/* Messages */}
        {error && (
          <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg">
            {error}
          </div>
        )}
        {success && (
          <div className="bg-green-500/10 text-green-600 px-4 py-3 rounded-lg">
            {success}
          </div>
        )}

        {/* Filters */}
        <div className="flex flex-wrap items-center gap-4 bg-card rounded-lg border border-border p-4">
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium">Kullanıcıya göre filtrele:</label>
            <select
              value={selectedUser}
              onChange={(e) => setSelectedUser(e.target.value)}
              className="px-3 py-2 rounded-lg border border-border bg-background text-sm"
            >
              <option value="">Tümü</option>
              {users.map((user) => (
                <option key={user} value={user}>{user}</option>
              ))}
            </select>
          </div>

          {selectedUser && (
            <Button
              variant="destructive"
              size="sm"
              onClick={handleKillUserProcesses}
            >
              <Skull className="w-4 h-4 mr-2" />
              Kullanıcının İşlemlerini Sonlandır
            </Button>
          )}

          <div className="flex-1" />

          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="İşlem ara..."
              className="pl-10 pr-4 py-2 rounded-lg border border-border bg-background text-sm w-64"
            />
          </div>
        </div>

        {/* Process Table */}
        <div className="bg-card rounded-lg border border-border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium">PID</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Owner</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Priority</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">CPU %</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Memory %</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Command</th>
                  <th className="px-4 py-3 text-right text-sm font-medium">İşlemler</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filteredProcesses.slice(0, 100).map((process) => (
                  <tr key={process.pid} className="hover:bg-muted/30">
                    <td className="px-4 py-2 text-sm font-mono">{process.pid}</td>
                    <td className="px-4 py-2 text-sm">{process.user}</td>
                    <td className="px-4 py-2 text-sm text-center">{process.priority}</td>
                    <td className={`px-4 py-2 text-sm ${getCpuColor(process.cpu_percent)}`}>
                      <div className="flex items-center gap-2">
                        <div className="w-16 bg-muted rounded-full h-2">
                          <div
                            className={`h-2 rounded-full ${
                              process.cpu_percent > 80 ? 'bg-red-500' :
                              process.cpu_percent > 50 ? 'bg-orange-500' :
                              process.cpu_percent > 20 ? 'bg-yellow-500' : 'bg-green-500'
                            }`}
                            style={{ width: `${Math.min(process.cpu_percent, 100)}%` }}
                          />
                        </div>
                        {process.cpu_percent.toFixed(2)}
                      </div>
                    </td>
                    <td className={`px-4 py-2 text-sm ${getMemColor(process.mem_percent)}`}>
                      {process.mem_percent.toFixed(2)}
                    </td>
                    <td className="px-4 py-2 text-sm">
                      <div className="max-w-md truncate font-mono text-xs" title={process.command}>
                        {process.command}
                      </div>
                    </td>
                    <td className="px-4 py-2 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleKillProcess(process.pid)}
                        className="text-destructive hover:text-destructive"
                        title="Sonlandır"
                      >
                        <Skull className="w-4 h-4" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {filteredProcesses.length > 100 && (
            <div className="px-4 py-2 bg-muted/50 text-sm text-muted-foreground text-center">
              İlk 100 işlem gösteriliyor (toplam: {filteredProcesses.length})
            </div>
          )}
        </div>
      </div>

      {/* Kill Confirmation Modal */}
      {showKillModal && killTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border border-border w-full max-w-md mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center">
                <AlertTriangle className="w-6 h-6 text-destructive" />
              </div>
              <div>
                <h2 className="text-lg font-semibold">İşlemi Sonlandır</h2>
                <p className="text-sm text-muted-foreground">
                  {killTarget.type === 'process'
                    ? `PID ${killTarget.pid} işlemini sonlandırmak istediğinize emin misiniz?`
                    : `${killTarget.username} kullanıcısının tüm işlemlerini sonlandırmak istediğinize emin misiniz?`}
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setShowKillModal(false)}>
                İptal
              </Button>
              <Button variant="destructive" onClick={confirmKill}>
                <Skull className="w-4 h-4 mr-2" />
                Sonlandır
              </Button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}
