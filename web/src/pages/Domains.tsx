import { useEffect, useState, useCallback } from 'react';
import { domainsAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import {
  Globe,
  Plus,
  Trash2,
  ExternalLink,
  Shield,
  ShieldOff,
  FolderOpen,
  X,
} from 'lucide-react';
import Layout from '@/components/Layout';

interface Domain {
  id: number;
  user_id: number;
  name: string;
  document_root: string;
  ssl_enabled: boolean;
  ssl_expiry: string | null;
  active: boolean;
  created_at: string;
}

export default function Domains() {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [newDomain, setNewDomain] = useState('');
  const [documentRoot, setDocumentRoot] = useState('');
  const [addingDomain, setAddingDomain] = useState(false);
  const [error, setError] = useState('');

  const fetchDomains = async () => {
    try {
      const response = await domainsAPI.list();
      if (response.data.success) {
        setDomains(response.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch domains:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDomains();
  }, []);

  // ESC key handler for modal
  const closeModal = useCallback(() => {
    setShowAddModal(false);
    setError('');
  }, []);

  useEffect(() => {
    if (!showAddModal) return;
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeModal();
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [showAddModal, closeModal]);

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setAddingDomain(true);

    try {
      const response = await domainsAPI.create({
        name: newDomain,
        document_root: documentRoot || undefined,
      });

      if (response.data.success) {
        setShowAddModal(false);
        setNewDomain('');
        setDocumentRoot('');
        fetchDomains();
      } else {
        setError(response.data.error || 'Domain eklenemedi');
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Domain eklenirken bir hata oluştu');
      }
    } finally {
      setAddingDomain(false);
    }
  };

  const handleDeleteDomain = async (id: number, name: string) => {
    if (!confirm(`"${name}" domaini silmek istediğinize emin misiniz?`)) {
      return;
    }

    try {
      const response = await domainsAPI.delete(id);
      if (response.data.success) {
        fetchDomains();
      }
    } catch (error) {
      console.error('Failed to delete domain:', error);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('tr-TR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Domainler</h1>
            <p className="text-muted-foreground">
              Website domainlerinizi yönetin
            </p>
          </div>
          <Button onClick={() => setShowAddModal(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Domain Ekle
          </Button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-blue-100">
                  <Globe className="w-5 h-5 text-blue-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{domains.length}</p>
                  <p className="text-sm text-muted-foreground">Toplam Domain</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-green-100">
                  <Shield className="w-5 h-5 text-green-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">
                    {domains.filter((d) => d.ssl_enabled).length}
                  </p>
                  <p className="text-sm text-muted-foreground">SSL Aktif</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-orange-100">
                  <ShieldOff className="w-5 h-5 text-orange-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">
                    {domains.filter((d) => !d.ssl_enabled).length}
                  </p>
                  <p className="text-sm text-muted-foreground">SSL Yok</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Domain List */}
        <Card>
          <CardHeader>
            <CardTitle>Domain Listesi</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
              </div>
            ) : domains.length === 0 ? (
              <div className="text-center py-8">
                <Globe className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
                <p className="text-muted-foreground">Henüz domain eklenmemiş</p>
                <Button
                  variant="outline"
                  className="mt-4"
                  onClick={() => setShowAddModal(true)}
                >
                  <Plus className="w-4 h-4 mr-2" />
                  İlk Domaini Ekle
                </Button>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-3 px-4 font-medium">Domain</th>
                      <th className="text-left py-3 px-4 font-medium">Document Root</th>
                      <th className="text-left py-3 px-4 font-medium">SSL</th>
                      <th className="text-left py-3 px-4 font-medium">Durum</th>
                      <th className="text-left py-3 px-4 font-medium">Oluşturulma</th>
                      <th className="text-right py-3 px-4 font-medium">İşlemler</th>
                    </tr>
                  </thead>
                  <tbody>
                    {domains.map((domain) => (
                      <tr key={domain.id} className="border-b hover:bg-muted/50">
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <Globe className="w-4 h-4 text-blue-500" />
                            <span className="font-medium">{domain.name}</span>
                            <a
                              href={`http://${domain.name}`}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-muted-foreground hover:text-blue-600"
                            >
                              <ExternalLink className="w-3 h-3" />
                            </a>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-1 text-sm text-muted-foreground">
                            <FolderOpen className="w-3 h-3" />
                            <span className="font-mono text-xs">
                              {domain.document_root || '/public_html'}
                            </span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          {domain.ssl_enabled ? (
                            <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700">
                              <Shield className="w-3 h-3" />
                              Aktif
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-700">
                              <ShieldOff className="w-3 h-3" />
                              Yok
                            </span>
                          )}
                        </td>
                        <td className="py-3 px-4">
                          {domain.active ? (
                            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700">
                              Aktif
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-700">
                              Pasif
                            </span>
                          )}
                        </td>
                        <td className="py-3 px-4 text-sm text-muted-foreground">
                          {formatDate(domain.created_at)}
                        </td>
                        <td className="py-3 px-4 text-right">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-red-500 hover:text-red-700 hover:bg-red-50"
                            onClick={() => handleDeleteDomain(domain.id, domain.name)}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Add Domain Modal */}
      {showAddModal && (
        <div 
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
          onClick={closeModal}
        >
          <Card className="w-full max-w-md mx-4" onClick={(e) => e.stopPropagation()}>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Yeni Domain Ekle</CardTitle>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowAddModal(false)}
              >
                <X className="w-4 h-4" />
              </Button>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleAddDomain} className="space-y-4">
                {error && (
                  <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm">
                    {error}
                  </div>
                )}

                <div className="space-y-2">
                  <label className="text-sm font-medium">Domain Adı *</label>
                  <Input
                    placeholder="ornek.com"
                    value={newDomain}
                    onChange={(e) => setNewDomain(e.target.value)}
                    required
                  />
                  <p className="text-xs text-muted-foreground">
                    www olmadan girin (örn: ornek.com)
                  </p>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">Document Root (Opsiyonel)</label>
                  <Input
                    placeholder="/home/user/public_html/ornek.com"
                    value={documentRoot}
                    onChange={(e) => setDocumentRoot(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Boş bırakırsanız otomatik oluşturulur
                  </p>
                </div>

                <div className="flex gap-3 pt-4">
                  <Button
                    type="button"
                    variant="outline"
                    className="flex-1"
                    onClick={() => setShowAddModal(false)}
                  >
                    İptal
                  </Button>
                  <Button type="submit" className="flex-1" isLoading={addingDomain}>
                    Domain Ekle
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      )}
    </Layout>
  );
}
