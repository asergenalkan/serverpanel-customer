import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import Layout from '../components/Layout';
import { Card, CardContent } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { domainsAPI, subdomainsAPI } from '../lib/api';
import {
  Globe,
  Plus,
  Trash2,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  X,
  FolderOpen,
  Shield,
  ExternalLink,
  Layers,
  ArrowRight,
  Search,
} from 'lucide-react';

interface DomainData {
  id: number;
  user_id: number;
  username?: string;
  name: string;
  domain_type: string;
  parent_domain?: string;
  document_root: string;
  php_version: string;
  ssl_enabled: boolean;
  ssl_expiry?: string;
  active: boolean;
  created_at: string;
  subdomain_count: number;
}

interface SubdomainData {
  id: number;
  user_id: number;
  domain_id: number;
  domain_name?: string;
  name: string;
  full_name: string;
  document_root: string;
  redirect_url?: string;
  redirect_type?: string;
  active: boolean;
  created_at: string;
}

interface UserLimits {
  max_domains: number;
  current_domains: number;
  max_subdomains: number;
  current_subdomains: number;
}

export default function DomainManagerPage() {
  const { user } = useAuth();
  const [domains, setDomains] = useState<DomainData[]>([]);
  const [subdomains, setSubdomains] = useState<SubdomainData[]>([]);
  const [limits, setLimits] = useState<UserLimits | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Modal states
  const [showAddDomainModal, setShowAddDomainModal] = useState(false);
  const [showAddSubdomainModal, setShowAddSubdomainModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'domain' | 'subdomain'; id: number; name: string; documentRoot?: string } | null>(null);
  const [deleteFiles, setDeleteFiles] = useState(false);

  // Form states
  const [domainForm, setDomainForm] = useState({ name: '', document_root: '' });
  const [subdomainForm, setSubdomainForm] = useState({ 
    domain_id: 0, 
    name: '', 
    document_root: '', 
    redirect_url: '', 
    redirect_type: '' 
  });

  // Active tab
  const [activeTab, setActiveTab] = useState<'domains' | 'subdomains'>('domains');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [domainsRes, subdomainsRes, limitsRes] = await Promise.all([
        domainsAPI.list(),
        subdomainsAPI.list(),
        domainsAPI.getLimits(),
      ]);
      setDomains(domainsRes.data.data || []);
      setSubdomains(subdomainsRes.data.data || []);
      setLimits(limitsRes.data.data);
    } catch (error) {
      console.error('Failed to fetch data:', error);
      setMessage({ type: 'error', text: 'Veriler yüklenemedi' });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateDomain = async () => {
    if (!domainForm.name.trim()) {
      setMessage({ type: 'error', text: 'Domain adı gerekli' });
      return;
    }

    try {
      setActionLoading(-1);
      await domainsAPI.create({
        name: domainForm.name.toLowerCase().trim(),
        domain_type: 'addon',
        document_root: domainForm.document_root || undefined,
      });
      setMessage({ type: 'success', text: 'Domain eklendi' });
      setShowAddDomainModal(false);
      setDomainForm({ name: '', document_root: '' });
      await fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Domain eklenemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleCreateSubdomain = async () => {
    if (!subdomainForm.name.trim() || !subdomainForm.domain_id) {
      setMessage({ type: 'error', text: 'Subdomain adı ve domain seçimi gerekli' });
      return;
    }

    try {
      setActionLoading(-1);
      await subdomainsAPI.create({
        domain_id: subdomainForm.domain_id,
        name: subdomainForm.name.toLowerCase().trim(),
        document_root: subdomainForm.document_root || undefined,
        redirect_url: subdomainForm.redirect_url || undefined,
        redirect_type: subdomainForm.redirect_type || undefined,
      });
      setMessage({ type: 'success', text: 'Subdomain eklendi' });
      setShowAddSubdomainModal(false);
      setSubdomainForm({ domain_id: 0, name: '', document_root: '', redirect_url: '', redirect_type: '' });
      await fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Subdomain eklenemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;

    try {
      setActionLoading(deleteTarget.id);
      if (deleteTarget.type === 'domain') {
        await domainsAPI.delete(deleteTarget.id, deleteFiles);
      } else {
        await subdomainsAPI.delete(deleteTarget.id, deleteFiles);
      }
      setMessage({ type: 'success', text: `${deleteTarget.type === 'domain' ? 'Domain' : 'Subdomain'} silindi${deleteFiles ? ' (dosyalar dahil)' : ''}` });
      setShowDeleteModal(false);
      setDeleteTarget(null);
      setDeleteFiles(false);
      await fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Silme işlemi başarısız' });
    } finally {
      setActionLoading(null);
    }
  };

  const openDeleteModal = (type: 'domain' | 'subdomain', id: number, name: string, documentRoot?: string) => {
    setDeleteTarget({ type, id, name, documentRoot });
    setDeleteFiles(false);
    setShowDeleteModal(true);
  };

  const getDomainTypeLabel = (type: string) => {
    switch (type) {
      case 'primary': return 'Ana Domain';
      case 'addon': return 'Ek Domain';
      case 'alias': return 'Alias';
      default: return type;
    }
  };

  const getDomainTypeBadge = (type: string) => {
    switch (type) {
      case 'primary':
        return 'bg-blue-500/20 text-blue-600 dark:text-blue-400 border border-blue-500/30';
      case 'addon':
        return 'bg-green-500/20 text-green-600 dark:text-green-400 border border-green-500/30';
      case 'alias':
        return 'bg-purple-500/20 text-purple-600 dark:text-purple-400 border border-purple-500/30';
      default:
        return 'bg-gray-500/20 text-gray-600 dark:text-gray-400 border border-gray-500/30';
    }
  };

  // Filter data based on search
  const filteredDomains = domains.filter(d => 
    d.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    d.document_root.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const filteredSubdomains = subdomains.filter(s =>
    s.full_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    s.document_root.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold">Domain Yönetimi</h1>
            <p className="text-muted-foreground text-sm">
              Domain ve subdomain'lerinizi yönetin
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={fetchData} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Yenile
            </Button>
          </div>
        </div>

        {/* Limits Card */}
        {limits && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-blue-500/10">
                      <Globe className="w-5 h-5 text-blue-500" />
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Domain Kullanımı</p>
                      <p className="text-lg font-semibold">{limits.current_domains} / {limits.max_domains}</p>
                    </div>
                  </div>
                  <div className="w-24 h-2 bg-muted rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-blue-500 transition-all"
                      style={{ width: `${Math.min((limits.current_domains / limits.max_domains) * 100, 100)}%` }}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-green-500/10">
                      <Layers className="w-5 h-5 text-green-500" />
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Subdomain Kullanımı</p>
                      <p className="text-lg font-semibold">{limits.current_subdomains} / {limits.max_subdomains}</p>
                    </div>
                  </div>
                  <div className="w-24 h-2 bg-muted rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-green-500 transition-all"
                      style={{ width: `${Math.min((limits.current_subdomains / limits.max_subdomains) * 100, 100)}%` }}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Message */}
        {message && (
          <div className={`p-4 rounded-lg flex items-center gap-2 border text-white ${
            message.type === 'success' 
              ? 'bg-emerald-600 border-emerald-700' 
              : 'bg-rose-600 border-rose-700'
          }`}>
            {message.type === 'success' ? <CheckCircle className="w-5 h-5" /> : <AlertCircle className="w-5 h-5" />}
            {message.text}
            <button onClick={() => setMessage(null)} className="ml-auto hover:opacity-80">×</button>
          </div>
        )}

        {/* Tabs */}
        <div className="border-b border-border">
          <div className="flex gap-4">
            <button
              onClick={() => setActiveTab('domains')}
              className={`pb-3 px-1 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'domains'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Globe className="w-4 h-4 inline mr-2" />
              Domain'ler ({domains.length})
            </button>
            <button
              onClick={() => setActiveTab('subdomains')}
              className={`pb-3 px-1 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'subdomains'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Layers className="w-4 h-4 inline mr-2" />
              Subdomain'ler ({subdomains.length})
            </button>
          </div>
        </div>

        {/* Search & Add */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Ara..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-4 py-2 text-sm border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
          </div>
          <Button onClick={() => activeTab === 'domains' ? setShowAddDomainModal(true) : setShowAddSubdomainModal(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {activeTab === 'domains' ? 'Domain Ekle' : 'Subdomain Ekle'}
          </Button>
        </div>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <RefreshCw className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : activeTab === 'domains' ? (
          /* Domains List */
          <Card>
            <CardContent className="p-0">
              {filteredDomains.length === 0 ? (
                <div className="p-12 text-center">
                  <Globe className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
                  <h3 className="font-medium mb-1">Domain Bulunamadı</h3>
                  <p className="text-sm text-muted-foreground mb-4">Henüz domain eklenmemiş.</p>
                  <Button onClick={() => setShowAddDomainModal(true)}>
                    <Plus className="w-4 h-4 mr-2" />
                    İlk Domain'i Ekle
                  </Button>
                </div>
              ) : (
                <div className="divide-y divide-border">
                  {filteredDomains.map((domain) => (
                    <div key={domain.id} className="p-4 hover:bg-muted/50 transition-colors">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4">
                          <div className="p-2 rounded-lg bg-primary/10">
                            <Globe className="w-5 h-5 text-primary" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <a 
                                href={`http://${domain.name}`} 
                                target="_blank" 
                                rel="noopener noreferrer"
                                className="font-medium hover:text-primary transition-colors flex items-center gap-1"
                              >
                                {domain.name}
                                <ExternalLink className="w-3 h-3" />
                              </a>
                              <span className={`px-2 py-0.5 text-xs rounded-full ${getDomainTypeBadge(domain.domain_type)}`}>
                                {getDomainTypeLabel(domain.domain_type)}
                              </span>
                              {domain.ssl_enabled && (
                                <span className="px-2 py-0.5 text-xs rounded-full bg-green-500/20 text-green-600 dark:text-green-400 border border-green-500/30">
                                  <Shield className="w-3 h-3 inline mr-1" />
                                  SSL
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-4 mt-1 text-sm text-muted-foreground">
                              <span className="flex items-center gap-1">
                                <FolderOpen className="w-3 h-3" />
                                {domain.document_root}
                              </span>
                              {domain.subdomain_count > 0 && (
                                <span className="flex items-center gap-1">
                                  <Layers className="w-3 h-3" />
                                  {domain.subdomain_count} subdomain
                                </span>
                              )}
                              {user?.role === 'admin' && domain.username && (
                                <span className="text-xs">({domain.username})</span>
                              )}
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          {domain.domain_type !== 'primary' && (
                            <Button 
                              variant="ghost" 
                              size="sm"
                              onClick={() => openDeleteModal('domain', domain.id, domain.name, domain.document_root)}
                              disabled={actionLoading === domain.id}
                            >
                              <Trash2 className="w-4 h-4 text-red-500" />
                            </Button>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        ) : (
          /* Subdomains List */
          <Card>
            <CardContent className="p-0">
              {filteredSubdomains.length === 0 ? (
                <div className="p-12 text-center">
                  <Layers className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
                  <h3 className="font-medium mb-1">Subdomain Bulunamadı</h3>
                  <p className="text-sm text-muted-foreground mb-4">Henüz subdomain eklenmemiş.</p>
                  <Button onClick={() => setShowAddSubdomainModal(true)}>
                    <Plus className="w-4 h-4 mr-2" />
                    İlk Subdomain'i Ekle
                  </Button>
                </div>
              ) : (
                <div className="divide-y divide-border">
                  {filteredSubdomains.map((subdomain) => (
                    <div key={subdomain.id} className="p-4 hover:bg-muted/50 transition-colors">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4">
                          <div className="p-2 rounded-lg bg-green-500/10">
                            <Layers className="w-5 h-5 text-green-500" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <a 
                                href={`http://${subdomain.full_name}`} 
                                target="_blank" 
                                rel="noopener noreferrer"
                                className="font-medium hover:text-primary transition-colors flex items-center gap-1"
                              >
                                {subdomain.full_name}
                                <ExternalLink className="w-3 h-3" />
                              </a>
                              {subdomain.redirect_url && (
                                <span className="px-2 py-0.5 text-xs rounded-full bg-orange-500/20 text-orange-600 dark:text-orange-400 border border-orange-500/30">
                                  <ArrowRight className="w-3 h-3 inline mr-1" />
                                  Yönlendirme
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-4 mt-1 text-sm text-muted-foreground">
                              {subdomain.redirect_url ? (
                                <span className="flex items-center gap-1">
                                  <ArrowRight className="w-3 h-3" />
                                  {subdomain.redirect_url}
                                </span>
                              ) : (
                                <span className="flex items-center gap-1">
                                  <FolderOpen className="w-3 h-3" />
                                  {subdomain.document_root}
                                </span>
                              )}
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Button 
                            variant="ghost" 
                            size="sm"
                            onClick={() => openDeleteModal('subdomain', subdomain.id, subdomain.full_name, subdomain.document_root)}
                            disabled={actionLoading === subdomain.id}
                          >
                            <Trash2 className="w-4 h-4 text-red-500" />
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* Add Domain Modal */}
        {showAddDomainModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h2 className="text-lg font-semibold">Yeni Domain Ekle</h2>
                <button onClick={() => setShowAddDomainModal(false)} className="p-1 hover:bg-muted rounded">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Domain Adı *</label>
                  <input
                    type="text"
                    value={domainForm.name}
                    onChange={(e) => setDomainForm({ ...domainForm, name: e.target.value })}
                    placeholder="ornek.com"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Domain adını www olmadan girin
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Document Root (Opsiyonel)</label>
                  <input
                    type="text"
                    value={domainForm.document_root}
                    onChange={(e) => setDomainForm({ ...domainForm, document_root: e.target.value })}
                    placeholder="/home/kullanici/public_html/ornek.com"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Boş bırakırsanız otomatik oluşturulur
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button variant="outline" onClick={() => setShowAddDomainModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleCreateDomain} disabled={actionLoading !== null}>
                  {actionLoading !== null && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                  Ekle
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Add Subdomain Modal */}
        {showAddSubdomainModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h2 className="text-lg font-semibold">Yeni Subdomain Ekle</h2>
                <button onClick={() => setShowAddSubdomainModal(false)} className="p-1 hover:bg-muted rounded">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Domain Seçin *</label>
                  <select
                    value={subdomainForm.domain_id}
                    onChange={(e) => setSubdomainForm({ ...subdomainForm, domain_id: parseInt(e.target.value) })}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                  >
                    <option value={0}>Domain seçin...</option>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>{d.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Subdomain Adı *</label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={subdomainForm.name}
                      onChange={(e) => setSubdomainForm({ ...subdomainForm, name: e.target.value })}
                      placeholder="blog"
                      className="flex-1 px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                    <span className="text-muted-foreground">
                      .{domains.find(d => d.id === subdomainForm.domain_id)?.name || 'domain.com'}
                    </span>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Yönlendirme URL (Opsiyonel)</label>
                  <input
                    type="text"
                    value={subdomainForm.redirect_url}
                    onChange={(e) => setSubdomainForm({ ...subdomainForm, redirect_url: e.target.value })}
                    placeholder="https://hedef-site.com"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Yönlendirme yapılacaksa URL girin, yoksa boş bırakın
                  </p>
                </div>
                {subdomainForm.redirect_url && (
                  <div>
                    <label className="block text-sm font-medium mb-1">Yönlendirme Tipi</label>
                    <select
                      value={subdomainForm.redirect_type}
                      onChange={(e) => setSubdomainForm({ ...subdomainForm, redirect_type: e.target.value })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    >
                      <option value="301">301 - Kalıcı Yönlendirme</option>
                      <option value="302">302 - Geçici Yönlendirme</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button variant="outline" onClick={() => setShowAddSubdomainModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleCreateSubdomain} disabled={actionLoading !== null}>
                  {actionLoading !== null && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                  Ekle
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Delete Confirmation Modal */}
        {showDeleteModal && deleteTarget && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="p-6">
                <div className="text-center">
                  <div className="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center mx-auto mb-4">
                    <Trash2 className="w-6 h-6 text-red-600" />
                  </div>
                  <h3 className="text-lg font-semibold mb-2">
                    {deleteTarget.type === 'domain' ? 'Domain' : 'Subdomain'} Sil
                  </h3>
                  <p className="text-muted-foreground mb-4">
                    <strong>{deleteTarget.name}</strong> {deleteTarget.type === 'domain' ? "domain'ini" : "subdomain'ini"} silmek istediğinizden emin misiniz?
                    {deleteTarget.type === 'domain' && ' Tüm subdomain\'ler de silinecektir.'}
                  </p>
                </div>

                {/* Delete files option */}
                {deleteTarget.type !== 'domain' || (deleteTarget.type === 'domain' && domains.find(d => d.id === deleteTarget.id)?.domain_type !== 'primary') ? (
                  <div className="bg-muted/50 rounded-lg p-4 mb-4">
                    <label className="flex items-start gap-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={deleteFiles}
                        onChange={(e) => setDeleteFiles(e.target.checked)}
                        className="mt-1 w-4 h-4 rounded border-border text-primary focus:ring-primary"
                      />
                      <div>
                        <span className="font-medium text-foreground">Dosyaları da sil</span>
                        <p className="text-sm text-muted-foreground mt-1">
                          {deleteTarget.documentRoot || 'İlgili dizin'} klasörü ve içindeki tüm dosyalar kalıcı olarak silinecektir.
                        </p>
                      </div>
                    </label>
                  </div>
                ) : null}

                {deleteFiles && (
                  <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 mb-4">
                    <p className="text-sm text-red-600 dark:text-red-400 flex items-center gap-2">
                      <AlertCircle className="w-4 h-4 flex-shrink-0" />
                      <span>Dikkat: Dosyalar geri alınamaz şekilde silinecektir!</span>
                    </p>
                  </div>
                )}

                <div className="flex justify-center gap-2">
                  <Button variant="outline" onClick={() => { setShowDeleteModal(false); setDeleteTarget(null); setDeleteFiles(false); }}>
                    İptal
                  </Button>
                  <Button variant="destructive" onClick={handleDelete} disabled={actionLoading !== null}>
                    {actionLoading !== null && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                    {deleteFiles ? 'Dosyalarla Birlikte Sil' : 'Sil'}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
