# ServerPanel Mimari Dökümanı

## Mevcut Durum vs Hedef

### ✅ Mevcut (ÇALIŞIYOR!) - Son Güncelleme: 5 Aralık 2025
```
Admin giriş yapar
├── Hesap oluşturur → ✅ Gerçekten oluşur!
│   ├── Linux user: useradd -m -d /home/username -s /bin/bash username
│   ├── Dizin yapısı: /home/username/{public_html, mail, logs, tmp, ssl}
│   ├── İzinler: home=711, public_html=755
│   ├── PHP-FPM pool: /etc/php/8.1/fpm/pool.d/username.conf
│   ├── Apache vhost: /etc/apache2/sites-available/domain.conf
│   ├── DNS zone: /etc/bind/zones/db.domain (SPF, DMARC dahil)
│   ├── DKIM key: /etc/opendkim/keys/domain/default.private
│   ├── OpenDKIM config: KeyTable, SigningTable, TrustedHosts
│   ├── Postfix vdomains: domain eklenir
│   ├── Mail dizini: /var/mail/vhosts/domain
│   ├── Webmail vhost: webmail.domain.com
│   └── Welcome page: index.html
│
├── Hesap siler → ✅ Gerçekten siliniyor!
│   ├── Apache vhost kaldırılır
│   ├── DNS zone silinir
│   ├── PHP-FPM pool silinir
│   ├── Linux user silinir
│   └── Home dizini silinir
│
├── Dosya Yöneticisi → ✅ Tam fonksiyonel!
│   ├── Dosya/klasör listeleme, oluşturma, silme
│   ├── Dosya yükleme (drag & drop, çoklu dosya, progress bar)
│   ├── Dosya indirme
│   ├── Dosya düzenleme (code editor)
│   ├── Kopyalama/Taşıma (Cut/Copy/Paste)
│   ├── Zip/Unzip (Archive)
│   ├── Dosya arama
│   ├── Resim önizleme
│   ├── 512MB yükleme limiti
│   └── Dark mode + ESC modal kapatma
│
├── SSL/Let's Encrypt → ✅ Tam fonksiyonel!
│   ├── Tek tıkla SSL sertifikası alma
│   ├── Subdomain/WWW/Mail/Webmail/FTP için ayrı SSL alma
│   ├── cPanel benzeri SSL Status tablosu
│   ├── SAN/Wildcard sertifika kontrolü
│   ├── Otomatik yenileme (cron job)
│   ├── SSL durumu görüntüleme (detaylı)
│   ├── Sertifika yenileme
│   ├── Sertifika iptal etme
│   └── **Otomatik Apache SSL vhost oluşturma** (webmail, mail, ftp, www)
│
├── E-posta Yönetimi → ✅ Tam fonksiyonel!
│   ├── Postfix MTA (mail gönderme)
│   ├── Dovecot MDA (IMAP/POP3)
│   ├── Roundcube Webmail (webmail.domain.com)
│   ├── OpenDKIM (mail imzalama)
│   ├── SpamAssassin (spam filtreleme)
│   ├── ClamAV (virüs tarama)
│   ├── E-posta hesabı oluşturma/silme
│   ├── E-posta yönlendirme (forwarders)
│   ├── Otomatik yanıtlayıcı (autoresponder)
│   ├── Rate limiting (saatte 100 mail)
│   ├── TLS/SSL güvenliği (587, 465, 993)
│   └── Otomatik DKIM/SPF/DMARC kurulumu
│
├── Veritabanı Yönetimi → ✅ Tam fonksiyonel!
│   ├── MySQL veritabanı oluşturma/silme
│   ├── Veritabanı kullanıcısı oluşturma
│   ├── phpMyAdmin SSO (tek tıkla giriş)
│   └── Veritabanı boyutu görüntüleme
│
├── MultiPHP Yönetimi → ✅ Tam fonksiyonel!
│   ├── PHP versiyon seçimi (7.4, 8.0, 8.1, 8.2, 8.3)
│   ├── PHP INI ayarları düzenleme
│   │   ├── memory_limit
│   │   ├── max_execution_time
│   │   ├── upload_max_filesize
│   │   ├── post_max_size
│   │   └── display_errors
│   ├── Paket bazlı PHP limitleri
│   └── PHP-FPM pool otomatik güncelleme
│
├── Yazılım Yöneticisi (Admin) → ✅ Tam fonksiyonel!
│   ├── PHP sürümleri kurma/kaldırma
│   ├── PHP eklentileri kurma/kaldırma
│   ├── Apache modülleri etkinleştirme/devre dışı bırakma
│   ├── Ek yazılımlar kurma/kaldırma
│   ├── Gerçek zamanlı log görüntüleme (WebSocket)
│   ├── Ondrej PHP PPA desteği (tüm PHP sürümleri)
│   ├── **ClamAV tam kurulum/kaldırma** (daemon + freshclam + temizlik)
│   ├── **ImageMagick tam kurulum/kaldırma** (config temizliği dahil)
│   └── **Kalıntısız kaldırma** (paketler, config, kullanıcılar, gruplar)
│
├── Sunucu Ayarları (Admin) → ✅ Tam fonksiyonel!
│   ├── MultiPHP aktif/pasif
│   ├── Domain bazlı PHP aktif/pasif
│   ├── Varsayılan PHP sürümü seçimi
│   └── İzin verilen PHP sürümlerini belirleme
│
├── Sunucu Özellikleri (Müşteri) → ✅ Tam fonksiyonel!
│   ├── Kurulu PHP sürümlerini görüntüleme
│   ├── Kurulu PHP eklentilerini görüntüleme
│   ├── Aktif Apache modüllerini görüntüleme
│   └── Kurulu ek yazılımları görüntüleme
│
├── Sunucu Durumu (Admin) → ✅ Tam fonksiyonel!
│   ├── Sunucu Bilgileri
│   ├── Günlük İşlem Günlüğü
│   ├── Top Processes
│   └── Task Queue
│
├── Sistem Sağlığı (Admin) → ✅ Tam fonksiyonel!
│   ├── Arka Plan İşlem Sonlandırıcı (tehlikeli işlemler, güvenilir kullanıcılar)
│   ├── İşlem Yöneticisi (CPU/Memory kullanımı, kill, kullanıcı filtreleme)
│   ├── Geçerli Disk Kullanımı (disk bilgisi, I/O istatistikleri)
│   └── Geçerli Çalışma İşlemleri (tüm işlemler listesi)
│
├── Spam Filtreleri → ✅ Tam fonksiyonel!
│   ├── SpamAssassin ayarları (spam skoru, otomatik silme)
│   ├── ClamAV antivirüs durumu görüntüleme
│   ├── Whitelist/Blacklist yönetimi
│   └── Veritabanı güncelleme tetikleme
│
├── Güvenlik Yönetimi → ✅ Tam fonksiyonel!
│   ├── Fail2ban Yönetimi
│   │   ├── Servis durumu ve jail listesi
│   │   ├── IP ban/unban
│   │   ├── Jail ayarları (bantime, findtime, maxretry)
│   │   └── Whitelist yönetimi
│   ├── UFW Firewall Yönetimi
│   │   ├── Firewall durumu görüntüleme
│   │   ├── Kural ekleme/silme
│   │   ├── Varsayılan portlar otomatik açılır
│   │   └── Güvenli etkinleştirme (kilitlenme önleme)
│   ├── SSH Güvenliği
│   │   ├── SSH port değiştirme
│   │   ├── Root login ayarları
│   │   ├── Şifre/Key authentication
│   │   └── Güvenlik puanı hesaplama
│   ├── SSH Key Yönetimi
│   │   ├── ED25519 key çifti oluşturma
│   │   ├── Private key tek seferlik indirme
│   │   ├── Mevcut public key ekleme
│   │   ├── Key listeleme (fingerprint)
│   │   └── Key silme
│   ├── Malware Tarama (ClamAV)
│   │   ├── Arka planda tarama (sayfa kapatılabilir)
│   │   ├── Canlı ilerleme gösterimi (progress bar)
│   │   ├── Taranan dosya adı gösterimi
│   │   ├── Hızlı/Tam tarama seçenekleri
│   │   ├── Tarama iptali
│   │   ├── Tehdit tespiti ve karantina
│   │   └── Tarama geçmişi (veritabanında)
│   └── ModSecurity WAF
│       ├── Web Application Firewall
│       ├── OWASP Core Rule Set (CRS)
│       ├── Tespit/Engelleme modları
│       ├── Audit log görüntüleme
│       ├── İstatistikler
│       ├── IP whitelist yönetimi
│       ├── CMS Exclusion kuralları (WordPress, Joomla, Drupal, PrestaShop, Magento)
│       ├── Manuel kural devre dışı bırakma (ID ile)
│       └── Detaylı bilgilendirme UI
│
├── Cron Jobs → ✅ Tam fonksiyonel!
│   ├── Cron işi oluşturma/düzenleme/silme
│   ├── Zamanlama şablonları (dakikalık, saatlik, günlük, haftalık, aylık)
│   ├── Özel cron ifadesi desteği
│   ├── Manuel çalıştırma ve çıktı görüntüleme
│   ├── Aktif/pasif durumu değiştirme
│   └── Sistem crontab senkronizasyonu
│
├── FTP Yönetimi (Pure-FTPd) → ✅ Tam fonksiyonel!
│   ├── FTP hesabı oluşturma/silme
│   ├── Hesap aktif/pasif yapma
│   ├── Dizin kısıtlaması (chroot)
│   ├── Kota yönetimi (sınırsız seçeneği)
│   ├── Şifre gücü göstergesi
│   └── Admin sunucu ayarları (TLS, bağlantı limitleri)
│
├── DNS Zone Editor (BIND9) → ✅ Tam fonksiyonel!
│   ├── A, AAAA, CNAME, MX, TXT, NS, SRV, CAA kayıtları
│   ├── TTL yönetimi (preset seçenekleri)
│   ├── Kayıt ekleme/düzenleme/silme
│   ├── Zone sıfırlama (varsayılana döndürme)
│   ├── Kullanıcı izolasyonu
│   ├── cPanel benzeri UI
│   └── Kayıt arama çubuğu (isim, içerik, tip filtreleme)
│
├── Paket Yönetimi → ✅ Tam fonksiyonel!
│   ├── Paket listesi (grid görünümü)
│   ├── Paket oluşturma/düzenleme/silme
│   ├── Disk, bant genişliği, domain, veritabanı, e-posta, FTP limitleri
│   ├── PHP ayarları (memory, upload, execution time)
│   └── Kullanıcı sayısı gösterimi
│
├── Domain & Subdomain Yönetimi → ✅ Tam fonksiyonel!
│   ├── Domain ekleme/silme (addon domain)
│   ├── Subdomain ekleme/silme
│   ├── Silme sırasında dosya silme seçeneği
│   ├── Modern hoşgeldin sayfası (domain ve subdomain için aynı)
│   ├── Yönlendirme desteği (301/302)
│   ├── Paket limitleri kontrolü
│   ├── Otomatik Apache vhost oluşturma
│   ├── Otomatik DNS A kaydı ekleme (subdomain için)
│   └── Kullanım limitleri gösterimi
```

