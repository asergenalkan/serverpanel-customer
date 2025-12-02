package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// FTPAccount represents an FTP account
type FTPAccount struct {
	ID                int64  `json:"id"`
	UserID            int64  `json:"user_id"`
	Username          string `json:"username"`
	HomeDirectory     string `json:"home_directory"`
	QuotaMB           int    `json:"quota_mb"`
	UploadBandwidth   int    `json:"upload_bandwidth"`
	DownloadBandwidth int    `json:"download_bandwidth"`
	Active            bool   `json:"active"`
	CreatedAt         string `json:"created_at"`
	// For display
	OwnerUsername string `json:"owner_username,omitempty"`
}

// FTPSettings represents FTP server configuration
type FTPSettings struct {
	TLSEncryption         string `json:"tls_encryption"`   // disabled, optional, required
	TLSCipherSuite        string `json:"tls_cipher_suite"` // HIGH, MEDIUM, etc.
	AllowAnonymousLogins  bool   `json:"allow_anonymous_logins"`
	AllowAnonymousUploads bool   `json:"allow_anonymous_uploads"`
	MaxIdleTime           int    `json:"max_idle_time"` // minutes
	MaxConnections        int    `json:"max_connections"`
	MaxConnectionsPerIP   int    `json:"max_connections_per_ip"`
	AllowRootLogin        bool   `json:"allow_root_login"`
	PassivePortMin        int    `json:"passive_port_min"`
	PassivePortMax        int    `json:"passive_port_max"`
}

// ListFTPAccounts returns all FTP accounts (admin sees all, user sees own)
func (h *Handler) ListFTPAccounts(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var accounts []FTPAccount
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT f.id, f.user_id, f.username, f.home_directory, f.quota_mb, 
			       f.upload_bandwidth, f.download_bandwidth, f.active, f.created_at,
			       u.username as owner_username
			FROM ftp_accounts f
			JOIN users u ON f.user_id = u.id
			ORDER BY f.created_at DESC
		`
	} else {
		query = `
			SELECT f.id, f.user_id, f.username, f.home_directory, f.quota_mb, 
			       f.upload_bandwidth, f.download_bandwidth, f.active, f.created_at,
			       '' as owner_username
			FROM ftp_accounts f
			WHERE f.user_id = ?
			ORDER BY f.created_at DESC
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch FTP accounts",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var acc FTPAccount
		var activeInt int
		err := rows.Scan(&acc.ID, &acc.UserID, &acc.Username, &acc.HomeDirectory,
			&acc.QuotaMB, &acc.UploadBandwidth, &acc.DownloadBandwidth,
			&activeInt, &acc.CreatedAt, &acc.OwnerUsername)
		if err != nil {
			continue
		}
		acc.Active = activeInt == 1
		accounts = append(accounts, acc)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    accounts,
	})
}

