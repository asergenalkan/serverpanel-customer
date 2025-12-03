import { useState, useEffect } from 'react';
import Layout from '@/components/Layout';
import LoadingAnimation from '@/components/LoadingAnimation';
import { Button } from '@/components/ui/Button';
import { useAuth } from '@/contexts/AuthContext';
import axios from 'axios';
import {
  Clock,
  Plus,
  Trash2,
  Play,
  Pause,
  Edit,
  PlayCircle,
  Terminal,
  Calendar,
  CheckCircle,
  XCircle,
  AlertCircle,
  X,
} from 'lucide-react';

interface CronJob {
  id: number;
  user_id: number;
  name: string;
  command: string;
  schedule: string;
  minute: string;
  hour: string;
  day: string;
  month: string;
  weekday: string;
  active: boolean;
  last_run: string | null;
  next_run: string | null;
  last_status: string | null;
  last_output: string | null;
  created_at: string;
  updated_at: string;
  owner_username?: string;
}

interface CronPreset {
  key: string;
  label: string;
  minute: string;
  hour: string;
  day: string;
  month: string;
  weekday: string;
}

export default function CronJobs() {
  const { user } = useAuth();
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [presets, setPresets] = useState<CronPreset[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [showOutputModal, setShowOutputModal] = useState(false);
  const [selectedJob, setSelectedJob] = useState<CronJob | null>(null);
  const [editingJob, setEditingJob] = useState<CronJob | null>(null);
  const [runOutput, setRunOutput] = useState<string>('');
  const [runStatus, setRunStatus] = useState<string>('');
  const [formData, setFormData] = useState({
    name: '',
    command: '',
    schedule: 'daily',
    minute: '0',
    hour: '0',
    day: '*',
    month: '*',
    weekday: '*',
  });
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    fetchJobs();
    fetchPresets();
  }, []);

  const fetchJobs = async () => {
    try {
      const response = await axios.get('/api/v1/cron/jobs');
      if (response.data.success) {
        setJobs(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch cron jobs:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchPresets = async () => {
    try {
      const response = await axios.get('/api/v1/cron/presets');
      if (response.data.success) {
        setPresets(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch presets:', err);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      if (editingJob) {
        await axios.put(`/api/v1/cron/jobs/${editingJob.id}`, formData);
        setSuccess('Cron işi başarıyla güncellendi');
      } else {
        await axios.post('/api/v1/cron/jobs', formData);
        setSuccess('Cron işi başarıyla oluşturuldu');
      }
      setShowModal(false);
      setEditingJob(null);
      resetForm();
      fetchJobs();
    } catch (err: any) {
      setError(err.response?.data?.error || 'İşlem başarısız');
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Bu cron işini silmek istediğinize emin misiniz?')) return;

    try {
      await axios.delete(`/api/v1/cron/jobs/${id}`);
      setSuccess('Cron işi silindi');
      fetchJobs();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Silme işlemi başarısız');
    }
  };

  const handleToggle = async (id: number) => {
    try {
      await axios.post(`/api/v1/cron/jobs/${id}/toggle`);
      fetchJobs();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Durum değiştirme başarısız');
    }
  };

  const handleRun = async (job: CronJob) => {
    setSelectedJob(job);
    setRunOutput('Çalıştırılıyor...');
    setRunStatus('running');
    setShowOutputModal(true);

    try {
      const response = await axios.post(`/api/v1/cron/jobs/${job.id}/run`);
      if (response.data.success) {
        setRunOutput(response.data.data?.output || 'Çıktı yok');
        setRunStatus(response.data.data?.status || 'success');
      } else {
        setRunOutput(response.data.error || 'Hata oluştu');
        setRunStatus('failed');
      }
      fetchJobs();
    } catch (err: any) {
      setRunOutput(err.response?.data?.error || 'Çalıştırma başarısız');
      setRunStatus('failed');
    }
  };

  const handleEdit = (job: CronJob) => {
    setEditingJob(job);
    setFormData({
      name: job.name,
      command: job.command,
      schedule: 'custom',
      minute: job.minute,
      hour: job.hour,
      day: job.day,
      month: job.month,
      weekday: job.weekday,
    });
    setShowModal(true);
  };

  const resetForm = () => {
    setFormData({
      name: '',
      command: '',
      schedule: 'daily',
      minute: '0',
      hour: '0',
      day: '*',
      month: '*',
      weekday: '*',
    });
  };

  const handlePresetChange = (presetKey: string) => {
    const preset = presets.find((p) => p.key === presetKey);
    if (preset) {
      setFormData({
        ...formData,
        schedule: presetKey,
        minute: preset.minute,
        hour: preset.hour,
        day: preset.day,
        month: preset.month,
        weekday: preset.weekday,
      });
    }
  };

  const formatSchedule = (job: CronJob) => {
    return `${job.minute} ${job.hour} ${job.day} ${job.month} ${job.weekday}`;
  };

  const getScheduleLabel = (job: CronJob) => {
    const schedule = formatSchedule(job);
    
    // Common patterns
    if (schedule === '* * * * *') return 'Her dakika';
    if (schedule === '*/5 * * * *') return 'Her 5 dakika';
    if (schedule === '*/15 * * * *') return 'Her 15 dakika';
    if (schedule === '*/30 * * * *') return 'Her 30 dakika';
    if (schedule === '0 * * * *') return 'Saatlik';
    if (schedule === '0 0 * * *') return 'Günlük (gece yarısı)';
    if (schedule === '0 0 * * 0') return 'Haftalık (Pazar)';
    if (schedule === '0 0 1 * *') return 'Aylık (ayın 1\'i)';
    
    return schedule;
  };

  const getStatusIcon = (status: string | null) => {
    if (!status) return <AlertCircle className="w-4 h-4 text-muted-foreground" />;
    if (status === 'success') return <CheckCircle className="w-4 h-4 text-green-500" />;
    return <XCircle className="w-4 h-4 text-red-500" />;
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
            <h1 className="text-2xl font-bold">Cron Jobs</h1>
            <p className="text-muted-foreground">
              Zamanlanmış görevlerinizi yönetin
            </p>
          </div>
          <Button
            onClick={() => {
              setEditingJob(null);
              resetForm();
              setShowModal(true);
            }}
          >
            <Plus className="w-4 h-4 mr-2" />
            Yeni Cron İşi
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

        {/* Jobs List */}
        <div className="bg-card rounded-lg border border-border overflow-hidden">
          {jobs.length === 0 ? (
            <div className="p-12 text-center">
              <Clock className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium mb-2">Henüz cron işi yok</h3>
              <p className="text-muted-foreground mb-4">
                Zamanlanmış görevler oluşturarak işlerinizi otomatikleştirin
              </p>
              <Button
                onClick={() => {
                  setEditingJob(null);
                  resetForm();
                  setShowModal(true);
                }}
              >
                <Plus className="w-4 h-4 mr-2" />
                İlk Cron İşinizi Oluşturun
              </Button>
            </div>
          ) : (
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium">
                    İş Adı
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium">
                    Komut
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium">
                    Zamanlama
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium">
                    Son Çalışma
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium">
                    Durum
                  </th>
                  {user?.role === 'admin' && (
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Sahip
                    </th>
                  )}
                  <th className="px-4 py-3 text-right text-sm font-medium">
                    İşlemler
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {jobs.map((job) => (
                  <tr
                    key={job.id}
                    className={`hover:bg-muted/30 ${!job.active ? 'opacity-50' : ''}`}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Clock className="w-4 h-4 text-primary" />
                        <span className="font-medium">{job.name}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <code className="text-xs bg-muted px-2 py-1 rounded max-w-xs truncate block">
                        {job.command}
                      </code>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Calendar className="w-4 h-4 text-muted-foreground" />
                        <span className="text-sm">{getScheduleLabel(job)}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {job.last_run || 'Henüz çalışmadı'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(job.last_status)}
                        <span className="text-sm">
                          {job.active ? 'Aktif' : 'Pasif'}
                        </span>
                      </div>
                    </td>
                    {user?.role === 'admin' && (
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        {job.owner_username}
                      </td>
                    )}
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRun(job)}
                          title="Şimdi Çalıştır"
                        >
                          <PlayCircle className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleToggle(job.id)}
                          title={job.active ? 'Durdur' : 'Başlat'}
                        >
                          {job.active ? (
                            <Pause className="w-4 h-4" />
                          ) : (
                            <Play className="w-4 h-4" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEdit(job)}
                          title="Düzenle"
                        >
                          <Edit className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(job.id)}
                          className="text-destructive hover:text-destructive"
                          title="Sil"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* Cron Expression Help */}
        <div className="bg-card rounded-lg border border-border p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Terminal className="w-5 h-5" />
            Cron İfadesi Yardımı
          </h3>
          <div className="grid grid-cols-5 gap-4 text-sm">
            <div className="text-center">
              <div className="font-mono bg-muted px-3 py-2 rounded mb-2">*</div>
              <div className="text-muted-foreground">Dakika</div>
              <div className="text-xs text-muted-foreground">(0-59)</div>
            </div>
            <div className="text-center">
              <div className="font-mono bg-muted px-3 py-2 rounded mb-2">*</div>
              <div className="text-muted-foreground">Saat</div>
              <div className="text-xs text-muted-foreground">(0-23)</div>
            </div>
            <div className="text-center">
              <div className="font-mono bg-muted px-3 py-2 rounded mb-2">*</div>
              <div className="text-muted-foreground">Gün</div>
              <div className="text-xs text-muted-foreground">(1-31)</div>
            </div>
            <div className="text-center">
              <div className="font-mono bg-muted px-3 py-2 rounded mb-2">*</div>
              <div className="text-muted-foreground">Ay</div>
              <div className="text-xs text-muted-foreground">(1-12)</div>
            </div>
            <div className="text-center">
              <div className="font-mono bg-muted px-3 py-2 rounded mb-2">*</div>
              <div className="text-muted-foreground">Hafta Günü</div>
              <div className="text-xs text-muted-foreground">(0-7, 0=Pazar)</div>
            </div>
          </div>
          <div className="mt-4 text-sm text-muted-foreground">
            <p><strong>Örnekler:</strong></p>
            <ul className="list-disc list-inside mt-2 space-y-1">
              <li><code className="bg-muted px-1 rounded">*/5 * * * *</code> - Her 5 dakikada bir</li>
              <li><code className="bg-muted px-1 rounded">0 * * * *</code> - Her saat başı</li>
              <li><code className="bg-muted px-1 rounded">0 0 * * *</code> - Her gece yarısı</li>
              <li><code className="bg-muted px-1 rounded">0 0 * * 0</code> - Her Pazar gece yarısı</li>
              <li><code className="bg-muted px-1 rounded">0 0 1 * *</code> - Her ayın 1'inde gece yarısı</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border border-border w-full max-w-lg mx-4">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <h2 className="text-lg font-semibold">
                {editingJob ? 'Cron İşini Düzenle' : 'Yeni Cron İşi'}
              </h2>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setShowModal(false);
                  setEditingJob(null);
                }}
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
            <form onSubmit={handleSubmit} className="p-4 space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">İş Adı</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                  placeholder="Örn: Veritabanı Yedekleme"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">Komut</label>
                <input
                  type="text"
                  value={formData.command}
                  onChange={(e) =>
                    setFormData({ ...formData, command: e.target.value })
                  }
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background font-mono text-sm"
                  placeholder="Örn: /usr/bin/php /home/user/backup.php"
                  required
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Tam yol kullanmanız önerilir
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  Zamanlama Şablonu
                </label>
                <select
                  value={formData.schedule}
                  onChange={(e) => handlePresetChange(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background"
                >
                  {presets.map((preset) => (
                    <option key={preset.key} value={preset.key}>
                      {preset.label}
                    </option>
                  ))}
                </select>
              </div>

              {formData.schedule === 'custom' && (
                <div className="grid grid-cols-5 gap-2">
                  <div>
                    <label className="block text-xs font-medium mb-1">
                      Dakika
                    </label>
                    <input
                      type="text"
                      value={formData.minute}
                      onChange={(e) =>
                        setFormData({ ...formData, minute: e.target.value })
                      }
                      className="w-full px-2 py-1 rounded border border-border bg-background text-center font-mono text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium mb-1">Saat</label>
                    <input
                      type="text"
                      value={formData.hour}
                      onChange={(e) =>
                        setFormData({ ...formData, hour: e.target.value })
                      }
                      className="w-full px-2 py-1 rounded border border-border bg-background text-center font-mono text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium mb-1">Gün</label>
                    <input
                      type="text"
                      value={formData.day}
                      onChange={(e) =>
                        setFormData({ ...formData, day: e.target.value })
                      }
                      className="w-full px-2 py-1 rounded border border-border bg-background text-center font-mono text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium mb-1">Ay</label>
                    <input
                      type="text"
                      value={formData.month}
                      onChange={(e) =>
                        setFormData({ ...formData, month: e.target.value })
                      }
                      className="w-full px-2 py-1 rounded border border-border bg-background text-center font-mono text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium mb-1">
                      Hafta Günü
                    </label>
                    <input
                      type="text"
                      value={formData.weekday}
                      onChange={(e) =>
                        setFormData({ ...formData, weekday: e.target.value })
                      }
                      className="w-full px-2 py-1 rounded border border-border bg-background text-center font-mono text-sm"
                    />
                  </div>
                </div>
              )}

              <div className="flex justify-end gap-2 pt-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setShowModal(false);
                    setEditingJob(null);
                  }}
                >
                  İptal
                </Button>
                <Button type="submit">
                  {editingJob ? 'Güncelle' : 'Oluştur'}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Run Output Modal */}
      {showOutputModal && selectedJob && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border border-border w-full max-w-2xl mx-4">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <div className="flex items-center gap-2">
                <Terminal className="w-5 h-5" />
                <h2 className="text-lg font-semibold">
                  {selectedJob.name} - Çalıştırma Çıktısı
                </h2>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowOutputModal(false)}
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
            <div className="p-4">
              <div className="flex items-center gap-2 mb-4">
                <span className="text-sm font-medium">Durum:</span>
                {runStatus === 'running' ? (
                  <span className="text-yellow-500 flex items-center gap-1">
                    <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse" />
                    Çalışıyor...
                  </span>
                ) : runStatus === 'success' ? (
                  <span className="text-green-500 flex items-center gap-1">
                    <CheckCircle className="w-4 h-4" />
                    Başarılı
                  </span>
                ) : (
                  <span className="text-red-500 flex items-center gap-1">
                    <XCircle className="w-4 h-4" />
                    Başarısız
                  </span>
                )}
              </div>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm max-h-96 overflow-auto">
                <pre className="whitespace-pre-wrap">{runOutput || 'Çıktı yok'}</pre>
              </div>
            </div>
            <div className="flex justify-end p-4 border-t border-border">
              <Button onClick={() => setShowOutputModal(false)}>Kapat</Button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}
