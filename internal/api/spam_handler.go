package api

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SpamSettings represents spam filter settings for a user
type SpamSettings struct {
	ID              int64    `json:"id"`
	UserID          int64    `json:"user_id"`
	Enabled         bool     `json:"enabled"`
	SpamScore       float64  `json:"spam_score"`
	AutoDelete      bool     `json:"auto_delete"`
	AutoDeleteScore float64  `json:"auto_delete_score"`
	SpamFolder      bool     `json:"spam_folder"`
	Whitelist       []string `json:"whitelist"`
	Blacklist       []string `json:"blacklist"`
}

// AntivirusStatus represents ClamAV status
type AntivirusStatus struct {
	ClamAVInstalled bool   `json:"clamav_installed"`
	ClamAVRunning   bool   `json:"clamav_running"`
	LastUpdate      string `json:"last_update"`
	VirusDBVersion  string `json:"virus_db_version"`
}

// SpamStats represents spam statistics
type SpamStats struct {
	TotalScanned    int `json:"total_scanned"`
	SpamDetected    int `json:"spam_detected"`
	VirusesDetected int `json:"viruses_detected"`
	Last24hSpam     int `json:"last_24h_spam"`
}

// GetSpamSettings returns spam settings for the current user
func (h *Handler) GetSpamSettings(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	// Get or create spam settings
	settings, err := h.getOrCreateSpamSettings(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ayarlar yüklenemedi"})
	}

	// Get antivirus status
	antivirus := h.getAntivirusStatus()

	// Get stats
	stats := h.getSpamStats(userID)

	return c.JSON(fiber.Map{
		"settings":  settings,
		"antivirus": antivirus,
		"stats":     stats,
	})
}

// UpdateSpamSettings updates spam settings for the current user
func (h *Handler) UpdateSpamSettings(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var input SpamSettings
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	// Convert whitelist/blacklist to JSON
	whitelistJSON, _ := json.Marshal(input.Whitelist)
	blacklistJSON, _ := json.Marshal(input.Blacklist)

	// Check if settings exist
	var existingID int64
	err := h.db.QueryRow("SELECT id FROM spam_settings WHERE user_id = ?", userID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new settings
		_, err = h.db.Exec(`
			INSERT INTO spam_settings (user_id, enabled, spam_score, auto_delete, auto_delete_score, spam_folder, whitelist, blacklist)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, input.Enabled, input.SpamScore, input.AutoDelete, input.AutoDeleteScore, input.SpamFolder, string(whitelistJSON), string(blacklistJSON))
	} else if err == nil {
		// Update existing settings
		_, err = h.db.Exec(`
			UPDATE spam_settings 
			SET enabled = ?, spam_score = ?, auto_delete = ?, auto_delete_score = ?, spam_folder = ?, whitelist = ?, blacklist = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, input.Enabled, input.SpamScore, input.AutoDelete, input.AutoDeleteScore, input.SpamFolder, string(whitelistJSON), string(blacklistJSON), userID)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ayarlar kaydedilemedi"})
	}

	// Apply SpamAssassin settings
	if err := h.applySpamAssassinSettings(userID, input); err != nil {
		// Log error but don't fail
		println("SpamAssassin ayarları uygulanamadı:", err.Error())
	}

	return c.JSON(fiber.Map{"message": "Ayarlar kaydedildi"})
}

// UpdateClamAV triggers ClamAV database update
func (h *Handler) UpdateClamAV(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	// Run freshclam in background
	go func() {
		exec.Command("systemctl", "stop", "clamav-freshclam").Run()
		exec.Command("freshclam").Run()
		exec.Command("systemctl", "start", "clamav-freshclam").Run()
	}()

	return c.JSON(fiber.Map{"message": "Güncelleme başlatıldı"})
}

// GetGlobalSpamSettings returns global spam settings (admin only)
func (h *Handler) GetGlobalSpamSettings(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	// Check SpamAssassin status
	spamassassinRunning := false
	if out, err := exec.Command("systemctl", "is-active", "spamassassin").Output(); err == nil {
		spamassassinRunning = strings.TrimSpace(string(out)) == "active"
	}

	// Check ClamAV status
	clamavRunning := false
	if out, err := exec.Command("systemctl", "is-active", "clamav-daemon").Output(); err == nil {
		clamavRunning = strings.TrimSpace(string(out)) == "active"
	}

	return c.JSON(fiber.Map{
		"spamassassin_running": spamassassinRunning,
		"clamav_running":       clamavRunning,
		"antivirus":            h.getAntivirusStatus(),
	})
}

