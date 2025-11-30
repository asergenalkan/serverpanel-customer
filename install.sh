#!/bin/bash
#
# ╔═══════════════════════════════════════════════════════════════════════════╗
# ║                         SERVERPANEL INSTALLER                             ║
# ║                    Tek Komutla Tam Kurulum Scripti                        ║
# ╚═══════════════════════════════════════════════════════════════════════════╝
#
# Kullanım:
#   curl -sSL https://raw.githubusercontent.com/asergenalkan/serverpanel/main/install.sh | bash
#

# ═══════════════════════════════════════════════════════════════════════════════
# RENK TANIMLARI
# ═══════════════════════════════════════════════════════════════════════════════
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
NC='\033[0m'
BOLD='\033[1m'

# ═══════════════════════════════════════════════════════════════════════════════
# YAPILANDIRMA
# ═══════════════════════════════════════════════════════════════════════════════
VERSION="1.0.0"
INSTALL_DIR="/opt/serverpanel"
DATA_DIR="/var/lib/serverpanel"
LOG_DIR="/var/log/serverpanel"
CONFIG_DIR="/root/.serverpanel"
GITHUB_REPO="asergenalkan/serverpanel"
RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}"

# PHP Sürümü (OS'a göre belirlenir)
PHP_VERSION=""

# Sayaçlar
STEP_CURRENT=0
STEP_TOTAL=14
ERRORS=0
WARNINGS=0
START_TIME=$(date +%s)

# ═══════════════════════════════════════════════════════════════════════════════
# YARDIMCI FONKSİYONLAR
# ═══════════════════════════════════════════════════════════════════════════════

