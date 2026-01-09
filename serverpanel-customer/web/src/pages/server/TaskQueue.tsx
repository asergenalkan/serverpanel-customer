import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { ListTodo, RefreshCw, Mail, Clock, AlertTriangle, CheckCircle, Trash2, RotateCcw, Users } from 'lucide-react';
import Layout from '@/components/Layout';
import api from '@/lib/api';

interface MailQueueItem {
  id: string;
  sender: string;
  recipient: string;
  size: string;
  time: string;
  status: string;
}

interface MailQueueItemDB {
  id: number;
  user_id: number;
  username: string;
  sender: string;
  recipient: string;
  subject: string;
  priority: number;
  retry_count: number;
  max_retries: number;
  scheduled_at: string;
  status: string;
  error_message: string;
  created_at: string;
}

interface MailStats {
  user_id: number;
  username: string;
  hourly_limit: number;
  daily_limit: number;
  sent_last_hour: number;
  sent_today: number;
  queued_count: number;
  hourly_remaining: number;
  daily_remaining: number;
}

interface MailQueueStats {
  total_queued: number;
  total_pending: number;
  total_processing: number;
  total_failed: number;
  total_sent_today: number;
  postfix_queue: number;
  user_stats: MailStats[];
}

interface CronJob {
  user: string;
  schedule: string;
  command: string;
  next_run: string;
}

interface QueueData {
  mail_queue: MailQueueItem[];
  mail_queue_count: number;
  cron_jobs: CronJob[];
  pending_tasks: number;
}

