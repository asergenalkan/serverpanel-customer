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

set -e

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
GITHUB_REPO="asergenalkan/serverpanel"
RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}"

# Sayaçlar
STEP_CURRENT=0
STEP_TOTAL=10
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

log_info() {
    echo -e "  ${GREEN}✓${NC} $1"
}

log_warn() {
    echo -e "  ${YELLOW}⚠${NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

log_error() {
    echo -e "  ${RED}✗${NC} $1"
    ERRORS=$((ERRORS + 1))
}

log_detail() {
    echo -e "    ${CYAN}→${NC} $1"
}

log_progress() {
    echo -ne "  ${MAGENTA}◌${NC} $1...\r"
}

log_done() {
    echo -e "  ${GREEN}●${NC} $1              "
}

# ═══════════════════════════════════════════════════════════════════════════════
# KONTROL FONKSİYONLARI
# ═══════════════════════════════════════════════════════════════════════════════

check_root() {
    log_step "Yetki Kontrolü"
    
    if [[ $EUID -ne 0 ]]; then
        log_error "Bu script root yetkisi gerektirir!"
        echo ""
        echo -e "  ${YELLOW}Kullanım:${NC}"
        echo -e "    ${WHITE}sudo bash install.sh${NC}"
        echo ""
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
    
    log_detail "Dağıtım: $NAME $VERSION_ID"
    
    if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
        log_error "Sadece Ubuntu ve Debian desteklenmektedir"
        exit 1
    fi
    
    if [[ "$ID" == "ubuntu" && "${VERSION_ID%%.*}" -lt 20 ]]; then
        log_error "Ubuntu 20.04 veya üzeri gereklidir"
        exit 1
    fi
    
    log_info "İşletim sistemi: $PRETTY_NAME"
    
    # Mimari
    ARCH=$(uname -m)
    if [[ "$ARCH" != "x86_64" ]]; then
        log_error "Sadece x86_64 (64-bit) desteklenmektedir"
        exit 1
    fi
    log_info "Mimari: $ARCH"
}

check_resources() {
    log_step "Sistem Kaynakları Kontrolü"
    
    # RAM
    local total_ram=$(free -m | awk '/^Mem:/{print $2}')
    if [[ $total_ram -lt 512 ]]; then
        log_error "Minimum 512MB RAM gerekli (Mevcut: ${total_ram}MB)"
        exit 1
    fi
    log_info "RAM: ${total_ram}MB"
    
    # Disk
    local free_disk=$(df -m / | awk 'NR==2 {print $4}')
    if [[ $free_disk -lt 2048 ]]; then
        log_error "Minimum 2GB boş alan gerekli (Mevcut: ${free_disk}MB)"
        exit 1
    fi
    log_info "Disk: ${free_disk}MB boş"
    
    # CPU
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
    
    # PHP sürümünü belirle
    source /etc/os-release
    if [[ "$VERSION_ID" == "24.04" ]]; then
        PHP_VERSION="8.3"
    else
        PHP_VERSION="8.1"
    fi
    
    local packages=(
        # Temel
        curl wget git unzip tar net-tools
        # Apache
        apache2 libapache2-mod-fcgid
        # PHP
        php${PHP_VERSION}-fpm php${PHP_VERSION}-cli php${PHP_VERSION}-mysql
        php${PHP_VERSION}-curl php${PHP_VERSION}-gd php${PHP_VERSION}-mbstring
        php${PHP_VERSION}-xml php${PHP_VERSION}-zip php${PHP_VERSION}-intl
        # MySQL
        mysql-server
        # DNS
        bind9 bind9-utils
        # SSL
        certbot python3-certbot-apache
    )
    
    log_progress "Paketler kuruluyor (bu biraz sürebilir)"
    DEBIAN_FRONTEND=noninteractive apt-get install -y "${packages[@]}" > /dev/null 2>&1
    log_done "Tüm paketler kuruldu"
    
    # Kurulu paketleri listele
    log_info "Apache: $(apache2 -v 2>/dev/null | head -1 | awk '{print $3}')"
    log_info "PHP: ${PHP_VERSION}"
    log_info "MySQL: $(mysql --version | awk '{print $3}')"
}

configure_apache() {
    log_step "Apache Yapılandırılıyor"
    
    log_progress "Apache modülleri aktifleştiriliyor"
    a2enmod proxy_fcgi setenvif rewrite headers ssl expires > /dev/null 2>&1
    log_done "Modüller aktif"
    
    # PHP-FPM entegrasyonu
    source /etc/os-release
    [[ "$VERSION_ID" == "24.04" ]] && PHP_VERSION="8.3" || PHP_VERSION="8.1"
    
    a2enconf php${PHP_VERSION}-fpm > /dev/null 2>&1 || true
    a2dissite 000-default > /dev/null 2>&1 || true
    
    systemctl enable apache2 > /dev/null 2>&1
    systemctl restart apache2
    
    log_info "Apache durumu: $(systemctl is-active apache2)"
}

configure_mysql() {
    log_step "MySQL Yapılandırılıyor"
    
    systemctl enable mysql > /dev/null 2>&1
    systemctl start mysql
    
    # Root şifresi
    local MYSQL_ROOT_PASS=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 16)
    
    mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}';" 2>/dev/null || true
    mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true
    
    # Şifreyi kaydet
    mkdir -p /root/.serverpanel
    echo "MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASS}" > /root/.serverpanel/mysql.conf
    chmod 600 /root/.serverpanel/mysql.conf
    
    log_info "MySQL durumu: $(systemctl is-active mysql)"
    log_info "Root şifresi kaydedildi: /root/.serverpanel/mysql.conf"
}

