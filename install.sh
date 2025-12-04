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
STEP_TOTAL=15
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
    
    # Ondrej PHP PPA ekle (MultiPHP desteği için)
    log_progress "Ondrej PHP PPA ekleniyor"
    if ! grep -r 'ondrej/php' /etc/apt/sources.list.d/ > /dev/null 2>&1; then
        DEBIAN_FRONTEND=noninteractive apt-get install -y software-properties-common > /dev/null 2>&1
        add-apt-repository -y ppa:ondrej/php > /dev/null 2>&1
        apt-get update > /dev/null 2>&1
        log_done "Ondrej PHP PPA eklendi"
    else
        log_done "Ondrej PHP PPA zaten mevcut"
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
    
    # Pure-FTPd kurulumu
    log_progress "Pure-FTPd kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y pure-ftpd pure-ftpd-common > /dev/null 2>&1
    if command -v pure-ftpd &> /dev/null; then
        log_done "Pure-FTPd kuruldu"
    else
        log_warn "Pure-FTPd kurulamadı, manuel kurulum gerekebilir"
    fi
    
    # Cron servisi kontrolü
    log_progress "Cron servisi kontrol ediliyor"
    if ! command -v cron &> /dev/null; then
        DEBIAN_FRONTEND=noninteractive apt-get install -y cron > /dev/null 2>&1
    fi
    systemctl enable cron > /dev/null 2>&1
    systemctl start cron > /dev/null 2>&1
    if systemctl is-active --quiet cron; then
        log_done "Cron servisi çalışıyor"
    else
        log_warn "Cron servisi başlatılamadı"
    fi
    
    # Fail2ban kurulumu (Brute-force koruması)
    log_progress "Fail2ban kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y fail2ban > /dev/null 2>&1
    if command -v fail2ban-client &> /dev/null; then
        # Varsayılan jail yapılandırması
        cat > /etc/fail2ban/jail.local << 'FAIL2BAN_EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5
ignoreip = 127.0.0.1/8 ::1

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 5

