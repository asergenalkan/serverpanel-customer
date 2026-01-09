import { useEffect, useState, useCallback } from 'react';
import { accountsAPI, packagesAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import {
  Users,
  Plus,
  Trash2,
  Globe,
  Package,
  FolderOpen,
  X,
  UserCheck,
  UserX,
  AlertTriangle,
  RefreshCw,
  Eye,
  EyeOff,
} from 'lucide-react';
import Layout from '@/components/Layout';

interface Account {
  id: number;
  username: string;
  email: string;
  domain: string;
  home_dir: string;
  package_id: number;
  package_name: string;
  disk_used: number;
  disk_quota: number;
  active: boolean;
  created_at: string;
}

interface HostingPackage {
  id: number;
  name: string;
  disk_quota: number;
  bandwidth_quota: number;
  max_domains: number;
}

export default function Accounts() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [packages, setPackages] = useState<HostingPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [formData, setFormData] = useState({
    domain: '',
    username: '',
    password: '',
    passwordConfirm: '',
    email: '',
    package_id: 0,
  });
  const [usernameEdited, setUsernameEdited] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [addingAccount, setAddingAccount] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [deletingAccount, setDeletingAccount] = useState<Account | null>(null);
  const [deleteProgress, setDeleteProgress] = useState<string[]>([]);
  const [error, setError] = useState('');

  // Domain'den kullanıcı adı üret
  const generateUsername = (domain: string): string => {
    if (!domain) return '';
    // domain.com -> domain
    // sub.domain.com -> subdomain
    const parts = domain.replace(/\.[^.]+$/, '').replace(/\./g, '');
    return parts.toLowerCase().replace(/[^a-z0-9]/g, '').slice(0, 16);
  };

  // Rastgele şifre üret
  const generatePassword = (): string => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';
    for (let i = 0; i < 16; i++) {
      password += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return password;
  };

  const handleDomainChange = (domain: string) => {
    const newDomain = domain.toLowerCase();
    setFormData(prev => ({
      ...prev,
      domain: newDomain,
      username: usernameEdited ? prev.username : generateUsername(newDomain),
    }));
  };

  const fetchAccounts = async () => {
    try {
      const response = await accountsAPI.list();
      if (response.data.success) {
        setAccounts(response.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch accounts:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchPackages = async () => {
    try {
      const response = await packagesAPI.list();
      if (response.data.success) {
        setPackages(response.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch packages:', error);
    }
  };

  useEffect(() => {
    fetchAccounts();
    fetchPackages();
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

  const handleAddAccount = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // Şifre doğrulama kontrolü
    if (formData.password !== formData.passwordConfirm) {
      setError('Şifreler eşleşmiyor');
      return;
    }

    if (formData.password.length < 8) {
      setError('Şifre en az 8 karakter olmalıdır');
      return;
    }

    setAddingAccount(true);

    try {
      const response = await accountsAPI.create({
        domain: formData.domain,
        username: formData.username,
        password: formData.password,
        email: formData.email,
        package_id: formData.package_id,
      });
      if (response.data.success) {
        setShowAddModal(false);
        setFormData({ domain: '', username: '', password: '', passwordConfirm: '', email: '', package_id: 0 });
        setUsernameEdited(false);
        fetchAccounts();
      } else {
        setError(response.data.error || 'Hesap oluşturulamadı');
      }
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Hesap oluşturulurken bir hata oluştu');
    } finally {
      setAddingAccount(false);
    }
  };

  const handleDeleteAccount = async (account: Account) => {
    if (!confirm(`"${account.username}" hesabını silmek istediğinize emin misiniz?\n\nBu işlem geri alınamaz ve tüm dosyalar, veritabanları silinecektir!`)) {
      return;
    }

    setDeletingId(account.id);
    setDeletingAccount(account);
    setDeleteProgress([
      '🔄 Hesap silme işlemi başlatılıyor...',
      '⏳ Apache vhost yapılandırması siliniyor...',
      '⏳ PHP-FPM pool durduruluyor...',
      '⏳ DNS zone siliniyor...',
      '⏳ MySQL veritabanları siliniyor...',
      '⏳ Sistem kullanıcısı siliniyor...',
      '⏳ Dosyalar temizleniyor...',
    ]);
    
    try {
      const response = await accountsAPI.delete(account.id);
      if (response.data.success) {
        setDeleteProgress(prev => [...prev.slice(0, -1), '✅ Hesap başarıyla silindi!']);
        // Wait a moment to show success
        await new Promise(resolve => setTimeout(resolve, 1000));
        // Remove from list
        setAccounts(prev => prev.filter(a => a.id !== account.id));
      } else {
        setDeleteProgress(prev => [...prev, `❌ Hata: ${response.data.error || 'Bilinmeyen hata'}`]);
        await new Promise(resolve => setTimeout(resolve, 2000));
        fetchAccounts();
      }
    } catch (error: any) {
      console.error('Failed to delete account:', error);
      setDeleteProgress(prev => [...prev, `❌ Hata: ${error.response?.data?.error || 'Sunucu hatası'}`]);
      await new Promise(resolve => setTimeout(resolve, 2000));
      fetchAccounts();
    } finally {
      setDeletingId(null);
      setDeletingAccount(null);
      setDeleteProgress([]);
    }
  };

  const handleSuspend = async (id: number) => {
    try {
      await accountsAPI.suspend(id);
      fetchAccounts();
    } catch (error) {
      console.error('Failed to suspend account:', error);
    }
  };

  const handleUnsuspend = async (id: number) => {
    try {
      await accountsAPI.unsuspend(id);
      fetchAccounts();
    } catch (error) {
      console.error('Failed to unsuspend account:', error);
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
            <h1 className="text-2xl font-bold">Hosting Hesapları</h1>
            <p className="text-muted-foreground text-sm">
              Müşteri hesaplarını oluşturun ve yönetin
            </p>
          </div>
          <Button onClick={() => setShowAddModal(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Hesap Oluştur
          </Button>
        </div>

        {/* Info Banner */}
        <Card className="bg-blue-500/10 border-blue-500/20 dark:bg-blue-500/5 dark:border-blue-500/10">
          <CardContent className="p-4">
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-5 h-5 text-primary mt-0.5" />
              <div className="text-sm">
                <p className="font-medium text-foreground">Hesap Oluşturma Hakkında</p>
                <p className="text-muted-foreground mt-1">
                  Hesap oluşturduğunuzda otomatik olarak:
                </p>
                <ul className="text-muted-foreground mt-1 list-disc list-inside">
                  <li>Linux kullanıcısı oluşturulur</li>
                  <li>Home dizini ve public_html klasörü oluşturulur</li>
                  <li>Apache virtual host konfigürasyonu yapılır</li>
                  <li>Hoşgeldin sayfası oluşturulur</li>
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-500/20">
                  <Users className="w-5 h-5 text-blue-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{accounts.length}</p>
                  <p className="text-sm text-muted-foreground">Toplam Hesap</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-green-100 dark:bg-green-500/20">
                  <UserCheck className="w-5 h-5 text-green-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">
                    {accounts.filter((a) => a.active).length}
                  </p>
                  <p className="text-sm text-muted-foreground">Aktif</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-orange-100 dark:bg-orange-500/20">
                  <UserX className="w-5 h-5 text-orange-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">
                    {accounts.filter((a) => !a.active).length}
                  </p>
                  <p className="text-sm text-muted-foreground">Askıda</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Account List */}
        <Card>
          <CardHeader>
            <CardTitle>Hesap Listesi</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
              </div>
            ) : accounts.length === 0 ? (
              <div className="text-center py-8">
                <Users className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
                <p className="text-muted-foreground">Henüz hesap oluşturulmamış</p>
                <Button
                  variant="outline"
                  className="mt-4"
                  onClick={() => setShowAddModal(true)}
                >
                  <Plus className="w-4 h-4 mr-2" />
                  İlk Hesabı Oluştur
                </Button>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-3 px-4 font-medium">Kullanıcı</th>
                      <th className="text-left py-3 px-4 font-medium">Domain</th>
                      <th className="text-left py-3 px-4 font-medium">Paket</th>
                      <th className="text-left py-3 px-4 font-medium">Home Dizini</th>
                      <th className="text-left py-3 px-4 font-medium">Durum</th>
                      <th className="text-left py-3 px-4 font-medium">Oluşturulma</th>
                      <th className="text-right py-3 px-4 font-medium">İşlemler</th>
                    </tr>
                  </thead>
                  <tbody>
                    {accounts.map((account) => (
                      <tr 
                        key={account.id} 
                        className={`border-b hover:bg-muted/50 transition-opacity ${
                          deletingId === account.id ? 'opacity-50 pointer-events-none' : ''
                        }`}
                      >
                        <td className="py-3 px-4">
                          <div>
                            <p className="font-medium">{account.username}</p>
                            <p className="text-xs text-muted-foreground">{account.email}</p>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-1">
                            <Globe className="w-4 h-4 text-blue-500" />
                            <span>{account.domain || '-'}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-1">
                            <Package className="w-4 h-4 text-purple-500" />
                            <span>{account.package_name}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-1 text-sm text-muted-foreground">
                            <FolderOpen className="w-3 h-3" />
                            <span className="font-mono text-xs">{account.home_dir}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          {account.active ? (
                            <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-green-100 dark:bg-green-500/20 text-green-700 dark:text-green-400">
                              <UserCheck className="w-3 h-3" />
                              Aktif
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-orange-100 dark:bg-orange-500/20 text-orange-700 dark:text-orange-400">
                              <UserX className="w-3 h-3" />
                              Askıda
                            </span>
                          )}
                        </td>
                        <td className="py-3 px-4 text-sm text-muted-foreground">
                          {formatDate(account.created_at)}
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center justify-end gap-1">
                            {account.active ? (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-orange-500 hover:text-orange-700"
                                onClick={() => handleSuspend(account.id)}
                                title="Askıya Al"
                              >
                                <UserX className="w-4 h-4" />
                              </Button>
                            ) : (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-green-500 hover:text-green-700"
                                onClick={() => handleUnsuspend(account.id)}
                                title="Aktifleştir"
                              >
                                <UserCheck className="w-4 h-4" />
                              </Button>
                            )}
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-red-500 hover:text-red-700"
                              onClick={() => handleDeleteAccount(account)}
                              disabled={deletingId === account.id}
                              title="Sil"
                            >
                              {deletingId === account.id ? (
                                <RefreshCw className="w-4 h-4 animate-spin" />
                              ) : (
                                <Trash2 className="w-4 h-4" />
                              )}
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
      </div>

      {/* Add Account Modal */}
      {showAddModal && (
        <div 
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={closeModal}
        >
          <Card className="w-full max-w-3xl max-h-[90vh] overflow-hidden flex flex-col" onClick={(e) => e.stopPropagation()}>
            <CardHeader className="flex flex-row items-center justify-between border-b shrink-0">
              <div>
                <CardTitle>Yeni Hosting Hesabı</CardTitle>
                <p className="text-sm text-muted-foreground mt-1">Müşteri için yeni bir hosting hesabı oluşturun</p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowAddModal(false)}
              >
                <X className="w-4 h-4" />
              </Button>
            </CardHeader>
            <CardContent className="overflow-y-auto flex-1 p-6">
              <form onSubmit={handleAddAccount}>
                {error && (
                  <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm mb-6">
                    {error}
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {/* Sol Kolon */}
                  <div className="space-y-4">
                    <h3 className="font-medium text-sm text-muted-foreground uppercase tracking-wide">Hesap Bilgileri</h3>
                    
                    {/* 1. Alan Adı */}
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Alan Adı (Domain) *</label>
                      <Input
                        placeholder="ornek.com"
                        value={formData.domain}
                        onChange={(e) => handleDomainChange(e.target.value)}
                        required
                      />
                      <p className="text-xs text-muted-foreground">
                        www olmadan girin
                      </p>
                    </div>

                    {/* 2. Kullanıcı Adı */}
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Kullanıcı Adı *</label>
                      <Input
                        placeholder="ornek"
                        value={formData.username}
                        onChange={(e) => {
                          setUsernameEdited(true);
                          setFormData({ ...formData, username: e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '') });
                        }}
                        required
                      />
                      <p className="text-xs text-muted-foreground">
                        Domain'den otomatik üretilir
                      </p>
                    </div>

                    {/* 5. E-posta */}
                    <div className="space-y-2">
                      <label className="text-sm font-medium">E-posta *</label>
                      <Input
                        type="email"
                        placeholder="ornek@email.com"
                        value={formData.email}
                        onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                        required
                      />
                    </div>

                    {/* 6. Paket Seçimi */}
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Hosting Paketi *</label>
                      <select
                        className="w-full h-9 rounded-md border border-input bg-transparent px-3 text-sm"
                        value={formData.package_id}
                        onChange={(e) => setFormData({ ...formData, package_id: parseInt(e.target.value) })}
                        required
                      >
                        <option value={0}>Paket Seçin</option>
                        {packages.map((pkg) => (
                          <option key={pkg.id} value={pkg.id}>
                            {pkg.name} ({pkg.disk_quota}MB, {pkg.max_domains} Domain)
                          </option>
                        ))}
                      </select>
                    </div>
                  </div>

                  {/* Sağ Kolon */}
                  <div className="space-y-4">
                    <h3 className="font-medium text-sm text-muted-foreground uppercase tracking-wide">Güvenlik</h3>
                    
                    {/* 3. Şifre */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <label className="text-sm font-medium">Şifre *</label>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-6 text-xs"
                          onClick={() => {
                            const pwd = generatePassword();
                            setFormData({ ...formData, password: pwd, passwordConfirm: pwd });
                            setShowPassword(true);
                          }}
                        >
                          <RefreshCw className="w-3 h-3 mr-1" />
                          Üret
                        </Button>
                      </div>
                      <div className="relative">
                        <Input
                          type={showPassword ? 'text' : 'password'}
                          placeholder="En az 8 karakter"
                          value={formData.password}
                          onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                          required
                          className="pr-10"
                        />
                        <button
                          type="button"
                          className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                          onClick={() => setShowPassword(!showPassword)}
                        >
                          {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </button>
                      </div>
                    </div>

                    {/* 4. Şifre Doğrulama */}
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Şifre Doğrulama *</label>
                      <Input
                        type={showPassword ? 'text' : 'password'}
                        placeholder="Şifreyi tekrar girin"
                        value={formData.passwordConfirm}
                        onChange={(e) => setFormData({ ...formData, passwordConfirm: e.target.value })}
                        required
                      />
                      {formData.password && formData.passwordConfirm && formData.password !== formData.passwordConfirm && (
                        <p className="text-xs text-red-500">Şifreler eşleşmiyor</p>
                      )}
                      {formData.password && formData.passwordConfirm && formData.password === formData.passwordConfirm && formData.password.length >= 8 && (
                        <p className="text-xs text-green-600">✓ Şifreler eşleşiyor</p>
                      )}
                    </div>

                    {/* Önizleme */}
                    <div className="p-3 rounded-lg bg-muted text-sm mt-4">
                      <p className="font-medium mb-2">Oluşturulacaklar:</p>
                      <ul className="text-muted-foreground space-y-1 text-xs">
                        <li className="flex items-center gap-2">
                          <span className="w-2 h-2 rounded-full bg-green-500"></span>
                          Linux user: <code className="bg-background px-1 rounded">{formData.username || '...'}</code>
                        </li>
                        <li className="flex items-center gap-2">
                          <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                          Home: <code className="bg-background px-1 rounded">/home/{formData.username || '...'}</code>
                        </li>
                        <li className="flex items-center gap-2">
                          <span className="w-2 h-2 rounded-full bg-purple-500"></span>
                          Apache: <code className="bg-background px-1 rounded">{formData.domain || '...'}.conf</code>
                        </li>
                      </ul>
                    </div>
                  </div>
                </div>

                {/* Butonlar */}
                <div className="flex gap-3 pt-6 mt-6 border-t">
                  <Button
                    type="button"
                    variant="outline"
                    className="flex-1"
                    onClick={() => setShowAddModal(false)}
                  >
                    İptal
                  </Button>
                  <Button 
                    type="submit" 
                    className="flex-1" 
                    isLoading={addingAccount}
                    disabled={!formData.domain || !formData.username || !formData.password || formData.password !== formData.passwordConfirm}
                  >
                    Hesap Oluştur
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Delete Progress Modal */}
      {deletingAccount && deleteProgress.length > 0 && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
          <Card className="w-full max-w-md">
            <CardHeader className="text-center">
              <div className="mx-auto mb-4 w-16 h-16 rounded-full bg-red-100 dark:bg-red-500/20 flex items-center justify-center">
                <RefreshCw className="w-8 h-8 text-red-600 animate-spin" />
              </div>
              <CardTitle className="text-red-600">Hesap Siliniyor</CardTitle>
              <p className="text-sm text-muted-foreground">
                <strong>{deletingAccount.username}</strong> hesabı siliniyor...
              </p>
            </CardHeader>
            <CardContent>
              <div className="space-y-2 text-sm">
                {deleteProgress.map((step, index) => (
                  <div 
                    key={index}
                    className={`p-2 rounded ${
                      step.startsWith('✅') 
                        ? 'bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400'
                        : step.startsWith('❌')
                        ? 'bg-red-50 dark:bg-red-500/10 text-red-700 dark:text-red-400'
                        : step.startsWith('🔄')
                        ? 'bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400'
                        : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {step}
                  </div>
                ))}
              </div>
              <p className="text-xs text-muted-foreground text-center mt-4">
                Bu işlem birkaç saniye sürebilir, lütfen bekleyin...
              </p>
            </CardContent>
          </Card>
        </div>
      )}
    </Layout>
  );
}
