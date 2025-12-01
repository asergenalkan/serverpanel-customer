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
)

// PHPVersion represents an installed PHP version
type PHPVersion struct {
	Version   string `json:"version"`
	Path      string `json:"path"`
	IsDefault bool   `json:"is_default"`
	IsActive  bool   `json:"is_active"`
}

// PHPSettings represents PHP configuration for a domain
type PHPSettings struct {
	DomainID          int64  `json:"domain_id"`
	Domain            string `json:"domain"`
	PHPVersion        string `json:"php_version"`
	MemoryLimit       string `json:"memory_limit"`
	MaxExecutionTime  int    `json:"max_execution_time"`
	MaxInputTime      int    `json:"max_input_time"`
	PostMaxSize       string `json:"post_max_size"`
	UploadMaxFilesize string `json:"upload_max_filesize"`
	MaxFileUploads    int    `json:"max_file_uploads"`
	DisplayErrors     bool   `json:"display_errors"`
	ErrorReporting    string `json:"error_reporting"`
	// Package limits (for UI display)
	MaxAllowedMemory   string `json:"max_allowed_memory,omitempty"`
	MaxAllowedUpload   string `json:"max_allowed_upload,omitempty"`
	MaxAllowedExecTime int    `json:"max_allowed_exec_time,omitempty"`
}

// GetInstalledPHPVersions returns all installed PHP versions on the server
func (h *Handler) GetInstalledPHPVersions(c *fiber.Ctx) error {
	versions := []PHPVersion{}

	// Check common PHP versions
	phpVersions := []string{"7.4", "8.0", "8.1", "8.2", "8.3"}
	cfg := config.Get()

	for _, v := range phpVersions {
		fpmPath := fmt.Sprintf("/etc/php/%s/fpm/php-fpm.conf", v)
		if _, err := os.Stat(fpmPath); err == nil {
			versions = append(versions, PHPVersion{
				Version:   v,
				Path:      fmt.Sprintf("/etc/php/%s", v),
				IsDefault: v == cfg.PHPVersion,
				IsActive:  h.isPHPVersionActive(v),
			})
		}
	}

	// If no versions found (development mode), return default
	if len(versions) == 0 {
		versions = append(versions, PHPVersion{
			Version:   cfg.PHPVersion,
			Path:      fmt.Sprintf("/etc/php/%s", cfg.PHPVersion),
			IsDefault: true,
			IsActive:  true,
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    versions,
	})
}

// isPHPVersionActive checks if a PHP-FPM version is running
func (h *Handler) isPHPVersionActive(version string) bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", fmt.Sprintf("php%s-fpm", version))
	return cmd.Run() == nil
}

// GetDomainPHPSettings returns PHP settings for a specific domain
func (h *Handler) GetDomainPHPSettings(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info with package limits
	var domain string
	var ownerID int64
	var phpVersion string
	var maxMemory, maxUpload string
	var maxExecTime int
	err = h.db.QueryRow(`
		SELECT d.name, d.user_id, COALESCE(d.php_version, '8.1'),
		       COALESCE(p.max_php_memory, '256M'), COALESCE(p.max_php_upload, '64M'), 
		       COALESCE(p.max_php_execution_time, 300)
		FROM domains d 
		LEFT JOIN user_packages up ON up.user_id = d.user_id
		LEFT JOIN packages p ON p.id = up.package_id
		WHERE d.id = ?
	`, domainID).Scan(&domain, &ownerID, &phpVersion, &maxMemory, &maxUpload, &maxExecTime)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && userID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Get PHP settings from database
	settings := PHPSettings{
		DomainID:           domainID,
		Domain:             domain,
		PHPVersion:         phpVersion,
		MaxAllowedMemory:   maxMemory,
		MaxAllowedUpload:   maxUpload,
		MaxAllowedExecTime: maxExecTime,
	}

	err = h.db.QueryRow(`
		SELECT memory_limit, max_execution_time, max_input_time, 
		       post_max_size, upload_max_filesize, max_file_uploads,
		       display_errors, error_reporting
		FROM php_settings WHERE domain_id = ?
	`, domainID).Scan(
		&settings.MemoryLimit, &settings.MaxExecutionTime, &settings.MaxInputTime,
		&settings.PostMaxSize, &settings.UploadMaxFilesize, &settings.MaxFileUploads,
		&settings.DisplayErrors, &settings.ErrorReporting,
	)

	if err != nil {
		// Return defaults if no settings exist
		settings.MemoryLimit = "256M"
		settings.MaxExecutionTime = 300
		settings.MaxInputTime = 300
		settings.PostMaxSize = "64M"
		settings.UploadMaxFilesize = "64M"
		settings.MaxFileUploads = 20
		settings.DisplayErrors = false
		settings.ErrorReporting = "E_ALL & ~E_DEPRECATED & ~E_STRICT"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    settings,
	})
}

