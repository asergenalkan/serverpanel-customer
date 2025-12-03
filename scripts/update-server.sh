#!/bin/bash
#
# ServerPanel Sunucu Güncelleme Scripti
# Bu script sunucudaki ServerPanel'i GitHub'dan günceller
#
# Kullanım: ./update-server.sh
#

set -e

# Renkler
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

INSTALL_DIR="/opt/serverpanel"
GITHUB_REPO="asergenalkan/serverpanel"

echo -e "${CYAN}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}              ServerPanel Güncelleme Scripti${NC}"
echo -e "${CYAN}════════════════════════════════════════════════════════════════════${NC}"
echo ""

# Root kontrolü
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}Bu script root olarak çalıştırılmalı!${NC}"
   exit 1
fi

# Mevcut kurulum kontrolü
if [[ ! -d "$INSTALL_DIR" ]]; then
    echo -e "${RED}ServerPanel kurulu değil: $INSTALL_DIR${NC}"
    exit 1
fi

# Servisleri durdur
echo -e "${YELLOW}[1/6] Servisler durduruluyor...${NC}"
systemctl stop serverpanel 2>/dev/null || true
systemctl stop serverpanel-queue 2>/dev/null || true
echo -e "${GREEN}✓ Servisler durduruldu${NC}"

# Yedek al
echo -e "${YELLOW}[2/6] Yedek alınıyor...${NC}"
BACKUP_DIR="/tmp/serverpanel-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp "$INSTALL_DIR/serverpanel" "$BACKUP_DIR/" 2>/dev/null || true
cp -r "$INSTALL_DIR/public" "$BACKUP_DIR/" 2>/dev/null || true
echo -e "${GREEN}✓ Yedek alındı: $BACKUP_DIR${NC}"

# GitHub'dan güncelle
echo -e "${YELLOW}[3/6] GitHub'dan güncelleniyor...${NC}"
cd "$INSTALL_DIR"

# Eğer git repo değilse, yeniden clone et
if [[ ! -d ".git" ]]; then
    echo -e "${YELLOW}Git repo bulunamadı, yeniden indiriliyor...${NC}"
    cd /tmp
    rm -rf serverpanel-temp
    git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" serverpanel-temp
    rm -rf "$INSTALL_DIR"
    mv serverpanel-temp "$INSTALL_DIR"
    cd "$INSTALL_DIR"
else
    git fetch origin main
    git reset --hard origin/main
fi
echo -e "${GREEN}✓ Kaynak kod güncellendi${NC}"

# Backend derle
echo -e "${YELLOW}[4/6] Backend derleniyor...${NC}"
export PATH=$PATH:/usr/local/go/bin

# Ana panel
CGO_ENABLED=1 /usr/local/go/bin/go build -o serverpanel ./cmd/panel
chmod +x serverpanel
echo -e "${GREEN}  ✓ Panel derlendi${NC}"

# Policy daemon
if [[ -d "cmd/policy-daemon" ]]; then
    /usr/local/go/bin/go build -o bin/policy-daemon ./cmd/policy-daemon 2>/dev/null || true
    chmod +x bin/policy-daemon 2>/dev/null || true
    echo -e "${GREEN}  ✓ Policy daemon derlendi${NC}"
fi

# Queue processor
if [[ -d "cmd/queue-processor" ]]; then
    /usr/local/go/bin/go build -o bin/queue-processor ./cmd/queue-processor 2>/dev/null || true
    chmod +x bin/queue-processor 2>/dev/null || true
    echo -e "${GREEN}  ✓ Queue processor derlendi${NC}"
fi

echo -e "${GREEN}✓ Backend derlendi${NC}"

# Frontend kopyala
echo -e "${YELLOW}[5/6] Frontend güncelleniyor...${NC}"
mkdir -p public
if [[ -d "web/dist" ]] && [[ -f "web/dist/index.html" ]]; then
    rm -rf public/*
    cp -r web/dist/* public/
    echo -e "${GREEN}✓ Frontend güncellendi${NC}"
else
    echo -e "${RED}Frontend bulunamadı, yedekten geri yükleniyor...${NC}"
    cp -r "$BACKUP_DIR/public/"* public/ 2>/dev/null || true
fi

# Servisleri başlat
echo -e "${YELLOW}[6/6] Servisler başlatılıyor...${NC}"
systemctl start serverpanel
sleep 2

# Queue processor servisi varsa başlat
if [[ -f "/etc/systemd/system/serverpanel-queue.service" ]]; then
    systemctl start serverpanel-queue 2>/dev/null || true
fi

# Postfix'i yeniden başlat (policy daemon için)
systemctl restart postfix 2>/dev/null || true

# Durum kontrolü
if systemctl is-active --quiet serverpanel; then
    echo -e "${GREEN}✓ ServerPanel aktif${NC}"
else
    echo -e "${RED}✗ ServerPanel başlatılamadı!${NC}"
    echo -e "${YELLOW}Yedekten geri yükleniyor...${NC}"
    cp "$BACKUP_DIR/serverpanel" "$INSTALL_DIR/" 2>/dev/null || true
    systemctl start serverpanel
fi

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}              Güncelleme Tamamlandı!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "Panel URL: ${CYAN}http://$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):8443${NC}"
echo ""