// CreateFTPAccount creates a new FTP account
func (h *Handler) CreateFTPAccount(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		HomeDirectory string `json:"home_directory"`
		QuotaMB       int    `json:"quota_mb"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate username
	if len(req.Username) < 3 || len(req.Username) > 32 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Username must be 3-32 characters",
		})
	}

	// Validate password
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Password must be at least 6 characters",
		})
	}

	// Get system username for home directory validation
	var systemUsername string
	err := h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&systemUsername)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to get user info",
		})
	}

	// User's home base directory
	userHome := filepath.Join("/home", systemUsername)

	// Set home directory - convert relative path to absolute
	var homeDir string
	if req.HomeDirectory == "" || req.HomeDirectory == "public_html" {
		homeDir = filepath.Join(userHome, "public_html")
	} else if strings.HasPrefix(req.HomeDirectory, "/") {
		// Absolute path provided - validate it's within user's home (admin only)
		if role != models.RoleAdmin {
			// For non-admin, force it to be within their home
			cleanPath := filepath.Clean(req.HomeDirectory)
			if !strings.HasPrefix(cleanPath, userHome) {
				return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
					Success: false,
					Error:   "Erişim dizini kendi klasörünüz içinde olmalıdır",
				})
			}
			homeDir = cleanPath
		} else {
			homeDir = filepath.Clean(req.HomeDirectory)
		}
	} else {
		// Relative path - append to user's home
		// Sanitize: remove any ../ attempts
		cleanRelPath := filepath.Clean(req.HomeDirectory)
		if strings.Contains(cleanRelPath, "..") {
			return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Error:   "Geçersiz dizin yolu",
			})
		}
		homeDir = filepath.Join(userHome, cleanRelPath)
	}

	// Final security check - ensure path is within user's home
	if role != models.RoleAdmin {
		if !strings.HasPrefix(homeDir, userHome) {
			return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Error:   "Erişim dizini kendi klasörünüz içinde olmalıdır",
			})
		}
	}

	// Generate FTP username (prefix with system username)
	ftpUsername := fmt.Sprintf("%s_%s", systemUsername, req.Username)

	// Check if username already exists
	var exists int
	h.db.QueryRow("SELECT COUNT(*) FROM ftp_accounts WHERE username = ?", ftpUsername).Scan(&exists)
	if exists > 0 {
		return c.Status(fiber.StatusConflict).JSON(models.APIResponse{
			Success: false,
			Error:   "FTP username already exists",
		})
	}

	// Hash password for database storage
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to hash password",
		})
	}

	// Insert into database
	result, err := h.db.Exec(`
		INSERT INTO ftp_accounts (user_id, username, password, home_directory, quota_mb, active)
		VALUES (?, ?, ?, ?, ?, 1)
	`, userID, ftpUsername, string(hashedPassword), homeDir, req.QuotaMB)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to create FTP account",
		})
	}

	accountID, _ := result.LastInsertId()

	// Create Pure-FTPd virtual user
	if !config.IsDevelopment() {
		// Use system username (not www-data) to avoid uid < 1000 rejection
		if err := createPureFTPdUser(ftpUsername, req.Password, homeDir, systemUsername, req.QuotaMB); err != nil {
			// Rollback database insert
			h.db.Exec("DELETE FROM ftp_accounts WHERE id = ?", accountID)
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Failed to create system FTP user: " + err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "FTP account created successfully",
		Data: map[string]interface{}{
			"id":       accountID,
			"username": ftpUsername,
			"host":     "Your server IP or hostname",
			"port":     21,
		},
	})
}

// DeleteFTPAccount deletes an FTP account
func (h *Handler) DeleteFTPAccount(c *fiber.Ctx) error {
	accountID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid account ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get account info
	var ownerID int64
	var ftpUsername string
	err = h.db.QueryRow("SELECT user_id, username FROM ftp_accounts WHERE id = ?", accountID).Scan(&ownerID, &ftpUsername)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "FTP account not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && userID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Delete from Pure-FTPd
	if !config.IsDevelopment() {
		deletePureFTPdUser(ftpUsername)
	}

	// Delete from database
	_, err = h.db.Exec("DELETE FROM ftp_accounts WHERE id = ?", accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to delete FTP account",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "FTP account deleted successfully",
	})
}

// UpdateFTPAccount updates an FTP account
func (h *Handler) UpdateFTPAccount(c *fiber.Ctx) error {
	accountID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid account ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		Password      string `json:"password"`
		HomeDirectory string `json:"home_directory"`
		QuotaMB       int    `json:"quota_mb"`
		Active        *bool  `json:"active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Get account info
	var ownerID int64
	var ftpUsername, currentHomeDir string
	err = h.db.QueryRow("SELECT user_id, username, home_directory FROM ftp_accounts WHERE id = ?", accountID).
		Scan(&ownerID, &ftpUsername, &currentHomeDir)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "FTP account not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && userID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Update password if provided
	if req.Password != "" {
		if len(req.Password) < 6 {
			return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Error:   "Password must be at least 6 characters",
			})
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		h.db.Exec("UPDATE ftp_accounts SET password = ? WHERE id = ?", string(hashedPassword), accountID)

		// Update Pure-FTPd password
		if !config.IsDevelopment() {
			updatePureFTPdPassword(ftpUsername, req.Password)
		}
	}

	// Update home directory if provided
	if req.HomeDirectory != "" && req.HomeDirectory != currentHomeDir {
		h.db.Exec("UPDATE ftp_accounts SET home_directory = ? WHERE id = ?", req.HomeDirectory, accountID)
	}

	// Update quota if provided
	if req.QuotaMB > 0 {
		h.db.Exec("UPDATE ftp_accounts SET quota_mb = ? WHERE id = ?", req.QuotaMB, accountID)
	}

	// Update active status if provided
	if req.Active != nil {
		activeInt := 0
		if *req.Active {
			activeInt = 1
		}
		h.db.Exec("UPDATE ftp_accounts SET active = ? WHERE id = ?", activeInt, accountID)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "FTP account updated successfully",
	})
}