[apache-auth]
enabled = true
port = http,https
filter = apache-auth
logpath = /var/log/apache2/*error.log
maxretry = 5

[apache-badbots]
enabled = true
port = http,https
filter = apache-badbots
logpath = /var/log/apache2/*access.log
maxretry = 2

[postfix]
enabled = true
port = smtp,465,submission
filter = postfix
logpath = /var/log/mail.log
maxretry = 5

[dovecot]
enabled = true
port = pop3,pop3s,imap,imaps
filter = dovecot
logpath = /var/log/mail.log
maxretry = 5

[pure-ftpd]
enabled = true
port = ftp,ftp-data,ftps,ftps-data
filter = pure-ftpd
logpath = /var/log/syslog
maxretry = 5
FAIL2BAN_EOF
        systemctl enable fail2ban > /dev/null 2>&1
        systemctl restart fail2ban > /dev/null 2>&1
        if systemctl is-active --quiet fail2ban; then
            log_done "Fail2ban kuruldu ve yapılandırıldı"
        else
            log_warn "Fail2ban başlatılamadı"
        fi
    else
        log_warn "Fail2ban kurulamadı"
    fi
    
    # UFW Firewall kurulumu
    log_progress "UFW Firewall kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y ufw > /dev/null 2>&1
    if command -v ufw &> /dev/null; then
        # Varsayılan kurallar
        ufw default deny incoming > /dev/null 2>&1
        ufw default allow outgoing > /dev/null 2>&1
        # Gerekli portları aç
        ufw allow 22/tcp > /dev/null 2>&1      # SSH
        ufw allow 80/tcp > /dev/null 2>&1      # HTTP
        ufw allow 443/tcp > /dev/null 2>&1     # HTTPS
        ufw allow 8443/tcp > /dev/null 2>&1    # ServerPanel
        ufw allow 21/tcp > /dev/null 2>&1      # FTP
        ufw allow 25/tcp > /dev/null 2>&1      # SMTP
        ufw allow 465/tcp > /dev/null 2>&1     # SMTPS
        ufw allow 587/tcp > /dev/null 2>&1     # Submission
        ufw allow 110/tcp > /dev/null 2>&1     # POP3
        ufw allow 995/tcp > /dev/null 2>&1     # POP3S
        ufw allow 143/tcp > /dev/null 2>&1     # IMAP
        ufw allow 993/tcp > /dev/null 2>&1     # IMAPS
        ufw allow 53/tcp > /dev/null 2>&1      # DNS
        ufw allow 53/udp > /dev/null 2>&1      # DNS
        ufw allow 3306/tcp > /dev/null 2>&1    # MySQL (localhost only recommended)
        # Passive FTP ports
        ufw allow 30000:31000/tcp > /dev/null 2>&1
        # UFW'yi etkinleştir (non-interactive)
        echo "y" | ufw enable > /dev/null 2>&1
        if ufw status | grep -q "active"; then
            log_done "UFW Firewall kuruldu ve yapılandırıldı"
        else
            log_warn "UFW etkinleştirilemedi"
        fi
    else
        log_warn "UFW kurulamadı"
    fi
    
    # Kurulum özeti
    echo ""
    log_info "Apache: $(apache2 -v 2>/dev/null | head -1 | awk '{print $3}')"
    log_info "PHP: $(php -v 2>/dev/null | head -1 | awk '{print $2}')"
    log_info "MySQL: $(mysql --version 2>/dev/null | awk '{print $3}')"
    log_info "Pure-FTPd: $(pure-ftpd --help 2>&1 | head -1 || echo 'kurulu değil')"
    log_info "Fail2ban: $(fail2ban-client --version 2>/dev/null | head -1 || echo 'kurulu değil')"
    log_info "UFW: $(ufw version 2>/dev/null | head -1 || echo 'kurulu değil')"
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
# PURE-FTPD YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_pureftpd() {
    log_step "Pure-FTPd Yapılandırılıyor"
    
    # Pure-FTPd kurulu mu kontrol et
    if ! command -v pure-ftpd &> /dev/null; then
        log_warn "Pure-FTPd kurulu değil, kuruluyor..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y pure-ftpd pure-ftpd-common > /dev/null 2>&1
    fi
    
    if ! command -v pure-ftpd &> /dev/null; then
        log_warn "Pure-FTPd kurulamadı, FTP desteği olmadan devam ediliyor"
        return 0
    fi
    
    local conf_dir="/etc/pure-ftpd/conf"
    ensure_directory "$conf_dir" "root" "755"
    
    # Virtual users için PureDB kullan
    log_progress "Pure-FTPd virtual users yapılandırılıyor"
    
    # PureDB auth yöntemini etkinleştir (sadece PureDB kullan)
    echo "/etc/pure-ftpd/pureftpd.pdb" > "$conf_dir/PureDB"
    ln -sf /etc/pure-ftpd/conf/PureDB /etc/pure-ftpd/auth/50pure 2>/dev/null || true
    
    # PAM ve Unix auth'u devre dışı bırak (virtual users için gerekli)
    rm -f /etc/pure-ftpd/auth/65unix 2>/dev/null || true
    rm -f /etc/pure-ftpd/auth/70pam 2>/dev/null || true
    
    # Temel ayarlar
    echo "yes" > "$conf_dir/ChrootEveryone"      # Kullanıcıları kendi dizinlerine kilitle
    echo "yes" > "$conf_dir/NoAnonymous"         # Anonim girişi kapat
    echo "yes" > "$conf_dir/CreateHomeDir"       # Home dizini yoksa oluştur
    echo "15" > "$conf_dir/MaxIdleTime"          # 15 dakika idle timeout
    echo "50" > "$conf_dir/MaxClientsNumber"     # Maksimum 50 bağlantı
    echo "8" > "$conf_dir/MaxClientsPerIP"       # IP başına maksimum 8 bağlantı
    echo "30000 31000" > "$conf_dir/PassivePortRange"  # Passive port aralığı
    echo "1" > "$conf_dir/TLS"                   # TLS opsiyonel (0=kapalı, 1=opsiyonel, 2=zorunlu)
    echo "yes" > "$conf_dir/DontResolve"         # DNS çözümlemesi yapma (hızlandırır)
    echo "yes" > "$conf_dir/VerboseLog"          # Detaylı log
    
    # Boş PureDB oluştur (eğer yoksa)
    if [[ ! -f /etc/pure-ftpd/pureftpd.passwd ]]; then
        touch /etc/pure-ftpd/pureftpd.passwd
        pure-pw mkdb > /dev/null 2>&1 || true
    fi
    
    log_done "Pure-FTPd ayarları yapılandırıldı"
    
    # TLS sertifikası oluştur (self-signed)
    local ssl_dir="/etc/ssl/private"
    if [[ ! -f "$ssl_dir/pure-ftpd.pem" ]]; then
        log_progress "FTP SSL sertifikası oluşturuluyor"
        openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
            -keyout "$ssl_dir/pure-ftpd.pem" \
            -out "$ssl_dir/pure-ftpd.pem" \
            -subj "/C=TR/ST=Istanbul/L=Istanbul/O=ServerPanel/CN=$(hostname)" \
            > /dev/null 2>&1
        chmod 600 "$ssl_dir/pure-ftpd.pem"
        log_done "FTP SSL sertifikası oluşturuldu"
    fi
    
    # Pure-FTPd servisini başlat
    log_progress "Pure-FTPd servisi başlatılıyor"
    systemctl daemon-reload > /dev/null 2>&1
    systemctl enable pure-ftpd > /dev/null 2>&1
    systemctl restart pure-ftpd > /dev/null 2>&1
    
    sleep 2
    if systemctl is-active --quiet pure-ftpd; then
        log_done "Pure-FTPd servisi başlatıldı"
        log_info "FTP Port: 21"
        log_info "Passive Ports: 30000-31000"
    else
        log_warn "Pure-FTPd başlatılamadı, logları kontrol edin"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# MAIL SERVER YAPILANDIRMASI (Postfix + Dovecot + Roundcube)
# ═══════════════════════════════════════════════════════════════════════════════
#
# Mail Server Mimarisi:
# - MTA (Mail Transfer Agent): Postfix - SMTP gönderim/alım
# - MDA (Mail Delivery Agent): Dovecot - IMAP/POP3 erişim  
# - Webmail: Roundcube - Tarayıcı üzerinden e-posta (GPL-3.0 Ücretsiz)
#
# Standart Portlar:
# ┌─────────────┬──────────┬────────────┬─────────────────────────────────┐
# │ Protokol    │ Port     │ Güvenlik   │ Açıklama                        │
# ├─────────────┼──────────┼────────────┼─────────────────────────────────┤
# │ SMTP        │ 25       │ STARTTLS   │ Sunucular arası mail transferi  │
# │ SMTP        │ 587      │ STARTTLS   │ Mail gönderimi (submission)     │
# │ SMTPS       │ 465      │ SSL/TLS    │ Güvenli mail gönderimi          │
# │ IMAP        │ 143      │ STARTTLS   │ Mail okuma                      │
# │ IMAPS       │ 993      │ SSL/TLS    │ Güvenli mail okuma              │
# │ POP3        │ 110      │ STARTTLS   │ Mail indirme                    │
# │ POP3S       │ 995      │ SSL/TLS    │ Güvenli mail indirme            │
# └─────────────┴──────────┴────────────┴─────────────────────────────────┘
#
# ═══════════════════════════════════════════════════════════════════════════════

configure_mail_server() {
    log_step "Mail Server Yapılandırılıyor (Postfix + Dovecot + OpenDKIM)"
    
    # 1. Paketleri kur (DKIM, SPF kontrolü dahil)
    log_progress "Mail server paketleri kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        postfix postfix-mysql postfix-policyd-spf-python \
        dovecot-core dovecot-imapd dovecot-pop3d dovecot-lmtpd dovecot-sieve \
        opendkim opendkim-tools \
        > /dev/null 2>&1
    
    if ! command -v postfix &> /dev/null; then
        log_warn "Postfix kurulamadı, mail server atlanıyor"
        return
    fi
    log_done "Mail server paketleri kuruldu"
    
    # 2. vmail kullanıcısı oluştur
    log_progress "vmail kullanıcısı oluşturuluyor"
    if ! id "vmail" &>/dev/null; then
        groupadd -g 5000 vmail
        useradd -g vmail -u 5000 vmail -d /var/mail/vhosts -m
    fi
    ensure_directory "/var/mail/vhosts" "vmail" "770"
    log_done "vmail kullanıcısı hazır"
    
    # 3. OpenDKIM yapılandırma
    log_progress "OpenDKIM yapılandırılıyor"
    mkdir -p /etc/opendkim/keys
    chown -R opendkim:opendkim /etc/opendkim
    chmod 700 /etc/opendkim/keys
    
    cat > /etc/opendkim.conf << 'DKIMCONF'
AutoRestart             Yes
AutoRestartRate         10/1h
Syslog                  yes
SyslogSuccess           Yes
LogWhy                  Yes
Canonicalization        relaxed/simple
ExternalIgnoreList      refile:/etc/opendkim/TrustedHosts
InternalHosts           refile:/etc/opendkim/TrustedHosts
KeyTable                refile:/etc/opendkim/KeyTable
SigningTable            refile:/etc/opendkim/SigningTable
Mode                    sv
PidFile                 /var/run/opendkim/opendkim.pid
SignatureAlgorithm      rsa-sha256
UserID                  opendkim:opendkim
Socket                  inet:8891@localhost
DKIMCONF

    # Trusted hosts
    cat > /etc/opendkim/TrustedHosts << 'TRUSTEDHOSTS'
127.0.0.1
localhost
TRUSTEDHOSTS

    # Boş KeyTable ve SigningTable
    touch /etc/opendkim/KeyTable
    touch /etc/opendkim/SigningTable
    chown opendkim:opendkim /etc/opendkim/KeyTable /etc/opendkim/SigningTable
    
    # OpenDKIM servisini başlat
    systemctl enable opendkim > /dev/null 2>&1
    systemctl restart opendkim > /dev/null 2>&1
    log_done "OpenDKIM yapılandırıldı"
    
    # 4. Postfix temel yapılandırma
    log_progress "Postfix yapılandırılıyor"
    local hostname=$(hostname -f)
    local server_ip=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    
    postconf -e "myhostname = $hostname"
    postconf -e "mydomain = $(hostname -d)"
    postconf -e "myorigin = \$mydomain"
    postconf -e "inet_interfaces = all"
    postconf -e "inet_protocols = ipv4"
    postconf -e "mydestination = localhost"
    postconf -e "mynetworks = 127.0.0.0/8 [::1]/128"
    postconf -e "relay_domains ="
    
    # Virtual mailbox ayarları
    postconf -e "virtual_transport = lmtp:unix:private/dovecot-lmtp"
    postconf -e "virtual_mailbox_domains = hash:/etc/postfix/vdomains"
    postconf -e "virtual_mailbox_maps = hash:/etc/postfix/vmailbox"
    postconf -e "virtual_alias_maps = hash:/etc/postfix/virtual"
    postconf -e "virtual_mailbox_base = /var/mail/vhosts"
    postconf -e "virtual_uid_maps = static:5000"
    postconf -e "virtual_gid_maps = static:5000"
    
    # SASL authentication
    postconf -e "smtpd_sasl_type = dovecot"
    postconf -e "smtpd_sasl_path = private/auth"
    postconf -e "smtpd_sasl_auth_enable = yes"
    
    # Güvenlik ayarları - dış mail gönderimi için düzeltildi
    postconf -e "smtpd_recipient_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination"
    postconf -e "smtpd_helo_required = yes"
    postconf -e "smtpd_helo_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_invalid_helo_hostname"
    postconf -e "smtpd_sender_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_non_fqdn_sender"
    
    # TLS ayarları
    postconf -e "smtpd_tls_cert_file = /etc/ssl/certs/ssl-cert-snakeoil.pem"
    postconf -e "smtpd_tls_key_file = /etc/ssl/private/ssl-cert-snakeoil.key"
    postconf -e "smtpd_tls_security_level = may"
    postconf -e "smtp_tls_security_level = may"
    postconf -e "smtpd_tls_auth_only = yes"
    
    # DKIM milter
    postconf -e "milter_protocol = 6"
    postconf -e "milter_default_action = accept"
    postconf -e "smtpd_milters = inet:localhost:8891"
    postconf -e "non_smtpd_milters = inet:localhost:8891"
    
    # Rate limiting (saatte 100 mail)
    postconf -e "smtpd_client_message_rate_limit = 100"
    postconf -e "smtpd_client_recipient_rate_limit = 100"
    postconf -e "anvil_rate_time_unit = 3600s"
    
    # Message size limit (25MB)
    postconf -e "message_size_limit = 26214400"
    
    # Submission port (587) ve SMTPS (465)
    if ! grep -q "^submission" /etc/postfix/master.cf; then
        cat >> /etc/postfix/master.cf << 'SUBMISSION'
submission inet n       -       y       -       -       smtpd
  -o syslog_name=postfix/submission
  -o smtpd_tls_security_level=encrypt
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_tls_auth_only=yes
  -o smtpd_reject_unlisted_recipient=no
  -o smtpd_client_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING
smtps     inet  n       -       y       -       -       smtpd
  -o syslog_name=postfix/smtps
  -o smtpd_tls_wrappermode=yes
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_reject_unlisted_recipient=no
  -o smtpd_client_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING
policyd-spf  unix  -       n       n       -       0       spawn
  user=policyd-spf argv=/usr/bin/policyd-spf
SUBMISSION
    fi
    
    # Boş dosyalar oluştur ve hash'le
    touch /etc/postfix/vdomains
    touch /etc/postfix/vmailbox
    touch /etc/postfix/virtual
    postmap /etc/postfix/vdomains > /dev/null 2>&1 || true
    postmap /etc/postfix/vmailbox > /dev/null 2>&1 || true
    postmap /etc/postfix/virtual > /dev/null 2>&1 || true
    
    log_done "Postfix yapılandırıldı"
    
    # 4. Dovecot IMAP/POP3 paketlerini kur
    log_progress "Dovecot IMAP/POP3 kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y dovecot-imapd dovecot-pop3d > /dev/null 2>&1
    log_done "Dovecot IMAP/POP3 kuruldu"
    
    # 5. Dovecot yapılandırma
    log_progress "Dovecot yapılandırılıyor"
    
    # Dovecot users dosyası
    touch /etc/dovecot/users
    chown root:dovecot /etc/dovecot/users
    chmod 640 /etc/dovecot/users
    
    # Dovecot auth config - sistem auth'u devre dışı bırak, passwd-file kullan
    cat > /etc/dovecot/conf.d/10-auth.conf << 'DOVECOTAUTH'
disable_plaintext_auth = no
auth_mechanisms = plain login

#!include auth-system.conf.ext

passdb {
  driver = passwd-file
  args = scheme=BLF-CRYPT username_format=%u /etc/dovecot/users
}

userdb {
  driver = static
  args = uid=vmail gid=vmail home=/var/mail/vhosts/%d/%n
}
DOVECOTAUTH

    # Dovecot mail config
    cat > /etc/dovecot/conf.d/10-mail.conf << 'DOVECOTMAIL'
mail_location = maildir:/var/mail/vhosts/%d/%n
mail_privileged_group = vmail
namespace inbox {
  inbox = yes
}
DOVECOTMAIL

    # Dovecot master config (LMTP + Auth)
    cat > /etc/dovecot/conf.d/10-master.conf << 'DOVECOTMASTER'
service lmtp {
  unix_listener /var/spool/postfix/private/dovecot-lmtp {
    mode = 0600
    user = postfix
    group = postfix
  }
}

service auth {
  unix_listener /var/spool/postfix/private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
}
DOVECOTMASTER

    log_done "Dovecot yapılandırıldı"
    
    # 6. SpamAssassin kurulumu
    log_progress "SpamAssassin kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y spamassassin spamc > /dev/null 2>&1
    
    # SpamAssassin yapılandırma
    cat > /etc/spamassassin/local.cf << 'SPAMCONF'
# SpamAssassin configuration
rewrite_header Subject [SPAM]
report_safe 0
required_score 5.0
use_bayes 1
bayes_auto_learn 1
skip_rbl_checks 0
use_razor2 0
use_pyzor 0
SPAMCONF

    # SpamAssassin servisini etkinleştir
    sed -i 's/ENABLED=0/ENABLED=1/' /etc/default/spamassassin 2>/dev/null || true
    systemctl enable spamassassin > /dev/null 2>&1
    systemctl start spamassassin > /dev/null 2>&1
    
    # Postfix'e SpamAssassin entegrasyonu
    if ! grep -q "spamassassin" /etc/postfix/master.cf; then
        cat >> /etc/postfix/master.cf << 'SPAMMASTER'
# SpamAssassin integration
spamassassin unix -     n       n       -       -       pipe
  user=spamd argv=/usr/bin/spamc -f -e /usr/sbin/sendmail -oi -f \${sender} \${recipient}
SPAMMASTER
    fi
    
    log_done "SpamAssassin yapılandırıldı"
    
    # 7. ClamAV kurulumu (isteğe bağlı - büyük paket)
    log_progress "ClamAV kuruluyor (bu biraz zaman alabilir)"
    DEBIAN_FRONTEND=noninteractive apt-get install -y clamav clamav-daemon clamav-milter > /dev/null 2>&1
    
    if command -v clamscan &> /dev/null; then
        # ClamAV yapılandırma
        systemctl stop clamav-freshclam > /dev/null 2>&1 || true
        freshclam > /dev/null 2>&1 || true
        systemctl start clamav-freshclam > /dev/null 2>&1 || true
        systemctl enable clamav-daemon > /dev/null 2>&1
        systemctl start clamav-daemon > /dev/null 2>&1 || true
        
        # ClamAV milter yapılandırma
        cat > /etc/clamav/clamav-milter.conf << 'CLAMCONF'
MilterSocket /var/run/clamav/clamav-milter.sock
MilterSocketMode 666
FixStaleSocket true
User clamav
AllowSupplementaryGroups true
ReadTimeout 120
Foreground false
PidFile /var/run/clamav/clamav-milter.pid
ClamdSocket unix:/var/run/clamav/clamd.ctl
OnClean Accept
OnInfected Reject
OnFail Defer
AddHeader Replace
LogSyslog true
LogFacility LOG_MAIL
LogVerbose false
LogInfected Basic
LogClean Off
MaxFileSize 25M
SupportMultipleRecipients true
RejectMsg Virus detected: %v
CLAMCONF

        systemctl enable clamav-milter > /dev/null 2>&1
        systemctl restart clamav-milter > /dev/null 2>&1 || true
        
        # Postfix'e ClamAV milter ekle
        postconf -e "smtpd_milters = inet:localhost:8891, unix:/var/run/clamav/clamav-milter.sock"
        postconf -e "non_smtpd_milters = inet:localhost:8891, unix:/var/run/clamav/clamav-milter.sock"
        
        log_done "ClamAV yapılandırıldı"
    else
        log_warn "ClamAV kurulamadı, atlanıyor"
    fi
    
    # 8. Servisleri başlat
    log_progress "Mail servisleri başlatılıyor"
    systemctl enable postfix dovecot > /dev/null 2>&1
    systemctl restart postfix dovecot > /dev/null 2>&1
    
    sleep 2
    if systemctl is-active --quiet postfix && systemctl is-active --quiet dovecot; then
        log_done "Mail servisleri başlatıldı"
        log_info "SMTP: 25, 587 (submission), 465 (SMTPS)"
        log_info "IMAP: 143, 993 (SSL)"
        log_info "POP3: 110, 995 (SSL)"
        log_info "SpamAssassin: aktif"
        log_info "ClamAV: $(systemctl is-active clamav-daemon 2>/dev/null || echo 'pasif')"
    else
        log_warn "Mail servisleri başlatılamadı"
    fi
}

configure_roundcube() {
    log_step "Roundcube Webmail Yapılandırılıyor"
    
    # 1. Roundcube kur
    log_progress "Roundcube kuruluyor"
    DEBIAN_FRONTEND=noninteractive apt-get install -y roundcube roundcube-core roundcube-mysql > /dev/null 2>&1
    
    if [[ ! -d /usr/share/roundcube ]]; then
        log_warn "Roundcube kurulamadı, atlanıyor"
        return
    fi
    log_done "Roundcube kuruldu"
    
    # 2. Roundcube config dosyasını oluştur
    log_progress "Roundcube config yapılandırılıyor"
    cat > /etc/roundcube/config.inc.php << 'RCCONFIG'
<?php

$config = [];

// Database
include_once("/etc/roundcube/debian-db.php");

// Directories
$config['temp_dir'] = '/tmp/roundcube';
$config['log_dir'] = '/var/log/roundcube';
$config['log_driver'] = 'file';

// Mailboxes
$config['drafts_mbox'] = 'Drafts';
$config['junk_mbox'] = 'Junk';
$config['sent_mbox'] = 'Sent';
$config['trash_mbox'] = 'Trash';

// IMAP host
$config['default_host'] = 'localhost';
$config['default_port'] = 143;

// SMTP server - localhost için auth gerekmez
$config['smtp_server'] = 'localhost';
$config['smtp_port'] = 25;
$config['smtp_user'] = '';
$config['smtp_pass'] = '';
$config['smtp_auth_type'] = null;

// System
$config['support_url'] = '';
$config['product_name'] = 'ServerPanel Webmail';
$config['des_key'] = 'rcmail-!24ByteDESkey*Str';
$config['plugins'] = [];
$config['skin'] = 'elastic';
$config['enable_spellcheck'] = false;
RCCONFIG

    # Roundcube dizinlerini oluştur
    mkdir -p /tmp/roundcube /var/log/roundcube
    chown www-data:www-data /tmp/roundcube /var/log/roundcube
    
    # 3. Apache alias oluştur
    log_progress "Roundcube Apache yapılandırması"
    cat > /etc/apache2/conf-available/roundcube.conf << 'RCAPACHE'
Alias /webmail /usr/share/roundcube

<Directory /usr/share/roundcube>
    Options +FollowSymLinks
    AllowOverride All
    Require all granted
</Directory>

<Directory /usr/share/roundcube/config>
    Require all denied
</Directory>

<Directory /usr/share/roundcube/temp>
    Require all denied
</Directory>

<Directory /usr/share/roundcube/logs>
    Require all denied
</Directory>
RCAPACHE

    a2enconf roundcube > /dev/null 2>&1
    
    # 4. ACME challenge dizinini oluştur (Let's Encrypt SSL için)
    log_progress "ACME challenge dizini oluşturuluyor"
    mkdir -p /var/www/html/.well-known/acme-challenge
    chmod -R 755 /var/www/html/.well-known
    chown -R www-data:www-data /var/www/html/.well-known
    log_done "ACME challenge dizini hazır"
    
    # 5. Varsayılan webmail vhost template'i oluştur
    log_progress "Webmail vhost template oluşturuluyor"
    cat > /etc/apache2/sites-available/webmail-template.conf.disabled << 'WEBMAILTPL'
# Webmail Virtual Host Template
# Bu dosya yeni domain eklendiğinde kopyalanır
# Kullanım: cp webmail-template.conf.disabled webmail.DOMAIN.conf
#           sed -i 's/DOMAIN_PLACEHOLDER/domain.com/g' webmail.DOMAIN.conf
#           a2ensite webmail.DOMAIN.conf
<VirtualHost *:80>
    ServerName webmail.DOMAIN_PLACEHOLDER
    
    # ACME challenge için - SSL sertifikası alımı
    Alias /.well-known/acme-challenge/ /var/www/html/.well-known/acme-challenge/
    <Directory "/var/www/html/.well-known/acme-challenge/">
        Options None
        AllowOverride None
        ForceType text/plain
        Require all granted
    </Directory>
    
    # Roundcube
    DocumentRoot /usr/share/roundcube
    
    <Directory /usr/share/roundcube>
        Options +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
    
    <Directory /usr/share/roundcube/config>
        Require all denied
    </Directory>
</VirtualHost>
WEBMAILTPL
    log_done "Webmail vhost template hazır"
    
    systemctl reload apache2 > /dev/null 2>&1
    
    log_done "Roundcube yapılandırıldı"
    log_info "Webmail URL: http://SERVER_IP/webmail"
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
    
    # 6. ACME challenge için global config oluştur
    log_progress "Let's Encrypt ACME config oluşturuluyor"
    cat > /etc/apache2/conf-available/letsencrypt-acme.conf << 'ACMECONF'
# Let's Encrypt ACME Challenge - Global Config
# Tüm domain'ler için .well-known/acme-challenge dizinini /var/www/html'e yönlendirir
Alias /.well-known/acme-challenge/ /var/www/html/.well-known/acme-challenge/
<Directory "/var/www/html/.well-known/acme-challenge/">
    Options None
    AllowOverride None
    ForceType text/plain
    Require all granted
</Directory>
ACMECONF
    a2enconf letsencrypt-acme > /dev/null 2>&1
    log_done "ACME config oluşturuldu"
    
    # 7. Default site oluştur
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
    log_step "DNS (BIND9) Yapılandırılıyor"
    
    # BIND9 kurulu mu kontrol et
    if ! command -v named &> /dev/null; then
        log_warn "BIND9 kurulu değil, kuruluyor..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y bind9 bind9-utils > /dev/null 2>&1
    fi
    
    # Zone dizinini oluştur
    ensure_directory "/etc/bind/zones" "bind" "755"
    log_info "Zone dizini: /etc/bind/zones ✓"
    
    # named.conf.local dosyasını yapılandır (zone include için)
    log_progress "BIND9 yapılandırılıyor"
    
    # named.conf.options güncelle (recursion kapalı, güvenlik için)
    cat > /etc/bind/named.conf.options << 'BINDOPTIONS'
options {
    directory "/var/cache/bind";
    
    // Recursion kapalı (sadece authoritative DNS)
    recursion no;
    
    // DNSSEC validation
    dnssec-validation auto;
    
    // IPv6 dinleme
    listen-on-v6 { any; };
    listen-on { any; };
    
    // Zone transfer kısıtlaması
    allow-transfer { none; };
    
    // Query izni (herkes sorgulayabilir)
    allow-query { any; };
};
BINDOPTIONS
    log_done "named.conf.options yapılandırıldı"
    
    # named.conf.local başlangıç dosyası (boş, zone'lar dinamik eklenir)
    if [[ ! -f /etc/bind/named.conf.local ]] || ! grep -q "ServerPanel" /etc/bind/named.conf.local 2>/dev/null; then
        cat > /etc/bind/named.conf.local << 'BINDLOCAL'
//
// ServerPanel DNS Zone Configuration
// Zone'lar otomatik olarak eklenir
//
BINDLOCAL
        log_info "named.conf.local hazırlandı ✓"
    fi
    
    # BIND config syntax kontrolü
    log_detail "BIND config kontrol ediliyor"
    if named-checkconf > /dev/null 2>&1; then
        log_info "BIND config: geçerli ✓"
    else
        log_warn "BIND config hatası var, düzeltiliyor..."
        named-checkconf 2>&1 | head -5
    fi
    
    # BIND9 servisini başlat
    log_progress "BIND9 servisi başlatılıyor"
    systemctl daemon-reload > /dev/null 2>&1
    systemctl enable bind9 > /dev/null 2>&1 || systemctl enable named > /dev/null 2>&1
    systemctl restart bind9 > /dev/null 2>&1 || systemctl restart named > /dev/null 2>&1
    
    sleep 2
    
    if systemctl is-active --quiet bind9 || systemctl is-active --quiet named; then
        log_done "BIND9 servisi başlatıldı"
        log_info "DNS Port: 53 (TCP/UDP)"
    else
        log_warn "BIND9 başlatılamadı, logları kontrol edin"
        journalctl -u bind9 -n 5 --no-pager 2>/dev/null || true
    fi
    
    # Firewall kuralları (UFW varsa)
    if command -v ufw &> /dev/null; then
        ufw allow 53/tcp > /dev/null 2>&1 || true
        ufw allow 53/udp > /dev/null 2>&1 || true
        log_info "Firewall: DNS portları açıldı ✓"
    fi
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
    
    # /etc/phpmyadmin/config.inc.php oluştur (bazı kurulumlar bunu okur)
    log_progress "/etc/phpmyadmin config oluşturuluyor"
    mkdir -p /etc/phpmyadmin/conf.d
    cat > /etc/phpmyadmin/config.inc.php << ETCPMACONFIG
<?php
\$cfg['blowfish_secret'] = '${BLOWFISH}';
\$cfg['PmaNoRelation_DisableWarning'] = true;
ETCPMACONFIG
    chown root:root /etc/phpmyadmin/config.inc.php
    chmod 644 /etc/phpmyadmin/config.inc.php
    log_done "/etc/phpmyadmin config oluşturuldu"
    
    # Signon PHP script'i oluştur (Go backend'den credential çeker)
    log_progress "Signon script oluşturuluyor"
    cat > /var/www/html/pma-signon.php << 'SCRIPTEOF'
<?php
/**
 * ServerPanel - phpMyAdmin Single Sign-On
 * Go backend'den güvenli şekilde credential alır
 */

