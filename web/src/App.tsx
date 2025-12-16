import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';
import { ThemeProvider } from '@/contexts/ThemeContext';
import Login from '@/pages/Login';
import Dashboard from '@/pages/Dashboard';
import Domains from '@/pages/Domains';
import Accounts from '@/pages/Accounts';
import FileManager from '@/pages/FileManager';
import Databases from '@/pages/Databases';
import SSL from '@/pages/SSL';
import PHPSettings from '@/pages/PHPSettings';
import FTPAccounts from '@/pages/FTPAccounts';
import DNSZoneEditor from '@/pages/DNSZoneEditor';
import Packages from '@/pages/Packages';
import DomainManager from '@/pages/DomainManager';
import Email from '@/pages/Email';
import ServerInfo from '@/pages/server/ServerInfo';
import DailyLog from '@/pages/server/DailyLog';
import TopProcesses from '@/pages/server/TopProcesses';
import TaskQueue from '@/pages/server/TaskQueue';
import SoftwareManager from '@/pages/SoftwareManager';
import ServerSettings from '@/pages/ServerSettings';
import ServerFeatures from '@/pages/ServerFeatures';
import SpamFilters from '@/pages/SpamFilters';
import CronJobs from '@/pages/CronJobs';
import BackgroundKiller from '@/pages/system/BackgroundKiller';
import ProcessManager from '@/pages/system/ProcessManager';
import DiskUsage from '@/pages/system/DiskUsage';
import RunningProcesses from '@/pages/system/RunningProcesses';
import Fail2ban from '@/pages/security/Fail2ban';
import Firewall from '@/pages/security/Firewall';
import SSHSecurity from '@/pages/security/SSHSecurity';
import ModSecurity from '@/pages/security/ModSecurity';
import Terminal from '@/pages/Terminal';
import NodejsApps from '@/pages/NodejsApps';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (user.role !== 'admin') {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (user) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}

function AppRoutes() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <PublicRoute>
            <Login />
          </PublicRoute>
        }
      />
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <Dashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/domains"
        element={
          <ProtectedRoute>
            <Domains />
          </ProtectedRoute>
        }
      />
      <Route
        path="/accounts"
        element={
          <AdminRoute>
            <Accounts />
          </AdminRoute>
        }
      />
      <Route
        path="/files"
        element={
          <ProtectedRoute>
            <FileManager />
          </ProtectedRoute>
        }
      />
      <Route
        path="/databases"
        element={
          <ProtectedRoute>
            <Databases />
          </ProtectedRoute>
        }
      />
      <Route
        path="/ssl"
        element={
          <ProtectedRoute>
            <SSL />
          </ProtectedRoute>
        }
      />
      <Route
        path="/php"
        element={
          <ProtectedRoute>
            <PHPSettings />
          </ProtectedRoute>
        }
      />
      <Route
        path="/ftp"
        element={
          <ProtectedRoute>
            <FTPAccounts />
          </ProtectedRoute>
        }
      />
      <Route
        path="/dns"
        element={
          <ProtectedRoute>
            <DNSZoneEditor />
          </ProtectedRoute>
        }
      />
      <Route
        path="/packages"
        element={
          <ProtectedRoute>
            <Packages />
          </ProtectedRoute>
        }
      />
      <Route
        path="/domain-manager"
        element={
          <ProtectedRoute>
            <DomainManager />
          </ProtectedRoute>
        }
      />
      <Route
        path="/email"
        element={
          <ProtectedRoute>
            <Email />
          </ProtectedRoute>
        }
      />
      {/* Server Status Routes (Admin Only) */}
      <Route
        path="/server/info"
        element={
          <AdminRoute>
            <ServerInfo />
          </AdminRoute>
        }
      />
      <Route
        path="/server/daily-log"
        element={
          <AdminRoute>
            <DailyLog />
          </AdminRoute>
        }
      />
      <Route
        path="/server/processes"
        element={
          <AdminRoute>
            <TopProcesses />
          </AdminRoute>
        }
      />
      <Route
        path="/server/queue"
        element={
          <AdminRoute>
            <TaskQueue />
          </AdminRoute>
        }
      />
      <Route
        path="/software"
        element={
          <AdminRoute>
            <SoftwareManager />
          </AdminRoute>
        }
      />
      <Route
        path="/settings/server"
        element={
          <AdminRoute>
            <ServerSettings />
          </AdminRoute>
        }
      />
      <Route
        path="/server/features"
        element={
          <ProtectedRoute>
            <ServerFeatures />
          </ProtectedRoute>
        }
      />
      <Route
        path="/spam-filters"
        element={
          <ProtectedRoute>
            <SpamFilters />
          </ProtectedRoute>
        }
      />
      <Route
        path="/cron"
        element={
          <ProtectedRoute>
            <CronJobs />
          </ProtectedRoute>
        }
      />
      <Route
        path="/system/background-killer"
        element={
          <AdminRoute>
            <BackgroundKiller />
          </AdminRoute>
        }
      />
      <Route
        path="/system/process-manager"
        element={
          <AdminRoute>
            <ProcessManager />
          </AdminRoute>
        }
      />
      <Route
        path="/system/disk-usage"
        element={
          <AdminRoute>
            <DiskUsage />
          </AdminRoute>
        }
      />
      <Route
        path="/system/running-processes"
        element={
          <AdminRoute>
            <RunningProcesses />
          </AdminRoute>
        }
      />
      <Route
        path="/security/fail2ban"
        element={
          <AdminRoute>
            <Fail2ban />
          </AdminRoute>
        }
      />
      <Route
        path="/security/firewall"
        element={
          <AdminRoute>
            <Firewall />
          </AdminRoute>
        }
      />
      <Route
        path="/security/ssh"
        element={
          <AdminRoute>
            <SSHSecurity />
          </AdminRoute>
        }
      />
      <Route
        path="/security/modsecurity"
        element={
          <AdminRoute>
            <ModSecurity />
          </AdminRoute>
        }
      />
      <Route
        path="/terminal"
        element={
          <ProtectedRoute>
            <Terminal />
          </ProtectedRoute>
        }
      />
      <Route
        path="/nodejs"
        element={
          <ProtectedRoute>
            <NodejsApps />
          </ProtectedRoute>
        }
      />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
}
