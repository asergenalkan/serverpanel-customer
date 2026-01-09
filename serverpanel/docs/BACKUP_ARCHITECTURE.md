# ServerPanel Backup & Migration Mimarisi

## 📋 Genel Bakış

Bu döküman, ServerPanel için tasarlanan kapsamlı backup ve migration sisteminin teknik mimarisini tanımlar. Sistem, cPanel/WHM, Plesk ve DirectAdmin'den daha iyi bir deneyim sunmayı hedefler.

---

## 🎯 Tasarım Hedefleri

1. **cPanel Uyumluluğu**: cPanel backup formatını okuyabilme (migration için)
2. **Granüler Restore**: Tek dosya, tek veritabanı, tek email restore
3. **Incremental Backup**: Sadece değişenleri yedekleme (disk/zaman tasarrufu)
4. **Multi-Destination**: Local, Remote FTP, S3, B2, Google Cloud
5. **Şifreleme**: AES-256 client-side encryption
6. **Kullanıcı Self-Service**: Müşteriler kendi backup'larını yönetebilmeli
7. **Sıfır Downtime**: Backup sırasında servis kesintisi olmamalı

---

## 👥 Admin vs Müşteri Yetkileri

### Admin (WHM Benzeri)

| Özellik | Açıklama |
|---------|----------|
| **Sunucu Geneli Backup** | Tüm hesapları tek seferde yedekleme |
| **Backup Politikaları** | Retention, zamanlama, hedef ayarları |
| **Remote Storage Yönetimi** | S3, FTP, B2 bağlantıları |
| **Müşteri Backup Limitleri** | Paket bazlı backup kotası |
| **Forced Backup** | Belirli hesapları zorla yedekleme |
| **Migration Import** | cPanel/Plesk/DA'dan hesap aktarma |
| **Disaster Recovery** | Tam sunucu restore |
| **Backup Monitoring** | Tüm backup job'ların izlenmesi |
| **Global Şifreleme Anahtarı** | Master encryption key yönetimi |

### Müşteri (cPanel Benzeri)

| Özellik | Açıklama |
|---------|----------|
| **Manuel Backup** | Kendi hesabını yedekleme |
| **Granüler Restore** | Tek dosya/DB/email restore |
| **Backup İndirme** | Yerel bilgisayara indirme |
| **Backup Geçmişi** | Önceki backup'ları görme |
| **Partial Backup** | Sadece dosya, sadece DB, sadece email |
| **Remote Storage (Kendi)** | Kendi S3/FTP hesabına yedekleme |
| **Restore Point Seçimi** | Hangi tarihten restore edileceği |

---

## 📦 Backup Formatı (ServerPanel Native)

### Dosya Yapısı

```
backup-USERNAME-2025-12-28T14-30-00.spbackup/
├── manifest.json              # Backup metadata (VERSİYON, TARİH, İÇERİK)
├── account.json               # Hesap bilgileri (paket, limitler, ayarlar)
├── homedir/                   # Home dizini
│   ├── public_html/
│   ├── domains/               # Addon domain'ler
│   │   └── example.com/
│   ├── ssl/                   # SSL sertifikaları
│   └── logs/
├── databases/                 # MySQL veritabanları
│   ├── db1.sql.gz            # Sıkıştırılmış SQL dump
│   ├── db1.meta.json         # DB metadata (user, grants, size)
│   ├── db2.sql.gz
│   └── db2.meta.json
├── email/                     # Email verileri
│   ├── accounts.json         # Email hesapları listesi
│   ├── forwarders.json       # Yönlendirmeler
│   ├── autoresponders.json   # Otomatik yanıtlar
│   └── mailboxes/            # Maildir formatında
│       ├── info@domain.com/
│       └── admin@domain.com/
├── dns/                       # DNS kayıtları
│   ├── domain.com.zone       # BIND zone dosyası
│   └── domain.com.json       # Yapılandırılmış DNS
├── ftp/                       # FTP hesapları
│   └── accounts.json
├── cron/                      # Cron jobs
│   └── crontab.txt
├── php/                       # PHP ayarları
│   └── settings.json
├── nodejs/                    # Node.js uygulamaları
│   ├── apps.json
│   └── pm2/
├── ssl_certificates/          # SSL sertifikaları (ayrı)
│   ├── domain.com.crt
│   ├── domain.com.key
│   └── domain.com.ca
├── security/                  # Güvenlik ayarları
│   ├── modsecurity_rules.json
│   └── custom_rules/
└── checksums.sha256           # Bütünlük kontrolü
```