print_banner() {
    clear
    echo -e "${CYAN}"
    cat << "EOF"
   ___                          ___                 _ 
  / __| ___ _ ___ _____ _ _    | _ \__ _ _ _  ___  | |
  \__ \/ -_) '_\ V / -_) '_|   |  _/ _` | ' \/ -_) | |
  |___/\___|_|  \_/\___|_|     |_| \__,_|_||_\___| |_|
                                                      
EOF
    echo -e "${WHITE}  ════════════════════════════════════════════════════${NC}"
    echo -e "${WHITE}        Web Hosting Control Panel - v${VERSION}${NC}"
    echo -e "${WHITE}  ════════════════════════════════════════════════════${NC}"
    echo ""
}

log_step() {
    STEP_CURRENT=$((STEP_CURRENT + 1))
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  [${STEP_CURRENT}/${STEP_TOTAL}] ${WHITE}${BOLD}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

log_info() { echo -e "  ${GREEN}✓${NC} $1"; }
log_warn() { echo -e "  ${YELLOW}⚠${NC} $1"; WARNINGS=$((WARNINGS + 1)); }
log_error() { echo -e "  ${RED}✗${NC} $1"; ERRORS=$((ERRORS + 1)); }
log_detail() { echo -e "    ${CYAN}→${NC} $1"; }
log_progress() { echo -ne "  ${MAGENTA}◌${NC} $1...\r"; }
log_done() { echo -e "  ${GREEN}●${NC} $1              "; }

ensure_service_running() {
    local service_name=$1
    local max_attempts=${2:-3}
    local attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        if systemctl is-active --quiet "$service_name"; then
            return 0
        fi
        systemctl start "$service_name" 2>/dev/null
        sleep 2
        attempt=$((attempt + 1))
    done
    return 1
}

ensure_directory() {
    local dir=$1
    local owner=${2:-root}
    local perms=${3:-755}
    mkdir -p "$dir"
    chown "$owner:$owner" "$dir" 2>/dev/null || true
    chmod "$perms" "$dir"
}

# ═══════════════════════════════════════════════════════════════════════════════
# KONTROL FONKSİYONLARI
# ═══════════════════════════════════════════════════════════════════════════════

check_root() {
    log_step "Yetki Kontrolü"
    if [[ $EUID -ne 0 ]]; then
        log_error "Bu script root yetkisi gerektirir!"
        echo -e "  ${YELLOW}Kullanım:${NC} sudo bash install.sh"
        exit 1
    fi
    log_info "Root yetkisi doğrulandı"
}

check_os() {
    log_step "İşletim Sistemi Kontrolü"
    
    if [[ ! -f /etc/os-release ]]; then
        log_error "Desteklenmeyen işletim sistemi!"
        exit 1
    fi
    source /etc/os-release
    
    if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
        log_error "Sadece Ubuntu ve Debian desteklenmektedir"
        exit 1
    fi
    
    log_info "İşletim sistemi: $PRETTY_NAME"
    
    # PHP sürümünü belirle
    if [[ "$VERSION_ID" == "24.04" ]]; then
        PHP_VERSION="8.3"
    else
        PHP_VERSION="8.1"
    fi
    log_info "PHP sürümü: $PHP_VERSION"
    
    ARCH=$(uname -m)
    if [[ "$ARCH" != "x86_64" ]]; then
        log_error "Sadece x86_64 desteklenmektedir"
        exit 1
    fi
    log_info "Mimari: $ARCH"
}

check_resources() {
    log_step "Sistem Kaynakları Kontrolü"
    
    local total_ram=$(free -m | awk '/^Mem:/{print $2}')
    log_info "RAM: ${total_ram}MB"
    
    local free_disk=$(df -m / | awk 'NR==2 {print $4}')
    log_info "Disk: ${free_disk}MB boş"
    
    log_info "CPU: $(nproc) çekirdek"
}

# ═══════════════════════════════════════════════════════════════════════════════
# KURULUM FONKSİYONLARI
# ═══════════════════════════════════════════════════════════════════════════════

install_packages() {
    log_step "Sistem Paketleri Kuruluyor"
    
    # dpkg durumunu kontrol et ve düzelt
    log_progress "dpkg durumu kontrol ediliyor"
    dpkg --configure -a > /dev/null 2>&1 || true
    apt-get -f install -y > /dev/null 2>&1 || true
    log_done "dpkg durumu kontrol edildi"
    
    log_progress "Paket listesi güncelleniyor"
    apt-get update -qq > /dev/null 2>&1
    if [[ $? -ne 0 ]]; then
        log_warn "Paket listesi güncellenemedi, tekrar deneniyor..."
        apt-get update > /dev/null 2>&1
    fi
    log_done "Paket listesi güncellendi"
    
    # Temel paketler (önce bunlar kurulmalı)
    local base_packages=(curl wget git unzip tar net-tools sqlite3 build-essential debconf-utils)
    log_progress "Temel paketler kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y "${base_packages[@]}" > /dev/null 2>&1
    log_done "Temel paketler kuruldu"
    
    # MySQL paketleri (mysql-common önce!)
    log_progress "MySQL kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-common > /dev/null 2>&1
    DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server mysql-client > /dev/null 2>&1
    
    # MySQL kurulumunu doğrula
    if ! command -v mysql &> /dev/null; then
        log_warn "MySQL kurulumu başarısız, tekrar deneniyor..."
        apt-get install -y mysql-server mysql-client
    fi
    
    if command -v mysql &> /dev/null; then
        log_done "MySQL kuruldu"
    else
        log_error "MySQL kurulamadı!"
        exit 1
    fi
    
    # Apache paketleri
    log_progress "Apache kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y apache2 libapache2-mod-fcgid > /dev/null 2>&1
    if command -v apache2 &> /dev/null; then
        log_done "Apache kuruldu"
    else
        log_error "Apache kurulamadı!"
        exit 1
    fi
    
    # PHP paketleri
    log_progress "PHP ${PHP_VERSION} kuruluyor"
    local php_packages=(
        php${PHP_VERSION}-fpm php${PHP_VERSION}-cli php${PHP_VERSION}-mysql
        php${PHP_VERSION}-curl php${PHP_VERSION}-gd php${PHP_VERSION}-mbstring
        php${PHP_VERSION}-xml php${PHP_VERSION}-zip php${PHP_VERSION}-intl
    )
    DEBIAN_FRONTEND=noninteractive apt-get install -y "${php_packages[@]}" > /dev/null 2>&1
    
    if [[ -f /usr/sbin/php-fpm${PHP_VERSION} ]] || [[ -f /usr/sbin/php-fpm ]]; then
        log_done "PHP ${PHP_VERSION} kuruldu"
    else
        log_error "PHP-FPM kurulamadı!"
        exit 1
    fi
    
    # Diğer servisler
    log_progress "Ek servisler kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y bind9 bind9-utils certbot python3-certbot-apache > /dev/null 2>&1
    log_done "Ek servisler kuruldu"
    
    # Kurulum özeti
    echo ""
    log_info "Apache: $(apache2 -v 2>/dev/null | head -1 | awk '{print $3}')"
    log_info "PHP: $(php -v 2>/dev/null | head -1 | awk '{print $2}')"
    log_info "MySQL: $(mysql --version 2>/dev/null | awk '{print $3}')"
}

# ═══════════════════════════════════════════════════════════════════════════════
# MYSQL YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_mysql() {
    log_step "MySQL Yapılandırılıyor"
    
    ensure_directory "$CONFIG_DIR" "root" "700"
    
    # 1. MySQL paketlerinin kurulu olduğunu doğrula
    log_detail "MySQL paketleri kontrol ediliyor"
    if ! dpkg -l | grep -q "mysql-server"; then
        log_warn "mysql-server paketi eksik, kuruluyor..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-common mysql-server mysql-client
    fi
    
    # 2. MySQL config dosyasını kontrol et
    if [[ ! -f /etc/mysql/my.cnf ]] && [[ ! -f /etc/mysql/mysql.cnf ]]; then
        log_warn "MySQL config dosyası bulunamadı, yeniden kurulum yapılıyor..."
        
        # Tamamen kaldır
        systemctl stop mysql > /dev/null 2>&1
        dpkg --purge --force-all mysql-server mysql-client mysql-server-8.0 mysql-client-8.0 mysql-server-core-8.0 mysql-client-core-8.0 mysql-common > /dev/null 2>&1
        rm -rf /var/lib/mysql /etc/mysql /var/run/mysqld > /dev/null 2>&1
        apt-get autoremove -y > /dev/null 2>&1
        apt-get update > /dev/null 2>&1
        
        # Yeniden kur (sırayla!)
        DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-common > /dev/null 2>&1
        DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server mysql-client > /dev/null 2>&1
        sleep 3
        
        if [[ ! -f /etc/mysql/my.cnf ]] && [[ ! -f /etc/mysql/mysql.cnf ]]; then
            log_error "MySQL kurulumu başarısız!"
            log_error "Lütfen manuel olarak kurun: apt-get install mysql-server"
            exit 1
        fi
        log_done "MySQL yeniden kuruldu"
    fi
    log_info "MySQL config dosyası mevcut ✓"
    
    # 3. MySQL kullanıcısı kontrol et
    if ! id mysql &>/dev/null; then
        log_warn "MySQL kullanıcısı eksik, oluşturuluyor..."
        useradd -r -s /bin/false mysql
    fi
    
    # 4. MySQL socket dizinini oluştur
    log_detail "Socket dizini hazırlanıyor"
    ensure_directory "/var/run/mysqld" "mysql" "755"
    
    # 5. MySQL data dizinini kontrol et
    if [[ ! -d /var/lib/mysql/mysql ]]; then
        log_warn "MySQL data dizini eksik, başlatılıyor..."
        mysqld --initialize-insecure --user=mysql > /dev/null 2>&1 || true
    fi
    
    # 6. MySQL servisini başlat
    log_progress "MySQL servisi başlatılıyor"
    systemctl daemon-reload > /dev/null 2>&1
    systemctl enable mysql > /dev/null 2>&1
    systemctl stop mysql > /dev/null 2>&1
    sleep 1
    systemctl start mysql > /dev/null 2>&1
    
    # Servisin başlamasını bekle
    local attempts=0
    while [[ $attempts -lt 20 ]]; do
        if systemctl is-active --quiet mysql && [[ -S /var/run/mysqld/mysqld.sock ]]; then
            break
        fi
        sleep 1
        attempts=$((attempts + 1))
    done
    
    if ! systemctl is-active --quiet mysql; then
        log_error "MySQL servisi başlatılamadı!"
        journalctl -u mysql -n 15 --no-pager
        exit 1
    fi
    log_done "MySQL servisi başlatıldı"
    
    # 7. Socket kontrolü
    if [[ ! -S /var/run/mysqld/mysqld.sock ]]; then
        log_error "MySQL socket dosyası bulunamadı!"
        exit 1
    fi
    log_info "MySQL socket: /var/run/mysqld/mysqld.sock ✓"
    
    # 8. Root şifresi oluştur
    local MYSQL_ROOT_PASS=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 16)
    
    # 9. Şifreyi değiştir
    log_progress "MySQL root şifresi ayarlanıyor"
    local password_set=false
    
    # Yöntem 1: debian.cnf ile
    if [[ -f /etc/mysql/debian.cnf ]] && [[ "$password_set" == "false" ]]; then
        if mysql --defaults-file=/etc/mysql/debian.cnf -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null; then
            password_set=true
        fi
    fi
    
    # Yöntem 2: sudo mysql (auth_socket)
    if [[ "$password_set" == "false" ]]; then
        if mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null; then
            password_set=true
        fi
    fi
    
    # Yöntem 3: Boş şifre ile
    if [[ "$password_set" == "false" ]]; then
        if mysql -uroot -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null; then
            password_set=true
        fi
    fi
    
    if [[ "$password_set" == "false" ]]; then
        log_error "MySQL root şifresi ayarlanamadı!"
        exit 1
    fi
    log_done "MySQL root şifresi ayarlandı"
    
    # 10. Bağlantı testi
    if mysql -uroot -p"${MYSQL_ROOT_PASS}" -e "SELECT 1;" > /dev/null 2>&1; then
        log_info "MySQL bağlantısı: başarılı ✓"
    else
        log_error "MySQL bağlantısı başarısız!"
        exit 1
    fi
    
    # 11. Test veritabanı oluştur
    mysql -uroot -p"${MYSQL_ROOT_PASS}" -e "CREATE DATABASE IF NOT EXISTS serverpanel_test;" > /dev/null 2>&1
    mysql -uroot -p"${MYSQL_ROOT_PASS}" -e "DROP DATABASE serverpanel_test;" > /dev/null 2>&1
    log_info "MySQL veritabanı oluşturma: çalışıyor ✓"
    
    # 12. Kaydet
    echo "MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASS}" > "${CONFIG_DIR}/mysql.conf"
    chmod 600 "${CONFIG_DIR}/mysql.conf"
    
    log_info "Root şifresi: ${CONFIG_DIR}/mysql.conf"
}

# ═══════════════════════════════════════════════════════════════════════════════
# PHP-FPM YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_php() {
    log_step "PHP-FPM Yapılandırılıyor"
    
    # 1. PHP-FPM paketinin kurulu olduğunu doğrula
    log_detail "PHP-FPM paketi kontrol ediliyor"
    if ! dpkg -l | grep -q "php${PHP_VERSION}-fpm"; then
        log_warn "PHP-FPM paketi eksik, kuruluyor..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y php${PHP_VERSION}-fpm php${PHP_VERSION}-cli
    fi
    
    # 2. PHP binary kontrol
    if ! command -v php &> /dev/null; then
        log_error "PHP bulunamadı!"
        exit 1
    fi
    log_info "PHP sürümü: $(php -v | head -1 | awk '{print $2}')"
    
    # 3. PHP-FPM config dizini kontrol
    local fpm_conf_dir="/etc/php/${PHP_VERSION}/fpm"
    if [[ ! -d "$fpm_conf_dir" ]]; then
        log_warn "PHP-FPM config dizini eksik, yeniden kuruluyor..."
        DEBIAN_FRONTEND=noninteractive apt-get install --reinstall -y php${PHP_VERSION}-fpm
    fi
    
    # 4. php-fpm.conf kontrol
    if [[ ! -f "${fpm_conf_dir}/php-fpm.conf" ]]; then
        log_warn "php-fpm.conf eksik, oluşturuluyor..."
        cat > "${fpm_conf_dir}/php-fpm.conf" << FPMCONF
[global]
pid = /run/php/php${PHP_VERSION}-fpm.pid
error_log = /var/log/php${PHP_VERSION}-fpm.log
include=/etc/php/${PHP_VERSION}/fpm/pool.d/*.conf
FPMCONF
        log_done "php-fpm.conf oluşturuldu"
    fi
    
    # 5. Socket dizini oluştur
    ensure_directory "/run/php" "www-data" "755"
    
    # 6. Pool dizinini kontrol et, yoksa oluştur
    local pool_dir="/etc/php/${PHP_VERSION}/fpm/pool.d"
    if [[ ! -d "$pool_dir" ]]; then
        mkdir -p "$pool_dir"
    fi
    
    # 7. Default www pool yoksa oluştur
    if [[ ! -f "${pool_dir}/www.conf" ]] || [[ ! -s "${pool_dir}/www.conf" ]]; then
        log_warn "PHP-FPM www pool bulunamadı veya boş, oluşturuluyor..."
        cat > "${pool_dir}/www.conf" << POOLEOF
[www]
user = www-data
group = www-data
listen = /run/php/php${PHP_VERSION}-fpm.sock
listen.owner = www-data
listen.group = www-data
listen.mode = 0660
pm = dynamic
pm.max_children = 10
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 5
pm.max_requests = 500
php_admin_value[error_log] = /var/log/php${PHP_VERSION}-fpm-www.log
php_admin_flag[log_errors] = on
POOLEOF
        log_done "www pool oluşturuldu"
    fi
    log_info "Pool config: ${pool_dir}/www.conf ✓"
    
    # 8. Pool dosyasını doğrula (en az 1 pool olmalı)
    local pool_count=$(ls -1 ${pool_dir}/*.conf 2>/dev/null | wc -l)
    if [[ $pool_count -eq 0 ]]; then
        log_error "Hiç PHP-FPM pool bulunamadı!"
        exit 1
    fi
    log_info "Toplam ${pool_count} pool tanımlı ✓"
    
    # 9. PHP-FPM servisi başlat
    log_progress "PHP-FPM servisi başlatılıyor"
    systemctl daemon-reload > /dev/null 2>&1
    systemctl enable php${PHP_VERSION}-fpm > /dev/null 2>&1
    systemctl stop php${PHP_VERSION}-fpm > /dev/null 2>&1
    sleep 1
    systemctl start php${PHP_VERSION}-fpm > /dev/null 2>&1
    
    # Bekle
    local attempts=0
    while [[ $attempts -lt 10 ]]; do
        if systemctl is-active --quiet php${PHP_VERSION}-fpm; then
            break
        fi
        sleep 1
        attempts=$((attempts + 1))
    done
    
    if ! systemctl is-active --quiet php${PHP_VERSION}-fpm; then
        log_error "PHP-FPM başlatılamadı!"
        journalctl -u php${PHP_VERSION}-fpm -n 10 --no-pager
        exit 1
    fi
    log_done "PHP-FPM servisi başlatıldı"
    
    # 10. Socket kontrolü
    local socket_path="/run/php/php${PHP_VERSION}-fpm.sock"
    sleep 2
    if [[ -S "$socket_path" ]]; then
        log_info "PHP-FPM socket: $socket_path ✓"
    else
        log_error "PHP-FPM socket oluşturulamadı: $socket_path"
        exit 1
    fi
    
    # 11. PHP-FPM çalışma testi
    log_detail "PHP-FPM test ediliyor"
    echo "<?php echo 'OK'; ?>" > /tmp/php-test.php
    local test_result=$(php /tmp/php-test.php 2>/dev/null)
    rm -f /tmp/php-test.php
    if [[ "$test_result" == "OK" ]]; then
        log_info "PHP CLI çalışıyor ✓"
    else
        log_warn "PHP CLI testi başarısız"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# APACHE YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_apache() {
    log_step "Apache Yapılandırılıyor"
    
    # 1. Apache kurulu mu kontrol et
    log_detail "Apache paketi kontrol ediliyor"
    if ! command -v apache2 &> /dev/null; then
        log_warn "Apache kurulu değil, kuruluyor..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y apache2 libapache2-mod-fcgid
    fi
    
    if ! command -v apache2 &> /dev/null; then
        log_error "Apache kurulamadı!"
        exit 1
    fi
    log_info "Apache sürümü: $(apache2 -v 2>/dev/null | head -1 | awk '{print $3}')"
    
    # 2. Apache config dizini kontrol
    if [[ ! -d /etc/apache2/sites-available ]]; then
        log_error "Apache config dizini bulunamadı!"
        exit 1
    fi
    
    # 3. Modülleri aktifleştir
    log_progress "Apache modülleri aktifleştiriliyor"
    local modules=(proxy_fcgi setenvif rewrite headers ssl expires proxy alias)
    for mod in "${modules[@]}"; do
        a2enmod "$mod" > /dev/null 2>&1 || true
    done
    log_done "Modüller aktif"
    
    # 4. PHP-FPM config
    a2enconf php${PHP_VERSION}-fpm > /dev/null 2>&1 || true
    
    # 5. Web dizinini oluştur
    ensure_directory "/var/www/html" "www-data" "755"
    echo "<h1>ServerPanel</h1><p>Server is running.</p>" > /var/www/html/index.html
    chown www-data:www-data /var/www/html/index.html
    
    # 6. Default site oluştur
    log_progress "Default site oluşturuluyor"
    cat > /etc/apache2/sites-available/000-default.conf << APACHEEOF
<VirtualHost *:80>
    ServerName localhost
    DocumentRoot /var/www/html
    
    <FilesMatch \.php\$>
        SetHandler "proxy:unix:/run/php/php${PHP_VERSION}-fpm.sock|fcgi://localhost"
    </FilesMatch>
    
    <Directory /var/www/html>
        AllowOverride All
        Require all granted
    </Directory>
    
    Alias /phpmyadmin /usr/share/phpmyadmin
    <Directory /usr/share/phpmyadmin>
        Options SymLinksIfOwnerMatch
        DirectoryIndex index.php
        AllowOverride All
        Require all granted
        <FilesMatch \.php\$>
            SetHandler "proxy:unix:/run/php/php${PHP_VERSION}-fpm.sock|fcgi://localhost"
        </FilesMatch>
    </Directory>
    <Directory /usr/share/phpmyadmin/templates>
        Require all denied
    </Directory>
    <Directory /usr/share/phpmyadmin/libraries>
        Require all denied
    </Directory>
    
    ErrorLog \${APACHE_LOG_DIR}/error.log
    CustomLog \${APACHE_LOG_DIR}/access.log combined
</VirtualHost>
APACHEEOF
    log_done "Default site oluşturuldu"
    
    # 7. Site'ı aktifleştir
    a2ensite 000-default > /dev/null 2>&1
    a2dissite default-ssl > /dev/null 2>&1 || true
    
    # 8. Config testi
    log_detail "Apache config test ediliyor"
    if ! apache2ctl configtest > /dev/null 2>&1; then
        log_warn "Apache config hatası var, düzeltiliyor..."
        apache2ctl configtest 2>&1 | head -5
    fi
    
    # 9. Servisi başlat
    log_progress "Apache servisi başlatılıyor"
    systemctl daemon-reload > /dev/null 2>&1
    systemctl enable apache2 > /dev/null 2>&1
    systemctl restart apache2 > /dev/null 2>&1
    
    sleep 2
    
    if systemctl is-active --quiet apache2; then
        log_done "Apache servisi başlatıldı"
    else
        log_error "Apache başlatılamadı!"
        journalctl -u apache2 -n 10 --no-pager
        exit 1
    fi
    
    # 10. HTTP test
    log_detail "HTTP erişimi test ediliyor"
    if curl -s http://localhost/ 2>/dev/null | grep -qi "ServerPanel"; then
        log_info "HTTP erişimi: çalışıyor ✓"
    else
        log_warn "HTTP erişimi doğrulanamadı"
    fi
}

configure_dns() {
    log_step "DNS (BIND) Yapılandırılıyor"
    ensure_directory "/etc/bind/zones" "bind" "755"
    systemctl enable bind9 > /dev/null 2>&1 || true
    systemctl start bind9 > /dev/null 2>&1 || true
    log_info "BIND durumu: $(systemctl is-active bind9 2>/dev/null || echo 'inactive')"
}

# ═══════════════════════════════════════════════════════════════════════════════
# PHPMYADMIN KURULUMU
# ═══════════════════════════════════════════════════════════════════════════════

install_phpmyadmin() {
    log_step "phpMyAdmin Kuruluyor"
    
    source "${CONFIG_DIR}/mysql.conf"
    local MYSQL_PASS="$MYSQL_ROOT_PASSWORD"
    
    # MySQL bağlantısını kontrol et
    if ! mysql -uroot -p"${MYSQL_PASS}" -e "SELECT 1;" > /dev/null 2>&1; then
        log_error "MySQL bağlantısı kurulamadı!"
        exit 1
    fi
    log_info "MySQL bağlantısı aktif ✓"
    
    # Debconf ayarları
    log_progress "phpMyAdmin yapılandırılıyor"
    echo "phpmyadmin phpmyadmin/dbconfig-install boolean true" | debconf-set-selections
    echo "phpmyadmin phpmyadmin/app-password-confirm password ${MYSQL_PASS}" | debconf-set-selections
    echo "phpmyadmin phpmyadmin/mysql/admin-pass password ${MYSQL_PASS}" | debconf-set-selections
    echo "phpmyadmin phpmyadmin/mysql/app-pass password ${MYSQL_PASS}" | debconf-set-selections
    echo "phpmyadmin phpmyadmin/reconfigure-webserver multiselect" | debconf-set-selections
    log_done "Debconf ayarlandı"
    
    log_progress "phpMyAdmin kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y phpmyadmin > /dev/null 2>&1
    log_done "phpMyAdmin kuruldu"
    
    # Ana config.inc.php oluştur (signon authentication ile)
    log_progress "phpMyAdmin config oluşturuluyor"
    local BLOWFISH=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
    cat > /usr/share/phpmyadmin/config.inc.php << PMACONFIG
<?php
\$cfg['blowfish_secret'] = '${BLOWFISH}';

\$i = 0;
\$i++;

\$cfg['Servers'][\$i]['auth_type'] = 'signon';
\$cfg['Servers'][\$i]['SignonSession'] = 'SignonSession';
\$cfg['Servers'][\$i]['SignonURL'] = '/pma-signon.php';
\$cfg['Servers'][\$i]['LogoutURL'] = '/phpmyadmin/';
\$cfg['Servers'][\$i]['host'] = 'localhost';
\$cfg['Servers'][\$i]['compress'] = false;
\$cfg['Servers'][\$i]['AllowNoPassword'] = false;

\$cfg['UploadDir'] = '';
\$cfg['SaveDir'] = '';
PMACONFIG
    chown root:root /usr/share/phpmyadmin/config.inc.php
    chmod 644 /usr/share/phpmyadmin/config.inc.php
    log_done "phpMyAdmin config oluşturuldu"
    
    # Signon PHP script'i oluştur
    log_progress "Signon script oluşturuluyor"
    cat > /var/www/html/pma-signon.php << 'SCRIPTEOF'
<?php
/**
 * ServerPanel - phpMyAdmin Single Sign-On
 */

// Cookie ayarları - path '/' olmalı ki phpMyAdmin okuyabilsin
ini_set('session.use_cookies', 'true');
session_set_cookie_params(0, '/', '', false, true);

// Session başlat
session_name('SignonSession');
@session_start();

$token = $_GET['token'] ?? '';
if (empty($token)) {
    header('Location: /phpmyadmin/');
    exit;
}

$decoded = base64_decode($token);
if ($decoded === false) {
    die('Geçersiz token');
}

$data = json_decode($decoded, true);
if (!$data || !isset($data['user']) || !isset($data['password'])) {
    die('Token parse hatası');
}

if (isset($data['exp']) && time() > $data['exp']) {
    die('Token süresi dolmuş');
}

// phpMyAdmin için gerekli session değişkenleri
$_SESSION['PMA_single_signon_user'] = $data['user'];
$_SESSION['PMA_single_signon_password'] = $data['password'];
$_SESSION['PMA_single_signon_host'] = 'localhost';
$_SESSION['PMA_single_signon_port'] = 3306;
$_SESSION['PMA_single_signon_HMAC_secret'] = hash('sha1', uniqid(strval(rand()), true));

// Session'ı kapat ve kaydet
@session_write_close();

// phpMyAdmin'e yönlendir
$pmaUrl = '/phpmyadmin/index.php';
if (!empty($data['db'])) {
    $pmaUrl .= '?db=' . urlencode($data['db']);
}

header('Location: ' . $pmaUrl);
exit;
SCRIPTEOF
    chown www-data:www-data /var/www/html/pma-signon.php
    chmod 644 /var/www/html/pma-signon.php
    log_done "Signon script oluşturuldu"
    
    systemctl reload apache2
    
    sleep 2
    if curl -s http://localhost/phpmyadmin/ 2>/dev/null | grep -qi "phpmyadmin"; then
        log_info "phpMyAdmin erişimi: başarılı ✓"
    else
        log_warn "phpMyAdmin erişimi doğrulanamadı"
    fi
    
    log_info "Auto-login özelliği aktif ✓"
}

install_go() {
    log_step "Go Programlama Dili Kuruluyor"
    
    local GO_VERSION="1.21.5"
    
    if command -v /usr/local/go/bin/go &> /dev/null; then
        log_info "Go zaten kurulu: $(/usr/local/go/bin/go version)"
        return
    fi
    
    log_progress "Go ${GO_VERSION} indiriliyor"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    log_done "Go indirildi"
    
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
    
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    
    log_info "Go sürümü: $(/usr/local/go/bin/go version | awk '{print $3}')"
}

install_serverpanel() {
    log_step "ServerPanel Kuruluyor"
    
    cd /tmp
    
    ensure_directory "$INSTALL_DIR" "root" "755"
    ensure_directory "$DATA_DIR" "root" "755"
    ensure_directory "$LOG_DIR" "root" "755"
    
    log_progress "Kaynak kod indiriliyor"
    rm -rf "$INSTALL_DIR"
    git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" "$INSTALL_DIR" > /dev/null 2>&1
    log_done "Kaynak kod indirildi"
    
    cd "$INSTALL_DIR" || exit 1
    
    log_progress "Backend derleniyor"
    export PATH=$PATH:/usr/local/go/bin
    CGO_ENABLED=1 /usr/local/go/bin/go build -o serverpanel ./cmd/panel 2>&1
    
    if [[ ! -f "$INSTALL_DIR/serverpanel" ]]; then
        log_error "Backend derlenemedi!"
        exit 1
    fi
    chmod +x "$INSTALL_DIR/serverpanel"
    log_done "Backend derlendi"
    
    log_progress "Frontend indiriliyor"
    mkdir -p "$INSTALL_DIR/public"
    
    local FRONTEND_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/serverpanel-frontend.tar.gz"
    rm -f /tmp/frontend.tar.gz
    
    # İndir ve boyut kontrol et
    if curl -fsSL "$FRONTEND_URL" -o /tmp/frontend.tar.gz 2>/dev/null; then
        local filesize=$(stat -c%s /tmp/frontend.tar.gz 2>/dev/null || echo "0")
        if [[ "$filesize" -gt 1000 ]]; then
            tar -xzf /tmp/frontend.tar.gz -C "$INSTALL_DIR/public" 2>/dev/null
            rm -f /tmp/frontend.tar.gz
            log_done "Frontend indirildi (${filesize} bytes)"
        else
            log_warn "Frontend dosyası çok küçük: ${filesize} bytes"
            rm -f /tmp/frontend.tar.gz
        fi
    else
        log_warn "Frontend indirilemedi, alternatif deneniyor..."
        # Alternatif: wget ile dene
        if wget -q "$FRONTEND_URL" -O /tmp/frontend.tar.gz 2>/dev/null; then
            tar -xzf /tmp/frontend.tar.gz -C "$INSTALL_DIR/public" 2>/dev/null
            rm -f /tmp/frontend.tar.gz
            log_done "Frontend indirildi (wget)"
        else
            log_warn "Frontend indirilemedi"
        fi
    fi
}

create_service() {
    log_step "Servis Oluşturuluyor"
    
    local SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    source "${CONFIG_DIR}/mysql.conf"
    
    cat > /etc/systemd/system/serverpanel.service << EOF
[Unit]
Description=ServerPanel - Web Hosting Control Panel
After=network.target mysql.service apache2.service php${PHP_VERSION}-fpm.service
Wants=mysql.service apache2.service php${PHP_VERSION}-fpm.service

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/serverpanel
Restart=always
RestartSec=5
StandardOutput=append:${LOG_DIR}/panel.log
StandardError=append:${LOG_DIR}/error.log
Environment="ENVIRONMENT=production"
Environment="PORT=8443"
Environment="MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}"
Environment="SERVER_IP=${SERVER_IP}"
Environment="PHP_VERSION=${PHP_VERSION}"
Environment="WEB_SERVER=apache"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable serverpanel > /dev/null 2>&1
    systemctl start serverpanel
    
    sleep 3
    
    if systemctl is-active --quiet serverpanel; then
        log_info "ServerPanel: aktif ✓"
    else
        log_error "ServerPanel başlatılamadı!"
    fi
}

health_check() {
    log_step "Sistem Sağlık Kontrolü"
    
    local services=("mysql" "apache2" "php${PHP_VERSION}-fpm" "serverpanel")
    for svc in "${services[@]}"; do
        if systemctl is-active --quiet "$svc"; then
            log_info "$svc: aktif ✓"
        else
            log_error "$svc: çalışmıyor!"
        fi
    done
    
    # Socket kontrolleri
    [[ -S /var/run/mysqld/mysqld.sock ]] && log_info "MySQL socket ✓" || log_error "MySQL socket yok!"
    [[ -S /run/php/php${PHP_VERSION}-fpm.sock ]] && log_info "PHP-FPM socket ✓" || log_error "PHP-FPM socket yok!"
    
    # MySQL bağlantı testi
    source "${CONFIG_DIR}/mysql.conf"
    if mysql -uroot -p"${MYSQL_ROOT_PASSWORD}" -e "SELECT 1;" > /dev/null 2>&1; then
        log_info "MySQL bağlantısı ✓"
    else
        log_error "MySQL bağlantısı başarısız!"
    fi
}

print_summary() {
    local SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    local END_TIME=$(date +%s)
    local DURATION=$((END_TIME - START_TIME))
    
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}                  KURULUM BAŞARIYLA TAMAMLANDI!${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  ${CYAN}Süre:${NC} ${DURATION} saniye"
    echo ""
    echo -e "${CYAN}Panel Erişimi:${NC}"
    echo -e "  URL:        ${GREEN}http://${SERVER_IP}:8443${NC}"
    echo -e "  Kullanıcı:  admin"
    echo -e "  Şifre:      admin123"
    echo ""
    echo -e "${CYAN}phpMyAdmin:${NC}"
    echo -e "  URL:        ${GREEN}http://${SERVER_IP}/phpmyadmin${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  İlk girişte şifrenizi değiştirin!${NC}"
    echo ""
}

main() {
    print_banner
    echo -e "${WHITE}Kurulum başlatılıyor...${NC}"
    
    check_root
    check_os
    check_resources
    install_packages
    configure_mysql
    configure_php
    configure_apache
    configure_dns
    install_phpmyadmin
    install_go
    install_serverpanel
    create_service
    health_check
    
    print_summary
}

main "$@"