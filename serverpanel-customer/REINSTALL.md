# ServerPanel Yeniden Kurulum Kılavuzu

## 🗑️ Hızlı Temizlik (Tek Komut)

```bash
systemctl stop serverpanel mysql apache2 php8.1-fpm php8.3-fpm bind9 pure-ftpd 2>/dev/null; for u in $(awk -F: '$3>=1000 && $1!="nobody" && $1!="ubuntu" {print $1}' /etc/passwd); do pkill -9 -u "$u"; userdel -r "$u"; done 2>/dev/null; rm -rf /opt/serverpanel /var/lib/serverpanel /var/log/serverpanel /root/.serverpanel /etc/systemd/system/serverpanel.service /var/www/html/pma-signon.php /etc/bind/zones/* /etc/pure-ftpd/pureftpd.passwd /etc/pure-ftpd/pureftpd.pdb; rm -f /etc/php/*/fpm/pool.d/*.conf 2>/dev/null; rm -f /etc/apache2/sites-enabled/*.conf /etc/apache2/sites-available/*.conf 2>/dev/null; systemctl daemon-reload; echo "✅ Temizlendi"
```

## 🚀 Hızlı Kurulum (Tek Komut)

```bash
curl -sSL https://raw.githubusercontent.com/asergenalkan/serverpanel/main/install.sh | bash
```

---

## 📋 Detaylı Temizlik

```bash
# 1. Servisleri durdur
systemctl stop serverpanel mysql apache2 php8.1-fpm bind9 2>/dev/null

# 2. Panel kullanıcılarını sil
for user in $(awk -F: '$3 >= 1000 && $1 != "nobody" && $1 != "ubuntu" {print $1}' /etc/passwd); do
    pkill -9 -u "$user" 2>/dev/null
    userdel -r "$user" 2>/dev/null
    echo "Silindi: $user"
done

# 3. PHP-FPM pool'ları temizle
rm -f /etc/php/*/fpm/pool.d/*.conf 2>/dev/null

# 4. ServerPanel dosyaları
rm -rf /opt/serverpanel
rm -rf /var/lib/serverpanel
rm -rf /var/log/serverpanel
rm -rf /root/.serverpanel
rm -f /etc/systemd/system/serverpanel.service

# 5. phpMyAdmin signon
rm -f /var/www/html/pma-signon.php
rm -f /usr/share/phpmyadmin/config.inc.php

# 6. Systemd güncelle
systemctl daemon-reload

echo "✅ Temizlik tamamlandı!"
```

## ⚠️ Tam Temizlik (MySQL dahil)

```bash
# Tüm servisleri durdur
systemctl stop serverpanel mysql apache2 php8.1-fpm php8.3-fpm bind9 pure-ftpd 2>/dev/null

# Kullanıcıları sil
for u in $(awk -F: '$3>=1000 && $1!="nobody" && $1!="ubuntu" {print $1}' /etc/passwd); do
    pkill -9 -u "$u"; userdel -r "$u"
done 2>/dev/null

# Panel dosyaları
rm -rf /opt/serverpanel /var/lib/serverpanel /var/log/serverpanel /root/.serverpanel
rm -f /etc/systemd/system/serverpanel.service

# MySQL tamamen kaldır (DİKKAT: Tüm veritabanları silinir!)
apt-get purge -y mysql-server mysql-client mysql-common phpmyadmin 2>/dev/null
rm -rf /var/lib/mysql /etc/mysql /var/run/mysqld

# PHP pool temizle
rm -f /etc/php/*/fpm/pool.d/*.conf

# Apache temizle
rm -f /etc/apache2/sites-available/*.conf /etc/apache2/sites-enabled/*.conf
rm -f /var/www/html/pma-signon.php
rm -rf /etc/phpmyadmin /usr/share/phpmyadmin/config.inc.php

# DNS zone temizle
rm -rf /etc/bind/zones/*
cat > /etc/bind/named.conf.local << 'EOF'
// ServerPanel DNS Zone Configuration
EOF

# FTP temizle
rm -f /etc/pure-ftpd/pureftpd.passwd /etc/pure-ftpd/pureftpd.pdb

# Cache temizle
apt-get autoremove -y
systemctl daemon-reload

echo "✅ Tam temizlik tamamlandı!"
```

---

## 🔄 Test Döngüsü

```bash
# Temizle + Kur (tek satır)
systemctl stop serverpanel 2>/dev/null; rm -rf /opt/serverpanel /var/lib/serverpanel /root/.serverpanel; rm -f /etc/systemd/system/serverpanel.service; systemctl daemon-reload; curl -sSL https://raw.githubusercontent.com/asergenalkan/serverpanel/main/install.sh | bash
```

---

## 📌 Notlar

- **Ubuntu kullanıcısı silinmez** (filter ile korunuyor)
- Kurulum ~60-90 saniye sürer
- Panel: `http://SUNUCU_IP:8443`
- phpMyAdmin: `http://SUNUCU_IP/phpmyadmin`
- Varsayılan: `admin` / `admin123`
