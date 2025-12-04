import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import {
  Shield,
  RefreshCw,
  Power,
  Plus,
  Trash2,
  CheckCircle,
  AlertTriangle,
  ArrowDownLeft,
  ArrowUpRight,
} from 'lucide-react';

interface FirewallRule {
  id: number;
  to: string;
  action: string;
  from: string;
  direction: string;
}

interface FirewallStatus {
  active: boolean;
  rules: FirewallRule[];
  default: {
    incoming: string;
    outgoing: string;
  };
}

export default function Firewall() {
  const [status, setStatus] = useState<FirewallStatus | null>(null);
  const [notInstalled, setNotInstalled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [ruleForm, setRuleForm] = useState({
    port: '',
    protocol: 'tcp',
    action: 'allow',
    from: '',
  });

  useEffect(() => {
    fetchStatus();
  }, []);

  const fetchStatus = async () => {
    try {
      const response = await api.get('/security/firewall/status');
      if (response.data.success) {
        setNotInstalled(false);
        setStatus(response.data.data);
      }
    } catch (err: any) {
      if (err.response?.status === 500 || err.message?.includes('ufw')) {
        setNotInstalled(true);
      } else {
        setError(err.response?.data?.error || 'Firewall durumu alınamadı');
      }
    } finally {
      setLoading(false);
    }
  };

  const toggleFirewall = async (enable: boolean) => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.post('/security/firewall/toggle', { enable });
      if (response.data.success) {
        setSuccess(enable ? 'Firewall etkinleştirildi' : 'Firewall devre dışı bırakıldı');
        fetchStatus();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'İşlem başarısız');
    } finally {
      setActionLoading(false);
    }
  };

  const addRule = async () => {
    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.post('/security/firewall/rule', ruleForm);
      if (response.data.success) {
        setSuccess('Kural eklendi');
        setShowAddModal(false);
        setRuleForm({ port: '', protocol: 'tcp', action: 'allow', from: '' });
        fetchStatus();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kural eklenemedi');
    } finally {
      setActionLoading(false);
    }
  };

  const deleteRule = async (id: number) => {
    if (!confirm('Bu kuralı silmek istediğinizden emin misiniz?')) return;

    setActionLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await api.delete(`/security/firewall/rule/${id}`);
      if (response.data.success) {
        setSuccess('Kural silindi');
        fetchStatus();
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kural silinemedi');
    } finally {
      setActionLoading(false);
    }
  };

  const getActionColor = (action: string) => {
    switch (action.toUpperCase()) {
      case 'ALLOW':
        return 'text-green-600 bg-green-500/10';
      case 'DENY':
      case 'REJECT':
        return 'text-red-600 bg-red-500/10';
      case 'LIMIT':
        return 'text-yellow-600 bg-yellow-500/10';
      default:
        return 'text-muted-foreground bg-muted';
    }
  };

  // Common ports for quick add
  const commonPorts = [
    { port: '22', name: 'SSH' },
    { port: '80', name: 'HTTP' },
    { port: '443', name: 'HTTPS' },
    { port: '21', name: 'FTP' },
    { port: '25', name: 'SMTP' },
    { port: '587', name: 'Submission' },
    { port: '993', name: 'IMAPS' },
    { port: '3306', name: 'MySQL' },
    { port: '8443', name: 'Panel' },
  ];

  if (loading) {
    return (
      <Layout>
        <LoadingAnimation />
      </Layout>
    );
  }

  // Not installed state
  if (notInstalled) {
    return (
      <Layout>
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Shield className="w-7 h-7" />
              Firewall (UFW)
            </h1>
            <p className="text-muted-foreground">
              Uncomplicated Firewall yönetimi
            </p>
          </div>

          <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-8 text-center">
            <AlertTriangle className="w-16 h-16 text-yellow-500 mx-auto mb-4" />
            <h2 className="text-xl font-semibold mb-2">UFW Kurulu Değil</h2>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">
              UFW (Uncomplicated Firewall), sunucunuza gelen ve giden trafiği kontrol etmenizi sağlayan bir güvenlik aracıdır.
            </p>
            <div className="bg-card border border-border rounded-lg p-4 max-w-lg mx-auto">
              <p className="text-sm font-medium mb-2">Kurulum Komutu:</p>
              <code className="block bg-muted p-3 rounded text-sm font-mono text-left">
                apt-get update && apt-get install -y ufw
              </code>
            </div>
            <Button onClick={fetchStatus} variant="outline" className="mt-4">
              <RefreshCw className="w-4 h-4 mr-2" />
              Tekrar Kontrol Et
            </Button>
          </div>
        </div>
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
              <Shield className="w-7 h-7" />
              Firewall (UFW)
            </h1>
            <p className="text-muted-foreground">
              Uncomplicated Firewall yönetimi
            </p>
          </div>
          <div className="flex gap-2">
            <Button onClick={fetchStatus} variant="outline" size="sm">
              <RefreshCw className="w-4 h-4 mr-2" />
              Yenile
            </Button>
            {status?.active ? (
              <Button
                onClick={() => toggleFirewall(false)}
                variant="destructive"
                size="sm"
                disabled={actionLoading}
              >
                <Power className="w-4 h-4 mr-2" />
                Devre Dışı Bırak
              </Button>
            ) : (
              <Button
                onClick={() => toggleFirewall(true)}
                variant="default"
                size="sm"
                disabled={actionLoading}
              >
                <Power className="w-4 h-4 mr-2" />
                Etkinleştir
              </Button>
            )}
          </div>
        </div>

        {/* Messages */}
        {error && (
          <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg flex items-center gap-2">
            <AlertTriangle className="w-4 h-4" />
            {error}
          </div>
        )}
        {success && (
          <div className="bg-green-500/10 text-green-600 px-4 py-3 rounded-lg flex items-center gap-2">
            <CheckCircle className="w-4 h-4" />
            {success}
          </div>
        )}

        {/* Status Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Durum</span>
              {status?.active ? (
                <span className="flex items-center gap-1 text-green-600">
                  <CheckCircle className="w-4 h-4" />
                  Aktif
                </span>
              ) : (
                <span className="flex items-center gap-1 text-destructive">
                  <AlertTriangle className="w-4 h-4" />
                  Pasif
                </span>
              )}
            </div>
          </div>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Kural Sayısı</span>
              <span className="text-xl font-bold">{status?.rules?.length || 0}</span>
            </div>
          </div>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground flex items-center gap-1">
                <ArrowDownLeft className="w-3 h-3" />
                Gelen
              </span>
              <span
                className={`px-2 py-0.5 rounded text-sm ${
                  status?.default?.incoming === 'deny'
                    ? 'bg-red-500/10 text-red-600'
                    : 'bg-green-500/10 text-green-600'
                }`}
              >
                {status?.default?.incoming || 'deny'}
              </span>
            </div>
          </div>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground flex items-center gap-1">
                <ArrowUpRight className="w-3 h-3" />
                Giden
              </span>
              <span
                className={`px-2 py-0.5 rounded text-sm ${
                  status?.default?.outgoing === 'deny'
                    ? 'bg-red-500/10 text-red-600'
                    : 'bg-green-500/10 text-green-600'
                }`}
              >
                {status?.default?.outgoing || 'allow'}
              </span>
            </div>
          </div>
        </div>

        {/* Add Rule Button */}
        <Button onClick={() => setShowAddModal(true)} disabled={!status?.active}>
          <Plus className="w-4 h-4 mr-2" />
          Yeni Kural Ekle
        </Button>

        {/* Rules Table */}
        <div className="bg-card rounded-lg border border-border overflow-hidden">
          <div className="p-4 border-b border-border">
            <h2 className="text-lg font-semibold">Firewall Kuralları</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium">#</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Hedef</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Aksiyon</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Kaynak</th>
                  <th className="px-4 py-3 text-left text-sm font-medium">Yön</th>
                  <th className="px-4 py-3 text-right text-sm font-medium">İşlem</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {status?.rules?.map((rule) => (
                  <tr key={rule.id} className="hover:bg-muted/30">
                    <td className="px-4 py-3 text-sm">{rule.id}</td>
                    <td className="px-4 py-3 text-sm font-mono">{rule.to}</td>
                    <td className="px-4 py-3">
                      <span
                        className={`px-2 py-0.5 rounded text-xs font-medium ${getActionColor(
                          rule.action
                        )}`}
                      >
                        {rule.action}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm font-mono">
                      {rule.from || 'Anywhere'}
                    </td>
                    <td className="px-4 py-3 text-sm">{rule.direction || 'IN'}</td>
                    <td className="px-4 py-3 text-right">
                      <Button
                        onClick={() => deleteRule(rule.id)}
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        disabled={actionLoading}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </td>
                  </tr>
                ))}
                {(!status?.rules || status.rules.length === 0) && (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                      Henüz kural tanımlanmamış
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Add Rule Modal */}
        {showAddModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card rounded-lg border border-border p-6 w-full max-w-md">
              <h3 className="text-lg font-semibold mb-4">Yeni Firewall Kuralı</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Port</label>
                  <input
                    type="text"
                    value={ruleForm.port}
                    onChange={(e) => setRuleForm({ ...ruleForm, port: e.target.value })}
                    placeholder="22 veya 8000:9000"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                  <div className="flex flex-wrap gap-1 mt-2">
                    {commonPorts.map((p) => (
                      <button
                        key={p.port}
                        onClick={() => setRuleForm({ ...ruleForm, port: p.port })}
                        className="text-xs px-2 py-1 bg-muted rounded hover:bg-muted/80"
                      >
                        {p.name} ({p.port})
                      </button>
                    ))}
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Protokol</label>
                  <select
                    value={ruleForm.protocol}
                    onChange={(e) => setRuleForm({ ...ruleForm, protocol: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  >
                    <option value="tcp">TCP</option>
                    <option value="udp">UDP</option>
                    <option value="">TCP/UDP</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Aksiyon</label>
                  <select
                    value={ruleForm.action}
                    onChange={(e) => setRuleForm({ ...ruleForm, action: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  >
                    <option value="allow">İzin Ver (ALLOW)</option>
                    <option value="deny">Reddet (DENY)</option>
                    <option value="limit">Sınırla (LIMIT)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">
                    Kaynak IP (opsiyonel)
                  </label>
                  <input
                    type="text"
                    value={ruleForm.from}
                    onChange={(e) => setRuleForm({ ...ruleForm, from: e.target.value })}
                    placeholder="Boş bırakın = Anywhere"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button variant="outline" onClick={() => setShowAddModal(false)}>
                  İptal
                </Button>
                <Button onClick={addRule} disabled={actionLoading || !ruleForm.port}>
                  Ekle
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
