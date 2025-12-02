import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { Card, CardContent } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { emailAPI, domainsAPI } from '../lib/api';
import {
  Mail,
  Plus,
  Trash2,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  X,
  Forward,
  MessageSquare,
  ExternalLink,
  Eye,
  EyeOff,
  HardDrive,
  Users,
  ArrowRight,
  Search,
  Power,
} from 'lucide-react';

interface EmailAccount {
  id: number;
  user_id: number;
  domain_id: number;
  domain_name: string;
  email: string;
  local_part: string;
  quota_mb: number;
  used_mb: number;
  active: boolean;
  created_at: string;
}

interface EmailForwarder {
  id: number;
  user_id: number;
  domain_id: number;
  domain_name: string;
  source: string;
  destination: string;
  active: boolean;
  created_at: string;
}

interface EmailAutoresponder {
  id: number;
  user_id: number;
  domain_id: number;
  email: string;
  subject: string;
  body: string;
  start_date?: string;
  end_date?: string;
  active: boolean;
  created_at: string;
}

interface EmailStats {
  total_accounts: number;
  total_forwarders: number;
  total_quota_mb: number;
  used_quota_mb: number;
  max_accounts: number;
}

interface Domain {
  id: number;
  name: string;
}

export default function EmailPage() {
  const [accounts, setAccounts] = useState<EmailAccount[]>([]);
  const [forwarders, setForwarders] = useState<EmailForwarder[]>([]);
  const [autoresponders, setAutoresponders] = useState<EmailAutoresponder[]>([]);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [stats, setStats] = useState<EmailStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState<'accounts' | 'forwarders' | 'autoresponders'>('accounts');

  // Modal states
  const [showAddAccountModal, setShowAddAccountModal] = useState(false);
  const [showAddForwarderModal, setShowAddForwarderModal] = useState(false);
  const [showAddAutoresponderModal, setShowAddAutoresponderModal] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState<EmailAccount | null>(null);

  // Form states
  const [accountForm, setAccountForm] = useState({
    domain_id: 0,
    username: '',
    password: '',
    quota_mb: 1024,
  });
  const [forwarderForm, setForwarderForm] = useState({
    domain_id: 0,
    source: '',
    destination: '',
  });
  const [autoresponderForm, setAutoresponderForm] = useState({
    email_account_id: 0,
    subject: '',
    body: '',
    start_date: '',
    end_date: '',
  });
  const [newPassword, setNewPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [accountsRes, forwardersRes, autorespondersRes, domainsRes, statsRes] = await Promise.all([
        emailAPI.listAccounts(),
        emailAPI.listForwarders(),
        emailAPI.listAutoresponders(),
        domainsAPI.list(),
        emailAPI.getStats(),
      ]);

      setAccounts(accountsRes.data.data || []);
      setForwarders(forwardersRes.data.data || []);
      setAutoresponders(autorespondersRes.data.data || []);
      setDomains(domainsRes.data.data || []);
      setStats(statsRes.data.data);
    } catch (error) {
      console.error('Error fetching email data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAccount = async () => {
    if (!accountForm.domain_id || !accountForm.username || !accountForm.password) {
      setMessage({ type: 'error', text: 'Tüm alanları doldurun' });
      return;
    }

    try {
      setActionLoading('create-account');
      const response = await emailAPI.createAccount(accountForm);
      if (response.data.success) {
        setMessage({ type: 'success', text: 'E-posta hesabı oluşturuldu' });
        setShowAddAccountModal(false);
        setAccountForm({ domain_id: 0, username: '', password: '', quota_mb: 1024 });
        fetchData();
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Hesap oluşturulamadı' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Hesap oluşturulamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteAccount = async (id: number) => {
    if (!confirm('Bu e-posta hesabını silmek istediğinizden emin misiniz?')) return;

    try {
      setActionLoading(`delete-account-${id}`);
      await emailAPI.deleteAccount(id);
      setMessage({ type: 'success', text: 'E-posta hesabı silindi' });
      fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Hesap silinemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleToggleAccount = async (id: number) => {
    try {
      setActionLoading(`toggle-${id}`);
      await emailAPI.toggleAccount(id);
      fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Durum değiştirilemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleChangePassword = async () => {
    if (!selectedAccount || !newPassword) return;

    try {
      setActionLoading('change-password');
      await emailAPI.updateAccount(selectedAccount.id, { password: newPassword });
      setMessage({ type: 'success', text: 'Şifre güncellendi' });
      setShowPasswordModal(false);
      setNewPassword('');
      setSelectedAccount(null);
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Şifre güncellenemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleCreateForwarder = async () => {
    if (!forwarderForm.domain_id || !forwarderForm.source || !forwarderForm.destination) {
      setMessage({ type: 'error', text: 'Tüm alanları doldurun' });
      return;
    }

    try {
      setActionLoading('create-forwarder');
      const response = await emailAPI.createForwarder(forwarderForm);
      if (response.data.success) {
        setMessage({ type: 'success', text: 'Yönlendirme oluşturuldu' });
        setShowAddForwarderModal(false);
        setForwarderForm({ domain_id: 0, source: '', destination: '' });
        fetchData();
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Yönlendirme oluşturulamadı' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Yönlendirme oluşturulamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteForwarder = async (id: number) => {
    if (!confirm('Bu yönlendirmeyi silmek istediğinizden emin misiniz?')) return;

    try {
      setActionLoading(`delete-forwarder-${id}`);
      await emailAPI.deleteForwarder(id);
      setMessage({ type: 'success', text: 'Yönlendirme silindi' });
      fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Yönlendirme silinemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleCreateAutoresponder = async () => {
    if (!autoresponderForm.email_account_id || !autoresponderForm.subject || !autoresponderForm.body) {
      setMessage({ type: 'error', text: 'E-posta, konu ve mesaj gerekli' });
      return;
    }

    try {
      setActionLoading('create-autoresponder');
      const response = await emailAPI.createAutoresponder(autoresponderForm);
      if (response.data.success) {
        setMessage({ type: 'success', text: 'Otomatik yanıtlayıcı oluşturuldu' });
        setShowAddAutoresponderModal(false);
        setAutoresponderForm({ email_account_id: 0, subject: '', body: '', start_date: '', end_date: '' });
        fetchData();
      } else {
        setMessage({ type: 'error', text: response.data.error || 'Otomatik yanıtlayıcı oluşturulamadı' });
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Otomatik yanıtlayıcı oluşturulamadı' });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteAutoresponder = async (id: number) => {
    if (!confirm('Bu otomatik yanıtlayıcıyı silmek istediğinizden emin misiniz?')) return;

    try {
      setActionLoading(`delete-autoresponder-${id}`);
      await emailAPI.deleteAutoresponder(id);
      setMessage({ type: 'success', text: 'Otomatik yanıtlayıcı silindi' });
      fetchData();
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Otomatik yanıtlayıcı silinemedi' });
    } finally {
      setActionLoading(null);
    }
  };

  const openWebmail = async () => {
    try {
      const response = await emailAPI.getWebmailURL();
      if (response.data.success && response.data.data?.url) {
        window.open(response.data.data.url, '_blank');
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Webmail URL alınamadı' });
    }
  };

  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';
    for (let i = 0; i < 16; i++) {
      password += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return password;
  };

  // Filter data
  const filteredAccounts = accounts.filter(a =>
    a.email.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const filteredForwarders = forwarders.filter(f =>
    f.source.toLowerCase().includes(searchQuery.toLowerCase()) ||
    f.destination.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const filteredAutoresponders = autoresponders.filter(a =>
    a.email.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Mail className="w-7 h-7 text-primary" />
              E-posta Yönetimi
            </h1>
            <p className="text-muted-foreground mt-1">
              E-posta hesapları, yönlendirmeler ve otomatik yanıtlayıcılar
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={openWebmail}>
              <ExternalLink className="w-4 h-4 mr-2" />
              Webmail
            </Button>
            <Button onClick={fetchData} variant="outline" disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Yenile
            </Button>
          </div>
        </div>

        {/* Message */}
        {message && (
          <div className={`p-4 rounded-lg flex items-center gap-2 ${
            message.type === 'success' 
              ? 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400' 
              : 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400'
          }`}>
            {message.type === 'success' ? <CheckCircle className="w-5 h-5" /> : <AlertCircle className="w-5 h-5" />}
            {message.text}
            <button onClick={() => setMessage(null)} className="ml-auto">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Stats Cards */}
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                    <Mail className="w-5 h-5 text-blue-600" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">E-posta Hesapları</p>
                    <p className="text-xl font-bold">
                      {stats.total_accounts}
                      {stats.max_accounts !== -1 && <span className="text-sm font-normal text-muted-foreground">/{stats.max_accounts}</span>}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                    <Forward className="w-5 h-5 text-green-600" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Yönlendirmeler</p>
                    <p className="text-xl font-bold">{stats.total_forwarders}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                    <HardDrive className="w-5 h-5 text-purple-600" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Toplam Kota</p>
                    <p className="text-xl font-bold">{(stats.total_quota_mb / 1024).toFixed(1)} GB</p>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
                    <Users className="w-5 h-5 text-orange-600" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Kullanılan Alan</p>
                    <p className="text-xl font-bold">{stats.used_quota_mb} MB</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-2 border-b border-border">
          <button
            onClick={() => setActiveTab('accounts')}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === 'accounts'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <Mail className="w-4 h-4 inline mr-2" />
            E-posta Hesapları ({accounts.length})
          </button>
          <button
            onClick={() => setActiveTab('forwarders')}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === 'forwarders'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <Forward className="w-4 h-4 inline mr-2" />
            Yönlendirmeler ({forwarders.length})
          </button>
          <button
            onClick={() => setActiveTab('autoresponders')}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === 'autoresponders'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <MessageSquare className="w-4 h-4 inline mr-2" />
            Otomatik Yanıt ({autoresponders.length})
          </button>
        </div>

        {/* Search and Add */}
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Ara..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-background"
            />
          </div>
          {activeTab === 'accounts' && (
            <Button onClick={() => setShowAddAccountModal(true)}>
              <Plus className="w-4 h-4 mr-2" />
              E-posta Hesabı Ekle
            </Button>
          )}
          {activeTab === 'forwarders' && (
            <Button onClick={() => setShowAddForwarderModal(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Yönlendirme Ekle
            </Button>
          )}
          {activeTab === 'autoresponders' && (
            <Button onClick={() => setShowAddAutoresponderModal(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Otomatik Yanıt Ekle
            </Button>
          )}
        </div>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <RefreshCw className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : (
          <>
            {/* Email Accounts Tab */}
            {activeTab === 'accounts' && (
              <Card>
                <CardContent className="p-0">
                  {filteredAccounts.length === 0 ? (
                    <div className="text-center py-12 text-muted-foreground">
                      <Mail className="w-12 h-12 mx-auto mb-4 opacity-50" />
                      <p>Henüz e-posta hesabı yok</p>
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full">
                        <thead className="bg-muted/50">
                          <tr>
                            <th className="text-left p-4 font-medium">E-posta</th>
                            <th className="text-left p-4 font-medium">Kota</th>
                            <th className="text-left p-4 font-medium">Durum</th>
                            <th className="text-right p-4 font-medium">İşlemler</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {filteredAccounts.map((account) => (
                            <tr key={account.id} className="hover:bg-muted/30">
                              <td className="p-4">
                                <div className="flex items-center gap-3">
                                  <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                                    <Mail className="w-4 h-4 text-blue-600" />
                                  </div>
                                  <div>
                                    <p className="font-medium">{account.email}</p>
                                    <p className="text-sm text-muted-foreground">{account.domain_name}</p>
                                  </div>
                                </div>
                              </td>
                              <td className="p-4">
                                <div className="flex items-center gap-2">
                                  <div className="w-24 h-2 bg-muted rounded-full overflow-hidden">
                                    <div
                                      className="h-full bg-blue-500"
                                      style={{ width: `${Math.min((account.used_mb / account.quota_mb) * 100, 100)}%` }}
                                    />
                                  </div>
                                  <span className="text-sm text-muted-foreground">
                                    {account.used_mb} / {account.quota_mb} MB
                                  </span>
                                </div>
                              </td>
                              <td className="p-4">
                                <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${
                                  account.active
                                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                    : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                                }`}>
                                  {account.active ? 'Aktif' : 'Pasif'}
                                </span>
                              </td>
                              <td className="p-4">
                                <div className="flex items-center justify-end gap-2">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleToggleAccount(account.id)}
                                    disabled={actionLoading === `toggle-${account.id}`}
                                  >
                                    <Power className={`w-4 h-4 ${account.active ? 'text-green-500' : 'text-red-500'}`} />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                      setSelectedAccount(account);
                                      setShowPasswordModal(true);
                                    }}
                                  >
                                    <Eye className="w-4 h-4" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleDeleteAccount(account.id)}
                                    disabled={actionLoading === `delete-account-${account.id}`}
                                  >
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

            {/* Forwarders Tab */}
            {activeTab === 'forwarders' && (
              <Card>
                <CardContent className="p-0">
                  {filteredForwarders.length === 0 ? (
                    <div className="text-center py-12 text-muted-foreground">
                      <Forward className="w-12 h-12 mx-auto mb-4 opacity-50" />
                      <p>Henüz yönlendirme yok</p>
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full">
                        <thead className="bg-muted/50">
                          <tr>
                            <th className="text-left p-4 font-medium">Kaynak</th>
                            <th className="text-left p-4 font-medium">Hedef</th>
                            <th className="text-right p-4 font-medium">İşlemler</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {filteredForwarders.map((forwarder) => (
                            <tr key={forwarder.id} className="hover:bg-muted/30">
                              <td className="p-4">
                                <div className="flex items-center gap-3">
                                  <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                                    <Mail className="w-4 h-4 text-green-600" />
                                  </div>
                                  <span className="font-medium">{forwarder.source}</span>
                                </div>
                              </td>
                              <td className="p-4">
                                <div className="flex items-center gap-2">
                                  <ArrowRight className="w-4 h-4 text-muted-foreground" />
                                  <span>{forwarder.destination}</span>
                                </div>
                              </td>
                              <td className="p-4">
                                <div className="flex items-center justify-end">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleDeleteForwarder(forwarder.id)}
                                    disabled={actionLoading === `delete-forwarder-${forwarder.id}`}
                                  >
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

            {/* Autoresponders Tab */}
            {activeTab === 'autoresponders' && (
              <Card>
                <CardContent className="p-0">
                  {filteredAutoresponders.length === 0 ? (
                    <div className="text-center py-12 text-muted-foreground">
                      <MessageSquare className="w-12 h-12 mx-auto mb-4 opacity-50" />
                      <p>Henüz otomatik yanıtlayıcı yok</p>
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full">
                        <thead className="bg-muted/50">
                          <tr>
                            <th className="text-left p-4 font-medium">E-posta</th>
                            <th className="text-left p-4 font-medium">Konu</th>
                            <th className="text-left p-4 font-medium">Durum</th>
                            <th className="text-right p-4 font-medium">İşlemler</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {filteredAutoresponders.map((ar) => (
                            <tr key={ar.id} className="hover:bg-muted/30">
                              <td className="p-4">
                                <div className="flex items-center gap-3">
                                  <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                                    <MessageSquare className="w-4 h-4 text-purple-600" />
                                  </div>
                                  <span className="font-medium">{ar.email}</span>
                                </div>
                              </td>
                              <td className="p-4">
                                <span className="text-muted-foreground">{ar.subject}</span>
                              </td>
                              <td className="p-4">
                                <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${
                                  ar.active
                                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                    : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                                }`}>
                                  {ar.active ? 'Aktif' : 'Pasif'}
                                </span>
                              </td>
                              <td className="p-4">
                                <div className="flex items-center justify-end">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleDeleteAutoresponder(ar.id)}
                                    disabled={actionLoading === `delete-autoresponder-${ar.id}`}
                                  >
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
          </>
        )}

        {/* Add Account Modal */}
        {showAddAccountModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h3 className="text-lg font-semibold">E-posta Hesabı Ekle</h3>
                <button onClick={() => setShowAddAccountModal(false)}>
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Domain</label>
                  <select
                    value={accountForm.domain_id}
                    onChange={(e) => setAccountForm({ ...accountForm, domain_id: parseInt(e.target.value) })}
                    className="w-full p-2 border border-border rounded-lg bg-background"
                  >
                    <option value={0}>Domain seçin</option>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>{d.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Kullanıcı Adı</label>
                  <div className="flex">
                    <input
                      type="text"
                      value={accountForm.username}
                      onChange={(e) => setAccountForm({ ...accountForm, username: e.target.value.toLowerCase() })}
                      placeholder="kullanici"
                      className="flex-1 p-2 border border-border rounded-l-lg bg-background"
                    />
                    <span className="px-3 py-2 bg-muted border border-l-0 border-border rounded-r-lg text-muted-foreground">
                      @{domains.find(d => d.id === accountForm.domain_id)?.name || 'domain.com'}
                    </span>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Şifre</label>
                  <div className="flex gap-2">
                    <div className="relative flex-1">
                      <input
                        type={showPassword ? 'text' : 'password'}
                        value={accountForm.password}
                        onChange={(e) => setAccountForm({ ...accountForm, password: e.target.value })}
                        placeholder="Şifre"
                        className="w-full p-2 pr-10 border border-border rounded-lg bg-background"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="absolute right-2 top-1/2 -translate-y-1/2"
                      >
                        {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    </div>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setAccountForm({ ...accountForm, password: generatePassword() })}
                    >
                      Oluştur
                    </Button>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Kota (MB)</label>
                  <input
                    type="number"
                    value={accountForm.quota_mb}
                    onChange={(e) => setAccountForm({ ...accountForm, quota_mb: parseInt(e.target.value) })}
                    className="w-full p-2 border border-border rounded-lg bg-background"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button variant="outline" onClick={() => setShowAddAccountModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleCreateAccount} disabled={actionLoading === 'create-account'}>
                  {actionLoading === 'create-account' && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                  Oluştur
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Add Forwarder Modal */}
        {showAddForwarderModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h3 className="text-lg font-semibold">Yönlendirme Ekle</h3>
                <button onClick={() => setShowAddForwarderModal(false)}>
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Domain</label>
                  <select
                    value={forwarderForm.domain_id}
                    onChange={(e) => setForwarderForm({ ...forwarderForm, domain_id: parseInt(e.target.value) })}
                    className="w-full p-2 border border-border rounded-lg bg-background"
                  >
                    <option value={0}>Domain seçin</option>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>{d.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Kaynak E-posta</label>
                  <div className="flex">
                    <input
                      type="text"
                      value={forwarderForm.source}
                      onChange={(e) => setForwarderForm({ ...forwarderForm, source: e.target.value.toLowerCase() })}
                      placeholder="kullanici"
                      className="flex-1 p-2 border border-border rounded-l-lg bg-background"
                    />
                    <span className="px-3 py-2 bg-muted border border-l-0 border-border rounded-r-lg text-muted-foreground">
                      @{domains.find(d => d.id === forwarderForm.domain_id)?.name || 'domain.com'}
                    </span>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Hedef E-posta</label>
                  <input
                    type="email"
                    value={forwarderForm.destination}
                    onChange={(e) => setForwarderForm({ ...forwarderForm, destination: e.target.value })}
                    placeholder="hedef@ornek.com"
                    className="w-full p-2 border border-border rounded-lg bg-background"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button variant="outline" onClick={() => setShowAddForwarderModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleCreateForwarder} disabled={actionLoading === 'create-forwarder'}>
                  {actionLoading === 'create-forwarder' && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                  Oluştur
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Add Autoresponder Modal */}
        {showAddAutoresponderModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-lg">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h3 className="text-lg font-semibold">Otomatik Yanıtlayıcı Ekle</h3>
                <button onClick={() => setShowAddAutoresponderModal(false)}>
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">E-posta Hesabı</label>
                  <select
                    value={autoresponderForm.email_account_id}
                    onChange={(e) => setAutoresponderForm({ ...autoresponderForm, email_account_id: parseInt(e.target.value) })}
                    className="w-full p-2 border border-border rounded-lg bg-background"
                  >
                    <option value={0}>E-posta hesabı seçin</option>
                    {accounts.map((a) => (
                      <option key={a.id} value={a.id}>{a.email}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Konu</label>
                  <input
                    type="text"
                    value={autoresponderForm.subject}
                    onChange={(e) => setAutoresponderForm({ ...autoresponderForm, subject: e.target.value })}
                    placeholder="Tatildeyim"
                    className="w-full p-2 border border-border rounded-lg bg-background"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Mesaj</label>
                  <textarea
                    value={autoresponderForm.body}
                    onChange={(e) => setAutoresponderForm({ ...autoresponderForm, body: e.target.value })}
                    placeholder="Şu anda ofis dışındayım..."
                    rows={4}
                    className="w-full p-2 border border-border rounded-lg bg-background resize-none"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Başlangıç Tarihi (Opsiyonel)</label>
                    <input
                      type="date"
                      value={autoresponderForm.start_date}
                      onChange={(e) => setAutoresponderForm({ ...autoresponderForm, start_date: e.target.value })}
                      className="w-full p-2 border border-border rounded-lg bg-background"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Bitiş Tarihi (Opsiyonel)</label>
                    <input
                      type="date"
                      value={autoresponderForm.end_date}
                      onChange={(e) => setAutoresponderForm({ ...autoresponderForm, end_date: e.target.value })}
                      className="w-full p-2 border border-border rounded-lg bg-background"
                    />
                  </div>
                </div>
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button variant="outline" onClick={() => setShowAddAutoresponderModal(false)}>
                  İptal
                </Button>
                <Button onClick={handleCreateAutoresponder} disabled={actionLoading === 'create-autoresponder'}>
                  {actionLoading === 'create-autoresponder' && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                  Oluştur
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Change Password Modal */}
        {showPasswordModal && selectedAccount && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-background rounded-lg shadow-xl w-full max-w-md">
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h3 className="text-lg font-semibold">Şifre Değiştir</h3>
                <button onClick={() => { setShowPasswordModal(false); setSelectedAccount(null); setNewPassword(''); }}>
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="p-4 space-y-4">
                <p className="text-muted-foreground">
                  <strong>{selectedAccount.email}</strong> için yeni şifre belirleyin
                </p>
                <div>
                  <label className="block text-sm font-medium mb-1">Yeni Şifre</label>
                  <div className="flex gap-2">
                    <div className="relative flex-1">
                      <input
                        type={showPassword ? 'text' : 'password'}
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        placeholder="Yeni şifre"
                        className="w-full p-2 pr-10 border border-border rounded-lg bg-background"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="absolute right-2 top-1/2 -translate-y-1/2"
                      >
                        {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    </div>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setNewPassword(generatePassword())}
                    >
                      Oluştur
                    </Button>
                  </div>
                </div>
              </div>
              <div className="flex justify-end gap-2 p-4 border-t border-border">
                <Button variant="outline" onClick={() => { setShowPasswordModal(false); setSelectedAccount(null); setNewPassword(''); }}>
                  İptal
                </Button>
                <Button onClick={handleChangePassword} disabled={actionLoading === 'change-password' || !newPassword}>
                  {actionLoading === 'change-password' && <RefreshCw className="w-4 h-4 mr-2 animate-spin" />}
                  Değiştir
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
