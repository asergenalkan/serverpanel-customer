package api

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// MailQueueItem represents an item in the mail queue
type MailQueueItemDB struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Sender       string `json:"sender"`
	Recipient    string `json:"recipient"`
	Subject      string `json:"subject"`
	Priority     int    `json:"priority"`
	RetryCount   int    `json:"retry_count"`
	MaxRetries   int    `json:"max_retries"`
	ScheduledAt  string `json:"scheduled_at"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	CreatedAt    string `json:"created_at"`
}

// MailStats represents email statistics for a user
type MailStats struct {
	UserID          int64  `json:"user_id"`
	Username        string `json:"username"`
	HourlyLimit     int    `json:"hourly_limit"`
	DailyLimit      int    `json:"daily_limit"`
	SentLastHour    int    `json:"sent_last_hour"`
	SentToday       int    `json:"sent_today"`
	QueuedCount     int    `json:"queued_count"`
	HourlyRemaining int    `json:"hourly_remaining"`
	DailyRemaining  int    `json:"daily_remaining"`
}

// MailQueueStats represents overall mail queue statistics
type MailQueueStats struct {
	TotalQueued     int         `json:"total_queued"`
	TotalPending    int         `json:"total_pending"`
	TotalProcessing int         `json:"total_processing"`
	TotalFailed     int         `json:"total_failed"`
	TotalSentToday  int         `json:"total_sent_today"`
	PostfixQueue    int         `json:"postfix_queue"`
	UserStats       []MailStats `json:"user_stats"`
}

// GetMailQueueStats returns mail queue statistics
func (h *Handler) GetMailQueueStats(c *fiber.Ctx) error {
	stats := MailQueueStats{}

	// Get queue counts by status
	h.db.QueryRow(`SELECT COUNT(*) FROM mail_queue WHERE status = 'pending'`).Scan(&stats.TotalPending)
	h.db.QueryRow(`SELECT COUNT(*) FROM mail_queue WHERE status = 'processing'`).Scan(&stats.TotalProcessing)
	h.db.QueryRow(`SELECT COUNT(*) FROM mail_queue WHERE status = 'failed'`).Scan(&stats.TotalFailed)
	stats.TotalQueued = stats.TotalPending + stats.TotalProcessing

	// Get total sent today
	today := time.Now().Format("2006-01-02")
	h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE DATE(sent_at) = ?`, today).Scan(&stats.TotalSentToday)

	// Get Postfix queue count
	if output, err := exec.Command("mailq").Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "requests") {
				// Parse "-- X Kbytes in Y requests."
				parts := strings.Fields(line)
				for i, p := range parts {
					if p == "requests." && i > 0 {
						stats.PostfixQueue, _ = strconv.Atoi(parts[i-1])
						break
					}
				}
			}
		}
	}

	// Get per-user statistics
	rows, err := h.db.Query(`
		SELECT u.id, u.username, 
		       COALESCE(p.max_emails_per_hour, 100) as hourly_limit,
		       COALESCE(p.max_emails_per_day, 500) as daily_limit
		FROM users u
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE u.role = 'user'
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "İstatistikler alınamadı",
		})
	}
	defer rows.Close()

	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	todayStart := now.Format("2006-01-02") + " 00:00:00"

	for rows.Next() {
		var us MailStats
		rows.Scan(&us.UserID, &us.Username, &us.HourlyLimit, &us.DailyLimit)

		// Get sent counts
		h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
			us.UserID, hourAgo).Scan(&us.SentLastHour)
		h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
			us.UserID, todayStart).Scan(&us.SentToday)
		h.db.QueryRow(`SELECT COUNT(*) FROM mail_queue WHERE user_id = ? AND status IN ('pending', 'processing')`,
			us.UserID).Scan(&us.QueuedCount)

		us.HourlyRemaining = us.HourlyLimit - us.SentLastHour
		if us.HourlyRemaining < 0 {
			us.HourlyRemaining = 0
		}
		us.DailyRemaining = us.DailyLimit - us.SentToday
		if us.DailyRemaining < 0 {
			us.DailyRemaining = 0
		}

		stats.UserStats = append(stats.UserStats, us)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

// GetMailQueue returns the mail queue items
func (h *Handler) GetMailQueue(c *fiber.Ctx) error {
	status := c.Query("status", "")
	userID := c.Query("user_id", "")
	limit := c.QueryInt("limit", 100)

	query := `
		SELECT mq.id, mq.user_id, u.username, mq.sender, mq.recipient, 
		       COALESCE(mq.subject, ''), mq.priority, mq.retry_count, mq.max_retries,
		       COALESCE(mq.scheduled_at, ''), mq.status, COALESCE(mq.error_message, ''),
		       mq.created_at
		FROM mail_queue mq
		JOIN users u ON mq.user_id = u.id
		WHERE 1=1
	`
	args := []interface{}{}

	if status != "" {
		query += " AND mq.status = ?"
		args = append(args, status)
	}
	if userID != "" {
		query += " AND mq.user_id = ?"
		args = append(args, userID)
	}

	query += " ORDER BY mq.created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Kuyruk alınamadı",
		})
	}
	defer rows.Close()

	var items []MailQueueItemDB
	for rows.Next() {
		var item MailQueueItemDB
		rows.Scan(&item.ID, &item.UserID, &item.Username, &item.Sender, &item.Recipient,
			&item.Subject, &item.Priority, &item.RetryCount, &item.MaxRetries,
			&item.ScheduledAt, &item.Status, &item.ErrorMessage, &item.CreatedAt)
		items = append(items, item)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    items,
	})
}

// DeleteMailQueueItem deletes a mail queue item
func (h *Handler) DeleteMailQueueItem(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	_, err = h.db.Exec("DELETE FROM mail_queue WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Silme başarısız",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Mail kuyruktan silindi",
	})
}

// RetryMailQueueItem retries a failed mail queue item
func (h *Handler) RetryMailQueueItem(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	_, err = h.db.Exec(`
		UPDATE mail_queue 
		SET status = 'pending', error_message = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'failed'
	`, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Yeniden deneme başarısız",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Mail yeniden kuyruğa alındı",
	})
}

// ClearMailQueue clears the mail queue
func (h *Handler) ClearMailQueue(c *fiber.Ctx) error {
	var req struct {
		Status string `json:"status"`
		UserID int64  `json:"user_id"`
	}
	c.BodyParser(&req)

	query := "DELETE FROM mail_queue WHERE 1=1"
	args := []interface{}{}

	if req.Status != "" {
		query += " AND status = ?"
		args = append(args, req.Status)
	}
	if req.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, req.UserID)
	}

	result, err := h.db.Exec(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Kuyruk temizlenemedi",
		})
	}

	affected, _ := result.RowsAffected()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%d mail kuyruktan silindi", affected),
	})
}

// GetUserMailStats returns mail statistics for a specific user
func (h *Handler) GetUserMailStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var stats MailStats
	stats.UserID = userID

	// Get user's package limits
	err := h.db.QueryRow(`
		SELECT COALESCE(p.max_emails_per_hour, 100), COALESCE(p.max_emails_per_day, 500)
		FROM users u
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE u.id = ?
	`, userID).Scan(&stats.HourlyLimit, &stats.DailyLimit)

	if err != nil {
		stats.HourlyLimit = 100
		stats.DailyLimit = 500
	}

	// Get sent counts
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	todayStart := now.Format("2006-01-02") + " 00:00:00"

	h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, hourAgo).Scan(&stats.SentLastHour)
	h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, todayStart).Scan(&stats.SentToday)
	h.db.QueryRow(`SELECT COUNT(*) FROM mail_queue WHERE user_id = ? AND status IN ('pending', 'processing')`,
		userID).Scan(&stats.QueuedCount)

	stats.HourlyRemaining = stats.HourlyLimit - stats.SentLastHour
	if stats.HourlyRemaining < 0 {
		stats.HourlyRemaining = 0
	}
	stats.DailyRemaining = stats.DailyLimit - stats.SentToday
	if stats.DailyRemaining < 0 {
		stats.DailyRemaining = 0
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

// CheckRateLimit checks if a user can send an email
func (h *Handler) CheckRateLimit(userID int64) (bool, string, error) {
	var hourlyLimit, dailyLimit int

	// Get user's package limits
	err := h.db.QueryRow(`
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

	// Check hourly limit
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	var sentLastHour int
	h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, hourAgo).Scan(&sentLastHour)

	if sentLastHour >= hourlyLimit {
		return false, "hourly", nil
	}

	// Check daily limit
	todayStart := now.Format("2006-01-02") + " 00:00:00"
	var sentToday int
	h.db.QueryRow(`SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ?`,
		userID, todayStart).Scan(&sentToday)

	if sentToday >= dailyLimit {
		return false, "daily", nil
	}

	return true, "", nil
}

// LogEmailSent logs a sent email
func (h *Handler) LogEmailSent(userID int64, sender, recipient, subject, messageID string, sizeBytes int) error {
	_, err := h.db.Exec(`
		INSERT INTO email_send_log (user_id, sender, recipient, subject, message_id, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, sender, recipient, subject, messageID, sizeBytes)
	return err
}

// AddToMailQueue adds an email to the queue
func (h *Handler) AddToMailQueue(userID int64, sender, recipient, subject, body, headers string, scheduledAt *time.Time) (int64, error) {
	var scheduled interface{}
	if scheduledAt != nil {
		scheduled = scheduledAt.Format("2006-01-02 15:04:05")
	}

	result, err := h.db.Exec(`
		INSERT INTO mail_queue (user_id, sender, recipient, subject, body, headers, scheduled_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, sender, recipient, subject, body, headers, scheduled)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}