// ToggleFTPAccount enables/disables an FTP account
func (h *Handler) ToggleFTPAccount(c *fiber.Ctx) error {
	accountID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid account ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get account info
	var ownerID int64
	var active int
	var ftpUsername string
	err = h.db.QueryRow("SELECT user_id, username, active FROM ftp_accounts WHERE id = ?", accountID).
		Scan(&ownerID, &ftpUsername, &active)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "FTP account not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && userID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Toggle status
	newActive := 0
	if active == 0 {
		newActive = 1
	}

	_, err = h.db.Exec("UPDATE ftp_accounts SET active = ? WHERE id = ?", newActive, accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to update FTP account",
		})
	}

	status := "disabled"
	if newActive == 1 {
		status = "enabled"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("FTP account %s", status),
	})
}

// GetFTPSettings returns FTP server settings (admin only)
func (h *Handler) GetFTPSettings(c *fiber.Ctx) error {
	settings := FTPSettings{
		TLSEncryption:         "optional",
		TLSCipherSuite:        "HIGH",
		AllowAnonymousLogins:  false,
		AllowAnonymousUploads: false,
		MaxIdleTime:           15,
		MaxConnections:        50,
		MaxConnectionsPerIP:   8,
		AllowRootLogin:        false,
		PassivePortMin:        30000,
		PassivePortMax:        31000,
	}

	// Load from database
	rows, err := h.db.Query("SELECT setting_key, setting_value FROM ftp_settings")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			switch key {
			case "tls_encryption":
				settings.TLSEncryption = value
			case "tls_cipher_suite":
				settings.TLSCipherSuite = value
			case "allow_anonymous_logins":
				settings.AllowAnonymousLogins = value == "1"
			case "allow_anonymous_uploads":
				settings.AllowAnonymousUploads = value == "1"
			case "max_idle_time":
				settings.MaxIdleTime, _ = strconv.Atoi(value)
			case "max_connections":
				settings.MaxConnections, _ = strconv.Atoi(value)
			case "max_connections_per_ip":
				settings.MaxConnectionsPerIP, _ = strconv.Atoi(value)
			case "allow_root_login":
				settings.AllowRootLogin = value == "1"
			case "passive_port_min":
				settings.PassivePortMin, _ = strconv.Atoi(value)
			case "passive_port_max":
				settings.PassivePortMax, _ = strconv.Atoi(value)
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    settings,
	})
}

// UpdateFTPSettings updates FTP server settings (admin only)
func (h *Handler) UpdateFTPSettings(c *fiber.Ctx) error {
	var req FTPSettings
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Save settings to database
	settings := map[string]string{
		"tls_encryption":          req.TLSEncryption,
		"tls_cipher_suite":        req.TLSCipherSuite,
		"allow_anonymous_logins":  boolToStr(req.AllowAnonymousLogins),
		"allow_anonymous_uploads": boolToStr(req.AllowAnonymousUploads),
		"max_idle_time":           strconv.Itoa(req.MaxIdleTime),
		"max_connections":         strconv.Itoa(req.MaxConnections),
		"max_connections_per_ip":  strconv.Itoa(req.MaxConnectionsPerIP),
		"allow_root_login":        boolToStr(req.AllowRootLogin),
		"passive_port_min":        strconv.Itoa(req.PassivePortMin),
		"passive_port_max":        strconv.Itoa(req.PassivePortMax),
	}

	for key, value := range settings {
		h.db.Exec(`
			INSERT INTO ftp_settings (setting_key, setting_value, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(setting_key) DO UPDATE SET
				setting_value = excluded.setting_value,
				updated_at = CURRENT_TIMESTAMP
		`, key, value)
	}

	// Apply settings to Pure-FTPd
	if !config.IsDevelopment() {
		if err := applyPureFTPdSettings(req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Failed to apply FTP settings: " + err.Error(),
			})
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "FTP settings updated successfully",
	})
}