// ToggleSpamService enables/disables spam services (admin only)
func (h *Handler) ToggleSpamService(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	var input struct {
		Service string `json:"service"` // spamassassin or clamav
		Enabled bool   `json:"enabled"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	var serviceName string
	switch input.Service {
	case "spamassassin":
		serviceName = "spamassassin"
	case "clamav":
		serviceName = "clamav-daemon"
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz servis"})
	}

	action := "stop"
	if input.Enabled {
		action = "start"
	}

	if err := exec.Command("systemctl", action, serviceName).Run(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Servis değiştirilemedi"})
	}

	return c.JSON(fiber.Map{"message": "Servis durumu değiştirildi"})
}

// Helper functions

func (h *Handler) getOrCreateSpamSettings(userID int64) (*SpamSettings, error) {
	settings := &SpamSettings{
		UserID:          userID,
		Enabled:         true,
		SpamScore:       5.0,
		AutoDelete:      false,
		AutoDeleteScore: 10.0,
		SpamFolder:      true,
		Whitelist:       []string{},
		Blacklist:       []string{},
	}

	var whitelistJSON, blacklistJSON string
	err := h.db.QueryRow(`
		SELECT id, enabled, spam_score, auto_delete, auto_delete_score, spam_folder, whitelist, blacklist
		FROM spam_settings WHERE user_id = ?
	`, userID).Scan(
		&settings.ID, &settings.Enabled, &settings.SpamScore,
		&settings.AutoDelete, &settings.AutoDeleteScore, &settings.SpamFolder,
		&whitelistJSON, &blacklistJSON,
	)

	if err == sql.ErrNoRows {
		// Create default settings
		result, err := h.db.Exec(`
			INSERT INTO spam_settings (user_id, enabled, spam_score, auto_delete, auto_delete_score, spam_folder, whitelist, blacklist)
			VALUES (?, ?, ?, ?, ?, ?, '[]', '[]')
		`, userID, settings.Enabled, settings.SpamScore, settings.AutoDelete, settings.AutoDeleteScore, settings.SpamFolder)
		if err != nil {
			return nil, err
		}
		settings.ID, _ = result.LastInsertId()
		return settings, nil
	} else if err != nil {
		return nil, err
	}

	// Parse JSON arrays
	json.Unmarshal([]byte(whitelistJSON), &settings.Whitelist)
	json.Unmarshal([]byte(blacklistJSON), &settings.Blacklist)

	if settings.Whitelist == nil {
		settings.Whitelist = []string{}
	}
	if settings.Blacklist == nil {
		settings.Blacklist = []string{}
	}

	return settings, nil
}

func (h *Handler) getAntivirusStatus() *AntivirusStatus {
	status := &AntivirusStatus{}

	// Check if ClamAV is installed
	if _, err := exec.LookPath("clamscan"); err == nil {
		status.ClamAVInstalled = true
	}

	// Check if ClamAV daemon is running
	if out, err := exec.Command("systemctl", "is-active", "clamav-daemon").Output(); err == nil {
		status.ClamAVRunning = strings.TrimSpace(string(out)) == "active"
	}

	// Get virus database info
	if status.ClamAVInstalled {
		// Check last update time
		if info, err := os.Stat("/var/lib/clamav/daily.cvd"); err == nil {
			status.LastUpdate = info.ModTime().Format("2006-01-02 15:04")
		} else if info, err := os.Stat("/var/lib/clamav/daily.cld"); err == nil {
			status.LastUpdate = info.ModTime().Format("2006-01-02 15:04")
		}

		// Get version
		if out, err := exec.Command("sigtool", "--info", "/var/lib/clamav/daily.cvd").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "Version:") {
					status.VirusDBVersion = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
					break
				}
			}
		}
	}

	return status
}

func (h *Handler) getSpamStats(userID int64) *SpamStats {
	stats := &SpamStats{}

	// Get total scanned (from email_send_log or a dedicated table)
	// For now, return placeholder values
	// In production, you would query actual spam logs

	h.db.QueryRow(`
		SELECT COUNT(*) FROM email_send_log WHERE user_id = ?
	`, userID).Scan(&stats.TotalScanned)

	// Get spam detected in last 24 hours
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	h.db.QueryRow(`
		SELECT COUNT(*) FROM email_send_log WHERE user_id = ? AND sent_at >= ? AND status = 'spam'
	`, userID, yesterday).Scan(&stats.Last24hSpam)

	return stats
}

func (h *Handler) applySpamAssassinSettings(userID int64, settings SpamSettings) error {
	// Get user's home directory or use a per-user SpamAssassin config
	// This is a simplified implementation

	// For per-user settings, SpamAssassin uses ~/.spamassassin/user_prefs
	// In a virtual mail setup, we might use a different approach

	// For now, we'll store settings in the database and apply them via spamd

	return nil
}

// ==================== MALWARE SCANNING ====================

// ScanResult represents a malware scan result
type ScanResult struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // clean, infected, error
	Threat    string `json:"threat,omitempty"`
	ScannedAt string `json:"scanned_at"`
}

// ScanSummary represents scan summary
type ScanSummary struct {
	TotalFiles    int          `json:"total_files"`
	ScannedFiles  int          `json:"scanned_files"`
	InfectedFiles int          `json:"infected_files"`
	Errors        int          `json:"errors"`
	Duration      string       `json:"duration"`
	Results       []ScanResult `json:"results"`
}

// ScanPath scans a specific path for malware
func (h *Handler) ScanPath(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var input struct {
		Path string `json:"path"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	// Determine scan path
	var scanPath string
	if role == "admin" && input.Path != "" {
		// Admin can scan any path
		scanPath = input.Path
	} else {
		// Regular users can only scan their home directory
		homeDir := "/home/" + username
		if input.Path != "" {
			// Ensure path is within home directory
			if !strings.HasPrefix(input.Path, homeDir) {
				return c.Status(403).JSON(fiber.Map{"error": "Bu dizini tarama yetkiniz yok"})
			}
			scanPath = input.Path
		} else {
			scanPath = homeDir + "/public_html"
		}
	}

	// Check if ClamAV is installed
	if _, err := exec.LookPath("clamscan"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ClamAV kurulu değil"})
	}

	// Run clamscan
	startTime := time.Now()
	cmd := exec.Command("clamscan", "-r", "--infected", "--no-summary", scanPath)
	output, _ := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Parse results
	lines := strings.Split(string(output), "\n")
	results := []ScanResult{}
	infectedCount := 0
	scannedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				path := parts[0]
				status := parts[1]

				result := ScanResult{
					Path:      path,
					ScannedAt: time.Now().Format("2006-01-02 15:04:05"),
				}

				if strings.Contains(status, "FOUND") {
					result.Status = "infected"
					result.Threat = strings.TrimSuffix(status, " FOUND")
					infectedCount++
				} else if strings.Contains(status, "OK") {
					result.Status = "clean"
					scannedCount++
					continue // Don't include clean files in results to reduce noise
				} else if strings.Contains(status, "ERROR") {
					result.Status = "error"
					result.Threat = status
				}

				if result.Status != "clean" {
					results = append(results, result)
				}
				scannedCount++
			}
		}
	}

	// Get total file count
	countCmd := exec.Command("find", scanPath, "-type", "f")
	countOutput, _ := countCmd.Output()
	totalFiles := len(strings.Split(strings.TrimSpace(string(countOutput)), "\n"))
	if totalFiles == 1 && string(countOutput) == "" {
		totalFiles = 0
	}

	summary := ScanSummary{
		TotalFiles:    totalFiles,
		ScannedFiles:  scannedCount,
		InfectedFiles: infectedCount,
		Errors:        0,
		Duration:      duration.Round(time.Second).String(),
		Results:       results,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    summary,
	})
}

