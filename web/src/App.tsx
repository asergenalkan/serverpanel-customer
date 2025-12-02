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
