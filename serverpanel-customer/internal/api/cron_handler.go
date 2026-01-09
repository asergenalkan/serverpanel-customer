package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// UserCronJob represents a user's cron job in the panel
type UserCronJob struct {
	ID         int64   `json:"id"`
	UserID     int64   `json:"user_id"`
	Name       string  `json:"name"`
	Command    string  `json:"command"`
	Schedule   string  `json:"schedule"`
	Minute     string  `json:"minute"`
	Hour       string  `json:"hour"`
	Day        string  `json:"day"`
	Month      string  `json:"month"`
	Weekday    string  `json:"weekday"`
	Active     bool    `json:"active"`
	LastRun    *string `json:"last_run"`
	NextRun    *string `json:"next_run"`
	LastStatus *string `json:"last_status"`
	LastOutput *string `json:"last_output"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	// For display
	OwnerUsername string `json:"owner_username,omitempty"`
}

// Common cron schedule presets
var cronPresets = map[string]struct {
	Minute  string
	Hour    string
	Day     string
	Month   string
	Weekday string
	Label   string
}{
	"every_minute":     {"*", "*", "*", "*", "*", "Her dakika"},
	"every_5_minutes":  {"*/5", "*", "*", "*", "*", "Her 5 dakika"},
	"every_15_minutes": {"*/15", "*", "*", "*", "*", "Her 15 dakika"},
	"every_30_minutes": {"*/30", "*", "*", "*", "*", "Her 30 dakika"},
	"hourly":           {"0", "*", "*", "*", "*", "Saatlik"},
	"daily":            {"0", "0", "*", "*", "*", "Günlük (gece yarısı)"},
	"weekly":           {"0", "0", "*", "*", "0", "Haftalık (Pazar)"},
	"monthly":          {"0", "0", "1", "*", "*", "Aylık (ayın 1'i)"},
	"custom":           {"*", "*", "*", "*", "*", "Özel"},
}

// ListCronJobs returns all cron jobs (admin sees all, user sees own)
func (h *Handler) ListCronJobs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var jobs []UserCronJob
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT c.id, c.user_id, c.name, c.command, c.schedule, 
			       c.minute, c.hour, c.day, c.month, c.weekday,
			       c.active, c.last_run, c.next_run, c.last_status, c.last_output,
			       c.created_at, c.updated_at, u.username as owner_username
			FROM cron_jobs c
			JOIN users u ON c.user_id = u.id
			ORDER BY c.created_at DESC
		`
	} else {
		query = `
			SELECT c.id, c.user_id, c.name, c.command, c.schedule, 
			       c.minute, c.hour, c.day, c.month, c.weekday,
			       c.active, c.last_run, c.next_run, c.last_status, c.last_output,
			       c.created_at, c.updated_at, '' as owner_username
			FROM cron_jobs c
			WHERE c.user_id = ?
			ORDER BY c.created_at DESC
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işleri alınamadı",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var job UserCronJob
		var activeInt int
		err := rows.Scan(&job.ID, &job.UserID, &job.Name, &job.Command, &job.Schedule,
			&job.Minute, &job.Hour, &job.Day, &job.Month, &job.Weekday,
			&activeInt, &job.LastRun, &job.NextRun, &job.LastStatus, &job.LastOutput,
			&job.CreatedAt, &job.UpdatedAt, &job.OwnerUsername)
		if err != nil {
			continue
		}
		job.Active = activeInt == 1
		jobs = append(jobs, job)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    jobs,
	})
}

// GetCronJob returns a single cron job
func (h *Handler) GetCronJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz iş ID",
		})
	}

	var job UserCronJob
	var activeInt int
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT c.id, c.user_id, c.name, c.command, c.schedule, 
			       c.minute, c.hour, c.day, c.month, c.weekday,
			       c.active, c.last_run, c.next_run, c.last_status, c.last_output,
			       c.created_at, c.updated_at, u.username as owner_username
			FROM cron_jobs c
			JOIN users u ON c.user_id = u.id
			WHERE c.id = ?
		`
		args = []interface{}{jobID}
	} else {
		query = `
			SELECT c.id, c.user_id, c.name, c.command, c.schedule, 
			       c.minute, c.hour, c.day, c.month, c.weekday,
			       c.active, c.last_run, c.next_run, c.last_status, c.last_output,
			       c.created_at, c.updated_at, '' as owner_username
			FROM cron_jobs c
			WHERE c.id = ? AND c.user_id = ?
		`
		args = []interface{}{jobID, userID}
	}

	err = h.db.QueryRow(query, args...).Scan(
		&job.ID, &job.UserID, &job.Name, &job.Command, &job.Schedule,
		&job.Minute, &job.Hour, &job.Day, &job.Month, &job.Weekday,
		&activeInt, &job.LastRun, &job.NextRun, &job.LastStatus, &job.LastOutput,
		&job.CreatedAt, &job.UpdatedAt, &job.OwnerUsername)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi bulunamadı",
		})
	}
	job.Active = activeInt == 1

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    job,
	})
}