// QuickScan performs a quick scan of common malware locations
func (h *Handler) QuickScan(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	homeDir := "/home/" + username
	scanPaths := []string{
		homeDir + "/public_html",
	}

	// Check if ClamAV is installed
	if _, err := exec.LookPath("clamscan"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ClamAV kurulu değil"})
	}

	// Run quick scan
	startTime := time.Now()
	args := append([]string{"-r", "--infected", "--no-summary"}, scanPaths...)
	cmd := exec.Command("clamscan", args...)
	output, _ := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Parse results
	lines := strings.Split(string(output), "\n")
	results := []ScanResult{}
	infectedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "FOUND") {
			continue
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			result := ScanResult{
				Path:      parts[0],
				Status:    "infected",
				Threat:    strings.TrimSuffix(parts[1], " FOUND"),
				ScannedAt: time.Now().Format("2006-01-02 15:04:05"),
			}
			results = append(results, result)
			infectedCount++
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"infected_count": infectedCount,
			"duration":       duration.Round(time.Second).String(),
			"results":        results,
		},
	})
}

// QuarantineFile moves an infected file to quarantine
func (h *Handler) QuarantineFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var input struct {
		Path string `json:"path"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	if input.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Dosya yolu gerekli"})
	}

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	homeDir := "/home/" + username

	// Security check - user can only quarantine files in their home directory
	if role != "admin" && !strings.HasPrefix(input.Path, homeDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Bu dosyayı karantinaya alma yetkiniz yok"})
	}

	// Check if file exists
	if _, err := os.Stat(input.Path); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "Dosya bulunamadı"})
	}

	// Create quarantine directory
	quarantineDir := homeDir + "/.quarantine"
	if err := os.MkdirAll(quarantineDir, 0700); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Karantina dizini oluşturulamadı"})
	}

	// Move file to quarantine with timestamp
	fileName := strings.ReplaceAll(input.Path, "/", "_")
	quarantinePath := quarantineDir + "/" + time.Now().Format("20060102_150405") + "_" + fileName

	if err := os.Rename(input.Path, quarantinePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Dosya karantinaya alınamadı: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"message":         "Dosya karantinaya alındı",
		"quarantine_path": quarantinePath,
	})
}

// DeleteInfectedFile permanently deletes an infected file
func (h *Handler) DeleteInfectedFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var input struct {
		Path string `json:"path"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	if input.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Dosya yolu gerekli"})
	}

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	homeDir := "/home/" + username

	// Security check
	if role != "admin" && !strings.HasPrefix(input.Path, homeDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Bu dosyayı silme yetkiniz yok"})
	}

	// Check if file exists
	if _, err := os.Stat(input.Path); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "Dosya bulunamadı"})
	}

	// Delete file
	if err := os.Remove(input.Path); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Dosya silinemedi: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Dosya silindi",
	})
}