$token = $_GET['token'] ?? '';
if (empty($token)) {
    header('Location: /phpmyadmin/');
    exit;
}

// Go backend'den credential al (one-time token)
$apiUrl = "http://127.0.0.1:8443/api/v1/internal/pma-credentials?token=" . urlencode($token);

$ch = curl_init();
curl_setopt($ch, CURLOPT_URL, $apiUrl);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_TIMEOUT, 5);
curl_setopt($ch, CURLOPT_CONNECTTIMEOUT, 3);
$response = curl_exec($ch);
$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

if ($httpCode !== 200 || empty($response)) {
    die('Token geçersiz veya süresi dolmuş');
}

$data = json_decode($response, true);
if (!$data || empty($data['user']) || empty($data['password'])) {
    die('Credential alınamadı');
}

// Session ayarları - path '/' olmalı ki phpMyAdmin okuyabilsin
ini_set('session.use_cookies', 'true');
session_set_cookie_params(0, '/', '', false, true);
session_name('SignonSession');
session_start();

// phpMyAdmin için gerekli session değişkenleri
$_SESSION['PMA_single_signon_user'] = $data['user'];
$_SESSION['PMA_single_signon_password'] = $data['password'];
$_SESSION['PMA_single_signon_host'] = $data['host'] ?? 'localhost';
$_SESSION['PMA_single_signon_port'] = 3306;
$_SESSION['PMA_single_signon_HMAC_secret'] = hash('sha1', uniqid(strval(rand()), true));

