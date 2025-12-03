import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import api from '@/lib/api';
import { Skull, Save, Plus, X, AlertTriangle } from 'lucide-react';

interface BackgroundKillerSettings {
  processes: string[];
  trusted_users: string[];
}

export default function BackgroundKiller() {
  const [settings, setSettings] = useState<BackgroundKillerSettings>({
    processes: [],
    trusted_users: [],
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [newProcess, setNewProcess] = useState('');
  const [newUser, setNewUser] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      const response = await api.get('/system/background-killer');
      if (response.data.success) {
        setSettings(response.data.data);
      }
    } catch (err) {
      console.error('Failed to fetch settings:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    setSuccess('');

    try {
      const response = await api.post('/system/background-killer', settings);
      if (response.data.success) {
        setSuccess('Ayarlar başarıyla kaydedildi');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kaydetme başarısız');
    } finally {
      setSaving(false);
    }
  };

  const addProcess = () => {
    if (newProcess && !settings.processes.includes(newProcess)) {
      setSettings({
        ...settings,
        processes: [...settings.processes, newProcess],
      });
      setNewProcess('');
    }
  };

  const removeProcess = (process: string) => {
    setSettings({
      ...settings,
      processes: settings.processes.filter((p) => p !== process),
    });
  };

  const addUser = () => {
    if (newUser && !settings.trusted_users.includes(newUser)) {
      setSettings({
        ...settings,
        trusted_users: [...settings.trusted_users, newUser],
      });
      setNewUser('');
    }
  };

  const removeUser = (user: string) => {
    setSettings({
      ...settings,
      trusted_users: settings.trusted_users.filter((u) => u !== user),
    });
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
              <Skull className="w-7 h-7" />
              Arka Plan İşlem Sonlandırıcı
            </h1>
            <p className="text-muted-foreground">
              Tehlikeli arka plan işlemlerini otomatik olarak sonlandırın
            </p>
          </div>
          <Button onClick={handleSave} disabled={saving}>
            <Save className="w-4 h-4 mr-2" />
            {saving ? 'Kaydediliyor...' : 'Kaydet'}
          </Button>
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

        {/* Warning */}
        <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-yellow-500 mt-0.5" />
            <div>
              <h3 className="font-medium text-yellow-600">Dikkat</h3>
              <p className="text-sm text-muted-foreground mt-1">
                ServerPanel'i aşağıdaki işlemlerden herhangi birini bulduğunda sonlandıracak ve size bir e-posta gönderecek şekilde yapılandırabilirsiniz. 
                Kötü niyetli kullanıcılar, shell hesaplarında IRC bouncer çalıştırabilir. ServerPanel, bouncer "pine" gibi zararsız görünen bir şeye yeniden adlandırılsa bile bu işlemleri doğru şekilde algılar.
              </p>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Processes */}
          <div className="bg-card rounded-lg border border-border p-6">
            <h2 className="text-lg font-semibold mb-4">İzlenen İşlemler</h2>
            <p className="text-sm text-muted-foreground mb-4">
              Sunucunuzda çalışmasını istemediğiniz programların adlarını işaretleyin
            </p>

            <div className="space-y-2 max-h-64 overflow-y-auto mb-4">
              {settings.processes.map((process) => (
                <div
                  key={process}
                  className="flex items-center justify-between bg-muted/50 px-3 py-2 rounded"
                >
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={true}
                      onChange={() => removeProcess(process)}
                      className="rounded"
                    />
                    <span className="text-sm font-mono">{process}</span>
                  </label>
                  <button
                    onClick={() => removeProcess(process)}
                    className="text-muted-foreground hover:text-destructive"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>

            <div className="flex gap-2">
              <input
                type="text"
                value={newProcess}
                onChange={(e) => setNewProcess(e.target.value)}
                placeholder="Yeni işlem adı..."
                className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-sm"
                onKeyDown={(e) => e.key === 'Enter' && addProcess()}
              />
              <Button onClick={addProcess} size="sm">
                <Plus className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {/* Trusted Users */}
          <div className="bg-card rounded-lg border border-border p-6">
            <h2 className="text-lg font-semibold mb-4">Güvenilir Kullanıcılar</h2>
            <p className="text-sm text-muted-foreground mb-4">
              İşlem sonlandırıcının yok saymasını istediğiniz kullanıcıları listeleyin. 
              root, mysql, named ve UID'si 99'un altındaki kullanıcılar zaten güvenilir kabul edilir.
            </p>

            <div className="space-y-2 max-h-64 overflow-y-auto mb-4">
              {settings.trusted_users.length === 0 ? (
                <p className="text-sm text-muted-foreground italic">
                  Henüz güvenilir kullanıcı eklenmedi
                </p>
              ) : (
                settings.trusted_users.map((user) => (
                  <div
                    key={user}
                    className="flex items-center justify-between bg-muted/50 px-3 py-2 rounded"
                  >
                    <span className="text-sm font-mono">{user}</span>
                    <button
                      onClick={() => removeUser(user)}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))
              )}
            </div>

            <div className="flex gap-2">
              <input
                type="text"
                value={newUser}
                onChange={(e) => setNewUser(e.target.value)}
                placeholder="Kullanıcı adı..."
                className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-sm"
                onKeyDown={(e) => e.key === 'Enter' && addUser()}
              />
              <Button onClick={addUser} size="sm">
                <Plus className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
