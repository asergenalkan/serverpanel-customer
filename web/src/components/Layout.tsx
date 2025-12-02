import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { useTheme } from '@/contexts/ThemeContext';
import { Button } from '@/components/ui/Button';
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
  Sun,
  Moon,
  Code,
  Upload,
  Globe2,
  Layers,
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
  { icon: HardDrive, label: 'Backup', href: '/backup', disabled: true },
  { icon: Clock, label: 'Cron Jobs', href: '/cron', disabled: true },
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
  { icon: Package, label: 'Paketler', href: '/packages' },
  { icon: Settings, label: 'Ayarlar', href: '/settings', disabled: true },
];

export default function Layout({ children }: LayoutProps) {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const location = useLocation();

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
          {/* Theme Toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            title={theme === 'light' ? 'Gece Modu' : 'Gündüz Modu'}
            className="w-8 h-8"
          >
            {theme === 'light' ? (
              <Moon className="w-4 h-4" />
            ) : (
              <Sun className="w-4 h-4 text-yellow-400" />
            )}
          </Button>
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
        <div className="p-4 border-t border-[var(--color-sidebar-border)]">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">{user?.username}</p>
              <p className="text-xs text-muted-foreground capitalize">{user?.role}</p>
            </div>
            <Button variant="ghost" size="icon" onClick={logout} title="Çıkış Yap">
              <LogOut className="w-5 h-5" />
            </Button>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="pl-64">
        <div className="p-8">{children}</div>
      </main>
    </div>
  );
}
