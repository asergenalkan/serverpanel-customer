import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import Layout from '../components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { dnsAPI } from '../lib/api';
import {
  Globe,
  Plus,
  Pencil,
  Trash2,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Info,
  Server,
  Mail,
  FileText,
  Link2,
  Shield,
  RotateCcw,
  X,
} from 'lucide-react';

interface DNSRecord {
  id: number;
  domain_id: number;
  name: string;
  type: string;
  content: string;
  ttl: number;
  priority: number;
  active: boolean;
}

interface DNSZone {
  domain_id: number;
  domain_name: string;
  records: DNSRecord[];
  server_ip: string;
}

interface ZoneInfo {
  domain_id: number;
  domain_name: string;
  username: string;
}

const RECORD_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA'];

const TTL_OPTIONS = [
  { value: 300, label: '5 dakika' },
  { value: 900, label: '15 dakika' },
  { value: 1800, label: '30 dakika' },
  { value: 3600, label: '1 saat' },
  { value: 7200, label: '2 saat' },
  { value: 14400, label: '4 saat' },
  { value: 28800, label: '8 saat' },
  { value: 43200, label: '12 saat' },
  { value: 86400, label: '1 gün' },
];

const getRecordIcon = (type: string) => {
  switch (type) {
    case 'A':
    case 'AAAA':
      return <Server className="w-4 h-4" />;
    case 'MX':
      return <Mail className="w-4 h-4" />;
    case 'TXT':
      return <FileText className="w-4 h-4" />;
    case 'CNAME':
      return <Link2 className="w-4 h-4" />;
    case 'NS':
      return <Globe className="w-4 h-4" />;
    case 'CAA':
      return <Shield className="w-4 h-4" />;
    default:
      return <Globe className="w-4 h-4" />;
  }
};

const getRecordTypeColor = (type: string) => {
  switch (type) {
    case 'A':
      return 'bg-blue-500/20 text-blue-600 dark:text-blue-400 border border-blue-500/30';
    case 'AAAA':
      return 'bg-indigo-500/20 text-indigo-600 dark:text-indigo-400 border border-indigo-500/30';
    case 'CNAME':
      return 'bg-purple-500/20 text-purple-600 dark:text-purple-400 border border-purple-500/30';
    case 'MX':
      return 'bg-green-500/20 text-green-600 dark:text-green-400 border border-green-500/30';
    case 'TXT':
      return 'bg-yellow-500/20 text-yellow-600 dark:text-yellow-400 border border-yellow-500/30';
    case 'NS':
      return 'bg-orange-500/20 text-orange-600 dark:text-orange-400 border border-orange-500/30';
    case 'SRV':
      return 'bg-pink-500/20 text-pink-600 dark:text-pink-400 border border-pink-500/30';
    case 'CAA':
      return 'bg-red-500/20 text-red-600 dark:text-red-400 border border-red-500/30';
    default:
      return 'bg-gray-500/20 text-gray-600 dark:text-gray-400 border border-gray-500/30';
  }
};