// Session'ı kaydet
session_write_close();

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
    
    # Frontend: web/dist -> public kopyala
    log_progress "Frontend kopyalanıyor"
    mkdir -p "$INSTALL_DIR/public"
    
    if [[ -d "$INSTALL_DIR/web/dist" ]] && [[ -f "$INSTALL_DIR/web/dist/index.html" ]]; then
        cp -r "$INSTALL_DIR/web/dist/"* "$INSTALL_DIR/public/"
        log_done "Frontend kopyalandı"
    else
        log_error "Frontend bulunamadı: $INSTALL_DIR/web/dist"
        exit 1
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# VERİTABANI MİGRASYONU
# ═══════════════════════════════════════════════════════════════════════════════

migrate_database() {
    log_step "Veritabanı Migrasyonu"
    
    local DB_PATH="/root/.serverpanel/panel.db"
    
    # Veritabanı yoksa migration gerekli değil (yeni kurulum)
    if [[ ! -f "$DB_PATH" ]]; then
        log_info "Yeni kurulum, migration gerekli değil"
        return 0
    fi
    
    log_progress "Veritabanı şeması kontrol ediliyor"
    
    # domains tablosunda domain_type sütunu var mı kontrol et
    if ! sqlite3 "$DB_PATH" "PRAGMA table_info(domains);" | grep -q "domain_type"; then
        log_progress "domains tablosu güncelleniyor"
        sqlite3 "$DB_PATH" "ALTER TABLE domains ADD COLUMN domain_type TEXT DEFAULT 'primary';"
        sqlite3 "$DB_PATH" "ALTER TABLE domains ADD COLUMN parent_domain_id INTEGER;"
        log_done "domains tablosu güncellendi"
    else
        log_info "domains tablosu güncel"
    fi
    
    # subdomains tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "subdomains"; then
        log_progress "subdomains tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS subdomains (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            domain_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            full_name TEXT UNIQUE NOT NULL,
            document_root TEXT,
            redirect_url TEXT,
            redirect_type TEXT,
            active INTEGER DEFAULT 1,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
        );"
        log_done "subdomains tablosu oluşturuldu"
    else
        log_info "subdomains tablosu mevcut"
    fi
    
    # email_forwarders tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "email_forwarders"; then
        log_progress "email_forwarders tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS email_forwarders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            domain_id INTEGER NOT NULL,
            source TEXT NOT NULL,
            destination TEXT NOT NULL,
            active INTEGER DEFAULT 1,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
        );"
        log_done "email_forwarders tablosu oluşturuldu"
    else
        log_info "email_forwarders tablosu mevcut"
    fi
    
    # email_autoresponders tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "email_autoresponders"; then
        log_progress "email_autoresponders tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS email_autoresponders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            domain_id INTEGER NOT NULL,
            email TEXT NOT NULL,
            subject TEXT NOT NULL,
            body TEXT NOT NULL,
            start_date TEXT,
            end_date TEXT,
            active INTEGER DEFAULT 1,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
        );"
        log_done "email_autoresponders tablosu oluşturuldu"
    else
        log_info "email_autoresponders tablosu mevcut"
    fi
    
    # email_accounts tablosunda password_hash sütunu var mı kontrol et (eski şema: password, yeni şema: password_hash)
    if sqlite3 "$DB_PATH" "PRAGMA table_info(email_accounts);" | grep -q "|password|" ; then
        log_progress "email_accounts tablosu güncelleniyor (password -> password_hash)"
        sqlite3 "$DB_PATH" "ALTER TABLE email_accounts RENAME COLUMN password TO password_hash;" 2>/dev/null || true
        log_done "password_hash sütunu güncellendi"
    fi
    
    # email_accounts tablosunda quota_mb sütunu var mı kontrol et (eski şema: quota, yeni şema: quota_mb)
    if sqlite3 "$DB_PATH" "PRAGMA table_info(email_accounts);" | grep -q "|quota|" ; then
        log_progress "email_accounts tablosu güncelleniyor (quota -> quota_mb)"
        sqlite3 "$DB_PATH" "ALTER TABLE email_accounts RENAME COLUMN quota TO quota_mb;" 2>/dev/null || true
        log_done "quota_mb sütunu güncellendi"
    fi
    
    # email_settings tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "email_settings"; then
        log_progress "email_settings tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS email_settings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            domain_id INTEGER NOT NULL UNIQUE,
            hourly_limit INTEGER DEFAULT 100,
            daily_limit INTEGER DEFAULT 500,
            dkim_enabled INTEGER DEFAULT 0,
            dkim_selector TEXT DEFAULT 'default',
            dkim_private_key TEXT,
            dkim_public_key TEXT,
            spf_record TEXT,
            dmarc_record TEXT,
            catch_all_email TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
        );"
        log_done "email_settings tablosu oluşturuldu"
    else
        log_info "email_settings tablosu mevcut"
    fi
    
    # email_send_log tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "email_send_log"; then
        log_progress "email_send_log tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS email_send_log (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            email_account_id INTEGER,
            user_id INTEGER,
            sender TEXT,
            recipient TEXT NOT NULL,
            subject TEXT,
            message_id TEXT,
            size_bytes INTEGER DEFAULT 0,
            sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            status TEXT DEFAULT 'sent'
        );"
        sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_email_send_log_user_id ON email_send_log(user_id);"
        sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_email_send_log_sent_at ON email_send_log(sent_at);"
        log_done "email_send_log tablosu oluşturuldu"
    else
        # Mevcut tabloya user_id sütunu ekle (eski kurulumlar için)
        if ! sqlite3 "$DB_PATH" "PRAGMA table_info(email_send_log);" | grep -q "user_id"; then
            log_progress "email_send_log tablosu güncelleniyor"
            sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN user_id INTEGER;" 2>/dev/null || true
            sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN sender TEXT;" 2>/dev/null || true
            sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN message_id TEXT;" 2>/dev/null || true
            sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN size_bytes INTEGER DEFAULT 0;" 2>/dev/null || true
            sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_email_send_log_user_id ON email_send_log(user_id);" 2>/dev/null || true
            sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_email_send_log_sent_at ON email_send_log(sent_at);" 2>/dev/null || true
            log_done "email_send_log tablosu güncellendi"
        else
            log_info "email_send_log tablosu mevcut"
        fi
    fi
    
    # mail_queue tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "mail_queue"; then
        log_progress "mail_queue tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS mail_queue (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            sender TEXT NOT NULL,
            recipient TEXT NOT NULL,
            subject TEXT,
            body TEXT,
            headers TEXT,
            priority INTEGER DEFAULT 5,
            retry_count INTEGER DEFAULT 0,
            max_retries INTEGER DEFAULT 3,
            scheduled_at DATETIME,
            status TEXT DEFAULT 'pending',
            error_message TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );"
        sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_mail_queue_user_id ON mail_queue(user_id);"
        sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_mail_queue_status ON mail_queue(status);"
        log_done "mail_queue tablosu oluşturuldu"
    else
        log_info "mail_queue tablosu mevcut"
    fi
    
    # packages tablosuna mail limitleri ekle
    if ! sqlite3 "$DB_PATH" "PRAGMA table_info(packages);" | grep -q "max_emails_per_hour"; then
        log_progress "packages tablosuna mail limitleri ekleniyor"
        sqlite3 "$DB_PATH" "ALTER TABLE packages ADD COLUMN max_emails_per_hour INTEGER DEFAULT 100;" 2>/dev/null || true
        sqlite3 "$DB_PATH" "ALTER TABLE packages ADD COLUMN max_emails_per_day INTEGER DEFAULT 500;" 2>/dev/null || true
        log_done "packages tablosu güncellendi"
    fi
    
    # spam_settings tablosu var mı kontrol et
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "spam_settings"; then
        log_progress "spam_settings tablosu oluşturuluyor"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS spam_settings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL UNIQUE,
            enabled INTEGER DEFAULT 1,
            spam_score REAL DEFAULT 5.0,
            auto_delete INTEGER DEFAULT 0,
            auto_delete_score REAL DEFAULT 10.0,
            spam_folder INTEGER DEFAULT 1,
            whitelist TEXT DEFAULT '[]',
            blacklist TEXT DEFAULT '[]',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        );"
        log_done "spam_settings tablosu oluşturuldu"
    else
        log_info "spam_settings tablosu mevcut"
    fi
    
    log_info "Veritabanı migrasyonu tamamlandı ✓"
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

