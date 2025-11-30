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
    
    log_progress "Paket listesi güncelleniyor"
    apt-get update -qq > /dev/null 2>&1
    log_done "Paket listesi güncellendi"
    
    local packages=(
        curl wget git unzip tar net-tools sqlite3
        apache2 libapache2-mod-fcgid
        php${PHP_VERSION}-fpm php${PHP_VERSION}-cli php${PHP_VERSION}-mysql
        php${PHP_VERSION}-curl php${PHP_VERSION}-gd php${PHP_VERSION}-mbstring
        php${PHP_VERSION}-xml php${PHP_VERSION}-zip php${PHP_VERSION}-intl
        mysql-server mysql-client
        bind9 bind9-utils
        certbot python3-certbot-apache
        build-essential
    )
    
    log_progress "Paketler kuruluyor (bu biraz sürebilir)"
    DEBIAN_FRONTEND=noninteractive apt-get install -y "${packages[@]}" > /dev/null 2>&1
    log_done "Tüm paketler kuruldu"
    
    log_info "Apache: $(apache2 -v 2>/dev/null | head -1 | awk '{print $3}')"
    log_info "PHP: ${PHP_VERSION}"
    log_info "MySQL: $(mysql --version 2>/dev/null | awk '{print $3}')"
}

# ═══════════════════════════════════════════════════════════════════════════════
# MYSQL YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_mysql() {
    log_step "MySQL Yapılandırılıyor"
    
    ensure_directory "$CONFIG_DIR" "root" "700"
    
    # MySQL socket dizinini oluştur
    log_detail "Socket dizini hazırlanıyor"
    ensure_directory "/var/run/mysqld" "mysql" "755"
    
    # MySQL servisini başlat
    log_progress "MySQL servisi başlatılıyor"
    systemctl enable mysql > /dev/null 2>&1
    systemctl stop mysql > /dev/null 2>&1
    sleep 1
    systemctl start mysql > /dev/null 2>&1
    
    # Servisin başlamasını bekle
    local attempts=0
    while [[ $attempts -lt 15 ]]; do
        if systemctl is-active --quiet mysql && [[ -S /var/run/mysqld/mysqld.sock ]]; then
            break
        fi
        sleep 1
        attempts=$((attempts + 1))
    done
    
    if ! systemctl is-active --quiet mysql; then
        log_error "MySQL servisi başlatılamadı!"
        journalctl -u mysql -n 10 --no-pager
        exit 1
    fi
    log_done "MySQL servisi başlatıldı"
    
    if [[ ! -S /var/run/mysqld/mysqld.sock ]]; then
        log_error "MySQL socket dosyası bulunamadı!"
        exit 1
    fi
    log_info "MySQL socket: /var/run/mysqld/mysqld.sock ✓"
    
    # Root şifresi oluştur
    local MYSQL_ROOT_PASS=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 16)
    
    # Şifreyi değiştir
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
    
    if [[ "$password_set" == "false" ]]; then
        log_error "MySQL root şifresi ayarlanamadı!"
        exit 1
    fi
    log_done "MySQL root şifresi ayarlandı"
    
    # Test et
    if mysql -uroot -p"${MYSQL_ROOT_PASS}" -e "SELECT 1;" > /dev/null 2>&1; then
        log_info "MySQL bağlantısı: başarılı ✓"
    else
        log_error "MySQL bağlantısı başarısız!"
        exit 1
    fi
    
    # Kaydet
    echo "MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASS}" > "${CONFIG_DIR}/mysql.conf"
    chmod 600 "${CONFIG_DIR}/mysql.conf"
    
    log_info "Root şifresi: ${CONFIG_DIR}/mysql.conf"
}

# ═══════════════════════════════════════════════════════════════════════════════
# PHP-FPM YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_php() {
    log_step "PHP-FPM Yapılandırılıyor"
    
    ensure_directory "/run/php" "www-data" "755"
    
    log_progress "PHP-FPM servisi başlatılıyor"
    systemctl enable php${PHP_VERSION}-fpm > /dev/null 2>&1
    systemctl restart php${PHP_VERSION}-fpm > /dev/null 2>&1
    
    sleep 2
    
    if ! systemctl is-active --quiet php${PHP_VERSION}-fpm; then
        log_error "PHP-FPM başlatılamadı!"
        exit 1
    fi
    log_done "PHP-FPM başlatıldı"
    
    local socket_path="/run/php/php${PHP_VERSION}-fpm.sock"
    if [[ -S "$socket_path" ]]; then
        log_info "PHP-FPM socket: $socket_path ✓"
    else
        log_warn "PHP-FPM socket bulunamadı"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# APACHE YAPILANDIRMASI
# ═══════════════════════════════════════════════════════════════════════════════

configure_apache() {
    log_step "Apache Yapılandırılıyor"
    
    log_progress "Apache modülleri aktifleştiriliyor"
    a2enmod proxy_fcgi setenvif rewrite headers ssl expires proxy alias > /dev/null 2>&1
    log_done "Modüller aktif"
    
    a2enconf php${PHP_VERSION}-fpm > /dev/null 2>&1 || true
    
    ensure_directory "/var/www/html" "www-data" "755"
    echo "<h1>ServerPanel</h1><p>Server is running.</p>" > /var/www/html/index.html
    
    # Default site
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
    
    a2ensite 000-default > /dev/null 2>&1
    
    systemctl enable apache2 > /dev/null 2>&1
    systemctl restart apache2
    
    if systemctl is-active --quiet apache2; then
        log_info "Apache durumu: aktif ✓"
    else
        log_error "Apache başlatılamadı!"
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
    
    # Blowfish secret
    if [[ -f /etc/phpmyadmin/config.inc.php ]]; then
        local BLOWFISH=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
        sed -i "s/\$cfg\['blowfish_secret'\] = ''/\$cfg['blowfish_secret'] = '${BLOWFISH}'/" /etc/phpmyadmin/config.inc.php 2>/dev/null || true
    fi
    
    # Signon authentication için config ekle
    log_progress "Signon authentication ayarlanıyor"
    mkdir -p /etc/phpmyadmin/conf.d
    cat > /etc/phpmyadmin/conf.d/serverpanel.php << 'SIGNONEOF'
<?php
// ServerPanel Signon Authentication
$cfg['Servers'][1]['auth_type'] = 'signon';
$cfg['Servers'][1]['SignonSession'] = 'SignonSession';
$cfg['Servers'][1]['SignonURL'] = '/pma-signon.php';
$cfg['Servers'][1]['LogoutURL'] = '/phpmyadmin/';
SIGNONEOF
    log_done "Signon config oluşturuldu"
    
    # Signon PHP script'i oluştur
    log_progress "Signon script oluşturuluyor"
    cat > /var/www/html/pma-signon.php << 'SCRIPTEOF'
<?php
session_name('SignonSession');
session_start();

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

$_SESSION['PMA_single_signon_user'] = $data['user'];
$_SESSION['PMA_single_signon_password'] = $data['password'];
$_SESSION['PMA_single_signon_host'] = 'localhost';

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
    curl -sSL "https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/serverpanel-frontend.tar.gz" -o /tmp/frontend.tar.gz 2>/dev/null
    if [[ -f /tmp/frontend.tar.gz ]]; then
        tar -xzf /tmp/frontend.tar.gz -C "$INSTALL_DIR/public" 2>/dev/null || true
        rm -f /tmp/frontend.tar.gz
        log_done "Frontend indirildi"
    else
        log_warn "Frontend indirilemedi"
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