### manifest.json Örneği

```json
{
  "version": "1.0.0",
  "format": "serverpanel",
  "created_at": "2025-12-28T14:30:00Z",
  "created_by": "admin",
  "type": "full",
  "encryption": {
    "enabled": true,
    "algorithm": "AES-256-GCM",
    "key_id": "master-key-2025"
  },
  "account": {
    "username": "customer1",
    "domain": "example.com",
    "package": "premium",
    "created_at": "2024-01-15T10:00:00Z"
  },
  "contents": {
    "homedir": {
      "included": true,
      "size_bytes": 1073741824,
      "file_count": 15420
    },
    "databases": {
      "included": true,
      "count": 3,
      "total_size_bytes": 52428800
    },
    "email": {
      "included": true,
      "account_count": 5,
      "mailbox_size_bytes": 209715200
    },
    "dns": {
      "included": true,
      "zone_count": 2
    },
    "ftp": {
      "included": true,
      "account_count": 2
    },
    "cron": {
      "included": true,
      "job_count": 4
    },
    "ssl": {
      "included": true,
      "cert_count": 2
    },
    "nodejs": {
      "included": true,
      "app_count": 1
    }
  },
  "checksums": {
    "algorithm": "sha256",
    "manifest_hash": "abc123..."
  },
  "incremental": {
    "enabled": false,
    "base_backup_id": null,
    "changed_files_only": false
  }
}
```

---

## 🔄 Backup Tipleri

### 1. Full Backup
- Tüm verilerin tam yedeği
- En yavaş, en çok disk kullanan
- Restore için tek başına yeterli

### 2. Incremental Backup
- Sadece son backup'tan sonra değişen dosyalar
- Çok hızlı, az disk
- Restore için full + tüm incremental'lar gerekli

### 3. Differential Backup
- Son FULL backup'tan sonra değişen tüm dosyalar
- Orta hız, orta disk
- Restore için full + son differential yeterli

### 4. Partial Backup
- Sadece seçilen bileşenler (files, db, email, dns)
- Kullanıcı seçimine göre

### 5. Snapshot Backup (Gelişmiş)
- LVM snapshot veya ZFS snapshot
- Anlık, tutarlı backup
- Büyük veritabanları için ideal

---

## 🗄️ Storage Backends

### Desteklenen Hedefler

```go
type StorageBackend interface {
    Upload(ctx context.Context, path string, reader io.Reader) error
    Download(ctx context.Context, path string, writer io.Writer) error
    Delete(ctx context.Context, path string) error
    List(ctx context.Context, prefix string) ([]BackupFile, error)
    GetMetadata(ctx context.Context, path string) (*FileMetadata, error)
}
```

| Backend | Açıklama | Öncelik |
|---------|----------|---------|
| **Local** | `/backup` dizini | Faz 1 |
| **FTP/SFTP** | Remote FTP sunucu | Faz 2 |
| **Amazon S3** | S3 compatible (Minio, Wasabi, DO Spaces) | Faz 2 |
| **Backblaze B2** | Ucuz, güvenilir | Faz 2 |
| **Google Cloud Storage** | GCS | Faz 3 |
| **Azure Blob** | Azure | Faz 3 |
| **Rclone** | 40+ cloud destegi | Faz 3 |

---

## 🔐 Şifreleme Mimarisi

### Client-Side Encryption

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Backup Data   │────▶│  AES-256-GCM    │────▶│ Encrypted Blob  │
│   (plaintext)   │     │  Encryption     │     │ (.spbackup.enc) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │  Key Derivation │
                        │  (Argon2id)     │
                        └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │  Master Key     │
                        │  (HSM/Vault)    │
                        └─────────────────┘
