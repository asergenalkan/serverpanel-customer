#!/bin/bash
#
# ServerPanel Kurulum Scripti
# https://github.com/asergenalkan/serverpanel
#
# Kullanım:
#   curl -sSL https://raw.githubusercontent.com/asergenalkan/serverpanel/main/install.sh | bash
#
# veya:
#   wget -qO- https://raw.githubusercontent.com/asergenalkan/serverpanel/main/install.sh | bash
#

set -e

# Renkler
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Versiyon
VERSION="1.0.0"
INSTALL_DIR="/opt/serverpanel"
DATA_DIR="/var/lib/serverpanel"
LOG_DIR="/var/log/serverpanel"
GITHUB_REPO="asergenalkan/serverpanel"

# Banner
print_banner() {
    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║                                                           ║"
    echo "║   ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗        ║"
    echo "║   ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗       ║"
    echo "║   ███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝       ║"
    echo "║   ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗       ║"
    echo "║   ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║       ║"
    echo "║   ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝       ║"
    echo "║                     PANEL v${VERSION}                          ║"
    echo "║                                                           ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Log fonksiyonları
log_info() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[!]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[→]${NC} $1"
}

# Root kontrolü
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "Bu script root olarak çalıştırılmalıdır!"
        echo "Kullanım: sudo bash install.sh"
        exit 1
    fi
}

# İşletim sistemi kontrolü
check_os() {
    log_step "İşletim sistemi kontrol ediliyor..."
    
    if [[ ! -f /etc/os-release ]]; then
        log_error "Desteklenmeyen işletim sistemi!"
        exit 1
    fi
    
    source /etc/os-release
    
    if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
        log_error "Bu script sadece Ubuntu ve Debian için desteklenmektedir."
        log_error "Tespit edilen: $ID $VERSION_ID"
        exit 1
    fi
    
    if [[ "$ID" == "ubuntu" && "${VERSION_ID%%.*}" -lt 20 ]]; then
        log_error "Ubuntu 20.04 veya üzeri gereklidir."
        exit 1
    fi
    
    log_info "İşletim sistemi: $PRETTY_NAME"
}

# Sistem gereksinimleri kontrolü
check_requirements() {
    log_step "Sistem gereksinimleri kontrol ediliyor..."
    
    # RAM kontrolü
    total_ram=$(free -m | awk '/^Mem:/{print $2}')
    if [[ $total_ram -lt 512 ]]; then
        log_error "Minimum 512MB RAM gereklidir. Mevcut: ${total_ram}MB"
        exit 1
    fi
    log_info "RAM: ${total_ram}MB"
    
    # Disk kontrolü
    free_disk=$(df -m / | awk 'NR==2 {print $4}')
    if [[ $free_disk -lt 2048 ]]; then
        log_error "Minimum 2GB boş disk alanı gereklidir. Mevcut: ${free_disk}MB"
        exit 1
    fi
    log_info "Boş disk: ${free_disk}MB"
    
    # Port kontrolü
    for port in 80 443 8443; do
        if netstat -tuln 2>/dev/null | grep -q ":$port " || ss -tuln | grep -q ":$port "; then
            log_warn "Port $port kullanımda. Bu servisi durdurun veya portu değiştirin."
        fi
    done
}

# Paketleri yükle
install_packages() {
    log_step "Paketler güncelleniyor ve yükleniyor..."
    
    # Repo güncelle
    apt-get update -qq
    
    # PHP sürümünü belirle - Ubuntu 22.04'te varsayılan 8.1
    source /etc/os-release
    if [[ "$VERSION_ID" == "24.04" ]]; then
        PHP_VERSION="8.3"
    elif [[ "$VERSION_ID" == "22.04" ]]; then
        PHP_VERSION="8.1"
    elif [[ "$VERSION_ID" == "20.04" ]]; then
        PHP_VERSION="7.4"
    else
        PHP_VERSION="8.1"
    fi
    
    log_info "PHP sürümü: ${PHP_VERSION}"
    
    # Paket listesi
    PACKAGES=(
        # Web Server
        apache2
        libapache2-mod-fcgid
        
        # PHP
        php${PHP_VERSION}-fpm
        php${PHP_VERSION}-cli
        php${PHP_VERSION}-mysql
        php${PHP_VERSION}-curl
        php${PHP_VERSION}-gd
        php${PHP_VERSION}-mbstring
        php${PHP_VERSION}-xml
        php${PHP_VERSION}-zip
        php${PHP_VERSION}-intl
        php${PHP_VERSION}-bcmath
        
        # DNS
        bind9
        bind9-utils
        bind9-dnsutils
        
        # Database
        mysql-server
        
        # SSL
        certbot
        python3-certbot-apache
        
        # Diğer
        curl
        wget
        git
        unzip
        net-tools
    )
    
    log_step "Paketler yükleniyor... (bu biraz zaman alabilir)"
    
    DEBIAN_FRONTEND=noninteractive apt-get install -y "${PACKAGES[@]}" > /dev/null 2>&1
    
    log_info "Tüm paketler yüklendi"
}