// UpdateDomainPHPVersion updates the PHP version for a domain
func (h *Handler) UpdateDomainPHPVersion(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	var req struct {
		PHPVersion string `json:"php_version"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info
	var domain, username string
	var ownerID int64
	var documentRoot string
	err = h.db.QueryRow(`
		SELECT d.name, d.user_id, d.document_root, u.username 
		FROM domains d 
		JOIN users u ON d.user_id = u.id 
		WHERE d.id = ?
	`, domainID).Scan(&domain, &ownerID, &documentRoot, &username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && userID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Validate PHP version
	validVersions := []string{"7.4", "8.0", "8.1", "8.2", "8.3"}
	isValid := false
	for _, v := range validVersions {
		if req.PHPVersion == v {
			isValid = true
			break
		}
	}
	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid PHP version",
		})
	}

	// Update database
	_, err = h.db.Exec("UPDATE domains SET php_version = ? WHERE id = ?", req.PHPVersion, domainID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to update PHP version",
		})
	}

	// Update Apache vhost and PHP-FPM pool
	cfg := config.Get()
	if !config.IsDevelopment() {
		// Update Apache vhost to use new PHP version
		if err := h.updateApacheVhostPHP(domain, username, documentRoot, req.PHPVersion); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Failed to update web server config: " + err.Error(),
			})
		}

		// Reload Apache
		exec.Command("systemctl", "reload", "apache2").Run()
	} else {
		_ = cfg // suppress unused warning in dev mode
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("PHP version updated to %s", req.PHPVersion),
	})
}

// UpdateDomainPHPSettings updates PHP INI settings for a domain
func (h *Handler) UpdateDomainPHPSettings(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	var req PHPSettings
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info with package limits
	var domain, username string
	var ownerID int64
	var phpVersion string
	var maxMemory, maxUpload string
	var maxExecTime int
	err = h.db.QueryRow(`
		SELECT d.name, d.user_id, u.username, COALESCE(d.php_version, '8.1'),
		       COALESCE(p.max_php_memory, '256M'), COALESCE(p.max_php_upload, '64M'), 
		       COALESCE(p.max_php_execution_time, 300)
		FROM domains d 
		JOIN users u ON d.user_id = u.id 
		LEFT JOIN user_packages up ON up.user_id = u.id
		LEFT JOIN packages p ON p.id = up.package_id
		WHERE d.id = ?
	`, domainID).Scan(&domain, &ownerID, &username, &phpVersion, &maxMemory, &maxUpload, &maxExecTime)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && userID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Validate settings (apply package limits for non-admin users)
	if role != models.RoleAdmin {
		// Apply package memory limit
		if !isValidMemoryLimit(req.MemoryLimit, maxMemory) {
			req.MemoryLimit = maxMemory
		}
		// Apply package execution time limit
		if req.MaxExecutionTime > maxExecTime {
			req.MaxExecutionTime = maxExecTime
		}
		// Apply package upload size limit
		if !isValidMemoryLimit(req.UploadMaxFilesize, maxUpload) {
			req.UploadMaxFilesize = maxUpload
		}
		// post_max_size should be at least upload_max_filesize
		if !isValidMemoryLimit(req.PostMaxSize, maxUpload) {
			req.PostMaxSize = maxUpload
		}
	}

	// Upsert PHP settings
	_, err = h.db.Exec(`
		INSERT INTO php_settings (domain_id, memory_limit, max_execution_time, max_input_time, 
		                          post_max_size, upload_max_filesize, max_file_uploads,
		                          display_errors, error_reporting, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(domain_id) DO UPDATE SET
			memory_limit = excluded.memory_limit,
			max_execution_time = excluded.max_execution_time,
			max_input_time = excluded.max_input_time,
			post_max_size = excluded.post_max_size,
			upload_max_filesize = excluded.upload_max_filesize,
			max_file_uploads = excluded.max_file_uploads,
			display_errors = excluded.display_errors,
			error_reporting = excluded.error_reporting,
			updated_at = CURRENT_TIMESTAMP
	`, domainID, req.MemoryLimit, req.MaxExecutionTime, req.MaxInputTime,
		req.PostMaxSize, req.UploadMaxFilesize, req.MaxFileUploads,
		req.DisplayErrors, req.ErrorReporting)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to save PHP settings",
		})
	}

	// Update PHP-FPM pool config
	if !config.IsDevelopment() {
		homeDir := filepath.Join("/home", username)
		if err := h.updatePHPFPMPoolSettings(username, homeDir, phpVersion, req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Failed to update PHP-FPM config: " + err.Error(),
			})
		}

		// Reload PHP-FPM
		exec.Command("systemctl", "reload", fmt.Sprintf("php%s-fpm", phpVersion)).Run()
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "PHP settings updated successfully",
	})
}

// Helper functions

func (h *Handler) updateApacheVhostPHP(domain, username, documentRoot, phpVersion string) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", domain+".conf")

	// Read current vhost
	content, err := os.ReadFile(vhostPath)
	if err != nil {
		return err
	}

	// Replace PHP version in SetHandler directive
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.Contains(line, "SetHandler") && strings.Contains(line, "php") {
			// Update the PHP-FPM socket path
			lines[i] = fmt.Sprintf(`        SetHandler "proxy:unix:/run/php/php%s-fpm-%s.sock|fcgi://localhost"`, phpVersion, username)
		}
	}

	// Write updated vhost
	return os.WriteFile(vhostPath, []byte(strings.Join(lines, "\n")), 0644)
}

func (h *Handler) updatePHPFPMPoolSettings(username, homeDir, phpVersion string, settings PHPSettings) error {
	poolPath := fmt.Sprintf("/etc/php/%s/fpm/pool.d/%s.conf", phpVersion, username)

	displayErrors := "off"
	if settings.DisplayErrors {
		displayErrors = "on"
	}

	socketPath := fmt.Sprintf("/run/php/php%s-fpm-%s.sock", phpVersion, username)

	poolConfig := fmt.Sprintf(`[%s]
; Pool for user %s

user = %s
group = %s

listen = %s
listen.owner = www-data
listen.group = www-data
listen.mode = 0660

pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
pm.max_requests = 500

; Logging
php_admin_value[error_log] = %s/logs/php-error.log
php_admin_flag[log_errors] = on

; Security
php_admin_value[open_basedir] = %s:/tmp:/usr/share/php
php_admin_value[upload_tmp_dir] = %s/tmp
php_admin_value[session.save_path] = %s/tmp

; Custom Settings
php_admin_value[memory_limit] = %s
php_admin_value[max_execution_time] = %d
php_admin_value[max_input_time] = %d
php_admin_value[post_max_size] = %s
php_admin_value[upload_max_filesize] = %s
php_admin_value[max_file_uploads] = %d
php_admin_flag[display_errors] = %s
php_admin_value[error_reporting] = %s
`,
		username, username, username, username, socketPath,
		homeDir, homeDir, homeDir, homeDir,
		settings.MemoryLimit, settings.MaxExecutionTime, settings.MaxInputTime,
		settings.PostMaxSize, settings.UploadMaxFilesize, settings.MaxFileUploads,
		displayErrors, settings.ErrorReporting,
	)

	return os.WriteFile(poolPath, []byte(poolConfig), 0644)
}

// isValidMemoryLimit checks if a memory limit is within allowed range
func isValidMemoryLimit(value, maxAllowed string) bool {
	parseSize := func(s string) int64 {
		s = strings.ToUpper(strings.TrimSpace(s))
		multiplier := int64(1)
		if strings.HasSuffix(s, "G") {
			multiplier = 1024 * 1024 * 1024
			s = strings.TrimSuffix(s, "G")
		} else if strings.HasSuffix(s, "M") {
			multiplier = 1024 * 1024
			s = strings.TrimSuffix(s, "M")
		} else if strings.HasSuffix(s, "K") {
			multiplier = 1024
			s = strings.TrimSuffix(s, "K")
		}
		val, _ := strconv.ParseInt(s, 10, 64)
		return val * multiplier
	}

	return parseSize(value) <= parseSize(maxAllowed)
}
