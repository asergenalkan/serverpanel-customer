import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import Layout from '../components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { packagesAPI } from '../lib/api';
import {
  Package,
  Plus,
  Pencil,
  Trash2,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  X,
  HardDrive,
  Globe,
  Database,
  Mail,
  Server,
  Users,
  Cpu,
  Upload,
  Clock,
} from 'lucide-react';

interface PackageData {
  id: number;
  name: string;
  disk_quota: number;
  bandwidth_quota: number;
  max_domains: number;
  max_databases: number;
  max_emails: number;
  max_ftp: number;
  max_php_memory: string;
  max_php_upload: string;
  max_php_execution_time: number;
  max_emails_per_hour: number;
  max_emails_per_day: number;
  created_at: string;
  user_count: number;
}

const defaultFormData = {
  name: '',
  disk_quota: 1024,
  bandwidth_quota: 10240,
  max_domains: 1,
  max_databases: 1,
  max_emails: 5,
  max_ftp: 1,
  max_php_memory: '256M',
  max_php_upload: '64M',
  max_php_execution_time: 300,
  max_emails_per_hour: 100,
  max_emails_per_day: 500,
};

export default function PackagesPage() {
  const { user } = useAuth();
  const [packages, setPackages] = useState<PackageData[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Modal states
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [editingPackage, setEditingPackage] = useState<PackageData | null>(null);
  const [deletingPackage, setDeletingPackage] = useState<PackageData | null>(null);

  // Form state
  const [formData, setFormData] = useState(defaultFormData);

  useEffect(() => {
    fetchPackages();
  }, []);

  const fetchPackages = async () => {
    try {
      setLoading(true);
      const response = await packagesAPI.list();
      setPackages(response.data.data || []);
    } catch (error) {
      console.error('Failed to fetch packages:', error);
      setMessage({ type: 'error', text: 'Paketler yüklenemedi' });
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!formData.name.trim()) {
      setMessage({ type: 'error', text: 'Paket adı gerekli' });
      return;
    }

    try {
      setActionLoading(-1);
      await packagesAPI.create(formData);
      setMessage({ type: 'success', text: 'Paket oluşturuldu' });
      setShowAddModal(false);
      resetForm();
      await fetchPackages();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Paket oluşturulamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleUpdate = async () => {
    if (!editingPackage) return;

    try {
      setActionLoading(editingPackage.id);
      await packagesAPI.update(editingPackage.id, formData);
      setMessage({ type: 'success', text: 'Paket güncellendi' });
      setShowEditModal(false);
      setEditingPackage(null);
      resetForm();
      await fetchPackages();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Paket güncellenemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!deletingPackage) return;

    try {
      setActionLoading(deletingPackage.id);
      await packagesAPI.delete(deletingPackage.id);
      setMessage({ type: 'success', text: 'Paket silindi' });
      setShowDeleteModal(false);
      setDeletingPackage(null);
      await fetchPackages();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Paket silinemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const openEditModal = (pkg: PackageData) => {
    setEditingPackage(pkg);
    setFormData({
      name: pkg.name,
      disk_quota: pkg.disk_quota,
      bandwidth_quota: pkg.bandwidth_quota,
      max_domains: pkg.max_domains,
      max_databases: pkg.max_databases,
      max_emails: pkg.max_emails,
      max_ftp: pkg.max_ftp,
      max_php_memory: pkg.max_php_memory,
      max_php_upload: pkg.max_php_upload,
      max_php_execution_time: pkg.max_php_execution_time,
      max_emails_per_hour: pkg.max_emails_per_hour || 100,
      max_emails_per_day: pkg.max_emails_per_day || 500,
    });
    setShowEditModal(true);
  };

  const openDeleteModal = (pkg: PackageData) => {
    setDeletingPackage(pkg);
    setShowDeleteModal(true);
  };

  const resetForm = () => {
    setFormData(defaultFormData);
  };

  const formatSize = (mb: number) => {
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(1)} GB`;
    }
    return `${mb} MB`;
  };

  // Redirect if not admin
  if (user?.role !== 'admin') {
    return (
      <Layout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="w-12 h-12 mx-auto mb-4 text-red-500" />
            <h2 className="text-xl font-semibold">Erişim Reddedildi</h2>
            <p className="text-muted-foreground">Bu sayfaya erişim yetkiniz yok.</p>
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
            <h1 className="text-2xl font-bold">Paket Yönetimi</h1>
            <p className="text-muted-foreground text-sm">
              Hosting paketlerini oluşturun ve yönetin
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={fetchPackages} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Yenile
            </Button>
            <Button onClick={() => { resetForm(); setShowAddModal(true); }}>
              <Plus className="w-4 h-4 mr-2" />
              Yeni Paket
            </Button>
          </div>
        </div>

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

        {/* Packages Grid */}
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <RefreshCw className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : packages.length === 0 ? (
          <Card>
            <CardContent className="p-12 text-center">
              <Package className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
              <h3 className="font-medium mb-1">Paket Bulunamadı</h3>
              <p className="text-sm text-muted-foreground mb-4">Henüz hiç paket oluşturulmamış.</p>
              <Button onClick={() => { resetForm(); setShowAddModal(true); }}>
                <Plus className="w-4 h-4 mr-2" />
                İlk Paketi Oluştur
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {packages.map((pkg) => (
              <Card key={pkg.id} className="hover:shadow-lg transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg flex items-center gap-2">
                      <Package className="w-5 h-5 text-primary" />
                      {pkg.name}
                    </CardTitle>
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" onClick={() => openEditModal(pkg)}>
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button 
                        variant="ghost" 
                        size="sm" 
                        onClick={() => openDeleteModal(pkg)}
                        disabled={pkg.user_count > 0}
                        title={pkg.user_count > 0 ? 'Kullanımda olan paket silinemez' : ''}
                      >
                        <Trash2 className="w-4 h-4 text-red-500" />
                      </Button>
                    </div>
                  </div>
                  {pkg.user_count > 0 && (
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Users className="w-3 h-3" />
                      {pkg.user_count} kullanıcı
                    </div>
                  )}
                </CardHeader>
                <CardContent className="space-y-3">
                  {/* Disk & Bandwidth */}
                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex items-center gap-2 text-sm">
                      <HardDrive className="w-4 h-4 text-blue-500" />
                      <span className="text-muted-foreground">Disk:</span>
                      <span className="font-medium">{formatSize(pkg.disk_quota)}</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Server className="w-4 h-4 text-green-500" />
                      <span className="text-muted-foreground">Bant:</span>
                      <span className="font-medium">{formatSize(pkg.bandwidth_quota)}</span>
                    </div>
                  </div>

                  {/* Limits */}
                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex items-center gap-2 text-sm">
                      <Globe className="w-4 h-4 text-purple-500" />
                      <span className="text-muted-foreground">Domain:</span>
                      <span className="font-medium">{pkg.max_domains}</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Database className="w-4 h-4 text-orange-500" />
                      <span className="text-muted-foreground">DB:</span>
                      <span className="font-medium">{pkg.max_databases}</span>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex items-center gap-2 text-sm">
                      <Mail className="w-4 h-4 text-pink-500" />
                      <span className="text-muted-foreground">E-posta:</span>
                      <span className="font-medium">{pkg.max_emails}</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Server className="w-4 h-4 text-cyan-500" />
                      <span className="text-muted-foreground">FTP:</span>
                      <span className="font-medium">{pkg.max_ftp}</span>
                    </div>
                  </div>

                  {/* PHP Settings */}
                  <div className="pt-2 border-t border-border">
                    <div className="text-xs text-muted-foreground mb-2">PHP Ayarları</div>
                    <div className="grid grid-cols-3 gap-2 text-xs">
                      <div className="flex items-center gap-1">
                        <Cpu className="w-3 h-3 text-indigo-500" />
                        <span>{pkg.max_php_memory}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Upload className="w-3 h-3 text-teal-500" />
                        <span>{pkg.max_php_upload}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3 text-amber-500" />
                        <span>{pkg.max_php_execution_time}s</span>
                      </div>
                    </div>
                  </div>

                  {/* Mail Rate Limits */}
                  <div className="pt-2 border-t border-border">
                    <div className="text-xs text-muted-foreground mb-2">Mail Limitleri</div>
                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div className="flex items-center gap-1">
                        <Mail className="w-3 h-3 text-pink-500" />
                        <span>Saatlik: {pkg.max_emails_per_hour || 100}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Mail className="w-3 h-3 text-rose-500" />
                        <span>Günlük: {pkg.max_emails_per_day || 500}</span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {/* Add/Edit Modal */}
        {(showAddModal || showEditModal) && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h2 className="text-lg font-semibold">
                  {showAddModal ? 'Yeni Paket Oluştur' : 'Paketi Düzenle'}
                </h2>
                <button 
                  onClick={() => { setShowAddModal(false); setShowEditModal(false); setEditingPackage(null); resetForm(); }}
                  className="p-1 hover:bg-muted rounded"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                {/* Name */}
                <div>
                  <label className="block text-sm font-medium mb-1">Paket Adı *</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    placeholder="Örn: Starter, Professional, Enterprise"
                  />
                </div>

                {/* Disk & Bandwidth */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Disk Kotası (MB)</label>
                    <input
                      type="number"
                      value={formData.disk_quota}
                      onChange={(e) => setFormData({ ...formData, disk_quota: parseInt(e.target.value) || 0 })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                    <p className="text-xs text-muted-foreground mt-1">{formatSize(formData.disk_quota)}</p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Bant Genişliği (MB)</label>
                    <input
                      type="number"
                      value={formData.bandwidth_quota}
                      onChange={(e) => setFormData({ ...formData, bandwidth_quota: parseInt(e.target.value) || 0 })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                    <p className="text-xs text-muted-foreground mt-1">{formatSize(formData.bandwidth_quota)}</p>
                  </div>
                </div>

                {/* Limits */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Maks. Domain</label>
                    <input
                      type="number"
                      value={formData.max_domains}
                      onChange={(e) => setFormData({ ...formData, max_domains: parseInt(e.target.value) || 0 })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Maks. Veritabanı</label>
                    <input
                      type="number"
                      value={formData.max_databases}
                      onChange={(e) => setFormData({ ...formData, max_databases: parseInt(e.target.value) || 0 })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Maks. E-posta</label>
                    <input
                      type="number"
                      value={formData.max_emails}
                      onChange={(e) => setFormData({ ...formData, max_emails: parseInt(e.target.value) || 0 })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Maks. FTP</label>
                    <input
                      type="number"
                      value={formData.max_ftp}
                      onChange={(e) => setFormData({ ...formData, max_ftp: parseInt(e.target.value) || 0 })}
                      className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                  </div>
                </div>

                {/* PHP Settings */}
                <div className="pt-4 border-t border-border">
                  <h3 className="text-sm font-medium mb-3">PHP Ayarları</h3>
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium mb-1">Memory Limit</label>
                      <select
                        value={formData.max_php_memory}
                        onChange={(e) => setFormData({ ...formData, max_php_memory: e.target.value })}
                        className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                      >
                        <option value="64M">64 MB</option>
                        <option value="128M">128 MB</option>
                        <option value="256M">256 MB</option>
                        <option value="512M">512 MB</option>
                        <option value="1024M">1 GB</option>
                        <option value="2048M">2 GB</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-1">Upload Limit</label>
                      <select
                        value={formData.max_php_upload}
                        onChange={(e) => setFormData({ ...formData, max_php_upload: e.target.value })}
                        className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                      >
                        <option value="8M">8 MB</option>
                        <option value="16M">16 MB</option>
                        <option value="32M">32 MB</option>
                        <option value="64M">64 MB</option>
                        <option value="128M">128 MB</option>
                        <option value="256M">256 MB</option>
                        <option value="512M">512 MB</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-1">Execution Time (s)</label>
                      <select
                        value={formData.max_php_execution_time}
                        onChange={(e) => setFormData({ ...formData, max_php_execution_time: parseInt(e.target.value) })}
                        className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                      >
                        <option value={30}>30 saniye</option>
                        <option value={60}>60 saniye</option>
                        <option value={120}>2 dakika</option>
                        <option value={300}>5 dakika</option>
                        <option value={600}>10 dakika</option>
                        <option value={900}>15 dakika</option>
                      </select>
                    </div>
                  </div>
                </div>

                {/* Mail Rate Limits */}
                <div className="pt-4 border-t border-border">
                  <h3 className="text-sm font-medium mb-3">Mail Gönderim Limitleri</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium mb-1">Saatlik Limit</label>
                      <input
                        type="number"
                        value={formData.max_emails_per_hour}
                        onChange={(e) => setFormData({ ...formData, max_emails_per_hour: parseInt(e.target.value) || 0 })}
                        className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                        min={0}
                      />
                      <p className="text-xs text-muted-foreground mt-1">Saatte gönderilebilecek maksimum mail sayısı</p>
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-1">Günlük Limit</label>
                      <input
                        type="number"
                        value={formData.max_emails_per_day}
                        onChange={(e) => setFormData({ ...formData, max_emails_per_day: parseInt(e.target.value) || 0 })}
                        className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
                        min={0}
                      />
                      <p className="text-xs text-muted-foreground mt-1">Günde gönderilebilecek maksimum mail sayısı</p>
                    </div>
                  </div>
                  <p className="text-xs text-muted-foreground mt-3 p-2 bg-muted rounded">
                    <strong>Not:</strong> Limit aşıldığında mailler otomatik olarak kuyruğa alınır ve sonraki saat/gün gönderilir.
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button 
                  variant="outline" 
                  onClick={() => { setShowAddModal(false); setShowEditModal(false); setEditingPackage(null); resetForm(); }}
                >
                  İptal
                </Button>
                <Button 
                  onClick={showAddModal ? handleCreate : handleUpdate}
                  disabled={actionLoading !== null}
                >
                  {actionLoading !== null ? (
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                  ) : null}
                  {showAddModal ? 'Oluştur' : 'Güncelle'}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Delete Confirmation Modal */}
        {showDeleteModal && deletingPackage && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="p-6 text-center">
                <div className="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center mx-auto mb-4">
                  <Trash2 className="w-6 h-6 text-red-600" />
                </div>
                <h3 className="text-lg font-semibold mb-2">Paketi Sil</h3>
                <p className="text-muted-foreground mb-4">
                  <strong>{deletingPackage.name}</strong> paketini silmek istediğinizden emin misiniz?
                  Bu işlem geri alınamaz.
                </p>
                <div className="flex justify-center gap-2">
                  <Button 
                    variant="outline" 
                    onClick={() => { setShowDeleteModal(false); setDeletingPackage(null); }}
                  >
                    İptal
                  </Button>
                  <Button 
                    variant="destructive"
                    onClick={handleDelete}
                    disabled={actionLoading !== null}
                  >
                    {actionLoading !== null ? (
                      <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    ) : null}
                    Sil
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