### 📋 Hedef (Devam Eden)
```
Admin (WHM benzeri):
├── ✅ Kullanıcı/Hesap oluşturur
│   ├── ✅ Linux user: useradd -m -d /home/username -s /bin/bash username
│   ├── ✅ Dizin yapısı: /home/username/{public_html, mail, logs, tmp}
│   ├── ✅ İzinler: chown -R username:username /home/username
│   ├── ⏳ Quota: setquota veya disk limit
│   └── ✅ PHP-FPM pool: /etc/php/8.x/fpm/pool.d/username.conf
│
├── ✅ Domain atar
│   ├── ✅ Apache vhost: /etc/apache2/sites-available/domain.com
│   ├── ✅ Document root: /home/username/public_html
│   ├── ✅ SSL config: Let's Encrypt entegrasyonu
│   └── ✅ DNS zone: BIND9
│
├── ✅ Paket limitleri belirler
│   ├── ✅ max_php_memory (PHP bellek limiti)
│   ├── ✅ max_php_upload (Yükleme boyutu limiti)
│   └── ✅ max_php_execution_time (Çalışma süresi limiti)

Kullanıcı (cPanel benzeri):
├── ✅ Kendi hesabına giriş yapar
├── ✅ Sadece kendi kaynaklarını görür
├── ✅ Dosya Yöneticisi ile dosyalarını yönetir
│   ├── ✅ Dosya/klasör listeleme, oluşturma, silme
│   ├── ✅ Dosya yükleme (drag & drop, çoklu, progress bar)
│   ├── ✅ Dosya düzenleme (code editor)
│   ├── ✅ Dosya kopyalama/taşıma
│   ├── ✅ Zip/Unzip (Archive)
│   └── ✅ Resim önizleme
├── ✅ SSL sertifikası alır/yönetir
├── ✅ Kendi veritabanlarını yönetir (phpMyAdmin SSO)
├── ✅ PHP ayarlarını düzenler (paket limitleri dahilinde)
└── ⏳ Kendi mail hesaplarını yönetir
```