// CreateCronJob creates a new cron job
func (h *Handler) CreateCronJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var req struct {
		Name     string `json:"name"`
		Command  string `json:"command"`
		Schedule string `json:"schedule"` // preset name or "custom"
		Minute   string `json:"minute"`
		Hour     string `json:"hour"`
		Day      string `json:"day"`
		Month    string `json:"month"`
		Weekday  string `json:"weekday"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate name
	if len(req.Name) < 1 || len(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "İş adı 1-100 karakter arasında olmalıdır",
		})
	}

	// Validate command
	if len(req.Command) < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Komut boş olamaz",
		})
	}

	// Validate command for security (basic checks)
	if err := validateCronCommand(req.Command); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Get system username
	var systemUsername string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&systemUsername)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Kullanıcı bilgisi alınamadı",
		})
	}

	// Apply preset if not custom
	minute, hour, day, month, weekday := req.Minute, req.Hour, req.Day, req.Month, req.Weekday
	if preset, ok := cronPresets[req.Schedule]; ok && req.Schedule != "custom" {
		minute = preset.Minute
		hour = preset.Hour
		day = preset.Day
		month = preset.Month
		weekday = preset.Weekday
	}

	// Validate cron expression
	if !isValidCronField(minute, 0, 59) || !isValidCronField(hour, 0, 23) ||
		!isValidCronField(day, 1, 31) || !isValidCronField(month, 1, 12) ||
		!isValidCronField(weekday, 0, 7) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz cron ifadesi",
		})
	}

	// Build schedule string
	schedule := fmt.Sprintf("%s %s %s %s %s", minute, hour, day, month, weekday)

	// Insert into database
	result, err := h.db.Exec(`
		INSERT INTO cron_jobs (user_id, name, command, schedule, minute, hour, day, month, weekday, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
	`, userID, req.Name, req.Command, schedule, minute, hour, day, month, weekday)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi oluşturulamadı",
		})
	}

	jobID, _ := result.LastInsertId()

	// Sync to system crontab
	if err := h.SyncUserCrontabFromDB(userID, systemUsername); err != nil {
		// Log error but don't fail - job is in DB
		fmt.Printf("Warning: Failed to sync crontab for user %s: %v\n", systemUsername, err)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Cron işi başarıyla oluşturuldu",
		Data:    map[string]int64{"id": jobID},
	})
}

// UpdateCronJob updates an existing cron job
func (h *Handler) UpdateCronJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz iş ID",
		})
	}

	var req struct {
		Name     string `json:"name"`
		Command  string `json:"command"`
		Schedule string `json:"schedule"`
		Minute   string `json:"minute"`
		Hour     string `json:"hour"`
		Day      string `json:"day"`
		Month    string `json:"month"`
		Weekday  string `json:"weekday"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Check ownership
	var ownerID int64
	var systemUsername string
	err = h.db.QueryRow(`
		SELECT c.user_id, u.username 
		FROM cron_jobs c 
		JOIN users u ON c.user_id = u.id 
		WHERE c.id = ?
	`, jobID).Scan(&ownerID, &systemUsername)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi bulunamadı",
		})
	}

	if role != models.RoleAdmin && ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu işlemi yapmaya yetkiniz yok",
		})
	}

	// Validate command
	if err := validateCronCommand(req.Command); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Apply preset if not custom
	minute, hour, day, month, weekday := req.Minute, req.Hour, req.Day, req.Month, req.Weekday
	if preset, ok := cronPresets[req.Schedule]; ok && req.Schedule != "custom" {
		minute = preset.Minute
		hour = preset.Hour
		day = preset.Day
		month = preset.Month
		weekday = preset.Weekday
	}

	// Validate cron expression
	if !isValidCronField(minute, 0, 59) || !isValidCronField(hour, 0, 23) ||
		!isValidCronField(day, 1, 31) || !isValidCronField(month, 1, 12) ||
		!isValidCronField(weekday, 0, 7) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz cron ifadesi",
		})
	}

	schedule := fmt.Sprintf("%s %s %s %s %s", minute, hour, day, month, weekday)

	// Update database
	_, err = h.db.Exec(`
		UPDATE cron_jobs 
		SET name = ?, command = ?, schedule = ?, minute = ?, hour = ?, day = ?, month = ?, weekday = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, req.Name, req.Command, schedule, minute, hour, day, month, weekday, jobID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi güncellenemedi",
		})
	}

	// Sync to system crontab
	if err := h.SyncUserCrontabFromDB(ownerID, systemUsername); err != nil {
		fmt.Printf("Warning: Failed to sync crontab for user %s: %v\n", systemUsername, err)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Cron işi başarıyla güncellendi",
	})
}

// DeleteCronJob deletes a cron job
func (h *Handler) DeleteCronJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz iş ID",
		})
	}

	// Check ownership and get username
	var ownerID int64
	var systemUsername string
	err = h.db.QueryRow(`
		SELECT c.user_id, u.username 
		FROM cron_jobs c 
		JOIN users u ON c.user_id = u.id 
		WHERE c.id = ?
	`, jobID).Scan(&ownerID, &systemUsername)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi bulunamadı",
		})
	}

	if role != models.RoleAdmin && ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu işlemi yapmaya yetkiniz yok",
		})
	}

	// Delete from database
	_, err = h.db.Exec("DELETE FROM cron_jobs WHERE id = ?", jobID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi silinemedi",
		})
	}

	// Sync to system crontab
	if err := h.SyncUserCrontabFromDB(ownerID, systemUsername); err != nil {
		fmt.Printf("Warning: Failed to sync crontab for user %s: %v\n", systemUsername, err)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Cron işi başarıyla silindi",
	})
}

// ToggleCronJob enables/disables a cron job
func (h *Handler) ToggleCronJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz iş ID",
		})
	}

	// Check ownership
	var ownerID int64
	var systemUsername string
	var currentActive int
	err = h.db.QueryRow(`
		SELECT c.user_id, u.username, c.active
		FROM cron_jobs c 
		JOIN users u ON c.user_id = u.id 
		WHERE c.id = ?
	`, jobID).Scan(&ownerID, &systemUsername, &currentActive)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi bulunamadı",
		})
	}

	if role != models.RoleAdmin && ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu işlemi yapmaya yetkiniz yok",
		})
	}

	// Toggle active status
	newActive := 0
	if currentActive == 0 {
		newActive = 1
	}

	_, err = h.db.Exec("UPDATE cron_jobs SET active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newActive, jobID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi durumu değiştirilemedi",
		})
	}

	// Sync to system crontab
	if err := h.SyncUserCrontabFromDB(ownerID, systemUsername); err != nil {
		fmt.Printf("Warning: Failed to sync crontab for user %s: %v\n", systemUsername, err)
	}

	status := "devre dışı bırakıldı"
	if newActive == 1 {
		status = "etkinleştirildi"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Cron işi %s", status),
	})
}

// RunCronJob runs a cron job manually
func (h *Handler) RunCronJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz iş ID",
		})
	}

	// Get job details
	var ownerID int64
	var systemUsername, command string
	err = h.db.QueryRow(`
		SELECT c.user_id, u.username, c.command
		FROM cron_jobs c 
		JOIN users u ON c.user_id = u.id 
		WHERE c.id = ?
	`, jobID).Scan(&ownerID, &systemUsername, &command)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Cron işi bulunamadı",
		})
	}

	if role != models.RoleAdmin && ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu işlemi yapmaya yetkiniz yok",
		})
	}

	// Run the command as the user
	userHome := filepath.Join("/home", systemUsername)
	cmd := exec.Command("sudo", "-u", systemUsername, "bash", "-c", command)
	cmd.Dir = userHome
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", userHome),
		fmt.Sprintf("USER=%s", systemUsername),
	)

	output, err := cmd.CombinedOutput()
	status := "success"
	if err != nil {
		status = "failed"
	}

	// Update last run info
	now := time.Now().Format("2006-01-02 15:04:05")
	outputStr := string(output)
	if len(outputStr) > 10000 {
		outputStr = outputStr[:10000] + "... (truncated)"
	}

	h.db.Exec(`
		UPDATE cron_jobs 
		SET last_run = ?, last_status = ?, last_output = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, now, status, outputStr, jobID)

	return c.JSON(models.APIResponse{
		Success: status == "success",
		Message: fmt.Sprintf("Cron işi çalıştırıldı (%s)", status),
		Data: map[string]interface{}{
			"output": outputStr,
			"status": status,
		},
	})
}