export default function TaskQueuePage() {
  const [data, setData] = useState<QueueData | null>(null);
  const [mailQueueStats, setMailQueueStats] = useState<MailQueueStats | null>(null);
  const [dbQueue, setDbQueue] = useState<MailQueueItemDB[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState<'mail' | 'queue' | 'stats' | 'cron'>('mail');

  useEffect(() => {
    fetchAllData();
    const interval = setInterval(fetchAllData, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchAllData = async () => {
    await Promise.all([fetchQueueData(), fetchMailQueueStats(), fetchDbQueue()]);
  };

  const fetchQueueData = async () => {
    try {
      const response = await api.get('/server/queue');
      if (response.data.success) {
        setData(response.data.data);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kuyruk bilgileri alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const fetchMailQueueStats = async () => {
    try {
      const response = await api.get('/mail-queue/stats');
      if (response.data.success) {
        setMailQueueStats(response.data.data);
      }
    } catch (err: any) {
      console.error('Mail queue stats error:', err);
    }
  };

  const fetchDbQueue = async () => {
    try {
      const response = await api.get('/mail-queue');
      if (response.data.success) {
        setDbQueue(response.data.data || []);
      }
    } catch (err: any) {
      console.error('DB queue error:', err);
    }
  };

  const flushMailQueue = async () => {
    try {
      await api.post('/server/queue/flush');
      fetchAllData();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kuyruk temizlenemedi');
    }
  };

  const deleteQueueItem = async (id: number) => {
    try {
      await api.delete(`/mail-queue/${id}`);
      fetchDbQueue();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Silme başarısız');
    }
  };

  const retryQueueItem = async (id: number) => {
    try {
      await api.post(`/mail-queue/${id}/retry`);
      fetchDbQueue();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Yeniden deneme başarısız');
    }
  };

  const clearQueue = async (status?: string) => {
    try {
      await api.post('/mail-queue/clear', { status });
      fetchDbQueue();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kuyruk temizlenemedi');
    }
  };

  return (
    <Layout>
      <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Task Queue</h1>
          <p className="text-muted-foreground">Mail kuyruğu ve zamanlanmış görevler</p>
        </div>
        <Button onClick={fetchQueueData} variant="outline" size="sm">
          <RefreshCw className="w-4 h-4 mr-2" />
          Yenile
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-blue-500/10 rounded-lg">
                <Mail className="w-6 h-6 text-blue-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{data?.mail_queue_count || 0}</p>
                <p className="text-sm text-muted-foreground">Mail Kuyruğunda</p>
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
                <p className="text-2xl font-bold">{data?.cron_jobs?.length || 0}</p>
                <p className="text-sm text-muted-foreground">Cron Job</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-orange-500/10 rounded-lg">
                <ListTodo className="w-6 h-6 text-orange-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{data?.pending_tasks || 0}</p>
                <p className="text-sm text-muted-foreground">Bekleyen Görev</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b">
        <button
          onClick={() => setActiveTab('mail')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'mail'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Mail className="w-4 h-4 inline mr-2" />
          Postfix Kuyruğu
        </button>
        <button
          onClick={() => setActiveTab('queue')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'queue'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <ListTodo className="w-4 h-4 inline mr-2" />
          Rate Limit Kuyruğu ({dbQueue.length})
        </button>
        <button
          onClick={() => setActiveTab('stats')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'stats'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Users className="w-4 h-4 inline mr-2" />
          Kullanıcı İstatistikleri
        </button>
        <button
          onClick={() => setActiveTab('cron')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'cron'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Clock className="w-4 h-4 inline mr-2" />
          Cron Jobs
        </button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-32">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : error ? (
        <div className="bg-destructive/10 text-destructive p-4 rounded-lg">{error}</div>
      ) : (
        <>
          {/* Mail Queue Tab */}
          {activeTab === 'mail' && (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <Mail className="w-5 h-5" />
                  Mail Kuyruğu
                </CardTitle>
                {(data?.mail_queue_count || 0) > 0 && (
                  <Button variant="destructive" size="sm" onClick={flushMailQueue}>
                    Kuyruğu Temizle
                  </Button>
                )}
              </CardHeader>
              <CardContent>
                {!data?.mail_queue || data.mail_queue.length === 0 ? (
                  <div className="text-center py-8">
                    <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
                    <p className="text-muted-foreground">Mail kuyruğu boş</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="border-b">
                          <th className="text-left py-3 px-4 font-medium">ID</th>
                          <th className="text-left py-3 px-4 font-medium">Gönderen</th>
                          <th className="text-left py-3 px-4 font-medium">Alıcı</th>
                          <th className="text-right py-3 px-4 font-medium">Boyut</th>
                          <th className="text-left py-3 px-4 font-medium">Zaman</th>
                          <th className="text-left py-3 px-4 font-medium">Durum</th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.mail_queue.map((item, index) => (
                          <tr key={index} className="border-b hover:bg-muted/50">
                            <td className="py-3 px-4 font-mono text-sm">{item.id}</td>
                            <td className="py-3 px-4 text-sm">{item.sender}</td>
                            <td className="py-3 px-4 text-sm">{item.recipient}</td>
                            <td className="py-3 px-4 text-right text-sm">{item.size}</td>
                            <td className="py-3 px-4 text-sm">{item.time}</td>
                            <td className="py-3 px-4">
                              {item.status === 'deferred' ? (
                                <span className="inline-flex items-center gap-1 text-yellow-500 text-sm">
                                  <AlertTriangle className="w-4 h-4" />
                                  Ertelendi
                                </span>
                              ) : (
                                <span className="text-sm">{item.status}</span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Rate Limit Queue Tab */}
          {activeTab === 'queue' && (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <ListTodo className="w-5 h-5" />
                  Rate Limit Kuyruğu
                </CardTitle>
                {dbQueue.length > 0 && (
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={() => clearQueue('failed')}>
                      Başarısızları Temizle
                    </Button>
                    <Button variant="destructive" size="sm" onClick={() => clearQueue()}>
                      Tümünü Temizle
                    </Button>
                  </div>
                )}
              </CardHeader>
              <CardContent>
                {dbQueue.length === 0 ? (
                  <div className="text-center py-8">
                    <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
                    <p className="text-muted-foreground">Rate limit kuyruğu boş</p>
                    <p className="text-xs text-muted-foreground mt-2">
                      Limit aşıldığında mailler buraya eklenir ve sonraki saat/gün otomatik gönderilir.
                    </p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="border-b">
                          <th className="text-left py-3 px-4 font-medium">Kullanıcı</th>
                          <th className="text-left py-3 px-4 font-medium">Gönderen</th>
                          <th className="text-left py-3 px-4 font-medium">Alıcı</th>
                          <th className="text-left py-3 px-4 font-medium">Konu</th>
                          <th className="text-left py-3 px-4 font-medium">Durum</th>
                          <th className="text-left py-3 px-4 font-medium">Zamanlandı</th>
                          <th className="text-right py-3 px-4 font-medium">İşlem</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dbQueue.map((item) => (
                          <tr key={item.id} className="border-b hover:bg-muted/50">
                            <td className="py-3 px-4 text-sm">{item.username}</td>
                            <td className="py-3 px-4 text-sm">{item.sender}</td>
                            <td className="py-3 px-4 text-sm">{item.recipient}</td>
                            <td className="py-3 px-4 text-sm max-w-xs truncate">{item.subject || '-'}</td>
                            <td className="py-3 px-4">
                              <span className={`inline-flex items-center gap-1 text-sm px-2 py-1 rounded ${
                                item.status === 'pending' ? 'bg-yellow-500/10 text-yellow-500' :
                                item.status === 'processing' ? 'bg-blue-500/10 text-blue-500' :
                                item.status === 'failed' ? 'bg-red-500/10 text-red-500' :
                                'bg-gray-500/10 text-gray-500'
                              }`}>
                                {item.status === 'pending' && <Clock className="w-3 h-3" />}
                                {item.status === 'processing' && <RefreshCw className="w-3 h-3 animate-spin" />}
                                {item.status === 'failed' && <AlertTriangle className="w-3 h-3" />}
                                {item.status}
                              </span>
                              {item.error_message && (
                                <p className="text-xs text-red-500 mt-1">{item.error_message}</p>
                              )}
                            </td>
                            <td className="py-3 px-4 text-sm">{item.scheduled_at || '-'}</td>
                            <td className="py-3 px-4 text-right">
                              <div className="flex justify-end gap-1">
                                {item.status === 'failed' && (
                                  <Button variant="ghost" size="sm" onClick={() => retryQueueItem(item.id)}>
                                    <RotateCcw className="w-4 h-4" />
                                  </Button>
                                )}
                                <Button variant="ghost" size="sm" onClick={() => deleteQueueItem(item.id)}>
                                  <Trash2 className="w-4 h-4 text-red-500" />
                                </Button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* User Stats Tab */}
          {activeTab === 'stats' && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Users className="w-5 h-5" />
                  Kullanıcı Mail İstatistikleri
                </CardTitle>
              </CardHeader>
              <CardContent>
                {!mailQueueStats?.user_stats || mailQueueStats.user_stats.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    Kullanıcı istatistiği bulunamadı.
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="border-b">
                          <th className="text-left py-3 px-4 font-medium">Kullanıcı</th>
                          <th className="text-center py-3 px-4 font-medium">Saatlik Limit</th>
                          <th className="text-center py-3 px-4 font-medium">Son 1 Saat</th>
                          <th className="text-center py-3 px-4 font-medium">Günlük Limit</th>
                          <th className="text-center py-3 px-4 font-medium">Bugün</th>
                          <th className="text-center py-3 px-4 font-medium">Kuyrukta</th>
                        </tr>
                      </thead>
                      <tbody>
                        {mailQueueStats.user_stats.map((stat) => (
                          <tr key={stat.user_id} className="border-b hover:bg-muted/50">
                            <td className="py-3 px-4 font-medium">{stat.username}</td>
                            <td className="py-3 px-4 text-center">{stat.hourly_limit}</td>
                            <td className="py-3 px-4 text-center">
                              <span className={stat.sent_last_hour >= stat.hourly_limit ? 'text-red-500 font-bold' : ''}>
                                {stat.sent_last_hour}
                              </span>
                              <span className="text-muted-foreground text-xs ml-1">
                                ({stat.hourly_remaining} kaldı)
                              </span>
                            </td>
                            <td className="py-3 px-4 text-center">{stat.daily_limit}</td>
                            <td className="py-3 px-4 text-center">
                              <span className={stat.sent_today >= stat.daily_limit ? 'text-red-500 font-bold' : ''}>
                                {stat.sent_today}
                              </span>
                              <span className="text-muted-foreground text-xs ml-1">
                                ({stat.daily_remaining} kaldı)
                              </span>
                            </td>
                            <td className="py-3 px-4 text-center">
                              {stat.queued_count > 0 ? (
                                <span className="text-yellow-500 font-bold">{stat.queued_count}</span>
                              ) : (
                                <span className="text-muted-foreground">0</span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Cron Jobs Tab */}
          {activeTab === 'cron' && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Clock className="w-5 h-5" />
                  Zamanlanmış Görevler (Cron)
                </CardTitle>
              </CardHeader>
              <CardContent>
                {!data?.cron_jobs || data.cron_jobs.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    Zamanlanmış görev bulunamadı.
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="border-b">
                          <th className="text-left py-3 px-4 font-medium">Kullanıcı</th>
                          <th className="text-left py-3 px-4 font-medium">Zamanlama</th>
                          <th className="text-left py-3 px-4 font-medium">Komut</th>
                          <th className="text-left py-3 px-4 font-medium">Sonraki Çalışma</th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.cron_jobs.map((job, index) => (
                          <tr key={index} className="border-b hover:bg-muted/50">
                            <td className="py-3 px-4 font-mono text-sm">{job.user}</td>
                            <td className="py-3 px-4 font-mono text-sm">{job.schedule}</td>
                            <td className="py-3 px-4 text-sm max-w-md truncate">
                              <code className="text-xs bg-muted px-2 py-1 rounded">{job.command}</code>
                            </td>
                            <td className="py-3 px-4 text-sm">{job.next_run}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
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