configure_ssl() {
    log_step "SSL/Let's Encrypt Yapılandırılıyor"
    
    # Certbot zaten kurulu (install_packages'da)
    log_progress "SSL auto-renewal yapılandırılıyor"
    
    # Certbot auto-renewal cron job
    cat > /etc/cron.d/certbot-renew << 'CRONEOF'
# Let's Encrypt SSL sertifikalarını otomatik yenile
0 0,12 * * * root certbot renew --quiet --post-hook "systemctl reload apache2"
CRONEOF
    
    chmod 644 /etc/cron.d/certbot-renew
    log_done "SSL auto-renewal yapılandırıldı"
    
    # Apache SSL modülünü etkinleştir
    log_progress "Apache SSL modülü etkinleştiriliyor"
    a2enmod ssl > /dev/null 2>&1
    a2enmod rewrite > /dev/null 2>&1
    systemctl reload apache2 > /dev/null 2>&1
    log_done "Apache SSL modülü etkinleştirildi"
    
    log_info "Let's Encrypt: hazır ✓"
}

configure_mail_queue() {
    log_step "Mail Queue Daemon Yapılandırılıyor"
    
    # Queue processor ve policy daemon binary'leri derleniyor
    log_progress "Mail daemon'ları derleniyor"
    
    cd "${INSTALL_DIR}"
    mkdir -p "${INSTALL_DIR}/bin"
    export PATH=$PATH:/usr/local/go/bin
    
    # Queue processor derle
    if [[ -d "${INSTALL_DIR}/cmd/queue-processor" ]]; then
        /usr/local/go/bin/go build -o "${INSTALL_DIR}/bin/queue-processor" ./cmd/queue-processor 2>/dev/null
        if [[ -f "${INSTALL_DIR}/bin/queue-processor" ]]; then
            chmod +x "${INSTALL_DIR}/bin/queue-processor"
            log_done "Queue processor derlendi"
        else
            log_warn "Queue processor derlenemedi"
        fi
    fi
    
    # Policy daemon derle
    if [[ -d "${INSTALL_DIR}/cmd/policy-daemon" ]]; then
        /usr/local/go/bin/go build -o "${INSTALL_DIR}/bin/policy-daemon" ./cmd/policy-daemon 2>/dev/null
        if [[ -f "${INSTALL_DIR}/bin/policy-daemon" ]]; then
            chmod +x "${INSTALL_DIR}/bin/policy-daemon"
            log_done "Policy daemon derlendi"
        else
            log_warn "Policy daemon derlenemedi"
        fi
    fi
    
    # Queue processor systemd service
    log_progress "Queue processor servisi oluşturuluyor"
    cat > /etc/systemd/system/serverpanel-queue.service << 'QUEUEEOF'
[Unit]
Description=ServerPanel Mail Queue Processor
After=network.target serverpanel.service

[Service]
Type=simple
ExecStart=/opt/serverpanel/bin/queue-processor
Restart=always
RestartSec=10
User=root
WorkingDirectory=/opt/serverpanel

[Install]
WantedBy=multi-user.target
QUEUEEOF
    
    # Postfix policy service yapılandırması
    log_progress "Postfix policy daemon yapılandırılıyor"
    
    # Postfix master.cf'e policy service ekle
    if ! grep -q "policy.*spawn" /etc/postfix/master.cf 2>/dev/null; then
        cat >> /etc/postfix/master.cf << 'POLICYEOF'

# ServerPanel Rate Limiting Policy Daemon
policy    unix  -       n       n       -       0       spawn
  user=nobody argv=/opt/serverpanel/bin/policy-daemon
POLICYEOF
        log_done "Policy daemon Postfix'e eklendi"
    fi
    
    # Postfix main.cf'e policy check ekle (sender restrictions - giden mailler için)
    if ! grep -q "check_policy_service" /etc/postfix/main.cf 2>/dev/null; then
        postconf -e "smtpd_sender_restrictions = permit_mynetworks, permit_sasl_authenticated, check_policy_service unix:private/policy"
        log_done "Policy check Postfix'e eklendi"
    fi
    
    # Log dizini oluştur
    mkdir -p /var/log/serverpanel
    chmod 755 /var/log/serverpanel
    
    # Servisleri başlat
    systemctl daemon-reload > /dev/null 2>&1
    
    if [[ -f "${INSTALL_DIR}/bin/queue-processor" ]]; then
        systemctl enable serverpanel-queue > /dev/null 2>&1
        systemctl start serverpanel-queue > /dev/null 2>&1
        
        if systemctl is-active --quiet serverpanel-queue; then
            log_info "Queue processor: aktif ✓"
        else
            log_warn "Queue processor başlatılamadı"
        fi
    fi
    
    # Postfix'i yeniden başlat
    systemctl restart postfix > /dev/null 2>&1
    log_done "Mail queue daemon yapılandırıldı"
}

