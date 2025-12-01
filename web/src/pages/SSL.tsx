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
  Clock,
  Calendar,
  Globe,
  Loader2,
  CheckCircle,
  AlertCircle,
} from 'lucide-react';
import Layout from '@/components/Layout';

interface SSLCertificate {
  id: number;
  domain_id: number;
  domain: string;
  issuer: string;
  status: 'active' | 'expired' | 'pending' | 'none';
  valid_from: string;
  valid_until: string;
  auto_renew: boolean;
  cert_path?: string;
  key_path?: string;
}

export default function SSL() {
  const [certificates, setCertificates] = useState<SSLCertificate[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

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
    setActionLoading(domainId);
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

  const handleRenew = async (domainId: number, domain: string) => {
    setActionLoading(domainId);
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

    setActionLoading(domainId);
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
        return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
      case 'expired':
        return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400';
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

  const getDaysRemaining = (validUntil: string) => {
    if (!validUntil) return null;
    const now = new Date();
    const expiry = new Date(validUntil);
    const diff = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
    return diff;
  };

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              SSL Sertifikaları
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">
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
              <p className="text-red-800 dark:text-red-200 font-medium">Hata</p>
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
              <p className="text-green-800 dark:text-green-200 font-medium">Başarılı</p>
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
              <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                <Shield className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900 dark:text-white">
                  Let's Encrypt SSL
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Tüm domainleriniz için ücretsiz SSL sertifikası alabilirsiniz. 
                  Sertifikalar 90 gün geçerlidir ve otomatik olarak yenilenir.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Certificates List */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
          </div>
        ) : certificates.length === 0 ? (
          <Card>
            <CardContent className="p-12 text-center">
              <Globe className="h-12 w-12 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                Domain Bulunamadı
              </h3>
              <p className="text-gray-600 dark:text-gray-400">
                SSL sertifikası almak için önce bir domain eklemeniz gerekiyor.
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {certificates.map((cert) => {
              const daysRemaining = getDaysRemaining(cert.valid_until);
              const isExpiringSoon = daysRemaining !== null && daysRemaining <= 30 && daysRemaining > 0;
              
              return (
                <Card key={cert.domain_id} className="overflow-hidden">
                  <CardContent className="p-0">
                    <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
                      <div className="flex items-center gap-4">
                        {getStatusIcon(cert.status)}
                        <div>
                          <h3 className="font-medium text-gray-900 dark:text-white">
                            {cert.domain}
                          </h3>
                          <div className="flex items-center gap-2 mt-1">
                            <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(cert.status)}`}>
                              {getStatusText(cert.status)}
                            </span>
                            {cert.issuer && (
                              <span className="text-xs text-gray-500 dark:text-gray-400">
                                {cert.issuer}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2">
                        {cert.status === 'none' ? (
                          <Button
                            onClick={() => handleIssue(cert.domain_id, cert.domain)}
                            disabled={actionLoading === cert.domain_id}
                            size="sm"
                          >
                            {actionLoading === cert.domain_id ? (
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
                              disabled={actionLoading === cert.domain_id}
                              variant="outline"
                              size="sm"
                            >
                              {actionLoading === cert.domain_id ? (
                                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                              ) : (
                                <RefreshCw className="h-4 w-4 mr-2" />
                              )}
                              Yenile
                            </Button>
                            <Button
                              onClick={() => handleRevoke(cert.domain_id, cert.domain)}
                              disabled={actionLoading === cert.domain_id}
                              variant="outline"
                              size="sm"
                              className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </>
                        )}
                      </div>
                    </div>
                    
                    {cert.status !== 'none' && (
                      <div className="p-4 bg-gray-50 dark:bg-gray-800/50 grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div className="flex items-center gap-2">
                          <Calendar className="h-4 w-4 text-gray-400" />
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Başlangıç</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {formatDate(cert.valid_from)}
                            </p>
                          </div>
                        </div>
                        
                        <div className="flex items-center gap-2">
                          <Clock className="h-4 w-4 text-gray-400" />
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Bitiş</p>
                            <p className={`text-sm font-medium ${
                              cert.status === 'expired' 
                                ? 'text-red-600 dark:text-red-400' 
                                : isExpiringSoon 
                                  ? 'text-yellow-600 dark:text-yellow-400'
                                  : 'text-gray-900 dark:text-white'
                            }`}>
                              {formatDate(cert.valid_until)}
                              {daysRemaining !== null && daysRemaining > 0 && (
                                <span className="text-xs ml-1">
                                  ({daysRemaining} gün kaldı)
                                </span>
                              )}
                            </p>
                          </div>
                        </div>
                        
                        <div className="flex items-center gap-2">
                          <ShieldCheck className="h-4 w-4 text-gray-400" />
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Otomatik Yenileme</p>
                            <p className="text-sm font-medium text-green-600 dark:text-green-400">
                              {cert.auto_renew ? 'Aktif' : 'Pasif'}
                            </p>
                          </div>
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </Layout>
  );
}