# Apache yapılandırması
configure_apache() {
    log_step "Apache yapılandırılıyor..."
    
    # Modülleri aktif et
    a2enmod proxy_fcgi setenvif rewrite headers ssl expires > /dev/null 2>&1
    
    # PHP-FPM yapılandırması
    source /etc/os-release
    if [[ "$VERSION_ID" == "22.04" || "$VERSION_ID" == "24.04" ]]; then
        PHP_VERSION="8.2"
    else
        PHP_VERSION="8.1"
    fi
    
    a2enconf php${PHP_VERSION}-fpm > /dev/null 2>&1 || true
    
    # Varsayılan site devre dışı
    a2dissite 000-default > /dev/null 2>&1 || true
    
    # Apache yeniden başlat
    systemctl restart apache2
    systemctl enable apache2 > /dev/null 2>&1
    
    log_info "Apache yapılandırıldı"
}

# MySQL yapılandırması
configure_mysql() {
    log_step "MySQL yapılandırılıyor..."
    
    # MySQL başlat
    systemctl start mysql
    systemctl enable mysql > /dev/null 2>&1
    
    # Root şifresi oluştur
    MYSQL_ROOT_PASS=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 16)
    
    # Root şifresini ayarla
    mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}';" 2>/dev/null || true
    mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true
    
    # Şifreyi kaydet
    echo "MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASS}" > /root/.serverpanel_mysql
    chmod 600 /root/.serverpanel_mysql
    
    log_info "MySQL yapılandırıldı"
    log_info "MySQL root şifresi: /root/.serverpanel_mysql"
}

# BIND yapılandırması
configure_bind() {
    log_step "BIND DNS yapılandırılıyor..."
    
    # Zones dizini oluştur
    mkdir -p /etc/bind/zones
    chown bind:bind /etc/bind/zones
    
    # BIND başlat
    systemctl start bind9
    systemctl enable bind9 > /dev/null 2>&1
    
    log_info "BIND DNS yapılandırıldı"
}

# PHP-FPM yapılandırması
configure_php() {
    log_step "PHP-FPM yapılandırılıyor..."
    
    source /etc/os-release
    if [[ "$VERSION_ID" == "22.04" || "$VERSION_ID" == "24.04" ]]; then
        PHP_VERSION="8.2"
    else
        PHP_VERSION="8.1"
    fi
    
    # PHP-FPM başlat
    systemctl start php${PHP_VERSION}-fpm
    systemctl enable php${PHP_VERSION}-fpm > /dev/null 2>&1
    
    log_info "PHP-FPM yapılandırıldı"
}

# ServerPanel yükle
install_serverpanel() {
    log_step "ServerPanel indiriliyor..."
    
    # Dizinleri oluştur
    mkdir -p $INSTALL_DIR
    mkdir -p $DATA_DIR
    mkdir -p $LOG_DIR
    
    # Binary indir
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/serverpanel-linux-amd64"
    
    # Release yoksa doğrudan repo'dan al
    if ! curl -sSLf -o $INSTALL_DIR/serverpanel "$DOWNLOAD_URL" 2>/dev/null; then
        log_warn "Release bulunamadı, manuel kurulum gerekiyor"
        log_step "Binary'yi manuel olarak yükleyin:"
        echo "  scp serverpanel-linux root@$(hostname -I | awk '{print $1}'):$INSTALL_DIR/serverpanel"
        touch $INSTALL_DIR/.manual_install_required
    else
        chmod +x $INSTALL_DIR/serverpanel
        log_info "ServerPanel indirildi"
    fi
}

