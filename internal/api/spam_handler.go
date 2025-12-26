package api

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Global scan manager for tracking active scans
var scanManager = &ScanManager{
	activeScans: make(map[int64]*ActiveScan),
}

// ScanManager manages active background scans
type ScanManager struct {
	mu          sync.RWMutex
	activeScans map[int64]*ActiveScan // key: scan ID
}

// ActiveScan represents a running scan
type ActiveScan struct {
	ID           int64
	UserID       int64
	Path         string
	Status       string
	TotalFiles   int
	ScannedFiles int
	CurrentFile  string
	Results      []ScanResult
	StartedAt    time.Time
	Cancel       chan struct{}
}

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
		// Check last update time - try both .cvd and .cld files
		dbFiles := []string{
			"/var/lib/clamav/daily.cvd",
			"/var/lib/clamav/daily.cld",
			"/var/lib/clamav/main.cvd",
			"/var/lib/clamav/main.cld",
		}

		for _, dbFile := range dbFiles {
			if info, err := os.Stat(dbFile); err == nil {
				status.LastUpdate = info.ModTime().Format("2006-01-02 15:04")
				break
			}
		}

		// Get version using clamscan --version (more reliable)
		if out, err := exec.Command("clamscan", "--version").Output(); err == nil {
			// Output format: ClamAV 0.103.8/27155/Wed Dec 25 09:24:01 2024
			parts := strings.Split(strings.TrimSpace(string(out)), "/")
			if len(parts) >= 2 {
				status.VirusDBVersion = parts[1]
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
	if role == "admin" {
		if input.Path != "" {
			// Admin can scan any path
			scanPath = input.Path
		} else {
			// Admin default: scan all user home directories
			scanPath = "/home"
		}
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
	role := c.Locals("role").(string)

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	var scanPaths []string
	if role == "admin" {
		// Admin: scan all user public_html directories
		scanPaths = []string{"/home"}
	} else {
		homeDir := "/home/" + username
		scanPaths = []string{homeDir + "/public_html"}
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

// ==================== BACKGROUND SCANNING ====================

// StartBackgroundScan starts a malware scan in background
func (h *Handler) StartBackgroundScan(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var input struct {
		Path     string `json:"path"`
		ScanType string `json:"scan_type"` // quick or full
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	if input.ScanType == "" {
		input.ScanType = "full"
	}

	// Get user's home directory
	var username string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	// Determine scan path
	var scanPath string
	if role == "admin" {
		if input.Path != "" {
			scanPath = input.Path
		} else {
			scanPath = "/home"
		}
	} else {
		homeDir := "/home/" + username
		if input.Path != "" {
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

	// Check if user already has an active scan
	scanManager.mu.RLock()
	for _, scan := range scanManager.activeScans {
		if scan.UserID == userID && scan.Status == "running" {
			scanManager.mu.RUnlock()
			return c.Status(400).JSON(fiber.Map{"error": "Zaten aktif bir taramanız var"})
		}
	}
	scanManager.mu.RUnlock()

	// Count total files
	totalFiles := 0
	countCmd := exec.Command("find", scanPath, "-type", "f")
	countOutput, _ := countCmd.Output()
	if len(countOutput) > 0 {
		totalFiles = len(strings.Split(strings.TrimSpace(string(countOutput)), "\n"))
	}

	// Create scan record in database
	result, err := h.db.Exec(`
		INSERT INTO malware_scans (user_id, scan_path, scan_type, status, total_files, started_at)
		VALUES (?, ?, ?, 'running', ?, CURRENT_TIMESTAMP)
	`, userID, scanPath, input.ScanType, totalFiles)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Tarama başlatılamadı"})
	}

	scanID, _ := result.LastInsertId()

	// Create active scan
	activeScan := &ActiveScan{
		ID:         scanID,
		UserID:     userID,
		Path:       scanPath,
		Status:     "running",
		TotalFiles: totalFiles,
		StartedAt:  time.Now(),
		Cancel:     make(chan struct{}),
		Results:    []ScanResult{},
	}

	scanManager.mu.Lock()
	scanManager.activeScans[scanID] = activeScan
	scanManager.mu.Unlock()

	// Start background scan
	go h.runBackgroundScan(activeScan)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"scan_id":     scanID,
			"status":      "running",
			"total_files": totalFiles,
			"path":        scanPath,
		},
	})
}

// runBackgroundScan executes the scan in background
func (h *Handler) runBackgroundScan(scan *ActiveScan) {
	defer func() {
		// Cleanup
		scanManager.mu.Lock()
		delete(scanManager.activeScans, scan.ID)
		scanManager.mu.Unlock()
	}()

	startTime := time.Now()

	// Run clamscan with progress output
	cmd := exec.Command("clamscan", "-r", "--infected", scan.Path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		h.updateScanStatus(scan.ID, "error", 0, "Tarama başlatılamadı")
		return
	}

	if err := cmd.Start(); err != nil {
		h.updateScanStatus(scan.ID, "error", 0, "Tarama başlatılamadı")
		return
	}

	scanner := bufio.NewScanner(stdout)
	scannedCount := 0
	infectedCount := 0

	for scanner.Scan() {
		select {
		case <-scan.Cancel:
			cmd.Process.Kill()
			h.updateScanStatus(scan.ID, "cancelled", int(time.Since(startTime).Seconds()), "")
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Update current file
		if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				filePath := parts[0]
				status := parts[1]

				scan.CurrentFile = filePath
				scannedCount++
				scan.ScannedFiles = scannedCount

				if strings.Contains(status, "FOUND") {
					infectedCount++
					scan.Results = append(scan.Results, ScanResult{
						Path:      filePath,
						Status:    "infected",
						Threat:    strings.TrimSuffix(status, " FOUND"),
						ScannedAt: time.Now().Format("2006-01-02 15:04:05"),
					})
				}

				// Update database every 100 files
				if scannedCount%100 == 0 {
					h.db.Exec(`
						UPDATE malware_scans 
						SET scanned_files = ?, infected_files = ?, current_file = ?
						WHERE id = ?
					`, scannedCount, infectedCount, filePath, scan.ID)
				}
			}
		}
	}

	cmd.Wait()
	duration := int(time.Since(startTime).Seconds())

	// Save results to database
	resultsJSON, _ := json.Marshal(scan.Results)
	h.db.Exec(`
		UPDATE malware_scans 
		SET status = 'completed', scanned_files = ?, infected_files = ?, 
		    results = ?, duration = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, scannedCount, infectedCount, string(resultsJSON), duration, scan.ID)

	scan.Status = "completed"
}

// updateScanStatus updates scan status in database
func (h *Handler) updateScanStatus(scanID int64, status string, duration int, errorMsg string) {
	if errorMsg != "" {
		h.db.Exec(`
			UPDATE malware_scans 
			SET status = ?, duration = ?, results = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, status, duration, fmt.Sprintf(`{"error": "%s"}`, errorMsg), scanID)
	} else {
		h.db.Exec(`
			UPDATE malware_scans 
			SET status = ?, duration = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, status, duration, scanID)
	}
}

// GetScanStatus returns current scan status
func (h *Handler) GetScanStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	scanID := c.Params("id")

	// First check active scans
	scanManager.mu.RLock()
	for _, scan := range scanManager.activeScans {
		if fmt.Sprintf("%d", scan.ID) == scanID && scan.UserID == userID {
			scanManager.mu.RUnlock()
			return c.JSON(fiber.Map{
				"success": true,
				"data": fiber.Map{
					"id":             scan.ID,
					"status":         scan.Status,
					"total_files":    scan.TotalFiles,
					"scanned_files":  scan.ScannedFiles,
					"infected_files": len(scan.Results),
					"current_file":   scan.CurrentFile,
					"results":        scan.Results,
					"duration":       int(time.Since(scan.StartedAt).Seconds()),
				},
			})
		}
	}
	scanManager.mu.RUnlock()

	// Check database for completed scan
	var scan struct {
		ID            int64
		ScanPath      string
		ScanType      string
		Status        string
		TotalFiles    int
		ScannedFiles  int
		InfectedFiles int
		CurrentFile   sql.NullString
		Results       sql.NullString
		Duration      int
		StartedAt     sql.NullTime
		CompletedAt   sql.NullTime
	}

	err := h.db.QueryRow(`
		SELECT id, scan_path, scan_type, status, total_files, scanned_files, 
		       infected_files, current_file, results, duration, started_at, completed_at
		FROM malware_scans WHERE id = ? AND user_id = ?
	`, scanID, userID).Scan(
		&scan.ID, &scan.ScanPath, &scan.ScanType, &scan.Status,
		&scan.TotalFiles, &scan.ScannedFiles, &scan.InfectedFiles,
		&scan.CurrentFile, &scan.Results, &scan.Duration,
		&scan.StartedAt, &scan.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Tarama bulunamadı"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}

	// Parse results
	var results []ScanResult
	if scan.Results.Valid && scan.Results.String != "" {
		json.Unmarshal([]byte(scan.Results.String), &results)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":             scan.ID,
			"path":           scan.ScanPath,
			"scan_type":      scan.ScanType,
			"status":         scan.Status,
			"total_files":    scan.TotalFiles,
			"scanned_files":  scan.ScannedFiles,
			"infected_files": scan.InfectedFiles,
			"current_file":   scan.CurrentFile.String,
			"results":        results,
			"duration":       scan.Duration,
		},
	})
}

// CancelScan cancels an active scan
func (h *Handler) CancelScan(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	scanID := c.Params("id")

	scanManager.mu.Lock()
	defer scanManager.mu.Unlock()

	for _, scan := range scanManager.activeScans {
		if fmt.Sprintf("%d", scan.ID) == scanID && scan.UserID == userID {
			close(scan.Cancel)
			return c.JSON(fiber.Map{
				"success": true,
				"message": "Tarama iptal edildi",
			})
		}
	}

	return c.Status(404).JSON(fiber.Map{"error": "Aktif tarama bulunamadı"})
}

// GetScanHistory returns scan history for user
func (h *Handler) GetScanHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == "admin" {
		// Admin can see all scans
		query = `
			SELECT ms.id, ms.user_id, u.username, ms.scan_path, ms.scan_type, ms.status, 
			       ms.total_files, ms.scanned_files, ms.infected_files, ms.duration, 
			       ms.started_at, ms.completed_at
			FROM malware_scans ms
			LEFT JOIN users u ON ms.user_id = u.id
			ORDER BY ms.created_at DESC
			LIMIT 50
		`
	} else {
		query = `
			SELECT ms.id, ms.user_id, u.username, ms.scan_path, ms.scan_type, ms.status, 
			       ms.total_files, ms.scanned_files, ms.infected_files, ms.duration, 
			       ms.started_at, ms.completed_at
			FROM malware_scans ms
			LEFT JOIN users u ON ms.user_id = u.id
			WHERE ms.user_id = ?
			ORDER BY ms.created_at DESC
			LIMIT 50
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}
	defer rows.Close()

	var scans []fiber.Map
	for rows.Next() {
		var scan struct {
			ID            int64
			UserID        int64
			Username      sql.NullString
			ScanPath      string
			ScanType      string
			Status        string
			TotalFiles    int
			ScannedFiles  int
			InfectedFiles int
			Duration      int
			StartedAt     sql.NullTime
			CompletedAt   sql.NullTime
		}

		err := rows.Scan(
			&scan.ID, &scan.UserID, &scan.Username, &scan.ScanPath, &scan.ScanType,
			&scan.Status, &scan.TotalFiles, &scan.ScannedFiles, &scan.InfectedFiles,
			&scan.Duration, &scan.StartedAt, &scan.CompletedAt,
		)
		if err != nil {
			continue
		}

		startedAt := ""
		if scan.StartedAt.Valid {
			startedAt = scan.StartedAt.Time.Format("2006-01-02 15:04:05")
		}
		completedAt := ""
		if scan.CompletedAt.Valid {
			completedAt = scan.CompletedAt.Time.Format("2006-01-02 15:04:05")
		}

		scans = append(scans, fiber.Map{
			"id":             scan.ID,
			"user_id":        scan.UserID,
			"username":       scan.Username.String,
			"path":           scan.ScanPath,
			"scan_type":      scan.ScanType,
			"status":         scan.Status,
			"total_files":    scan.TotalFiles,
			"scanned_files":  scan.ScannedFiles,
			"infected_files": scan.InfectedFiles,
			"duration":       scan.Duration,
			"started_at":     startedAt,
			"completed_at":   completedAt,
		})
	}

	if scans == nil {
		scans = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    scans,
	})
}

// GetActiveScan returns the active scan for user (if any)
func (h *Handler) GetActiveScan(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	scanManager.mu.RLock()
	defer scanManager.mu.RUnlock()

	for _, scan := range scanManager.activeScans {
		if scan.UserID == userID && scan.Status == "running" {
			return c.JSON(fiber.Map{
				"success": true,
				"data": fiber.Map{
					"id":             scan.ID,
					"status":         scan.Status,
					"path":           scan.Path,
					"total_files":    scan.TotalFiles,
					"scanned_files":  scan.ScannedFiles,
					"infected_files": len(scan.Results),
					"current_file":   scan.CurrentFile,
					"duration":       int(time.Since(scan.StartedAt).Seconds()),
				},
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    nil,
	})
}
