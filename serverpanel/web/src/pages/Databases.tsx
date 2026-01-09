import { useEffect, useState, useCallback } from 'react';
import { databasesAPI, databaseUsersAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { SimpleModal } from '@/components/ui/Modal';
import {
  Database,
  Plus,
  Trash2,
  User,
  ExternalLink,
  Copy,
  Eye,
  EyeOff,
  RefreshCw,
  Server,
  HardDrive,
} from 'lucide-react';
import Layout from '@/components/Layout';
import { useAuth } from '@/contexts/AuthContext';

interface DatabaseItem {
  id: number;
  user_id: number;
  username?: string;
  name: string;
  type: string;
  size: number;
  created_at: string;
}

interface DatabaseUser {
  id: number;
  user_id: number;
  database_id: number;
  db_username: string;
  host: string;
  database_name: string;
  created_at: string;
}

export default function Databases() {
  const { user } = useAuth();
  const [databases, setDatabases] = useState<DatabaseItem[]>([]);
  const [dbUsers, setDbUsers] = useState<DatabaseUser[]>([]);
  const [loading, setLoading] = useState(true);
  
  // Modals
  const [showAddDB, setShowAddDB] = useState(false);
  const [showAddUser, setShowAddUser] = useState(false);
  const [showCredentials, setShowCredentials] = useState<{
    name: string;
    username: string;
    password: string;
    host: string;
  } | null>(null);
  
  // Form states
  const [newDBName, setNewDBName] = useState('');
  const [newDBPassword, setNewDBPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [selectedDB, setSelectedDB] = useState<number>(0);
  const [newUsername, setNewUsername] = useState('');
  const [newUserPassword, setNewUserPassword] = useState('');
  
  const [addingDB, setAddingDB] = useState(false);
  const [addingUser, setAddingUser] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [error, setError] = useState('');

  const fetchDatabases = async () => {
    try {
      const response = await databasesAPI.list();
      if (response.data.success) {
        setDatabases(response.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch databases:', error);
    }
  };

  const fetchDBUsers = async () => {
    try {
      const response = await databaseUsersAPI.list();
      if (response.data.success) {
        setDbUsers(response.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch database users:', error);
    }
  };

  useEffect(() => {
    Promise.all([fetchDatabases(), fetchDBUsers()]).finally(() => setLoading(false));
  }, []);

  // ESC handler for modals
  const closeAddDB = useCallback(() => {
    setShowAddDB(false);
    setNewDBName('');
    setNewDBPassword('');
    setError('');
  }, []);

  const closeAddUser = useCallback(() => {
    setShowAddUser(false);
    setSelectedDB(0);
    setNewUsername('');
    setNewUserPassword('');
    setError('');
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (showAddDB) closeAddDB();
        if (showAddUser) closeAddUser();
        if (showCredentials) setShowCredentials(null);
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [showAddDB, showAddUser, showCredentials, closeAddDB, closeAddUser]);

  const handleCreateDatabase = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setAddingDB(true);

    try {
      const response = await databasesAPI.create({
        name: newDBName,
        password: newDBPassword || undefined,
      });

      if (response.data.success) {
        // Show credentials
        setShowCredentials({
          name: response.data.data.name,
          username: response.data.data.username,
          password: response.data.data.password,
          host: response.data.data.host,
        });
        closeAddDB();
        fetchDatabases();
        fetchDBUsers();
      } else {
        setError(response.data.error || 'Veritabanı oluşturulamadı');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Veritabanı oluşturulurken hata oluştu');
    } finally {
      setAddingDB(false);
    }
  };

  const handleDeleteDatabase = async (id: number, name: string) => {
    if (!confirm(`"${name}" veritabanını silmek istediğinize emin misiniz?\n\nBu işlem geri alınamaz!`)) {
      return;
    }

    setDeletingId(id);
    try {
      const response = await databasesAPI.delete(id);
      if (response.data.success) {
        fetchDatabases();
        fetchDBUsers();
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'Silme işlemi başarısız');
    } finally {
      setDeletingId(null);
    }
  };

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setAddingUser(true);

    try {
      const response = await databaseUsersAPI.create({
        database_id: selectedDB,
        username: newUsername,
        password: newUserPassword,
      });

      if (response.data.success) {
        closeAddUser();
        fetchDBUsers();
        alert(`Kullanıcı oluşturuldu!\n\nKullanıcı Adı: ${response.data.data.username}\nHost: ${response.data.data.host}`);
      } else {
        setError(response.data.error || 'Kullanıcı oluşturulamadı');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kullanıcı oluşturulurken hata oluştu');
    } finally {
      setAddingUser(false);
    }
  };

  const handleDeleteUser = async (id: number, username: string) => {
    if (!confirm(`"${username}" kullanıcısını silmek istediğinize emin misiniz?`)) {
      return;
    }

    try {
      const response = await databaseUsersAPI.delete(id);
      if (response.data.success) {
        fetchDBUsers();
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'Silme işlemi başarısız');
    }
  };

  const openPhpMyAdmin = async (dbId: number) => {
    try {
      const response = await databasesAPI.getPhpMyAdminURL(dbId);
      if (response.data.success) {
        // Open phpMyAdmin with auto-login in new tab
        window.open(response.data.data.url, '_blank');
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'phpMyAdmin açılamadı');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const formatDate = (dateString: string): string => {
    return new Date(dateString).toLocaleDateString('tr-TR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  // Generate random password
  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';
    for (let i = 0; i < 16; i++) {
      password += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return password;
  };

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">MySQL Veritabanları</h1>
            <p className="text-muted-foreground text-sm">
              {user?.role === 'admin' ? 'Tüm veritabanlarını yönetin' : 'Veritabanlarınızı yönetin'}
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setShowAddUser(true)} disabled={databases.length === 0}>
              <User className="w-4 h-4 mr-2" />
              Kullanıcı Ekle
            </Button>
            <Button onClick={() => setShowAddDB(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Veritabanı Oluştur
            </Button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-500/20">
                  <Database className="w-5 h-5 text-blue-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{databases.length}</p>
                  <p className="text-sm text-muted-foreground">Veritabanı</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-green-100 dark:bg-green-500/20">
                  <User className="w-5 h-5 text-green-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{dbUsers.length}</p>
                  <p className="text-sm text-muted-foreground">DB Kullanıcısı</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-purple-100 dark:bg-purple-500/20">
                  <HardDrive className="w-5 h-5 text-purple-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">
                    {formatSize(databases.reduce((acc, db) => acc + db.size, 0))}
                  </p>
                  <p className="text-sm text-muted-foreground">Toplam Boyut</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Databases Table */}
        <Card>
          <CardHeader>
            <CardTitle>Veritabanları</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="w-6 h-6 animate-spin text-blue-600" />
              </div>
            ) : databases.length === 0 ? (
              <div className="text-center py-8">
                <Database className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
                <p className="text-muted-foreground">Henüz veritabanı oluşturulmamış</p>
                <Button className="mt-4" onClick={() => setShowAddDB(true)}>
                  <Plus className="w-4 h-4 mr-2" />
                  İlk Veritabanını Oluştur
                </Button>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-muted/50 border-b">
                    <tr>
                      <th className="text-left py-3 px-4 font-medium">Veritabanı</th>
                      {user?.role === 'admin' && (
                        <th className="text-left py-3 px-4 font-medium">Sahip</th>
                      )}
                      <th className="text-left py-3 px-4 font-medium">Boyut</th>
                      <th className="text-left py-3 px-4 font-medium">Oluşturulma</th>
                      <th className="text-right py-3 px-4 font-medium">İşlemler</th>
                    </tr>
                  </thead>
                  <tbody>
                    {databases.map((db) => (
                      <tr
                        key={db.id}
                        className={`border-b hover:bg-muted/50 ${
                          deletingId === db.id ? 'opacity-50' : ''
                        }`}
                      >
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <Database className="w-4 h-4 text-blue-500" />
                            <span className="font-medium font-mono">{db.name}</span>
                          </div>
                        </td>
                        {user?.role === 'admin' && (
                          <td className="py-3 px-4 text-muted-foreground">
                            {db.username || '-'}
                          </td>
                        )}
                        <td className="py-3 px-4 text-muted-foreground">
                          {formatSize(db.size)}
                        </td>
                        <td className="py-3 px-4 text-muted-foreground">
                          {formatDate(db.created_at)}
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openPhpMyAdmin(db.id)}
                              title="phpMyAdmin'de Aç"
                            >
                              <ExternalLink className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-500/10"
                              onClick={() => handleDeleteDatabase(db.id, db.name)}
                              disabled={deletingId === db.id}
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

        {/* Database Users Table */}
        {dbUsers.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Veritabanı Kullanıcıları</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-muted/50 border-b">
                    <tr>
                      <th className="text-left py-3 px-4 font-medium">Kullanıcı Adı</th>
                      <th className="text-left py-3 px-4 font-medium">Veritabanı</th>
                      <th className="text-left py-3 px-4 font-medium">Host</th>
                      <th className="text-left py-3 px-4 font-medium">Oluşturulma</th>
                      <th className="text-right py-3 px-4 font-medium">İşlemler</th>
                    </tr>
                  </thead>
                  <tbody>
                    {dbUsers.map((dbUser) => (
                      <tr key={dbUser.id} className="border-b hover:bg-muted/50">
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <User className="w-4 h-4 text-green-500" />
                            <span className="font-mono">{dbUser.db_username}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4 font-mono text-muted-foreground">
                          {dbUser.database_name}
                        </td>
                        <td className="py-3 px-4 text-muted-foreground">
                          {dbUser.host}
                        </td>
                        <td className="py-3 px-4 text-muted-foreground">
                          {formatDate(dbUser.created_at)}
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-500/10"
                              onClick={() => handleDeleteUser(dbUser.id, dbUser.db_username)}
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
            </CardContent>
          </Card>
        )}

        {/* phpMyAdmin Info */}
        <Card className="bg-blue-500/10 border-blue-500/20 dark:bg-blue-500/5 dark:border-blue-500/10">
          <CardContent className="p-4">
            <div className="flex items-start gap-3">
              <Server className="w-5 h-5 text-primary mt-0.5" />
              <div className="text-sm">
                <p className="font-medium text-foreground">phpMyAdmin Erişimi</p>
                <p className="text-muted-foreground mt-1">
                  Veritabanınızı yönetmek için phpMyAdmin'i kullanabilirsiniz. "phpMyAdmin'de Aç" butonuna tıkladığınızda
                  giriş bilgileri panoya kopyalanacaktır.
                </p>
                <p className="text-muted-foreground mt-2 font-mono text-xs">
                  Host: localhost | Port: 3306
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Create Database Modal */}
      <SimpleModal
        isOpen={showAddDB}
        onClose={closeAddDB}
        title="Yeni Veritabanı Oluştur"
      >
        <form onSubmit={handleCreateDatabase} className="space-y-4">
          {error && (
            <div className="p-3 rounded-lg bg-red-50 dark:bg-red-500/10 border border-red-200 dark:border-red-500/20 text-red-700 dark:text-red-400 text-sm">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium">Veritabanı Adı *</label>
            <div className="flex items-center">
              <span className="px-3 py-2 bg-muted rounded-l-lg border border-r-0 text-sm text-muted-foreground">
                {user?.username}_
              </span>
              <Input
                className="rounded-l-none"
                placeholder="veritabani_adi"
                value={newDBName}
                onChange={(e) => setNewDBName(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
                required
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Sadece küçük harf, rakam ve alt çizgi kullanın
            </p>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Şifre (Opsiyonel)</label>
            <div className="relative">
              <Input
                type={showPassword ? 'text' : 'password'}
                placeholder="Boş bırakılırsa otomatik oluşturulur"
                value={newDBPassword}
                onChange={(e) => setNewDBPassword(e.target.value)}
              />
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setNewDBPassword(generatePassword())}
            >
              <RefreshCw className="w-3 h-3 mr-1" />
              Şifre Oluştur
            </Button>
          </div>

          <div className="p-3 rounded-lg bg-muted text-sm">
            <p className="font-medium mb-2">Oluşturulacaklar:</p>
            <ul className="text-muted-foreground space-y-1 text-xs">
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                Veritabanı: <code className="bg-background px-1 rounded">{user?.username}_{newDBName || '...'}</code>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-green-500"></span>
                Kullanıcı: <code className="bg-background px-1 rounded">{user?.username}_{newDBName || '...'}</code>
              </li>
            </ul>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button type="button" variant="ghost" onClick={closeAddDB}>
              İptal
            </Button>
            <Button type="submit" isLoading={addingDB}>
              <Plus className="w-4 h-4 mr-2" />
              Oluştur
            </Button>
          </div>
        </form>
      </SimpleModal>

      {/* Create User Modal */}
      <SimpleModal
        isOpen={showAddUser}
        onClose={closeAddUser}
        title="Veritabanı Kullanıcısı Ekle"
      >
        <form onSubmit={handleCreateUser} className="space-y-4">
          {error && (
            <div className="p-3 rounded-lg bg-red-50 dark:bg-red-500/10 border border-red-200 dark:border-red-500/20 text-red-700 dark:text-red-400 text-sm">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium">Veritabanı *</label>
            <select
              className="w-full px-3 py-2 rounded-lg border bg-background"
              value={selectedDB}
              onChange={(e) => setSelectedDB(Number(e.target.value))}
              required
            >
              <option value={0}>Seçin...</option>
              {databases.map((db) => (
                <option key={db.id} value={db.id}>
                  {db.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Kullanıcı Adı *</label>
            <div className="flex items-center">
              <span className="px-3 py-2 bg-muted rounded-l-lg border border-r-0 text-sm text-muted-foreground">
                {user?.username}_
              </span>
              <Input
                className="rounded-l-none"
                placeholder="kullanici"
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
                required
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Şifre *</label>
            <div className="relative">
              <Input
                type={showPassword ? 'text' : 'password'}
                placeholder="Güçlü bir şifre girin"
                value={newUserPassword}
                onChange={(e) => setNewUserPassword(e.target.value)}
                required
              />
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setNewUserPassword(generatePassword())}
            >
              <RefreshCw className="w-3 h-3 mr-1" />
              Şifre Oluştur
            </Button>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button type="button" variant="ghost" onClick={closeAddUser}>
              İptal
            </Button>
            <Button type="submit" isLoading={addingUser} disabled={!selectedDB}>
              <Plus className="w-4 h-4 mr-2" />
              Kullanıcı Ekle
            </Button>
          </div>
        </form>
      </SimpleModal>

      {/* Credentials Modal */}
      {showCredentials && (
        <div 
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
          onClick={() => setShowCredentials(null)}
        >
          <div 
            className="bg-card rounded-lg p-6 w-full max-w-md shadow-xl border"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-semibold mb-4 text-green-600">
              ✅ Veritabanı Oluşturuldu!
            </h3>
            
            <div className="space-y-3 bg-muted p-4 rounded-lg font-mono text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Veritabanı:</span>
                <div className="flex items-center gap-2">
                  <span>{showCredentials.name}</span>
                  <button onClick={() => copyToClipboard(showCredentials.name)}>
                    <Copy className="w-3 h-3 text-muted-foreground hover:text-foreground" />
                  </button>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Kullanıcı:</span>
                <div className="flex items-center gap-2">
                  <span>{showCredentials.username}</span>
                  <button onClick={() => copyToClipboard(showCredentials.username)}>
                    <Copy className="w-3 h-3 text-muted-foreground hover:text-foreground" />
                  </button>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Şifre:</span>
                <div className="flex items-center gap-2">
                  <span className="text-green-600">{showCredentials.password}</span>
                  <button onClick={() => copyToClipboard(showCredentials.password)}>
                    <Copy className="w-3 h-3 text-muted-foreground hover:text-foreground" />
                  </button>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Host:</span>
                <span>{showCredentials.host}</span>
              </div>
            </div>

            <div className="mt-4 p-3 bg-orange-50 dark:bg-orange-500/10 border border-orange-200 dark:border-orange-500/20 rounded-lg">
              <p className="text-sm text-orange-700 dark:text-orange-400">
                ⚠️ Bu bilgileri güvenli bir yere kaydedin. Şifre tekrar gösterilmeyecektir!
              </p>
            </div>

            <div className="flex justify-end gap-2 mt-4">
              <Button
                variant="outline"
                onClick={() => {
                  const text = `Veritabanı: ${showCredentials.name}\nKullanıcı: ${showCredentials.username}\nŞifre: ${showCredentials.password}\nHost: ${showCredentials.host}`;
                  copyToClipboard(text);
                }}
              >
                <Copy className="w-4 h-4 mr-2" />
                Tümünü Kopyala
              </Button>
              <Button onClick={() => setShowCredentials(null)}>
                Tamam
              </Button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}
