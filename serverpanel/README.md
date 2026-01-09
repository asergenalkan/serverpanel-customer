# ServerPanel

Modern, tek port üzerinden çalışan sunucu yönetim paneli. cPanel/WHM ve Plesk benzeri özellikler sunar.

## Özellikler

- 🔐 **Rol Tabanlı Erişim**: Admin, Reseller, User rolleri
- 🌐 **Domain Yönetimi**: Website ve DNS yönetimi
- 🗄️ **Veritabanı Yönetimi**: MySQL/MariaDB/PostgreSQL desteği
- 📧 **E-posta Yönetimi**: Mail hesapları ve alias'lar
- 📦 **Paket Yönetimi**: Hosting paketleri oluşturma
- 📊 **Sistem İzleme**: CPU, RAM, Disk kullanımı
- 🔧 **Servis Yönetimi**: Nginx, Apache, MySQL vs. kontrol

## Teknoloji Stack

### Backend
- **Go** (Fiber framework)
- **SQLite** (panel veritabanı)
- **JWT** authentication

### Frontend
- **React** + **TypeScript**
- **Vite** build tool
- **TailwindCSS** styling
- **Lucide** icons

## Kurulum

### Gereksinimler
- Go 1.21+
- Node.js 20+
- npm veya yarn

### Backend

```bash
# Bağımlılıkları indir
go mod tidy

# Derle
go build -o serverpanel ./cmd/panel

# Çalıştır
./serverpanel
```

### Frontend

```bash
cd web

# Bağımlılıkları indir
npm install

# Geliştirme sunucusu
npm run dev

# Üretim build
npm run build
```

## Kullanım

1. Backend'i başlat: `./serverpanel` (Port: 8443)
2. Frontend'i başlat: `cd web && npm run dev` (Port: 3000)
3. Tarayıcıda aç: http://localhost:3000
4. Giriş yap:
   - **Kullanıcı**: admin
   - **Şifre**: admin123

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - Giriş
- `GET /api/v1/auth/me` - Mevcut kullanıcı
- `POST /api/v1/auth/logout` - Çıkış

### Dashboard
- `GET /api/v1/dashboard/stats` - İstatistikler

### Users (Admin)
- `GET /api/v1/users` - Kullanıcı listesi
- `POST /api/v1/users` - Kullanıcı oluştur
- `PUT /api/v1/users/:id` - Kullanıcı güncelle
- `DELETE /api/v1/users/:id` - Kullanıcı sil

### Domains
- `GET /api/v1/domains` - Domain listesi
- `POST /api/v1/domains` - Domain ekle
- `DELETE /api/v1/domains/:id` - Domain sil

### Databases
- `GET /api/v1/databases` - Veritabanı listesi
- `POST /api/v1/databases` - Veritabanı oluştur
- `DELETE /api/v1/databases/:id` - Veritabanı sil

### System (Admin)
- `GET /api/v1/system/stats` - Sistem istatistikleri
- `GET /api/v1/system/services` - Servis durumları
- `POST /api/v1/system/services/:name/restart` - Servis yeniden başlat

## Proje Yapısı

```
whm-clone/
├── cmd/panel/          # Ana uygulama
│   └── main.go
├── internal/           # Backend modülleri
│   ├── api/           # HTTP handlers
│   ├── auth/          # JWT authentication
│   ├── config/        # Konfigürasyon
│   ├── database/      # SQLite işlemleri
│   ├── middleware/    # Fiber middleware
│   ├── models/        # Veri modelleri
│   └── system/        # Sistem komutları
├── web/               # React frontend
│   ├── src/
│   │   ├── components/  # UI componentleri
│   │   ├── contexts/    # React contexts
│   │   ├── hooks/       # Custom hooks
│   │   ├── lib/         # Utilities
│   │   └── pages/       # Sayfa componentleri
│   └── ...
├── configs/           # Konfigürasyon şablonları
└── scripts/           # Yardımcı scriptler
```

## Lisans

MIT
