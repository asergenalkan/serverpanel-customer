#!/bin/bash
#
# ╔═══════════════════════════════════════════════════════════════════════════╗
# ║                    SERVERPANEL UPDATE SCRIPT                              ║
# ║              Sunucudaki ServerPanel'i GitHub'dan Günceller                ║
# ╚═══════════════════════════════════════════════════════════════════════════╝
#
# Kullanım:
#   curl -sSL https://raw.githubusercontent.com/asergenalkan/serverpanel/main/scripts/update-server.sh | bash
#
# veya sunucuda:
#   bash /opt/serverpanel/scripts/update-server.sh
#

set -e

# Renkler
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

INSTALL_DIR="/opt/serverpanel"
DATA_DIR="/root/.serverpanel"
DB_PATH="$DATA_DIR/panel.db"
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

# ═══════════════════════════════════════════════════════════════════════════════
# 1. SERVİSLERİ DURDUR
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[1/7] Servisler durduruluyor...${NC}"
systemctl stop serverpanel 2>/dev/null || true
systemctl stop serverpanel-queue 2>/dev/null || true
echo -e "${GREEN}✓ Servisler durduruldu${NC}"

# ═══════════════════════════════════════════════════════════════════════════════
# 2. YEDEK AL
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[2/7] Yedek alınıyor...${NC}"
BACKUP_DIR="/tmp/serverpanel-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp "$INSTALL_DIR/serverpanel" "$BACKUP_DIR/" 2>/dev/null || true
cp -r "$INSTALL_DIR/public" "$BACKUP_DIR/" 2>/dev/null || true
cp "$DB_PATH" "$BACKUP_DIR/panel.db" 2>/dev/null || true
echo -e "${GREEN}✓ Yedek alındı: $BACKUP_DIR${NC}"

# ═══════════════════════════════════════════════════════════════════════════════
# 3. GITHUB'DAN GÜNCELLE
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[3/7] GitHub'dan güncelleniyor...${NC}"
cd "$INSTALL_DIR"