```

### Key Hierarchy

1. **Master Key**: Admin tarafından yönetilen ana anahtar
2. **Account Key**: Her hesap için türetilen anahtar
3. **Backup Key**: Her backup için unique anahtar

```json
{
  "key_management": {
    "master_key_location": "/root/.serverpanel/backup_master.key",
    "key_rotation_days": 90,
    "algorithm": "AES-256-GCM",
    "kdf": "Argon2id",
    "allow_user_keys": true
  }
}
```

---

## ⏰ Zamanlama Sistemi

### Retention Policies

```json
{
  "retention": {
    "daily": {
      "keep": 7,
      "time": "02:00"
    },
    "weekly": {
      "keep": 4,
      "day": "sunday",
      "time": "03:00"
    },
    "monthly": {
      "keep": 12,
      "day": 1,
      "time": "04:00"
    },
    "yearly": {
      "keep": 3,
      "month": 1,
      "day": 1
    }
  }
}
```

### Grandfather-Father-Son (GFS) Stratejisi

```
Günlük (Son):     D1 D2 D3 D4 D5 D6 D7
Haftalık (Baba):  W1 W2 W3 W4
Aylık (Dede):     M1 M2 M3 ... M12
Yıllık:           Y1 Y2 Y3
```

---

## 🔄 Restore Senaryoları

### 1. Full Account Restore
```
Senaryo: Hesap tamamen silindi/bozuldu
İşlem: Tüm backup restore edilir
Sonuç: Hesap orijinal haline döner
```

### 2. Granüler File Restore
```
Senaryo: Tek dosya yanlışlıkla silindi
İşlem: Backup'tan sadece o dosya çıkarılır
Sonuç: Dosya geri gelir, diğerleri etkilenmez
```

### 3. Database Point-in-Time Recovery
```
Senaryo: Veritabanında yanlış UPDATE çalıştırıldı
İşlem: Binary log + backup ile belirli ana dönüş
Sonuç: Veritabanı istenen zamana döner
```

### 4. Email Restore
```
Senaryo: Email hesabı yanlışlıkla silindi
İşlem: Sadece email verileri restore
Sonuç: Mailbox ve ayarlar geri gelir
```

### 5. Cross-Server Migration
```
Senaryo: Hesap başka sunucuya taşınacak
İşlem: Backup al → Transfer → Restore
Sonuç: Hesap yeni sunucuda çalışır
```

---

## 🔀 Migration (İçe Aktarma)

### Desteklenen Formatlar

| Kaynak | Format | Öncelik |
|--------|--------|---------|
| **cPanel/WHM** | `cpmove-USERNAME.tar.gz` | Faz 1 |
| **Plesk** | Plesk backup format | Faz 2 |
| **DirectAdmin** | DA backup format | Faz 2 |
| **ServerPanel** | `.spbackup` | Faz 1 |

### cPanel Import Mimarisi

```go
type CPanelImporter struct {
    backupPath string
}

func (c *CPanelImporter) Import() (*Account, error) {
    // 1. Tar.gz aç
    // 2. manifest/version dosyasını oku
    // 3. Account bilgilerini parse et (cp/ dizini)
    // 4. Homedir'i kopyala
    // 5. MySQL dump'ları import et
    // 6. Email hesaplarını oluştur
    // 7. DNS zone'ları import et
    // 8. SSL sertifikalarını kur
    // 9. Cron job'ları ekle
    // 10. FTP hesaplarını oluştur
}
```

### cPanel Backup Yapısı (Referans)

```
cpmove-USERNAME.tar.gz
├── homedir/              → /home/USERNAME/
├── mysql/                → Database dumps
│   ├── USERNAME_db1.sql
│   └── grants_USERNAME.sql
├── cp/                   → /var/cpanel/users/USERNAME
├── dnszones/             → DNS zone dosyaları
├── apache_tls/           → SSL sertifikaları
├── sslkeys/              → SSL private keys
├── cron/                 → Crontab
├── shadow                → Password hash
├── quota                 → Disk quota
├── pds                   → Parked domains
├── sds                   → Subdomains
├── sds2                  → Subdomain details
├── proftpdpasswd         → FTP accounts
└── version               → Backup version
```

---

## 📊 Veritabanı Şeması

### Backup Jobs Table

```sql
CREATE TABLE backup_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER,                    -- NULL = sunucu geneli
    user_id INTEGER NOT NULL,              -- İşlemi başlatan
    type TEXT NOT NULL,                    -- full, incremental, partial
    status TEXT NOT NULL,                  -- pending, running, completed, failed
    
    -- İçerik seçimi
    include_files BOOLEAN DEFAULT TRUE,
    include_databases BOOLEAN DEFAULT TRUE,
    include_email BOOLEAN DEFAULT TRUE,
    include_dns BOOLEAN DEFAULT TRUE,
    include_ftp BOOLEAN DEFAULT TRUE,
    include_cron BOOLEAN DEFAULT TRUE,
    include_ssl BOOLEAN DEFAULT TRUE,
    include_nodejs BOOLEAN DEFAULT TRUE,
    
    -- Sonuç bilgileri
    backup_path TEXT,
    backup_size INTEGER,
    file_count INTEGER,
    
    -- Zaman bilgileri
    started_at DATETIME,
    completed_at DATETIME,
    duration_seconds INTEGER,
    
    -- Hata bilgisi
    error_message TEXT,
    
    -- Metadata
    storage_backend TEXT,                  -- local, s3, ftp, b2
    encrypted BOOLEAN DEFAULT FALSE,
    encryption_key_id TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_backup_jobs_account ON backup_jobs(account_id);