export default function DNSZoneEditorPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [zones, setZones] = useState<ZoneInfo[]>([]);
  const [selectedZone, setSelectedZone] = useState<DNSZone | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Modal states
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingRecord, setEditingRecord] = useState<DNSRecord | null>(null);

  // Form state
  const [formData, setFormData] = useState({
    name: '',
    type: 'A',
    content: '',
    ttl: 3600,
    priority: 0,
  });

  // Filter state
  const [filterType, setFilterType] = useState<string>('all');

  useEffect(() => {
    fetchZones();
  }, []);

  const fetchZones = async () => {
    try {
      setLoading(true);
      const response = await dnsAPI.listZones();
      setZones(response.data || []);
      
      // Auto-select first zone if available
      if (response.data && response.data.length > 0 && !selectedZone) {
        await fetchZoneDetails(response.data[0].domain_id);
      }
    } catch (error) {
      console.error('Failed to fetch zones:', error);
      setMessage({ type: 'error', text: 'DNS zone listesi alınamadı' });
    } finally {
      setLoading(false);
    }
  };

  const fetchZoneDetails = async (domainId: number) => {
    try {
      setActionLoading(domainId);
      const response = await dnsAPI.getZone(domainId);
      setSelectedZone(response.data);
    } catch (error) {
      console.error('Failed to fetch zone details:', error);
      setMessage({ type: 'error', text: 'DNS zone detayları alınamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleAddRecord = async () => {
    if (!selectedZone) return;

    try {
      setActionLoading(-1);
      await dnsAPI.createRecord({
        domain_id: selectedZone.domain_id,
        name: formData.name || '@',
        type: formData.type,
        content: formData.content,
        ttl: formData.ttl,
        priority: formData.priority,
      });

      setMessage({ type: 'success', text: 'DNS kaydı oluşturuldu' });
      setShowAddModal(false);
      resetForm();
      await fetchZoneDetails(selectedZone.domain_id);
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Kayıt oluşturulamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleUpdateRecord = async () => {
    if (!editingRecord || !selectedZone) return;

    try {
      setActionLoading(editingRecord.id);
      await dnsAPI.updateRecord(editingRecord.id, {
        name: formData.name || '@',
        type: formData.type,
        content: formData.content,
        ttl: formData.ttl,
        priority: formData.priority,
      });

      setMessage({ type: 'success', text: 'DNS kaydı güncellendi' });
      setShowEditModal(false);
      setEditingRecord(null);
      resetForm();
      await fetchZoneDetails(selectedZone.domain_id);
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Kayıt güncellenemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteRecord = async (record: DNSRecord) => {
    if (!selectedZone) return;
    if (!confirm(`"${record.name}" kaydını silmek istediğinize emin misiniz?`)) return;

    try {
      setActionLoading(record.id);
      await dnsAPI.deleteRecord(record.id);
      setMessage({ type: 'success', text: 'DNS kaydı silindi' });
      await fetchZoneDetails(selectedZone.domain_id);
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Kayıt silinemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleResetZone = async () => {
    if (!selectedZone) return;
    if (!confirm('Tüm DNS kayıtlarını varsayılana sıfırlamak istediğinize emin misiniz? Bu işlem geri alınamaz!')) return;

    try {
      setActionLoading(-2);
      await dnsAPI.resetZone(selectedZone.domain_id);
      setMessage({ type: 'success', text: 'DNS zone varsayılana sıfırlandı' });
      await fetchZoneDetails(selectedZone.domain_id);
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Zone sıfırlanamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const openEditModal = (record: DNSRecord) => {
    setEditingRecord(record);
    setFormData({
      name: record.name,
      type: record.type,
      content: record.content,
      ttl: record.ttl,
      priority: record.priority,
    });
    setShowEditModal(true);
  };

  const resetForm = () => {
    setFormData({
      name: '',
      type: 'A',
      content: '',
      ttl: 3600,
      priority: 0,
    });
  };

  const filteredRecords = selectedZone?.records?.filter(
    (r) => filterType === 'all' || r.type === filterType
  ) || [];


  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">DNS Zone Editor</h1>
            <p className="text-muted-foreground text-sm">
              Domain DNS kayıtlarınızı yönetin
            </p>
          </div>
          <Button variant="outline" onClick={fetchZones} disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Yenile
          </Button>
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

        {/* Domain Selector + Zone Editor */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Domain List */}
          <Card className="lg:col-span-1">
            <CardHeader className="pb-3">
              <CardTitle className="text-lg flex items-center gap-2">
                <Globe className="w-5 h-5" />
                Domainler
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {zones.length === 0 ? (
                <div className="p-4 text-center text-muted-foreground">
                  <Globe className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Domain bulunamadı</p>
                </div>
              ) : (
                <div className="divide-y divide-border">
                  {zones.map((zone) => (
                    <button
                      key={zone.domain_id}
                      onClick={() => fetchZoneDetails(zone.domain_id)}
                      className={`w-full px-4 py-3 text-left hover:bg-muted transition-colors ${
                        selectedZone?.domain_id === zone.domain_id ? 'bg-primary/10 border-l-2 border-primary' : ''
                      }`}
                    >
                      <div className="font-medium text-foreground">{zone.domain_name}</div>
                      {isAdmin && (
                        <div className="text-xs text-muted-foreground">{zone.username}</div>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Zone Editor */}
          <div className="lg:col-span-3 space-y-4">
            {selectedZone ? (
              <>
                {/* Zone Info */}
                <Card className="bg-primary/5 border-primary/20">
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-primary/10 rounded-lg">
                          <Globe className="w-6 h-6 text-primary" />
                        </div>
                        <div>
                          <h3 className="font-semibold text-foreground">{selectedZone.domain_name}</h3>
                          <p className="text-sm text-muted-foreground">
                            Sunucu IP: {selectedZone.server_ip} • {selectedZone.records?.length || 0} kayıt
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={handleResetZone}
                          disabled={actionLoading === -2}
                          className="text-orange-600 hover:text-orange-700 hover:bg-orange-50 dark:hover:bg-orange-900/20"
                        >
                          <RotateCcw className="w-4 h-4 mr-1" />
                          Sıfırla
                        </Button>
                        <Button size="sm" onClick={() => setShowAddModal(true)}>
                          <Plus className="w-4 h-4 mr-1" />
                          Kayıt Ekle
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Filter */}
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-sm text-muted-foreground">Filtrele:</span>
                  <button
                    onClick={() => setFilterType('all')}
                    className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
                      filterType === 'all'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:bg-muted/80'
                    }`}
                  >
                    Tümü
                  </button>
                  {RECORD_TYPES.map((type) => (
                    <button
                      key={type}
                      onClick={() => setFilterType(type)}
                      className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
                        filterType === type
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-muted-foreground hover:bg-muted/80'
                      }`}
                    >
                      {type}
                    </button>
                  ))}
                </div>

                {/* Records Table */}
                <Card>
                  <CardContent className="p-0">
                    {filteredRecords.length === 0 ? (
                      <div className="p-8 text-center text-muted-foreground">
                        <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
                        <h3 className="font-medium mb-1">Kayıt Bulunamadı</h3>
                        <p className="text-sm">Bu filtreye uygun DNS kaydı yok.</p>
                      </div>
                    ) : (
                      <div className="overflow-x-auto">
                        <table className="w-full">
                          <thead className="bg-muted/50">
                            <tr>
                              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Tip</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Ad</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">İçerik</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">TTL</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Öncelik</th>
                              <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">İşlemler</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-border">
                            {filteredRecords.map((record) => (
                              <tr key={record.id} className="hover:bg-muted/30 transition-colors">
                                <td className="px-4 py-3">
                                  <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${getRecordTypeColor(record.type)}`}>
                                    {getRecordIcon(record.type)}
                                    {record.type}
                                  </span>
                                </td>
                                <td className="px-4 py-3">
                                  <span className="font-mono text-sm text-foreground">
                                    {record.name === '@' ? selectedZone.domain_name : `${record.name}.${selectedZone.domain_name}`}
                                  </span>
                                </td>
                                <td className="px-4 py-3">
                                  <span className="font-mono text-sm text-muted-foreground max-w-xs truncate block">
                                    {record.content}
                                  </span>
                                </td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">
                                  {record.ttl}s
                                </td>
                                <td className="px-4 py-3 text-sm text-muted-foreground">
                                  {record.type === 'MX' || record.type === 'SRV' ? record.priority : '-'}
                                </td>
                                <td className="px-4 py-3 text-right">
                                  <div className="flex items-center justify-end gap-1">
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      onClick={() => openEditModal(record)}
                                      disabled={actionLoading === record.id}
                                    >
                                      <Pencil className="w-4 h-4" />
                                    </Button>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      onClick={() => handleDeleteRecord(record)}
                                      disabled={actionLoading === record.id || record.type === 'NS' && record.name === '@'}
                                      className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                                    >
                                      <Trash2 className="w-4 h-4" />
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

                {/* Info Box */}
                <Card className="bg-primary/5 border-primary/20">
                  <CardContent className="p-4">
                    <div className="flex items-start gap-3">
                      <Info className="w-5 h-5 text-primary mt-0.5" />
                      <div className="text-sm">
                        <p className="font-medium text-foreground">DNS Kayıt Tipleri</p>
                        <ul className="text-muted-foreground mt-1 space-y-1">
                          <li><strong>A:</strong> Domain'i IPv4 adresine yönlendirir</li>
                          <li><strong>AAAA:</strong> Domain'i IPv6 adresine yönlendirir</li>
                          <li><strong>CNAME:</strong> Domain'i başka bir domain'e yönlendirir</li>
                          <li><strong>MX:</strong> E-posta sunucusunu belirtir</li>
                          <li><strong>TXT:</strong> Metin kaydı (SPF, DKIM vb.)</li>
                          <li><strong>NS:</strong> Nameserver kaydı</li>
                        </ul>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </>
            ) : (
              <Card>
                <CardContent className="p-8 text-center">
                  <Globe className="w-16 h-16 mx-auto mb-4 text-muted-foreground opacity-50" />
                  <h3 className="text-lg font-medium text-foreground mb-2">Domain Seçin</h3>
                  <p className="text-muted-foreground">
                    DNS kayıtlarını düzenlemek için sol taraftan bir domain seçin.
                  </p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>

      {/* Add Record Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-background rounded-lg shadow-xl w-full max-w-lg">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <h2 className="text-lg font-semibold">Yeni DNS Kaydı</h2>
              <button onClick={() => { setShowAddModal(false); resetForm(); }} className="text-muted-foreground hover:text-foreground">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              {/* Record Type */}
              <div>
                <label className="block text-sm font-medium mb-1">Kayıt Tipi</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value, priority: e.target.value === 'MX' ? 10 : 0 })}
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  {RECORD_TYPES.map((type) => (
                    <option key={type} value={type}>{type}</option>
                  ))}
                </select>
              </div>

              {/* Name */}
              <div>
                <label className="block text-sm font-medium mb-1">Ad (Subdomain)</label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="@ veya www"
                    className="flex-1 px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                  <span className="text-muted-foreground">.{selectedZone?.domain_name}</span>
                </div>
                <p className="text-xs text-muted-foreground mt-1">@ işareti ana domain'i temsil eder</p>
              </div>

              {/* Content */}
              <div>
                <label className="block text-sm font-medium mb-1">
                  {formData.type === 'A' ? 'IPv4 Adresi' :
                   formData.type === 'AAAA' ? 'IPv6 Adresi' :
                   formData.type === 'CNAME' || formData.type === 'MX' || formData.type === 'NS' ? 'Hedef Hostname' :
                   formData.type === 'TXT' ? 'Metin İçeriği' : 'İçerik'}
                </label>
                <input
                  type="text"
                  value={formData.content}
                  onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                  placeholder={
                    formData.type === 'A' ? '192.168.1.1' :
                    formData.type === 'AAAA' ? '2001:db8::1' :
                    formData.type === 'CNAME' ? 'example.com.' :
                    formData.type === 'MX' ? 'mail.example.com.' :
                    formData.type === 'TXT' ? 'v=spf1 include:_spf.google.com ~all' : ''
                  }
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              {/* Priority (for MX/SRV) */}
              {(formData.type === 'MX' || formData.type === 'SRV') && (
                <div>
                  <label className="block text-sm font-medium mb-1">Öncelik</label>
                  <input
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                    min="0"
                    max="65535"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                  <p className="text-xs text-muted-foreground mt-1">Düşük değer = yüksek öncelik</p>
                </div>
              )}

              {/* TTL */}
              <div>
                <label className="block text-sm font-medium mb-1">TTL (Time To Live)</label>
                <select
                  value={formData.ttl}
                  onChange={(e) => setFormData({ ...formData, ttl: parseInt(e.target.value) })}
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  {TTL_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label} ({opt.value}s)</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 p-4 border-t border-border">
              <Button variant="outline" onClick={() => { setShowAddModal(false); resetForm(); }}>
                İptal
              </Button>
              <Button onClick={handleAddRecord} disabled={actionLoading === -1 || !formData.content}>
                {actionLoading === -1 ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Ekleniyor...
                  </>
                ) : (
                  <>
                    <Plus className="w-4 h-4 mr-2" />
                    Kayıt Ekle
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Record Modal */}
      {showEditModal && editingRecord && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-background rounded-lg shadow-xl w-full max-w-lg">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <h2 className="text-lg font-semibold">DNS Kaydını Düzenle</h2>
              <button onClick={() => { setShowEditModal(false); setEditingRecord(null); resetForm(); }} className="text-muted-foreground hover:text-foreground">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              {/* Record Type */}
              <div>
                <label className="block text-sm font-medium mb-1">Kayıt Tipi</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  {RECORD_TYPES.map((type) => (
                    <option key={type} value={type}>{type}</option>
                  ))}
                </select>
              </div>

              {/* Name */}
              <div>
                <label className="block text-sm font-medium mb-1">Ad (Subdomain)</label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="@ veya www"
                    className="flex-1 px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                  <span className="text-muted-foreground">.{selectedZone?.domain_name}</span>
                </div>
              </div>

              {/* Content */}
              <div>
                <label className="block text-sm font-medium mb-1">İçerik</label>
                <input
                  type="text"
                  value={formData.content}
                  onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              {/* Priority (for MX/SRV) */}
              {(formData.type === 'MX' || formData.type === 'SRV') && (
                <div>
                  <label className="block text-sm font-medium mb-1">Öncelik</label>
                  <input
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                    min="0"
                    max="65535"
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              )}

              {/* TTL */}
              <div>
                <label className="block text-sm font-medium mb-1">TTL (Time To Live)</label>
                <select
                  value={formData.ttl}
                  onChange={(e) => setFormData({ ...formData, ttl: parseInt(e.target.value) })}
                  className="w-full px-3 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  {TTL_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label} ({opt.value}s)</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 p-4 border-t border-border">
              <Button variant="outline" onClick={() => { setShowEditModal(false); setEditingRecord(null); resetForm(); }}>
                İptal
              </Button>
              <Button onClick={handleUpdateRecord} disabled={actionLoading === editingRecord.id || !formData.content}>
                {actionLoading === editingRecord.id ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Güncelleniyor...
                  </>
                ) : (
                  <>
                    <CheckCircle className="w-4 h-4 mr-2" />
                    Güncelle
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}