// GetCronPresets returns available cron schedule presets
func (h *Handler) GetCronPresets(c *fiber.Ctx) error {
	presets := []map[string]interface{}{}
	for key, preset := range cronPresets {
		presets = append(presets, map[string]interface{}{
			"key":     key,
			"label":   preset.Label,
			"minute":  preset.Minute,
			"hour":    preset.Hour,
			"day":     preset.Day,
			"month":   preset.Month,
			"weekday": preset.Weekday,
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    presets,
	})
}

// Helper functions

// validateCronCommand checks if a command is safe to run
func validateCronCommand(command string) error {
	// Block dangerous commands
	dangerous := []string{
		"rm -rf /",
		"mkfs",
		"dd if=",
		":(){:|:&};:",
		"> /dev/sd",
		"chmod -R 777 /",
		"chown -R",
	}

	lowerCmd := strings.ToLower(command)
	for _, d := range dangerous {
		if strings.Contains(lowerCmd, strings.ToLower(d)) {
			return fmt.Errorf("tehlikeli komut tespit edildi")
		}
	}

	return nil
}

// isValidCronField validates a cron field
func isValidCronField(field string, min, max int) bool {
	if field == "*" {
		return true
	}

	// Handle */n format
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil || n < 1 {
			return false
		}
		return true
	}

	// Handle comma-separated values
	parts := strings.Split(field, ",")
	for _, part := range parts {
		// Handle range (e.g., 1-5)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return false
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start < min || end > max || start > end {
				return false
			}
		} else {
			// Single value
			val, err := strconv.Atoi(part)
			if err != nil || val < min || val > max {
				return false
			}
		}
	}

	return true
}