---

## Gerekli Sistem Servisleri

### 1. Web Server (Nginx veya Apache)
```bash
# Nginx kurulumu
apt install nginx

# Her domain için vhost
/etc/nginx/sites-available/domain.com
/etc/nginx/sites-enabled/domain.com -> symlink
```

### 2. PHP-FPM (Her kullanıcı için ayrı pool)
```bash
# PHP-FPM kurulumu
apt install php8.2-fpm

# Her kullanıcı için pool
/etc/php/8.2/fpm/pool.d/username.conf
```

### 3. MySQL/MariaDB
```bash
# MariaDB kurulumu
apt install mariadb-server

# Veritabanı ve kullanıcı oluşturma
CREATE DATABASE username_dbname;
CREATE USER 'username_dbuser'@'localhost' IDENTIFIED BY 'password';
GRANT ALL ON username_dbname.* TO 'username_dbuser'@'localhost';
```

### 4. Mail Server (Postfix + Dovecot)
```bash
# Mail kurulumu
apt install postfix dovecot-core dovecot-imapd
```

### 5. FTP Server (ProFTPD veya Pure-FTPd)
```bash
apt install proftpd
```

### 6. DNS Server (BIND9 veya PowerDNS)
```bash
apt install bind9
```

---

## Kullanıcı İzolasyonu