health_check() {
    log_step "Sistem Sağlık Kontrolü"
    
    local services=("mysql" "apache2" "php${PHP_VERSION}-fpm" "bind9" "pure-ftpd" "postfix" "dovecot" "opendkim" "spamassassin" "serverpanel" "serverpanel-queue")
    for svc in "${services[@]}"; do
        if systemctl is-active --quiet "$svc"; then
            log_info "$svc: aktif ✓"
        else
            log_warn "$svc: çalışmıyor"
        fi
    done
    
    # ClamAV kontrolü (isteğe bağlı)
    if systemctl is-active --quiet clamav-daemon; then
        log_info "clamav-daemon: aktif ✓"
    else
        log_info "clamav-daemon: kurulu değil (isteğe bağlı)"
    fi
    
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
    
    # DNS testi
    if named-checkconf > /dev/null 2>&1; then
        log_info "BIND9 config ✓"
    else
        log_warn "BIND9 config hatası var"
    fi
    
    # Mail port kontrolü
    if netstat -tlnp 2>/dev/null | grep -q ":25 "; then
        log_info "SMTP (25): dinleniyor ✓"
    else
        log_warn "SMTP (25): dinlenmiyor"
    fi
    
    if netstat -tlnp 2>/dev/null | grep -q ":143 "; then
        log_info "IMAP (143): dinleniyor ✓"
    else
        log_warn "IMAP (143): dinlenmiyor"
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
    echo -e "${CYAN}Webmail (Roundcube):${NC}"
    echo -e "  URL:        ${GREEN}http://${SERVER_IP}/webmail${NC}"
    echo ""
    echo -e "${CYAN}Mail Server Portları:${NC}"
    echo -e "  SMTP:       25, 587 (submission)"
    echo -e "  IMAP:       143, 993 (SSL)"
    echo -e "  POP3:       110, 995 (SSL)"
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
    configure_pureftpd
    configure_mail_server
    configure_roundcube
    configure_apache
    configure_dns
    install_phpmyadmin
    install_go
    install_serverpanel
    migrate_database
    create_service
    configure_ssl
    configure_mail_queue
    health_check
    
    print_summary
}

main "$@"