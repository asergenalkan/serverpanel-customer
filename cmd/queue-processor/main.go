package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

/*
Mail Queue Processor for ServerPanel

Bu daemon, mail_queue tablosundaki bekleyen mailleri işler.
Her dakika çalışır ve rate limit kontrolü yaparak mailleri gönderir.

Systemd Service:
/etc/systemd/system/serverpanel-queue.service
*/

const (
	dbPath        = "/root/.serverpanel/panel.db"
	logPath       = "/var/log/serverpanel/queue-processor.log"
	checkInterval = 1 * time.Minute
)

var db *sql.DB

type QueueItem struct {
	ID         int64
	UserID     int64
	Sender     string
	Recipient  string
	Subject    string
	Body       string
	Headers    string
	RetryCount int
	MaxRetries int
}

func main() {
	// Setup logging
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Log dosyası açılamadı: %v, stdout kullanılıyor", err)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Println("Queue Processor başlatılıyor...")

	// Connect to database
	db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		log.Fatalf("Veritabanı bağlantısı başarısız: %v", err)
	}
	defer db.Close()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Printf("Queue Processor çalışıyor (interval: %v)", checkInterval)

	// Process immediately on start
	processQueue()

	for {
		select {
		case <-ticker.C:
			processQueue()
		case sig := <-sigChan:
			log.Printf("Sinyal alındı: %v, kapatılıyor...", sig)
			return
		}
	}
}

func processQueue() {
	log.Println("Kuyruk kontrol ediliyor...")

	// Get pending items that are scheduled for now or earlier
	now := time.Now().Format("2006-01-02 15:04:05")

	rows, err := db.Query(`
		SELECT id, user_id, sender, recipient, COALESCE(subject, ''), 
		       COALESCE(body, ''), COALESCE(headers, ''), retry_count, max_retries
		FROM mail_queue 
		WHERE status = 'pending' 
		  AND (scheduled_at IS NULL OR scheduled_at <= ?)
		ORDER BY priority ASC, created_at ASC
		LIMIT 50
	`, now)

	if err != nil {
		log.Printf("Kuyruk sorgusu başarısız: %v", err)
		return
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		err := rows.Scan(&item.ID, &item.UserID, &item.Sender, &item.Recipient,
			&item.Subject, &item.Body, &item.Headers, &item.RetryCount, &item.MaxRetries)
		if err != nil {
			log.Printf("Satır okuma hatası: %v", err)
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		log.Println("Kuyrukta bekleyen mail yok")
		return
	}

	log.Printf("%d mail işlenecek", len(items))

	for _, item := range items {
		processItem(item)
	}
}

func processItem(item QueueItem) {
	log.Printf("Mail işleniyor: id=%d, sender=%s, recipient=%s", item.ID, item.Sender, item.Recipient)

	// Check rate limit before sending
	canSend, limitType := checkRateLimit(item.UserID)

	if !canSend {
		// Reschedule based on limit type
		var rescheduleTime time.Time
		if limitType == "hourly" {
			rescheduleTime = time.Now().Add(1 * time.Hour)
		} else {
			// Daily limit - schedule for next day
			tomorrow := time.Now().AddDate(0, 0, 1)
			rescheduleTime = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
		}

		log.Printf("Rate limit aktif (%s), yeniden zamanlandı: %v", limitType, rescheduleTime)

		db.Exec(`
			UPDATE mail_queue 
			SET scheduled_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, rescheduleTime.Format("2006-01-02 15:04:05"), item.ID)

		return
	}

	// Mark as processing
	db.Exec(`UPDATE mail_queue SET status = 'processing', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, item.ID)

	// Send the email using sendmail
	err := sendEmail(item)

	if err != nil {
		log.Printf("Mail gönderimi başarısız: %v", err)

		item.RetryCount++
		if item.RetryCount >= item.MaxRetries {
			// Max retries reached, mark as failed
			db.Exec(`
				UPDATE mail_queue 
				SET status = 'failed', error_message = ?, retry_count = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, err.Error(), item.RetryCount, item.ID)
			log.Printf("Mail başarısız olarak işaretlendi (max retry): id=%d", item.ID)
		} else {
			// Schedule retry
			retryTime := time.Now().Add(time.Duration(item.RetryCount*5) * time.Minute)
			db.Exec(`
				UPDATE mail_queue 
				SET status = 'pending', error_message = ?, retry_count = ?, scheduled_at = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, err.Error(), item.RetryCount, retryTime.Format("2006-01-02 15:04:05"), item.ID)
			log.Printf("Mail yeniden zamanlandı: id=%d, retry=%d, time=%v", item.ID, item.RetryCount, retryTime)
		}
		return
	}

	// Success - log and delete from queue
	logEmail(item.UserID, item.Sender, item.Recipient, item.Subject)

	db.Exec(`DELETE FROM mail_queue WHERE id = ?`, item.ID)
	log.Printf("Mail başarıyla gönderildi ve kuyruktan silindi: id=%d", item.ID)
}

func checkRateLimit(userID int64) (bool, string) {
	var hourlyLimit, dailyLimit int

	err := db.QueryRow(`
		SELECT COALESCE(p.max_emails_per_hour, 100), COALESCE(p.max_emails_per_day, 500)
		FROM users u
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE u.id = ?
	`, userID).Scan(&hourlyLimit, &dailyLimit)

	if err != nil {
		hourlyLimit = 100
		dailyLimit = 500
	}

	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	todayStart := now.Format("2006-01-02") + " 00:00:00"

	var sentLastHour, sentToday int
	db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, hourAgo).Scan(&sentLastHour)
	db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, todayStart).Scan(&sentToday)

	if sentLastHour >= hourlyLimit {
		return false, "hourly"
	}
	if sentToday >= dailyLimit {
		return false, "daily"
	}

	return true, ""
}

func sendEmail(item QueueItem) error {
	// Create email content
	emailContent := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n", item.Sender, item.Recipient, item.Subject)

	if item.Headers != "" {
		emailContent += item.Headers + "\n"
	}

	emailContent += "\n" + item.Body

	// Use sendmail to send
	cmd := exec.Command("/usr/sbin/sendmail", "-t", "-oi", "-f", item.Sender)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe hatası: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("sendmail başlatılamadı: %v", err)
	}

	_, err = stdin.Write([]byte(emailContent))
	if err != nil {
		return fmt.Errorf("email yazma hatası: %v", err)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("sendmail hatası: %v", err)
	}

	return nil
}

func logEmail(userID int64, sender, recipient, subject string) {
	_, err := db.Exec(`
		INSERT INTO email_send_log (user_id, sender, recipient, subject)
		VALUES (?, ?, ?, ?)
	`, userID, sender, recipient, subject)

	if err != nil {
		log.Printf("Email log kaydedilemedi: %v", err)
	}
}
