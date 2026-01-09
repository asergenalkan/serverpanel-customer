import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { FileText, RefreshCw, ChevronLeft, ChevronRight } from 'lucide-react';
import Layout from '@/components/Layout';
import api from '@/lib/api';

interface DailyLogEntry {
  user: string;
  domain: string;
  cpu_percent: number;
  mem_percent: number;
  db_processes: number;
}

export default function DailyLogPage() {
  const [logs, setLogs] = useState<DailyLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);

  useEffect(() => {
    fetchDailyLog();
  }, [date]);

  const fetchDailyLog = async () => {
    setLoading(true);
    try {
      const response = await api.get(`/server/daily-log?date=${date}`);
      if (response.data.success) {
        setLogs(response.data.data || []);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Günlük veriler alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const changeDate = (days: number) => {
    const newDate = new Date(date);
    newDate.setDate(newDate.getDate() + days);
    if (newDate <= new Date()) {
      setDate(newDate.toISOString().split('T')[0]);
    }
  };

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    return d.toLocaleDateString('tr-TR', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Günlük İşlem Günlüğü</h1>
          <p className="text-muted-foreground">Kullanıcı bazlı kaynak kullanımı</p>
        </div>
        <Button onClick={fetchDailyLog} variant="outline" size="sm">
          <RefreshCw className="w-4 h-4 mr-2" />
          Yenile
        </Button>
      </div>

      {/* Date Navigation */}
      <Card>
        <CardContent className="py-4">
          <div className="flex items-center justify-center gap-4">
            <Button variant="outline" size="icon" onClick={() => changeDate(-1)}>
              <ChevronLeft className="w-4 h-4" />
            </Button>
            <div className="text-center min-w-[200px]">
              <p className="font-semibold">{formatDate(date)}</p>
              <p className="text-xs text-muted-foreground">
                Bu rakamlar bugün saat 00:00'dan itibaren ortalamalardır.
              </p>
            </div>
            <Button
              variant="outline"
              size="icon"
              onClick={() => changeDate(1)}
              disabled={date === new Date().toISOString().split('T')[0]}
            >
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Info Note */}
      <div className="bg-blue-500/10 text-blue-600 dark:text-blue-400 p-4 rounded-lg text-sm">
        <p>
          <strong>Not:</strong> Bu sayfa her kullanıcının CPU ve bellek kullanımını gösterir. 
          Değerler, kullanıcıya ait tüm işlemlerin (PHP-FPM, web istekleri, cron jobs vb.) 
          toplam kaynak tüketimini yansıtır.
        </p>
      </div>

      {/* Logs Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="w-5 h-5" />
            Kullanıcı Kaynak Kullanımı
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : error ? (
            <div className="text-center text-muted-foreground py-8">{error}</div>
          ) : logs.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              Bu tarih için kayıt bulunamadı.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-3 px-4 font-medium">Kullanıcı</th>
                    <th className="text-left py-3 px-4 font-medium">Domain</th>
                    <th className="text-right py-3 px-4 font-medium">% CPU</th>
                    <th className="text-right py-3 px-4 font-medium">% MEM</th>
                    <th className="text-right py-3 px-4 font-medium">DB Processes</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log, index) => (
                    <tr key={index} className="border-b hover:bg-muted/50">
                      <td className="py-3 px-4 font-mono text-sm">{log.user}</td>
                      <td className="py-3 px-4 text-blue-500">{log.domain}</td>
                      <td className="py-3 px-4 text-right">
                        <span
                          className={`font-mono ${
                            log.cpu_percent > 5
                              ? 'text-red-500'
                              : log.cpu_percent > 1
                              ? 'text-yellow-500'
                              : ''
                          }`}
                        >
                          {log.cpu_percent.toFixed(2)}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right">
                        <span
                          className={`font-mono ${
                            log.mem_percent > 5
                              ? 'text-red-500'
                              : log.mem_percent > 1
                              ? 'text-yellow-500'
                              : ''
                          }`}
                        >
                          {log.mem_percent.toFixed(2)}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right font-mono">{log.db_processes.toFixed(1)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
    </Layout>
  );
}