if [[ ! -d ".git" ]]; then
    echo -e "${YELLOW}  Git repo bulunamadı, yeniden indiriliyor...${NC}"
    cd /tmp
    rm -rf serverpanel-temp
    git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" serverpanel-temp
    # Eski dosyaları sil ama public ve bin koru
    find "$INSTALL_DIR" -mindepth 1 -maxdepth 1 ! -name 'public' ! -name 'bin' -exec rm -rf {} +
    cp -r serverpanel-temp/* "$INSTALL_DIR/"
    rm -rf serverpanel-temp
    cd "$INSTALL_DIR"
else
    git fetch origin main
    git reset --hard origin/main
fi
echo -e "${GREEN}✓ Kaynak kod güncellendi${NC}"

# ═══════════════════════════════════════════════════════════════════════════════
# 4. VERİTABANI MİGRASYONU
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[4/7] Veritabanı kontrol ediliyor...${NC}"

if [[ -f "$DB_PATH" ]]; then
    # email_send_log tablosunda user_id var mı?
    if ! sqlite3 "$DB_PATH" "PRAGMA table_info(email_send_log);" | grep -q "user_id"; then
        echo -e "${YELLOW}  email_send_log tablosu güncelleniyor...${NC}"
        sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN user_id INTEGER;" 2>/dev/null || true
        sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN sender TEXT;" 2>/dev/null || true
        sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN message_id TEXT;" 2>/dev/null || true
        sqlite3 "$DB_PATH" "ALTER TABLE email_send_log ADD COLUMN size_bytes INTEGER DEFAULT 0;" 2>/dev/null || true
    fi
    
    # mail_queue tablosu var mı?
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "mail_queue"; then
        echo -e "${YELLOW}  mail_queue tablosu oluşturuluyor...${NC}"
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
    fi
    
    # packages tablosunda mail limitleri var mı?
    if ! sqlite3 "$DB_PATH" "PRAGMA table_info(packages);" | grep -q "max_emails_per_hour"; then
        echo -e "${YELLOW}  packages tablosuna mail limitleri ekleniyor...${NC}"
        sqlite3 "$DB_PATH" "ALTER TABLE packages ADD COLUMN max_emails_per_hour INTEGER DEFAULT 100;" 2>/dev/null || true
        sqlite3 "$DB_PATH" "ALTER TABLE packages ADD COLUMN max_emails_per_day INTEGER DEFAULT 500;" 2>/dev/null || true
    fi
    
    # İndexler
    sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_email_send_log_user_id ON email_send_log(user_id);" 2>/dev/null || true
    sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_email_send_log_sent_at ON email_send_log(sent_at);" 2>/dev/null || true
    sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_mail_queue_user_id ON mail_queue(user_id);" 2>/dev/null || true
    sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_mail_queue_status ON mail_queue(status);" 2>/dev/null || true
    
    # spam_settings tablosu var mı?
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "spam_settings"; then
        echo -e "${YELLOW}  spam_settings tablosu oluşturuluyor...${NC}"
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
    fi
    
    # cron_jobs tablosu var mı?
    if ! sqlite3 "$DB_PATH" ".tables" | grep -q "cron_jobs"; then
        echo -e "${YELLOW}  cron_jobs tablosu oluşturuluyor...${NC}"
        sqlite3 "$DB_PATH" "CREATE TABLE IF NOT EXISTS cron_jobs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            command TEXT NOT NULL,
            schedule TEXT NOT NULL,
            minute TEXT DEFAULT '*',
            hour TEXT DEFAULT '*',
            day TEXT DEFAULT '*',
            month TEXT DEFAULT '*',
            weekday TEXT DEFAULT '*',
            active INTEGER DEFAULT 1,
            last_run DATETIME,
            next_run DATETIME,
            last_status TEXT,
            last_output TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        );"
        sqlite3 "$DB_PATH" "CREATE INDEX IF NOT EXISTS idx_cron_jobs_user_id ON cron_jobs(user_id);" 2>/dev/null || true
    fi
    
    echo -e "${GREEN}✓ Veritabanı güncellendi${NC}"
else
    echo -e "${GREEN}✓ Yeni kurulum, migration gerekli değil${NC}"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 5. BACKEND DERLE
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[5/7] Backend derleniyor...${NC}"
export PATH=$PATH:/usr/local/go/bin
mkdir -p bin

# Ana panel
CGO_ENABLED=1 /usr/local/go/bin/go build -o serverpanel ./cmd/panel
chmod +x serverpanel
echo -e "${GREEN}  ✓ Panel derlendi${NC}"

# Policy daemon
if [[ -d "cmd/policy-daemon" ]]; then
    /usr/local/go/bin/go build -o bin/policy-daemon ./cmd/policy-daemon 2>/dev/null && \
    chmod +x bin/policy-daemon && \
    echo -e "${GREEN}  ✓ Policy daemon derlendi${NC}" || true
fi

# Queue processor
if [[ -d "cmd/queue-processor" ]]; then
    /usr/local/go/bin/go build -o bin/queue-processor ./cmd/queue-processor 2>/dev/null && \
    chmod +x bin/queue-processor && \
    echo -e "${GREEN}  ✓ Queue processor derlendi${NC}" || true
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 6. FRONTEND GÜNCELLE
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[6/7] Frontend güncelleniyor...${NC}"
mkdir -p public
if [[ -d "web/dist" ]] && [[ -f "web/dist/index.html" ]]; then
    rm -rf public/*
    cp -r web/dist/* public/
    echo -e "${GREEN}✓ Frontend güncellendi${NC}"
else
    echo -e "${RED}Frontend bulunamadı, yedekten geri yükleniyor...${NC}"
    cp -r "$BACKUP_DIR/public/"* public/ 2>/dev/null || true
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 7. SERVİSLERİ BAŞLAT
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${YELLOW}[7/7] Servisler başlatılıyor...${NC}"
systemctl daemon-reload
systemctl start serverpanel
sleep 2

# Queue processor servisi
if [[ -f "bin/queue-processor" ]]; then
    # Servis dosyası yoksa oluştur
    if [[ ! -f "/etc/systemd/system/serverpanel-queue.service" ]]; then
        cat > /etc/systemd/system/serverpanel-queue.service << 'EOF'
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
EOF
        systemctl daemon-reload
        systemctl enable serverpanel-queue
    fi
    systemctl start serverpanel-queue 2>/dev/null || true
fi

# Postfix yeniden başlat (policy daemon için)
systemctl restart postfix 2>/dev/null || true

# Durum kontrolü
echo ""
if systemctl is-active --quiet serverpanel; then
    echo -e "${GREEN}✓ ServerPanel: aktif${NC}"
else
    echo -e "${RED}✗ ServerPanel başlatılamadı!${NC}"
    echo -e "${YELLOW}Log kontrol ediliyor...${NC}"
    tail -5 /var/log/serverpanel/error.log 2>/dev/null || true
    echo -e "${YELLOW}Yedekten geri yükleniyor...${NC}"
    cp "$BACKUP_DIR/serverpanel" "$INSTALL_DIR/" 2>/dev/null || true
    cp "$BACKUP_DIR/panel.db" "$DB_PATH" 2>/dev/null || true
    systemctl start serverpanel
fi

if systemctl is-active --quiet serverpanel-queue 2>/dev/null; then
    echo -e "${GREEN}✓ Queue Processor: aktif${NC}"
fi

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}              Güncelleme Tamamlandı!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "Panel URL: ${CYAN}http://$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):8443${NC}"
echo -e "Yedek: ${CYAN}$BACKUP_DIR${NC}"
echo ""
