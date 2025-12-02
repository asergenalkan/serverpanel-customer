import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/Card';
import { Server, Lock, User } from 'lucide-react';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      await login(username, password);
      navigate('/dashboard');
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Giriş başarısız. Lütfen bilgilerinizi kontrol edin.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col bg-[var(--color-page-bg)] px-4 py-6">
      <div className="flex-1 flex items-center justify-center">
        <div className="w-full max-w-md space-y-6">
          <div className="flex items-center justify-center flex-col gap-3 text-center">
            <div className="inline-flex items-center justify-center w-14 h-14 rounded-xl bg-blue-600 shadow-sm">
              <Server className="w-7 h-7 text-white" />
            </div>
            <div className="space-y-1">
              <h1 className="text-2xl font-semibold tracking-tight">EticPanel</h1>
              <p className="text-sm text-muted-foreground">
                Hosting ve sunucu yönetiminiz için modern kontrol paneli
              </p>
            </div>
          </div>

          <Card className="shadow-lg">
            <CardHeader className="space-y-1 pb-4">
              <CardTitle className="text-xl text-center">Panele Giriş Yap</CardTitle>
              <CardDescription className="text-center">
                Hesabınıza erişmek için bilgilerinizi girin
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="p-3 rounded-md border border-destructive/40 bg-destructive/5 text-destructive text-sm">
                  {error}
                </div>
              )}

              <div className="space-y-1.5">
                <label className="text-sm font-medium flex items-center gap-1.5">
                  <User className="w-4 h-4 text-muted-foreground" />
                  Kullanıcı Adı
                </label>
                <Input
                  type="text"
                  placeholder="admin"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  required
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium flex items-center gap-1.5">
                  <Lock className="w-4 h-4 text-muted-foreground" />
                  Şifre
                </label>
                <Input
                  type="password"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                />
              </div>

              <Button type="submit" className="w-full" isLoading={isLoading}>
                Giriş Yap
              </Button>
            </form>

            <div className="mt-6 pt-4 border-t border-border">
              <p className="text-xs text-center text-muted-foreground">
                Varsayılan giriş bilgileri yalnızca demo amaçlıdır.
              </p>
              <p className="text-xs text-center text-muted-foreground mt-1">
                <span className="font-mono font-medium">admin / admin123</span>
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
      </div>

      <div className="w-full max-w-5xl mx-auto mt-4 flex items-center justify-between text-xs text-muted-foreground">
        <span>
          © {new Date().getFullYear()} EticWeb
        </span>
        <div className="flex items-center gap-4">
          <a
            href="https://eticweb.com.tr"
            target="_blank"
            rel="noreferrer"
            className="hover:text-foreground transition-colors"
          >
            Dokümantasyon
          </a>
          <a
            href="https://eticweb.com.tr"
            target="_blank"
            rel="noreferrer"
            className="hover:text-foreground transition-colors"
          >
            Destek
          </a>
        </div>
      </div>
    </div>
  );
}