### Yöntem 1: Linux Kullanıcıları + PHP-FPM Pools
```
/home/
├── user1/
│   ├── public_html/
│   ├── mail/
│   ├── logs/
│   └── tmp/
├── user2/
│   ├── public_html/
│   └── ...
```

Her kullanıcı:
- Kendi Linux kullanıcısı
- Kendi PHP-FPM pool'u (farklı uid/gid ile çalışır)
- Kendi dizin izinleri (700 veya 750)

### Yöntem 2: Docker Containerization (Gelişmiş)
Her kullanıcı ayrı container'da çalışır.

---

## Güvenlik Kontrolleri

### Domain Ekleme Güvenliği
```go
// 1. Domain formatı kontrolü
func isValidDomain(domain string) bool {
    // Regex ile kontrol
    // Sadece alfanumerik, tire ve nokta
}

// 2. Kullanıcı yetkisi kontrolü
func canUserAddDomain(userID int, domain string) bool {
    // Kullanıcının paketinde domain hakkı var mı?
    // Limit aşılmış mı?
}

// 3. Path traversal koruması
func sanitizePath(path string) string {
    // ../../../etc/passwd gibi saldırıları engelle
    // Sadece /home/username/ altına izin ver
}
```

---

## Uygulama Akışı

### Hesap Oluşturma (Admin)
```
1. Admin "Hesap Oluştur" der
2. Form: kullanıcı adı, email, şifre, paket seçimi
3. Backend:
   a. Kullanıcı adı uygun mu? (sistemde var mı, geçerli mi)
   b. Linux user oluştur
   c. Home dizini oluştur
   d. PHP-FPM pool oluştur
   e. Nginx default config oluştur
   f. Veritabanına kaydet
   g. Hoşgeldin emaili gönder
```