# Systemd service oluştur
create_service() {
    log_step "Systemd service oluşturuluyor..."
    
    # MySQL root şifresini oku
    if [[ -f /root/.serverpanel_mysql ]]; then
        source /root/.serverpanel_mysql
    fi
    
    # Sunucu IP'sini al
    SERVER_IP=$(curl -s ifconfig.me || hostname -I | awk '{print $1}')
    
    cat > /etc/systemd/system/serverpanel.service << EOF
[Unit]
Description=ServerPanel - Web Hosting Control Panel
Documentation=https://github.com/${GITHUB_REPO}
After=network.target mysql.service apache2.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/serverpanel
Restart=always
RestartSec=5
StandardOutput=append:${LOG_DIR}/panel.log
StandardError=append:${LOG_DIR}/error.log

# Environment
Environment="ENVIRONMENT=production"
Environment="PORT=8443"
Environment="MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASS}"
Environment="SERVER_IP=${SERVER_IP}"
Environment="PHP_VERSION=${PHP_VERSION:-8.2}"
Environment="WEB_SERVER=apache"

# Security
NoNewPrivileges=false
ProtectSystem=false
ProtectHome=false

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_info "Systemd service oluşturuldu"
}

# Firewall yapılandırması
configure_firewall() {
    log_step "Firewall yapılandırılıyor..."
    
    if command -v ufw &> /dev/null; then
        ufw allow 22/tcp > /dev/null 2>&1 || true
        ufw allow 80/tcp > /dev/null 2>&1 || true
        ufw allow 443/tcp > /dev/null 2>&1 || true
        ufw allow 8443/tcp > /dev/null 2>&1 || true
        ufw allow 53/tcp > /dev/null 2>&1 || true
        ufw allow 53/udp > /dev/null 2>&1 || true
        log_info "UFW firewall yapılandırıldı"
    else
        log_warn "UFW bulunamadı, firewall manuel yapılandırın"
    fi
}

# Servisi başlat
start_service() {
    if [[ -f $INSTALL_DIR/.manual_install_required ]]; then
        log_warn "Binary manuel olarak yüklenene kadar servis başlatılamaz"
        return
    fi
    
    log_step "ServerPanel başlatılıyor..."
    
    systemctl enable serverpanel > /dev/null 2>&1
    systemctl start serverpanel
    
    sleep 2
    
    if systemctl is-active --quiet serverpanel; then
        log_info "ServerPanel başlatıldı"
    else
        log_error "ServerPanel başlatılamadı!"
        log_error "Loglar için: journalctl -u serverpanel -f"
    fi
}

# Kurulum özeti
print_summary() {
    SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║              KURULUM TAMAMLANDI!                          ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${CYAN}Panel Erişimi:${NC}"
    echo -e "  URL: ${GREEN}http://${SERVER_IP}:8443${NC}"
    echo -e "  Kullanıcı: ${GREEN}admin${NC}"
    echo -e "  Şifre: ${GREEN}admin123${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  İlk girişte şifrenizi değiştirin!${NC}"
    echo ""
    echo -e "${CYAN}Önemli Dosyalar:${NC}"
    echo -e "  Panel: ${INSTALL_DIR}/serverpanel"
    echo -e "  Loglar: ${LOG_DIR}/"
    echo -e "  MySQL şifresi: /root/.serverpanel_mysql"
    echo ""
    echo -e "${CYAN}Servis Komutları:${NC}"
    echo -e "  Başlat: ${GREEN}systemctl start serverpanel${NC}"
    echo -e "  Durdur: ${GREEN}systemctl stop serverpanel${NC}"
    echo -e "  Durum: ${GREEN}systemctl status serverpanel${NC}"
    echo -e "  Loglar: ${GREEN}journalctl -u serverpanel -f${NC}"
    echo ""
    
    if [[ -f $INSTALL_DIR/.manual_install_required ]]; then
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${YELLOW}Binary manuel olarak yüklenmeli:${NC}"
        echo -e "  ${CYAN}scp serverpanel-linux root@${SERVER_IP}:${INSTALL_DIR}/serverpanel${NC}"
        echo -e "  ${CYAN}chmod +x ${INSTALL_DIR}/serverpanel${NC}"
        echo -e "  ${CYAN}systemctl start serverpanel${NC}"
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    fi
    
    echo ""
}

# Ana kurulum fonksiyonu
main() {
    print_banner
    
    echo -e "${CYAN}ServerPanel kurulumu başlıyor...${NC}"
    echo ""
    
    check_root
    check_os
    check_requirements
    
    echo ""
    log_step "Kurulum başlatılıyor..."
    echo ""
    
    install_packages
    configure_apache
    configure_mysql
    configure_bind
    configure_php
    install_serverpanel
    create_service
    configure_firewall
    start_service
    
    print_summary
}

# Scripti çalıştır
main "$@"
