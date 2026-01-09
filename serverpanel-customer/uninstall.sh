#!/bin/bash
#
# ServerPanel - Tam Temizleme Scripti
#

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}⚠️  ServerPanel Tam Temizleme${NC}"
echo "Bu işlem TÜM verileri silecek!"

# --force flag kontrolü
if [[ "$1" != "--force" && "$1" != "-f" ]]; then
    read -p "Devam etmek istiyor musunuz? (yes/no): " confirm
    if [[ "$confirm" != "yes" ]]; then
        echo "İptal edildi."
        echo "Force modda çalıştırmak için: uninstall.sh --force"
        exit 0
    fi
fi

echo -e "\n${YELLOW}Temizleme başlıyor...${NC}\n"

# 1. ServerPanel servisini durdur
echo "→ ServerPanel servisi durduruluyor..."
systemctl stop serverpanel 2>/dev/null
systemctl disable serverpanel 2>/dev/null
rm -f /etc/systemd/system/serverpanel.service

# 2. Panel tarafından oluşturulan kullanıcıları sil
echo "→ Panel kullanıcıları siliniyor..."
if [[ -f /var/lib/serverpanel/panel.db ]]; then
    # SQLite'dan kullanıcı listesini al
    users=$(sqlite3 /var/lib/serverpanel/panel.db "SELECT username FROM users WHERE role='user';" 2>/dev/null)
    for user in $users; do
        echo "  Siliniyor: $user"
        # PHP-FPM pool sil
        rm -f /etc/php/*/fpm/pool.d/${user}.conf 2>/dev/null
        # Process'leri öldür
        pkill -9 -u "$user" 2>/dev/null
        sleep 1
        # Sistem kullanıcısını sil
        userdel -r "$user" 2>/dev/null
    done
fi

# 3. Manuel olarak oluşturulmuş olabilecek kullanıcıları da sil
# (UID 1000+ ve sistem kullanıcıları hariç)
echo "→ Kalan hosting kullanıcıları kontrol ediliyor..."
for user in $(awk -F: '$3 >= 1000 && $3 < 65000 && $1 != "nobody" && $1 != "ubuntu" && $1 != "root" {print $1}' /etc/passwd); do
    # Home dizini /home altındaysa ve panel tarafından oluşturulmuş olabilir
    if [[ -d "/home/$user/public_html" ]]; then
        echo "  Siliniyor: $user"
        pkill -9 -u "$user" 2>/dev/null
        sleep 1
        userdel -r "$user" 2>/dev/null
    fi
done

# 4. PHP-FPM pool dosyalarını temizle ve servisi yeniden başlat
echo "→ PHP-FPM temizleniyor..."
rm -f /etc/php/*/fpm/pool.d/*.conf 2>/dev/null
# www pool'u geri koy
PHP_VERSION=$(php -v 2>/dev/null | head -1 | awk '{print $2}' | cut -d. -f1,2)
[[ -z "$PHP_VERSION" ]] && PHP_VERSION="8.1"
if [[ -f "/etc/php/${PHP_VERSION}/fpm/pool.d/www.conf.bak" ]]; then
    mv /etc/php/${PHP_VERSION}/fpm/pool.d/www.conf.bak /etc/php/${PHP_VERSION}/fpm/pool.d/www.conf
fi
systemctl restart php${PHP_VERSION}-fpm 2>/dev/null

# 5. Apache vhost'ları temizle
echo "→ Apache yapılandırması temizleniyor..."
rm -f /etc/apache2/sites-enabled/*.conf 2>/dev/null
rm -f /etc/apache2/sites-available/*.conf 2>/dev/null
# Default site'ı geri koy
a2ensite 000-default 2>/dev/null || true
systemctl restart apache2 2>/dev/null

# 6. DNS zone'ları temizle
echo "→ DNS zone'ları temizleniyor..."
rm -rf /etc/bind/zones/* 2>/dev/null
systemctl restart bind9 2>/dev/null

# 7. MySQL veritabanlarını temizle (opsiyonel - tehlikeli)
echo "→ MySQL veritabanları temizleniyor..."
if [[ -f /root/.serverpanel/mysql.conf ]]; then
    source /root/.serverpanel/mysql.conf
    # Panel tarafından oluşturulan veritabanlarını sil
    mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SHOW DATABASES;" 2>/dev/null | grep -v "Database\|information_schema\|mysql\|performance_schema\|sys\|phpmyadmin" | while read db; do
        echo "  Veritabanı siliniyor: $db"
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "DROP DATABASE IF EXISTS \`$db\`;" 2>/dev/null
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "DROP USER IF EXISTS '$db'@'localhost';" 2>/dev/null
    done
fi

# 8. ServerPanel dosyalarını sil
echo "→ ServerPanel dosyaları siliniyor..."
rm -rf /opt/serverpanel
rm -rf /var/lib/serverpanel
rm -rf /var/log/serverpanel
rm -rf /root/.serverpanel

# 9. Home dizinlerini temizle
echo "→ Home dizinleri temizleniyor..."
rm -rf /home/*/public_html 2>/dev/null

# 10. Systemd'yi yenile
systemctl daemon-reload

echo -e "\n${GREEN}✅ Temizleme tamamlandı!${NC}"
echo -e "\nYeniden kurulum için:"
echo -e "  curl -sSL https://raw.githubusercontent.com/asergenalkan/serverpanel-customer/main/install.sh | bash"