### Domain Ekleme (Kullanıcı)
```
1. Kullanıcı kendi panelinde "Domain Ekle" der
2. Backend:
   a. Bu kullanıcı domain ekleyebilir mi? (paket limiti)
   b. Domain geçerli mi?
   c. /home/username/public_html/domain.com oluştur
   d. Nginx vhost oluştur
   e. Nginx reload
   f. Veritabanına kaydet
```

---

## Dosya Yapısı (Revize)

```
/internal/
├── api/           # HTTP handlers
├── auth/          # Authentication
├── database/      # SQLite (panel verisi)
├── models/        # Data models
├── services/      # İş mantığı
│   ├── account/   # Hesap oluşturma/silme
│   ├── domain/    # Domain yönetimi
│   ├── database/  # MySQL veritabanı yönetimi
│   ├── email/     # Mail hesap yönetimi
│   └── ssl/       # Let's Encrypt
└── system/        # Linux komutları
    ├── user.go    # useradd, userdel
    ├── nginx.go   # vhost yönetimi
    ├── php.go     # PHP-FPM pool
    ├── mysql.go   # MySQL yönetimi
    └── dns.go     # DNS zone yönetimi
```

---

## Öncelik Sırası (Güncellendi)

### ✅ Faz 0 - Temel Altyapı (TAMAMLANDI!)
1. [x] Linux user yönetimi (useradd/userdel)
2. [x] Dizin yapısı oluşturma (711/755 izinlerle)
3. [x] Apache vhost yönetimi (a2ensite/a2dissite)
4. [x] PHP-FPM pool yönetimi
5. [x] DNS zone yönetimi (BIND9)
6. [x] Hesap oluşturma akışı (tam entegrasyon)
7. [x] Hesap silme akışı (tam temizlik)
8. [x] Tek komutla kurulum scripti

### ✅ Faz 1 - MVP (TAMAMLANDI!)
1. [x] Hesap yönetimi UI (Admin)
2. [x] Kullanıcının kendi paneli
3. [x] Domain ekleme (gerçek)
4. [x] Dosya yöneticisi (tam fonksiyonel)
5. [x] MySQL veritabanı UI + phpMyAdmin SSO
6. [x] SSL/Let's Encrypt entegrasyonu
7. [x] MultiPHP yönetimi (versiyon + INI ayarları)
8. [x] Paket bazlı PHP limitleri

### 🔄 Faz 2 - Temel Hosting (Devam Ediyor)
1. [x] MySQL veritabanı yönetimi ✅
2. [x] SSL/Let's Encrypt ✅
3. [x] FTP hesapları (Pure-FTPd) ✅
4. [x] DNS Zone Editor (BIND9) + Arama ✅
5. [x] Paket Yönetimi UI ✅
6. [x] Domain & Subdomain Yönetimi ✅
7. [x] E-posta yönetimi (Postfix + Dovecot + Roundcube) ✅
8. [x] Spam Filtreleri (SpamAssassin + ClamAV) ✅
9. [x] Cron Jobs ✅
10. [x] Güvenlik Yönetimi (Fail2ban + UFW + SSH) ✅
11. [ ] Backup

### ✅ UI/UX İyileştirmeleri (TAMAMLANDI!)
1. [x] Merkezi tema renk sistemi (CSS variables)
2. [x] Light/Dark mode tutarlılığı
3. [x] Tüm sayfalarda tutarlı başlık boyutları
4. [x] Badge ve alert renkleri düzeltildi
5. [x] phpMyAdmin blowfish_secret otomatik yapılandırma
6. [x] DNS Zone Editor kayıt arama çubuğu
7. [x] Paket Yönetimi sayfası (grid görünümü, modal'lar)
8. [x] Domain & Subdomain Yönetimi (tab görünümü, limit kontrolü)
9. [x] **Lottie Loading Animasyonları** (tema uyumlu)
10. [x] **Spam Filtreleri Sayfası** (SpamAssassin + ClamAV UI)
11. [x] **Cron Jobs Sayfası** (zamanlama şablonları + manuel çalıştırma)
