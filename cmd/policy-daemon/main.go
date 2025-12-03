package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

/*
Postfix Policy Daemon for ServerPanel

Bu daemon, Postfix'in smtpd_recipient_restrictions içinde çağrılır.
Her mail gönderiminde rate limiting kontrolü yapar.

Postfix Konfigürasyonu:
/etc/postfix/main.cf:
  smtpd_recipient_restrictions =
    permit_mynetworks,
    permit_sasl_authenticated,
    check_policy_service unix:private/policy,
    reject_unauth_destination

/etc/postfix/master.cf:
  policy unix  -       n       n       -       0       spawn
    user=nobody argv=/opt/serverpanel/bin/policy-daemon

Protokol:
- Postfix, key=value formatında veri gönderir
- Daemon, "action=DUNNO" (izin ver) veya "action=DEFER_IF_PERMIT ..." (ertele) döner
*/

const (
	dbPath     = "/var/lib/serverpanel/panel.db"
	socketPath = "/var/spool/postfix/private/policy"
	logPath    = "/var/log/serverpanel/policy-daemon.log"
)

var db *sql.DB

func main() {
	// Setup logging
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Log dosyası açılamadı: %v, stdout kullanılıyor", err)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Println("Policy Daemon başlatılıyor...")

	// Connect to database
	db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=ro")
	if err != nil {
		log.Fatalf("Veritabanı bağlantısı başarısız: %v", err)
	}
	defer db.Close()

	// Check if running from stdin (Postfix spawn)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Running from Postfix spawn - handle single request from stdin
		handleStdinRequest()
		return
	}

	// Running as standalone daemon - listen on Unix socket
	// Remove existing socket
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Socket oluşturulamadı: %v", err)
	}
	defer listener.Close()

	// Set socket permissions
	os.Chmod(socketPath, 0666)

	log.Printf("Policy Daemon dinleniyor: %s", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Bağlantı hatası: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleStdinRequest() {
	scanner := bufio.NewScanner(os.Stdin)
	attrs := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// End of request
			action := checkPolicy(attrs)
			fmt.Printf("action=%s\n\n", action)
			return
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			attrs[parts[0]] = parts[1]
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	attrs := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// End of request, process it
			action := checkPolicy(attrs)
			fmt.Fprintf(conn, "action=%s\n\n", action)
			attrs = make(map[string]string) // Reset for next request
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			attrs[parts[0]] = parts[1]
		}
	}
}

func checkPolicy(attrs map[string]string) string {
	sender := attrs["sender"]
	recipient := attrs["recipient"]
	saslUsername := attrs["sasl_username"]

	// Skip if no sender (shouldn't happen)
	if sender == "" {
		return "DUNNO"
	}

	log.Printf("Rate limit kontrolü: sender=%s, recipient=%s, sasl_user=%s", sender, recipient, saslUsername)

	// Extract domain from sender
	senderParts := strings.Split(sender, "@")
	if len(senderParts) != 2 {
		return "DUNNO"
	}
	senderDomain := senderParts[1]

	// Find user by domain
	var userID int64
	var hourlyLimit, dailyLimit int

	err := db.QueryRow(`
		SELECT u.id, COALESCE(p.max_emails_per_hour, 100), COALESCE(p.max_emails_per_day, 500)
		FROM users u
		JOIN domains d ON u.id = d.user_id
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE d.name = ?
	`, senderDomain).Scan(&userID, &hourlyLimit, &dailyLimit)

	if err != nil {
		log.Printf("Kullanıcı bulunamadı (domain: %s): %v", senderDomain, err)
		return "DUNNO" // Allow if user not found (might be system mail)
	}

	// Check hourly limit
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")

	var sentLastHour int
	db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, hourAgo).Scan(&sentLastHour)

	if sentLastHour >= hourlyLimit {
		log.Printf("Saatlik limit aşıldı: user_id=%d, sent=%d, limit=%d", userID, sentLastHour, hourlyLimit)
		// Queue the email instead of rejecting
		queueEmail(userID, sender, recipient, attrs["subject"])
		return fmt.Sprintf("DEFER_IF_PERMIT Saatlik mail limiti aşıldı (%d/%d). Mail kuyruğa alındı.", sentLastHour, hourlyLimit)
	}

	// Check daily limit
	todayStart := now.Format("2006-01-02") + " 00:00:00"

	var sentToday int
	db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, todayStart).Scan(&sentToday)

	if sentToday >= dailyLimit {
		log.Printf("Günlük limit aşıldı: user_id=%d, sent=%d, limit=%d", userID, sentToday, dailyLimit)
		// Queue the email for next day
		queueEmail(userID, sender, recipient, attrs["subject"])
		return fmt.Sprintf("DEFER_IF_PERMIT Günlük mail limiti aşıldı (%d/%d). Mail kuyruğa alındı.", sentToday, dailyLimit)
	}

	// Log the email
	logEmail(userID, sender, recipient, attrs["subject"])

	log.Printf("Mail izin verildi: user_id=%d, hourly=%d/%d, daily=%d/%d",
		userID, sentLastHour+1, hourlyLimit, sentToday+1, dailyLimit)

	return "DUNNO"
}

func logEmail(userID int64, sender, recipient, subject string) {
	// Use a write connection for logging
	writeDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		log.Printf("Write DB bağlantısı başarısız: %v", err)
		return
	}
	defer writeDB.Close()

	_, err = writeDB.Exec(`
		INSERT INTO email_send_log (user_id, sender, recipient, subject)
		VALUES (?, ?, ?, ?)
	`, userID, sender, recipient, subject)

	if err != nil {
		log.Printf("Email log kaydedilemedi: %v", err)
	}
}

func queueEmail(userID int64, sender, recipient, subject string) {
	// Use a write connection for queuing
	writeDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		log.Printf("Write DB bağlantısı başarısız: %v", err)
		return
	}
	defer writeDB.Close()

	// Schedule for next hour
	scheduledAt := time.Now().Add(1 * time.Hour).Format("2006-01-02 15:04:05")

	_, err = writeDB.Exec(`
		INSERT INTO mail_queue (user_id, sender, recipient, subject, scheduled_at, status)
		VALUES (?, ?, ?, ?, ?, 'pending')
	`, userID, sender, recipient, subject, scheduledAt)

	if err != nil {
		log.Printf("Email kuyruğa eklenemedi: %v", err)
	} else {
		log.Printf("Email kuyruğa eklendi: user_id=%d, scheduled=%s", userID, scheduledAt)
	}
}
