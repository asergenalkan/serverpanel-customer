import { useEffect, useState } from 'react';
import { sslAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Card, CardContent } from '@/components/ui/Card';
import {
  Shield,
  ShieldCheck,
  ShieldAlert,
  ShieldX,
  RefreshCw,
  Trash2,
  Plus,
  Globe,
  Loader2,
  CheckCircle,
  AlertCircle,
} from 'lucide-react';
import Layout from '@/components/Layout';

interface SSLCertificate {
  id: number;
  domain_id: number;
  subdomain_id?: number;
  domain: string;
  domain_type: 'domain' | 'subdomain' | 'www' | 'mail';
  parent_domain?: string;
  issuer: string;
  status: 'active' | 'expired' | 'pending' | 'none' | 'error';
  status_detail?: string;
  valid_from: string;
  valid_until: string;
  auto_renew: boolean;
  cert_path?: string;
  key_path?: string;
}

export default function SSL() {
  const [certificates, setCertificates] = useState<SSLCertificate[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const fetchCertificates = async () => {
    try {
      const response = await sslAPI.list();
      if (response.data.success) {
        setCertificates(response.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch SSL certificates:', err);
      setError('SSL sertifikaları yüklenemedi');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCertificates();
  }, []);

  const handleIssue = async (domainId: number, domain: string) => {
    setActionLoading(`issue-${domainId}-${domain}`);
    setError('');
    setSuccess('');
    
    try {
      const response = await sslAPI.issue(domainId);
      if (response.data.success) {
        setSuccess(`${domain} için SSL sertifikası başarıyla oluşturuldu!`);
        fetchCertificates();
      } else {
        setError(response.data.error || 'SSL sertifikası oluşturulamadı');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'SSL sertifikası oluşturulurken hata oluştu');
    } finally {
      setActionLoading(null);
    }
  };

  const handleIssueFQDN = async (fqdn: string, domainId: number, domainType: string) => {
    setActionLoading(`issue-fqdn-${fqdn}`);
    setError('');
    setSuccess('');
    
    try {
      const response = await sslAPI.issueFQDN({ fqdn, domain_id: domainId, domain_type: domainType });
      if (response.data.success) {
        setSuccess(`${fqdn} için SSL sertifikası başarıyla oluşturuldu!`);
        fetchCertificates();
      } else {
        setError(response.data.error || 'SSL sertifikası oluşturulamadı');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'SSL sertifikası oluşturulurken hata oluştu');
    } finally {
      setActionLoading(null);
    }
  };

  const handleRenew = async (domainId: number, domain: string) => {
    setActionLoading(`renew-${domainId}-${domain}`);
    setError('');
    setSuccess('');
    
    try {
      const response = await sslAPI.renew(domainId);
      if (response.data.success) {
        setSuccess(`${domain} için SSL sertifikası yenilendi!`);
        fetchCertificates();
      } else {
        setError(response.data.error || 'SSL sertifikası yenilenemedi');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'SSL sertifikası yenilenirken hata oluştu');
    } finally {
      setActionLoading(null);
    }
  };

  const handleRevoke = async (domainId: number, domain: string) => {
    if (!confirm(`${domain} için SSL sertifikasını silmek istediğinize emin misiniz?`)) {
      return;
    }

    setActionLoading(`revoke-${domainId}-${domain}`);
    setError('');
    setSuccess('');
    
    try {
      const response = await sslAPI.revoke(domainId);
      if (response.data.success) {
        setSuccess(`${domain} için SSL sertifikası silindi`);
        fetchCertificates();
      } else {
        setError(response.data.error || 'SSL sertifikası silinemedi');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'SSL sertifikası silinirken hata oluştu');
    } finally {
      setActionLoading(null);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <ShieldCheck className="h-5 w-5 text-green-500" />;
      case 'expired':
        return <ShieldAlert className="h-5 w-5 text-red-500" />;
      case 'pending':
        return <Shield className="h-5 w-5 text-yellow-500" />;
      default:
        return <ShieldX className="h-5 w-5 text-gray-400" />;
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'active':
        return 'Aktif';
      case 'expired':
        return 'Süresi Dolmuş';
      case 'pending':
        return 'Beklemede';
      default:
        return 'SSL Yok';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'px-2.5 py-0.5 rounded-full text-xs font-semibold border text-white bg-emerald-600 border-emerald-700 dark:bg-emerald-600 dark:border-emerald-700';
      case 'expired':
        return 'px-2.5 py-0.5 rounded-full text-xs font-semibold border text-white bg-rose-600 border-rose-700 dark:bg-rose-600 dark:border-rose-700';
      case 'pending':
        return 'px-2.5 py-0.5 rounded-full text-xs font-semibold border text-white bg-amber-600 border-amber-700 dark:bg-amber-600 dark:border-amber-700';
      default:
        return 'px-2.5 py-0.5 rounded-full text-xs font-semibold border text-white bg-gray-600 border-gray-700 dark:bg-gray-600 dark:border-gray-700';
    }
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString('tr-TR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };


  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-foreground">
              SSL Sertifikaları
            </h1>
            <p className="text-muted-foreground text-sm">
              Let's Encrypt ile ücretsiz SSL sertifikası yönetimi
            </p>
          </div>
          <Button
            onClick={() => fetchCertificates()}
            variant="outline"
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Yenile
          </Button>
        </div>

        {/* Alerts */}
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 flex items-start gap-3">
            <AlertCircle className="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-red-700 dark:text-red-200 font-medium">Hata</p>
              <p className="text-red-600 dark:text-red-300 text-sm">{error}</p>
            </div>
            <button
              onClick={() => setError('')}
              className="ml-auto text-red-500 hover:text-red-700"
            >
              ×
            </button>
          </div>
        )}

        {success && (
          <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-green-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-green-700 dark:text-green-200 font-medium">Başarılı</p>
              <p className="text-green-600 dark:text-green-300 text-sm">{success}</p>
            </div>
            <button
              onClick={() => setSuccess('')}
              className="ml-auto text-green-500 hover:text-green-700"
            >
              ×
            </button>
          </div>
        )}

        {/* Info Card */}
        <Card>
          <CardContent className="p-4">
            <div className="flex items-start gap-4">
              <div className="p-3 bg-primary/10 rounded-lg">
                <Shield className="h-6 w-6 text-primary" />
              </div>
              <div>
                <h3 className="font-medium text-foreground">
                  Let's Encrypt SSL
                </h3>
                <p className="text-sm text-muted-foreground mt-1">
                  Tüm domainleriniz için ücretsiz SSL sertifikası alabilirsiniz. 
                  Sertifikalar 90 gün geçerlidir ve otomatik olarak yenilenir.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Search and Filter */}
        <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
          <div className="relative w-full sm:w-64">
            <input
              type="text"
              placeholder="Domain ara..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-4 pr-4 py-2 text-sm border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
          </div>
          <div className="flex gap-2">
            {['all', 'active', 'none', 'expired'].map((status) => (
              <button
                key={status}
                onClick={() => setFilterStatus(status)}
                className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${
                  filterStatus === status
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'bg-background border-border hover:bg-muted'
                }`}
              >
                {status === 'all' ? 'Tümü' : status === 'active' ? 'Aktif' : status === 'none' ? 'SSL Yok' : 'Süresi Dolmuş'}
              </button>
            ))}
          </div>
        </div>

        {/* Certificates Table */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
          </div>
        ) : certificates.length === 0 ? (
          <Card>
            <CardContent className="p-12 text-center">
              <Globe className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium text-foreground mb-2">
                Domain Bulunamadı
              </h3>
              <p className="text-muted-foreground">
                SSL sertifikası almak için önce bir domain eklemeniz gerekiyor.
              </p>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="p-0">
              {/* Stats */}
              <div className="p-4 border-b border-border bg-muted/30">
                <p className="text-sm text-muted-foreground">
                  Toplam <span className="font-medium text-foreground">{certificates.length}</span> domain gösteriliyor
                  {filterStatus !== 'all' && ` (${filterStatus === 'active' ? 'Aktif' : filterStatus === 'none' ? 'SSL Yok' : 'Süresi Dolmuş'} filtresi)`}
                </p>
              </div>

              {/* Table */}
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border bg-muted/50">
                      <th className="text-left p-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                        Alan Adı
                      </th>
                      <th className="text-left p-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                        Sertifika Durumu
                      </th>
                      <th className="text-right p-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                        İşlemler
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {certificates
                      .filter(cert => {
                        const matchesSearch = cert.domain.toLowerCase().includes(searchQuery.toLowerCase());
                        const matchesFilter = filterStatus === 'all' || cert.status === filterStatus;
                        return matchesSearch && matchesFilter;
                      })
                      .map((cert, index) => {
                        const loadingKey = `issue-${cert.domain_id}-${cert.domain}`;
                        const renewKey = `renew-${cert.domain_id}-${cert.domain}`;
                        const revokeKey = `revoke-${cert.domain_id}-${cert.domain}`;
                        const isLoading = actionLoading === loadingKey || actionLoading === renewKey || actionLoading === revokeKey;

                        return (
                          <tr key={`${cert.domain}-${index}`} className="hover:bg-muted/30 transition-colors">
                            {/* Domain Name */}
                            <td className="p-4">
                              <div className="flex items-center gap-3">
                                {getStatusIcon(cert.status)}
                                <div>
                                  <p className="font-medium text-foreground">{cert.domain}</p>
                                  {cert.domain_type !== 'domain' && (
                                    <span className={`text-xs px-1.5 py-0.5 rounded ${
                                      cert.domain_type === 'subdomain' 
                                        ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                                        : cert.domain_type === 'www'
                                          ? 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
                                          : cert.domain_type === 'mail'
                                            ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                                            : cert.domain_type === 'webmail'
                                              ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
                                              : cert.domain_type === 'ftp'
                                                ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                                                : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
                                    }`}>
                                      {cert.domain_type === 'subdomain' ? 'Subdomain' 
                                        : cert.domain_type === 'www' ? 'WWW' 
                                        : cert.domain_type === 'mail' ? 'Mail'
                                        : cert.domain_type === 'webmail' ? 'Webmail'
                                        : cert.domain_type === 'ftp' ? 'FTP'
                                        : cert.domain_type}
                                    </span>
                                  )}
                                </div>
                              </div>
                            </td>

                            {/* Certificate Status */}
                            <td className="p-4">
                              <div>
                                <span className={getStatusColor(cert.status)}>
                                  {getStatusText(cert.status)}
                                </span>
                                {cert.status_detail && (
                                  <p className="text-xs text-muted-foreground mt-1">
                                    {cert.status_detail}
                                  </p>
                                )}
                                {cert.status === 'active' && cert.valid_until && (
                                  <p className="text-xs text-muted-foreground mt-1">
                                    Bitiş: {formatDate(cert.valid_until)}
                                  </p>
                                )}
                              </div>
                            </td>

                            {/* Actions */}
                            <td className="p-4 text-right">
                              <div className="flex items-center justify-end gap-2">
                                {cert.domain_type === 'domain' ? (
                                  // Ana domain için butonlar
                                  cert.status === 'none' ? (
                                    <Button
                                      onClick={() => handleIssue(cert.domain_id, cert.domain)}
                                      disabled={isLoading}
                                      size="sm"
                                    >
                                      {actionLoading === loadingKey ? (
                                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                                      ) : (
                                        <Plus className="h-4 w-4 mr-2" />
                                      )}
                                      SSL Al
                                    </Button>
                                  ) : (
                                    <>
                                      <Button
                                        onClick={() => handleRenew(cert.domain_id, cert.domain)}
                                        disabled={isLoading}
                                        variant="outline"
                                        size="sm"
                                      >
                                        {actionLoading === renewKey ? (
                                          <Loader2 className="h-4 w-4 animate-spin" />
                                        ) : (
                                          <RefreshCw className="h-4 w-4" />
                                        )}
                                      </Button>
                                      <Button
                                        onClick={() => handleRevoke(cert.domain_id, cert.domain)}
                                        disabled={isLoading}
                                        variant="outline"
                                        size="sm"
                                        className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                                      >
                                        <Trash2 className="h-4 w-4" />
                                      </Button>
                                    </>
                                  )
                                ) : (
                                  // Subdomain, www, mail için butonlar
                                  cert.status === 'none' && (
                                    <Button
                                      onClick={() => handleIssueFQDN(cert.domain, cert.domain_id, cert.domain_type)}
                                      disabled={actionLoading === `issue-fqdn-${cert.domain}`}
                                      size="sm"
                                      variant="outline"
                                    >
                                      {actionLoading === `issue-fqdn-${cert.domain}` ? (
                                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                                      ) : (
                                        <Plus className="h-4 w-4 mr-2" />
                                      )}
                                      SSL Al
                                    </Button>
                                  )
                                )}
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </Layout>
  );
}