configure_dns() {
    log_step "DNS Yapılandırılıyor"
    
    mkdir -p /etc/bind/zones
    chown bind:bind /etc/bind/zones
    
    systemctl enable bind9 > /dev/null 2>&1
    systemctl start bind9
    
    log_info "BIND durumu: $(systemctl is-active bind9)"
}

configure_php() {
    log_step "PHP-FPM Yapılandırılıyor"
    
    source /etc/os-release
    [[ "$VERSION_ID" == "24.04" ]] && PHP_VERSION="8.3" || PHP_VERSION="8.1"
    
    systemctl enable php${PHP_VERSION}-fpm > /dev/null 2>&1
    systemctl restart php${PHP_VERSION}-fpm
    
    log_info "PHP-FPM durumu: $(systemctl is-active php${PHP_VERSION}-fpm)"
}

install_serverpanel() {
    log_step "ServerPanel Kuruluyor"
    
    # Dizinleri oluştur
    mkdir -p $INSTALL_DIR/public
    mkdir -p $DATA_DIR
    mkdir -p $LOG_DIR
    
    # Backend binary indir
    log_progress "Backend indiriliyor"
    curl -sSL "${RELEASE_URL}/serverpanel-linux-amd64" -o $INSTALL_DIR/serverpanel
    chmod +x $INSTALL_DIR/serverpanel
    log_done "Backend indirildi"
    
    # Frontend indir
    log_progress "Frontend indiriliyor"
    curl -sSL "${RELEASE_URL}/serverpanel-frontend.tar.gz" -o /tmp/frontend.tar.gz
    tar -xzf /tmp/frontend.tar.gz -C $INSTALL_DIR/public
    rm /tmp/frontend.tar.gz
    log_done "Frontend indirildi"
    
    log_info "Binary: $INSTALL_DIR/serverpanel"
    log_info "Frontend: $INSTALL_DIR/public/"
}