// SyncUserCrontabFromDB syncs crontab from database - called by handlers
func (h *Handler) SyncUserCrontabFromDB(userID int64, username string) error {
	cronDir := "/var/spool/cron/crontabs"
	cronFile := filepath.Join(cronDir, username)

	// Ensure directory exists
	os.MkdirAll(cronDir, 0755)

	// Get active cron jobs for user
	rows, err := h.db.Query(`
		SELECT schedule, command FROM cron_jobs 
		WHERE user_id = ? AND active = 1
		ORDER BY id
	`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Build crontab content
	var content strings.Builder
	content.WriteString("# ServerPanel managed crontab - DO NOT EDIT MANUALLY\n")
	content.WriteString("# Changes will be overwritten by the panel\n")
	content.WriteString(fmt.Sprintf("# User: %s\n\n", username))

	// Add environment variables
	content.WriteString(fmt.Sprintf("HOME=/home/%s\n", username))
	content.WriteString("SHELL=/bin/bash\n")
	content.WriteString("PATH=/usr/local/bin:/usr/bin:/bin\n\n")

	for rows.Next() {
		var schedule, command string
		if err := rows.Scan(&schedule, &command); err != nil {
			continue
		}
		content.WriteString(fmt.Sprintf("%s %s\n", schedule, command))
	}

	// Write to temp file first
	tempFile := cronFile + ".tmp"
	if err := os.WriteFile(tempFile, []byte(content.String()), 0600); err != nil {
		return err
	}

	// Set proper ownership (crontab files need specific permissions)
	exec.Command("chown", fmt.Sprintf("%s:crontab", username), tempFile).Run()
	exec.Command("chmod", "600", tempFile).Run()

	// Move to final location
	if err := os.Rename(tempFile, cronFile); err != nil {
		os.Remove(tempFile)
		return err
	}

	// Reload cron daemon to pick up changes
	exec.Command("systemctl", "reload", "cron").Run()

	return nil
}
