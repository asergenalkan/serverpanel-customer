import { ReactNode, useState, useRef, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { systemAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { ThemeDropdown } from '@/components/ui/ThemeDropdown';
import {
  Server,
  LayoutDashboard,
  Globe,
  Database,
  Mail,
  Users,
  Package,
  Settings,
  LogOut,
  FolderOpen,
  Shield,
  HardDrive,
  Clock,
  Code,
  Upload,
  Globe2,
  Layers,
  Activity,
  ChevronDown,
  ChevronRight,
  Cpu,
  FileText,
  ListTodo,
  Skull,
  Terminal,
  HeartPulse,
  Lock,
  ShieldCheck,
  Key,
  Box,
  User,
} from 'lucide-react';

interface LayoutProps {
  children: ReactNode;
}

// User menu items (for hosting users)
const userMenuItems = [
  { icon: LayoutDashboard, label: 'Dashboard', href: '/dashboard' },
  { icon: Layers, label: 'Domain Yönetimi', href: '/domain-manager' },
  { icon: FolderOpen, label: 'Dosya Yöneticisi', href: '/files' },
  { icon: Database, label: 'Veritabanları', href: '/databases' },
  { icon: Upload, label: 'FTP Hesapları', href: '/ftp' },
  { icon: Shield, label: 'SSL/TLS', href: '/ssl' },
  { icon: Code, label: 'PHP Ayarları', href: '/php' },
  { icon: Globe2, label: 'DNS Zone Editor', href: '/dns' },
  { icon: Mail, label: 'E-posta', href: '/email' },
  { icon: Shield, label: 'Spam Filtreleri', href: '/spam-filters' },
  { icon: Server, label: 'Sunucu Özellikleri', href: '/server/features' },
  { icon: HardDrive, label: 'Backup', href: '/backup', disabled: true },
  { icon: Clock, label: 'Cron Jobs', href: '/cron' },
  { icon: Box, label: 'Node.js Uygulamaları', href: '/nodejs' },
  { icon: Terminal, label: 'Terminal', href: '/terminal' },
];

// Admin menu items
const adminMenuItems = [
  { icon: LayoutDashboard, label: 'Dashboard', href: '/dashboard' },
  { icon: Users, label: 'Hosting Hesapları', href: '/accounts' },
  { icon: Layers, label: 'Domain Yönetimi', href: '/domain-manager' },
  { icon: Globe, label: 'Tüm Domainler', href: '/domains' },
  { icon: Database, label: 'Veritabanları', href: '/databases' },
  { icon: Upload, label: 'FTP Hesapları', href: '/ftp' },
  { icon: Shield, label: 'SSL/TLS', href: '/ssl' },
  { icon: Code, label: 'PHP Ayarları', href: '/php' },
  { icon: Globe2, label: 'DNS Zone Editor', href: '/dns' },
  { icon: Mail, label: 'E-posta', href: '/email' },
  { icon: Shield, label: 'Spam Filtreleri', href: '/spam-filters' },
  { icon: Clock, label: 'Cron Jobs', href: '/cron' },
  { icon: Box, label: 'Node.js Uygulamaları', href: '/nodejs' },
  { icon: Terminal, label: 'Terminal', href: '/terminal' },
  { icon: Package, label: 'Paketler', href: '/packages' },
  { icon: Settings, label: 'Sunucu Ayarları', href: '/settings/server' },
];

// Server status submenu items (admin only)
const serverStatusItems = [
  { icon: Cpu, label: 'Sunucu Bilgileri', href: '/server/info' },
  { icon: FileText, label: 'Günlük İşlem Günlüğü', href: '/server/daily-log' },
  { icon: ListTodo, label: 'Task Queue', href: '/server/queue' },
];

// System Health submenu items (admin only)
const systemHealthItems = [
  { icon: Skull, label: 'Arka Plan İşlem Sonlandırıcı', href: '/system/background-killer' },
  { icon: Activity, label: 'İşlem Yöneticisi', href: '/system/process-manager' },
  { icon: HardDrive, label: 'Geçerli Disk Kullanımı', href: '/system/disk-usage' },
  { icon: Terminal, label: 'Geçerli Çalışma İşlemleri', href: '/system/running-processes' },
];

// Software manager menu item (admin only)
const softwareMenuItem = { icon: Package, label: 'Yazılım Yöneticisi', href: '/software' };

// Security submenu items (admin only)
const securityItems = [
  { icon: ShieldCheck, label: 'Fail2ban Yönetimi', href: '/security/fail2ban' },
  { icon: Shield, label: 'Firewall (UFW)', href: '/security/firewall' },
  { icon: Key, label: 'SSH Güvenliği', href: '/security/ssh' },
  { icon: Shield, label: 'ModSecurity WAF', href: '/security/modsecurity' },
];

// TopBar Component
function TopBar() {
  const navigate = useNavigate();
  const [loadAverage, setLoadAverage] = useState<number[]>([0, 0, 0]);

  useEffect(() => {
    const fetchLoadAverage = async () => {
      try {
        const response = await systemAPI.getStats();
        if (response.data?.load_average) {
          setLoadAverage(response.data.load_average);
        }
      } catch (err) {
        // Ignore errors
      }
    };

    fetchLoadAverage();
    const interval = setInterval(fetchLoadAverage, 30000); // Her 30 saniyede bir güncelle
    return () => clearInterval(interval);
  }, []);

  const getLoadColor = (load: number) => {
    if (load < 1) return 'text-green-500';
    if (load < 2) return 'text-yellow-500';
    return 'text-red-500';
  };

  return (
    <div className="h-14 border-b border-[var(--color-sidebar-border)] bg-[var(--color-sidebar)] flex items-center justify-between px-6">
      <div className="flex items-center gap-4">
        {/* Breadcrumb veya sayfa başlığı buraya eklenebilir */}
      </div>
      
      <button
        onClick={() => navigate('/system/process-manager')}
        className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-accent hover:text-accent-foreground transition-colors cursor-pointer"
        title="İşlem Yöneticisi'ni aç"
      >
        <Activity className="w-4 h-4 text-muted-foreground" />
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">Load Average:</span>
          <span className={`font-mono font-medium ${getLoadColor(loadAverage[0])}`}>
            {loadAverage[0]?.toFixed(2) || '0.00'}
          </span>
          <span className="text-muted-foreground">,</span>
          <span className={`font-mono font-medium ${getLoadColor(loadAverage[1])}`}>
            {loadAverage[1]?.toFixed(2) || '0.00'}
          </span>
          <span className="text-muted-foreground">,</span>
          <span className={`font-mono font-medium ${getLoadColor(loadAverage[2])}`}>
            {loadAverage[2]?.toFixed(2) || '0.00'}
          </span>
        </div>
      </button>
    </div>
  );
}

export default function Layout({ children }: LayoutProps) {
  const { user, logout } = useAuth();
  const location = useLocation();
  const [serverStatusOpen, setServerStatusOpen] = useState(false);
  const [systemHealthOpen, setSystemHealthOpen] = useState(false);
  const [securityOpen, setSecurityOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);

  // Close user menu when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (userMenuRef.current && !userMenuRef.current.contains(event.target as Node)) {
        setUserMenuOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Check if current path is in server status section
  const isServerStatusActive = location.pathname.startsWith('/server/');
  const isSystemHealthActive = location.pathname.startsWith('/system/');
  const isSecurityActive = location.pathname.startsWith('/security/');

  return (
    <div className="min-h-screen bg-[var(--color-page-bg)] transition-colors">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 h-full w-64 bg-[var(--color-sidebar)] border-r border-[var(--color-sidebar-border)] z-50 flex flex-col transition-colors">
        {/* Logo */}
        <div className="h-16 flex items-center justify-between px-6 border-b border-[var(--color-sidebar-border)]">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center">
              <Server className="w-5 h-5 text-white" />
            </div>
            <span className="font-bold text-lg">EticPanel</span>
          </div>
          {/* Theme Dropdown */}
          <ThemeDropdown />
        </div>

        {/* Menu */}
        <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
          {/* Admin Menu */}
          {user?.role === 'admin' ? (
            <>
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-4 mb-2">
                Yönetim
              </p>
              {adminMenuItems.map((item) => (
                <Link
                  key={item.href}
                  to={item.href}
                  className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    location.pathname === item.href
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  <item.icon className="w-5 h-5" />
                  {item.label}
                </Link>
              ))}

              {/* Server Status Dropdown */}
              <div className="mt-4">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-4 mb-2">
                  Sunucu
                </p>
                <button
                  onClick={() => setServerStatusOpen(!serverStatusOpen)}
                  className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    isServerStatusActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  <Activity className="w-5 h-5" />
                  Sunucu Durumu
                  {serverStatusOpen || isServerStatusActive ? (
                    <ChevronDown className="w-4 h-4 ml-auto" />
                  ) : (
                    <ChevronRight className="w-4 h-4 ml-auto" />
                  )}
                </button>
                {(serverStatusOpen || isServerStatusActive) && (
                  <div className="ml-4 mt-1 space-y-1">
                    {serverStatusItems.map((item) => (
                      <Link
                        key={item.href}
                        to={item.href}
                        className={`w-full flex items-center gap-3 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                          location.pathname === item.href
                            ? 'bg-primary/10 text-primary'
                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                        }`}
                      >
                        <item.icon className="w-4 h-4" />
                        {item.label}
                      </Link>
                    ))}
                  </div>
                )}
              </div>

              {/* System Health Dropdown */}
              <div className="mt-2">
                <button
                  onClick={() => setSystemHealthOpen(!systemHealthOpen)}
                  className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    isSystemHealthActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  <HeartPulse className="w-5 h-5" />
                  Sistem Sağlığı
                  {systemHealthOpen || isSystemHealthActive ? (
                    <ChevronDown className="w-4 h-4 ml-auto" />
                  ) : (
                    <ChevronRight className="w-4 h-4 ml-auto" />
                  )}
                </button>
                {(systemHealthOpen || isSystemHealthActive) && (
                  <div className="ml-4 mt-1 space-y-1">
                    {systemHealthItems.map((item) => (
                      <Link
                        key={item.href}
                        to={item.href}
                        className={`w-full flex items-center gap-3 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                          location.pathname === item.href
                            ? 'bg-primary/10 text-primary'
                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                        }`}
                      >
                        <item.icon className="w-4 h-4" />
                        {item.label}
                      </Link>
                    ))}
                  </div>
                )}
              </div>

              {/* Security Dropdown */}
              <div className="mt-2">
                <button
                  onClick={() => setSecurityOpen(!securityOpen)}
                  className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    isSecurityActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  <Lock className="w-5 h-5" />
                  Güvenlik
                  {securityOpen || isSecurityActive ? (
                    <ChevronDown className="w-4 h-4 ml-auto" />
                  ) : (
                    <ChevronRight className="w-4 h-4 ml-auto" />
                  )}
                </button>
                {(securityOpen || isSecurityActive) && (
                  <div className="ml-4 mt-1 space-y-1">
                    {securityItems.map((item) => (
                      <Link
                        key={item.href}
                        to={item.href}
                        className={`w-full flex items-center gap-3 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                          location.pathname === item.href
                            ? 'bg-primary/10 text-primary'
                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                        }`}
                      >
                        <item.icon className="w-4 h-4" />
                        {item.label}
                      </Link>
                    ))}
                  </div>
                )}
              </div>

              {/* Software Manager */}
              <Link
                to={softwareMenuItem.href}
                className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  location.pathname === softwareMenuItem.href
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                }`}
              >
                <softwareMenuItem.icon className="w-5 h-5" />
                {softwareMenuItem.label}
              </Link>
            </>
          ) : (
            <>
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-4 mb-2">
                Hesabım
              </p>
              {userMenuItems.map((item) => (
                <Link
                  key={item.href}
                  to={item.disabled ? '#' : item.href}
                  className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    location.pathname === item.href
                      ? 'bg-primary/10 text-primary'
                      : item.disabled
                      ? 'text-muted-foreground/50 cursor-not-allowed'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                  onClick={(e) => item.disabled && e.preventDefault()}
                >
                  <item.icon className="w-5 h-5" />
                  {item.label}
                  {item.disabled && (
                    <span className="ml-auto text-xs bg-muted text-muted-foreground px-1.5 py-0.5 rounded">
                      Yakında
                    </span>
                  )}
                </Link>
              ))}
            </>
          )}
        </nav>

        {/* User */}
        <div className="p-4 border-t border-[var(--color-sidebar-border)]" ref={userMenuRef}>
          <div className="flex items-center justify-between relative">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex items-center gap-2 hover:bg-muted/80 p-2 pr-3 rounded-lg transition-colors duration-200"
            >
              <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                <User className="w-4 h-4 text-primary" />
              </div>
              <div className="text-left">
                <p className="text-sm font-medium">{user?.username}</p>
                <p className="text-xs text-muted-foreground capitalize">{user?.role}</p>
              </div>
            </button>
            <Button variant="ghost" size="icon" onClick={logout} title="Çıkış Yap">
              <LogOut className="w-5 h-5" />
            </Button>

            {/* Profile Dropdown */}
            {userMenuOpen && (
              <div className="absolute bottom-full left-0 mb-2 min-w-[160px] overflow-hidden rounded-lg border bg-background shadow-lg z-50">
                <div className="p-1">
                  <Link
                    to="/profile"
                    onClick={() => setUserMenuOpen(false)}
                    className="flex w-full cursor-pointer items-center rounded-md px-3 py-2 text-sm transition-colors duration-150 hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-muted-foreground/20"
                  >
                    <Settings className="mr-3 h-4 w-4 text-muted-foreground" />
                    <span className="flex-1 text-left">Profil Ayarları</span>
                  </Link>
                </div>
              </div>
            )}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="pl-64">
        {/* Top Bar */}
        <TopBar />
        <div className="p-8">{children}</div>
      </main>
    </div>
  );
}