create_service() {
    log_step "Servis Oluşturuluyor"
    
    local SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    
    # MySQL şifresini oku
    local MYSQL_PASS=""
    [[ -f /root/.serverpanel/mysql.conf ]] && source /root/.serverpanel/mysql.conf && MYSQL_PASS=$MYSQL_ROOT_PASSWORD
    
    # PHP sürümü
    source /etc/os-release
    [[ "$VERSION_ID" == "24.04" ]] && PHP_VERSION="8.3" || PHP_VERSION="8.1"
    
    cat > /etc/systemd/system/serverpanel.service << EOF
[Unit]
Description=ServerPanel - Web Hosting Control Panel
After=network.target mysql.service apache2.service

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
Environment="MYSQL_ROOT_PASSWORD=${MYSQL_PASS}"
Environment="SERVER_IP=${SERVER_IP}"
Environment="PHP_VERSION=${PHP_VERSION}"
Environment="WEB_SERVER=apache"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable serverpanel > /dev/null 2>&1
    systemctl start serverpanel
    
    sleep 2
    
    if systemctl is-active --quiet serverpanel; then
        log_info "ServerPanel durumu: ${GREEN}aktif${NC}"
    else
        log_error "ServerPanel başlatılamadı!"
        log_error "Hata: journalctl -u serverpanel -n 20"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# KURULUM ÖZETİ
# ═══════════════════════════════════════════════════════════════════════════════

print_summary() {
    local SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    local END_TIME=$(date +%s)
    local DURATION=$((END_TIME - START_TIME))
    
    echo ""
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    ${WHITE}${BOLD}KURULUM BAŞARIYLA TAMAMLANDI!${NC}${GREEN}                       ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  ${CYAN}Süre:${NC} ${DURATION} saniye"
    echo ""
    echo -e "${CYAN}┌─────────────────────────────────────────────────────────────────────────────┐${NC}"
    echo -e "${CYAN}│${NC} ${WHITE}${BOLD}Panel Erişimi${NC}                                                              ${CYAN}│${NC}"
    echo -e "${CYAN}├─────────────────────────────────────────────────────────────────────────────┤${NC}"
    echo -e "${CYAN}│${NC}                                                                             ${CYAN}│${NC}"
    echo -e "${CYAN}│${NC}   ${YELLOW}URL:${NC}        ${GREEN}http://${SERVER_IP}:8443${NC}                             ${CYAN}│${NC}"
    echo -e "${CYAN}│${NC}   ${YELLOW}Kullanıcı:${NC}  ${WHITE}admin${NC}                                                       ${CYAN}│${NC}"
    echo -e "${CYAN}│${NC}   ${YELLOW}Şifre:${NC}      ${WHITE}admin123${NC}                                                    ${CYAN}│${NC}"
    echo -e "${CYAN}│${NC}                                                                             ${CYAN}│${NC}"
    echo -e "${CYAN}└─────────────────────────────────────────────────────────────────────────────┘${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  GÜVENLİK: İlk girişte şifrenizi değiştirin!${NC}"
    echo ""
    echo -e "${CYAN}Servis Komutları:${NC}"
    echo -e "  systemctl status serverpanel   # Durum"
    echo -e "  systemctl restart serverpanel  # Yeniden başlat"
    echo -e "  journalctl -u serverpanel -f   # Loglar"
    echo ""
    
    if [[ $WARNINGS -gt 0 ]]; then
        echo -e "${YELLOW}ℹ️  $WARNINGS uyarı oluştu${NC}"
    fi
    
    echo -e "${GREEN}Kurulum tamamlandı!${NC}"
    echo ""
}

# ═══════════════════════════════════════════════════════════════════════════════
# ANA FONKSİYON
# ═══════════════════════════════════════════════════════════════════════════════

main() {
    print_banner
    
    echo -e "${WHITE}Kurulum başlatılıyor...${NC}"
    
    check_root
    check_os
    check_resources
    install_packages
    configure_apache
    configure_mysql
    configure_dns
    configure_php
    install_serverpanel
    create_service
    
    print_summary
}

main "$@"