CREATE INDEX idx_backup_jobs_status ON backup_jobs(status);
CREATE INDEX idx_backup_jobs_created ON backup_jobs(created_at);
```

### Backup Schedules Table

```sql
CREATE TABLE backup_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER,                    -- NULL = tüm hesaplar
    name TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    
    -- Zamanlama
    schedule_type TEXT NOT NULL,           -- daily, weekly, monthly, custom
    cron_expression TEXT,                  -- custom için: "0 2 * * *"
    
    -- Backup tipi
    backup_type TEXT NOT NULL,             -- full, incremental
    
    -- Retention
    retention_count INTEGER DEFAULT 7,
    
    -- Storage
    storage_backend TEXT DEFAULT 'local',
    storage_path TEXT,
    
    -- Seçenekler
    include_files BOOLEAN DEFAULT TRUE,
    include_databases BOOLEAN DEFAULT TRUE,
    include_email BOOLEAN DEFAULT TRUE,
    encrypted BOOLEAN DEFAULT FALSE,
    
    -- Metadata
    last_run_at DATETIME,
    next_run_at DATETIME,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (account_id) REFERENCES accounts(id)
);
```

### Restore Jobs Table

```sql
CREATE TABLE restore_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    backup_job_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    
    restore_type TEXT NOT NULL,            -- full, files, databases, email, partial
    status TEXT NOT NULL,                  -- pending, running, completed, failed
    
    -- Partial restore için
    selected_files TEXT,                   -- JSON array of paths
    selected_databases TEXT,               -- JSON array of db names
    selected_email_accounts TEXT,          -- JSON array of email addresses
    
    -- Seçenekler
    overwrite_existing BOOLEAN DEFAULT FALSE,
    restore_permissions BOOLEAN DEFAULT TRUE,
    
    -- Sonuç
    restored_files INTEGER,
    restored_databases INTEGER,
    restored_email_accounts INTEGER,
    
    started_at DATETIME,
    completed_at DATETIME,
    error_message TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (backup_job_id) REFERENCES backup_jobs(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Storage Backends Table

```sql
CREATE TABLE storage_backends (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,                    -- local, ftp, sftp, s3, b2, gcs
    
    -- Bağlantı bilgileri (şifreli)
    config_encrypted TEXT NOT NULL,        -- JSON, AES encrypted
    
    -- Durum
    enabled BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    
    -- Test sonucu
    last_test_at DATETIME,
    last_test_success BOOLEAN,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 🚀 API Endpoints

### Admin Endpoints

```
# Backup Yönetimi
POST   /api/admin/backups/create              # Backup oluştur
GET    /api/admin/backups                     # Tüm backup'ları listele
GET    /api/admin/backups/:id                 # Backup detayı
DELETE /api/admin/backups/:id                 # Backup sil
POST   /api/admin/backups/:id/download        # Backup indir
POST   /api/admin/backups/server              # Sunucu geneli backup

# Zamanlama
GET    /api/admin/backup-schedules            # Zamanlamaları listele
POST   /api/admin/backup-schedules            # Zamanlama oluştur
PUT    /api/admin/backup-schedules/:id        # Zamanlama güncelle
DELETE /api/admin/backup-schedules/:id        # Zamanlama sil
POST   /api/admin/backup-schedules/:id/run    # Manuel çalıştır

# Storage Backends
GET    /api/admin/storage-backends            # Backend'leri listele
POST   /api/admin/storage-backends            # Backend ekle
PUT    /api/admin/storage-backends/:id        # Backend güncelle
DELETE /api/admin/storage-backends/:id        # Backend sil
POST   /api/admin/storage-backends/:id/test   # Bağlantı test

# Restore
POST   /api/admin/restore                     # Restore başlat
GET    /api/admin/restore/:id                 # Restore durumu

# Migration
POST   /api/admin/migration/import            # cPanel/Plesk import
GET    /api/admin/migration/status/:id        # Import durumu
POST   /api/admin/migration/analyze           # Backup analiz et

# Monitoring
GET    /api/admin/backups/stats               # Backup istatistikleri
GET    /api/admin/backups/disk-usage          # Disk kullanımı
```

### Müşteri Endpoints

```
# Backup
POST   /api/backups/create                    # Kendi hesabını yedekle
GET    /api/backups                           # Kendi backup'larını listele
GET    /api/backups/:id                       # Backup detayı
POST   /api/backups/:id/download              # Backup indir

# Partial Backup
POST   /api/backups/files                     # Sadece dosyaları yedekle
POST   /api/backups/databases                 # Sadece DB'leri yedekle
POST   /api/backups/email                     # Sadece email'leri yedekle

# Granüler Restore
POST   /api/restore/file                      # Tek dosya restore
POST   /api/restore/database                  # Tek DB restore
POST   /api/restore/email                     # Email restore
POST   /api/restore/full                      # Tam restore

# Backup İçeriği
GET    /api/backups/:id/contents              # Backup içeriğini listele
GET    /api/backups/:id/files                 # Dosyaları listele
GET    /api/backups/:id/databases             # DB'leri listele
GET    /api/backups/:id/preview/:path         # Dosya önizleme
```

---

## 🔧 Uygulama Fazları

### Faz 1: Temel Backup (2 hafta)
- [ ] Local storage backend
- [ ] Full backup (homedir + DB + email)
- [ ] Backup job management
- [ ] Basic restore
- [ ] Admin UI
- [ ] Müşteri UI (basit)

### Faz 2: Gelişmiş Özellikler (2 hafta)
- [ ] Incremental backup
- [ ] Granüler restore
- [ ] Zamanlama sistemi
- [ ] Retention policies
- [ ] Remote storage (S3, FTP)

### Faz 3: Migration (1 hafta)
- [ ] cPanel import
- [ ] Backup format dönüşümü
- [ ] Conflict resolution

### Faz 4: Enterprise (1 hafta)
- [ ] Şifreleme
- [ ] Compression optimization
- [ ] Parallel backup/restore
- [ ] Snapshot backup

---

## 📈 Performans Hedefleri

| Metrik | Hedef |
|--------|-------|
| 1GB hesap backup süresi | < 2 dakika |
| 10GB hesap backup süresi | < 15 dakika |
| Incremental backup | < 30 saniye |
| Granüler dosya restore | < 5 saniye |
| DB restore (100MB) | < 30 saniye |
| Paralel backup sayısı | 5 eşzamanlı |

---

## 🛡️ Güvenlik Gereksinimleri

1. **Backup dosyaları root:root ownership**
2. **0600 permissions** (sadece root okuyabilir)
3. **Şifreli backup opsiyonel ama önerilen**
4. **Backup anahtarları ayrı saklanmalı**
5. **Müşteriler sadece kendi backup'larına erişebilir**
6. **Audit log tutulmalı** (kim, ne zaman, ne yaptı)
7. **Checksum doğrulama** (bütünlük kontrolü)

---

*Son güncelleme: 28 Aralık 2025*