// RestartFTPServer restarts the FTP server (admin only)
func (h *Handler) RestartFTPServer(c *fiber.Ctx) error {
	if !config.IsDevelopment() {
		cmd := exec.Command("systemctl", "restart", "pure-ftpd")
		if err := cmd.Run(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Failed to restart FTP server",
			})
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "FTP server restarted successfully",
	})
}

// GetFTPServerStatus returns FTP server status
func (h *Handler) GetFTPServerStatus(c *fiber.Ctx) error {
	status := "unknown"

	if !config.IsDevelopment() {
		cmd := exec.Command("systemctl", "is-active", "pure-ftpd")
		output, _ := cmd.Output()
		status = strings.TrimSpace(string(output))
	} else {
		status = "development"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]string{
			"status": status,
		},
	})
}

// Helper functions

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Pure-FTPd integration functions

func createPureFTPdUser(username, password, homeDir, systemUser string, quotaMB int) error {
	// Create virtual user using pure-pw
	// pure-pw useradd username -u systemuser -g systemuser -d /home/user/public_html -m
	// IMPORTANT: Must use system user (uid >= 1000) not www-data (uid 33)
	// Pure-FTPd rejects users with uid < 1000 by default

	// First, ensure the home directory exists
	os.MkdirAll(homeDir, 0755)

	// Create the user with pure-pw using the system username
	cmd := exec.Command("pure-pw", "useradd", username,
		"-u", systemUser,
		"-g", systemUser,
		"-d", homeDir,
		"-m")

	// Pipe the password
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		fmt.Fprintf(stdin, "%s\n%s\n", password, password)
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create FTP user: %v", err)
	}

	// Set quota if specified
	if quotaMB > 0 {
		quotaBytes := quotaMB * 1024 * 1024
		exec.Command("pure-pw", "usermod", username, "-n", strconv.Itoa(quotaBytes), "-m").Run()
	}

	// Rebuild the PureDB
	exec.Command("pure-pw", "mkdb").Run()

	return nil
}

func deletePureFTPdUser(username string) error {
	cmd := exec.Command("pure-pw", "userdel", username, "-m")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Rebuild the PureDB
	exec.Command("pure-pw", "mkdb").Run()

	return nil
}

func updatePureFTPdPassword(username, password string) error {
	cmd := exec.Command("pure-pw", "passwd", username, "-m")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		fmt.Fprintf(stdin, "%s\n%s\n", password, password)
	}()

	return cmd.Run()
}

func applyPureFTPdSettings(settings FTPSettings) error {
	// Pure-FTPd configuration is done via command-line options or config files
	// For Ubuntu/Debian, settings are in /etc/pure-ftpd/conf/

	confDir := "/etc/pure-ftpd/conf"

	// TLS settings
	tlsValue := "0"
	switch settings.TLSEncryption {
	case "optional":
		tlsValue = "1"
	case "required":
		tlsValue = "2"
	}
	os.WriteFile(filepath.Join(confDir, "TLS"), []byte(tlsValue+"\n"), 0644)

	// Anonymous settings
	if settings.AllowAnonymousLogins {
		os.WriteFile(filepath.Join(confDir, "NoAnonymous"), []byte("no\n"), 0644)
	} else {
		os.WriteFile(filepath.Join(confDir, "NoAnonymous"), []byte("yes\n"), 0644)
	}

	// Max idle time
	os.WriteFile(filepath.Join(confDir, "MaxIdleTime"), []byte(strconv.Itoa(settings.MaxIdleTime)+"\n"), 0644)

	// Max connections
	os.WriteFile(filepath.Join(confDir, "MaxClientsNumber"), []byte(strconv.Itoa(settings.MaxConnections)+"\n"), 0644)

	// Max connections per IP
	os.WriteFile(filepath.Join(confDir, "MaxClientsPerIP"), []byte(strconv.Itoa(settings.MaxConnectionsPerIP)+"\n"), 0644)

	// Passive port range
	passiveRange := fmt.Sprintf("%d %d", settings.PassivePortMin, settings.PassivePortMax)
	os.WriteFile(filepath.Join(confDir, "PassivePortRange"), []byte(passiveRange+"\n"), 0644)

	// Restart Pure-FTPd to apply changes
	return exec.Command("systemctl", "restart", "pure-ftpd").Run()
}