// GetQuarantinedFiles lists quarantined files
func (h *Handler) GetQuarantinedFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	quarantineDir := "/home/" + username + "/.quarantine"

	// Check if quarantine directory exists
	if _, err := os.Stat(quarantineDir); os.IsNotExist(err) {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []string{},
		})
	}

	// List files
	files, err := os.ReadDir(quarantineDir)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Karantina dizini okunamadı"})
	}

	quarantinedFiles := []fiber.Map{}
	for _, file := range files {
		info, _ := file.Info()
		quarantinedFiles = append(quarantinedFiles, fiber.Map{
			"name":           file.Name(),
			"path":           quarantineDir + "/" + file.Name(),
			"size":           info.Size(),
			"quarantined_at": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    quarantinedFiles,
	})
}

// RestoreFromQuarantine restores a file from quarantine
func (h *Handler) RestoreFromQuarantine(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var input struct {
		QuarantinePath string `json:"quarantine_path"`
		RestorePath    string `json:"restore_path"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	homeDir := "/home/" + username
	quarantineDir := homeDir + "/.quarantine"

	// Security check
	if !strings.HasPrefix(input.QuarantinePath, quarantineDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Geçersiz karantina yolu"})
	}

	if !strings.HasPrefix(input.RestorePath, homeDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Dosya sadece home dizinine geri yüklenebilir"})
	}

	// Check if quarantined file exists
	if _, err := os.Stat(input.QuarantinePath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "Karantina dosyası bulunamadı"})
	}

	// Restore file
	if err := os.Rename(input.QuarantinePath, input.RestorePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Dosya geri yüklenemedi: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Dosya geri yüklendi",
	})
}
