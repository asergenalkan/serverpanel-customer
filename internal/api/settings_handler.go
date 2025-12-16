package api

import (
	"os/exec"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// ServerSettings represents server configuration
type ServerSettings struct {
	MultiPHPEnabled    bool     `json:"multiphp_enabled"`
	DefaultPHPVersion  string   `json:"default_php_version"`
	AllowedPHPVersions []string `json:"allowed_php_versions"`
	DomainBasedPHP     bool     `json:"domain_based_php"`
	NodejsEnabled      bool     `json:"nodejs_enabled"`
}

// GetServerSettings returns server settings (admin only)
func (h *Handler) GetServerSettings(c *fiber.Ctx) error {
	settings := ServerSettings{
		MultiPHPEnabled:    true,
		DefaultPHPVersion:  "8.1",
		AllowedPHPVersions: []string{"7.4", "8.0", "8.1", "8.2", "8.3"},
		DomainBasedPHP:     true,
	}

	// Load from database
	rows, err := h.db.Query("SELECT key, value FROM server_settings")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			switch key {
			case "multiphp_enabled":
				settings.MultiPHPEnabled = value == "true"
			case "default_php_version":
				settings.DefaultPHPVersion = value
			case "allowed_php_versions":
				settings.AllowedPHPVersions = strings.Split(value, ",")
			case "domain_based_php":
				settings.DomainBasedPHP = value == "true"
			case "nodejs_enabled":
				settings.NodejsEnabled = value == "true"
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    settings,
	})
}

// UpdateServerSettings updates server settings (admin only)
func (h *Handler) UpdateServerSettings(c *fiber.Ctx) error {
	var req ServerSettings
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate default PHP version is in allowed list
	found := false
	for _, v := range req.AllowedPHPVersions {
		if v == req.DefaultPHPVersion {
			found = true
			break
		}
	}
	if !found && len(req.AllowedPHPVersions) > 0 {
		req.DefaultPHPVersion = req.AllowedPHPVersions[0]
	}

	// Update settings
	updates := map[string]string{
		"multiphp_enabled":     boolToString(req.MultiPHPEnabled),
		"default_php_version":  req.DefaultPHPVersion,
		"allowed_php_versions": strings.Join(req.AllowedPHPVersions, ","),
		"domain_based_php":     boolToString(req.DomainBasedPHP),
		"nodejs_enabled":       boolToString(req.NodejsEnabled),
	}

	for key, value := range updates {
		_, err := h.db.Exec(`
			INSERT INTO server_settings (key, value, updated_at) 
			VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
		`, key, value)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Ayarlar kaydedilemedi",
			})
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Ayarlar kaydedildi",
	})
}

// GetAllowedPHPVersions returns allowed PHP versions for users
func (h *Handler) GetAllowedPHPVersions(c *fiber.Ctx) error {
	// Get settings
	var allowedVersionsStr string
	var domainBasedPHP string
	h.db.QueryRow("SELECT value FROM server_settings WHERE key = 'allowed_php_versions'").Scan(&allowedVersionsStr)
	h.db.QueryRow("SELECT value FROM server_settings WHERE key = 'domain_based_php'").Scan(&domainBasedPHP)

	if allowedVersionsStr == "" {
		allowedVersionsStr = "7.4,8.0,8.1,8.2,8.3"
	}

	allowedVersions := strings.Split(allowedVersionsStr, ",")

	// Filter to only installed versions
	installedVersions := h.getInstalledPHPVersionsList()
	var availableVersions []string
	for _, v := range allowedVersions {
		for _, installed := range installedVersions {
			if v == installed {
				availableVersions = append(availableVersions, v)
				break
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"versions":         availableVersions,
			"domain_based_php": domainBasedPHP == "true",
		},
	})
}

// GetServerFeatures returns server features for users (read-only view)
func (h *Handler) GetServerFeatures(c *fiber.Ctx) error {
	// Get nodejs_enabled from server_settings
	nodejsEnabled := false
	var value string
	if err := h.db.QueryRow("SELECT value FROM server_settings WHERE key = 'nodejs_enabled'").Scan(&value); err == nil {
		nodejsEnabled = value == "true"
	}

	features := map[string]interface{}{
		"php_versions":        h.getPHPVersions(),
		"php_extensions":      h.getPHPExtensionsSimple(),
		"apache_modules":      h.getApacheModulesSimple(),
		"additional_software": h.getAdditionalSoftwareSimple(),
		"nodejs_enabled":      nodejsEnabled,
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    features,
	})
}

// getPHPExtensionsSimple returns simplified PHP extensions list
func (h *Handler) getPHPExtensionsSimple() []map[string]interface{} {
	extensions := []map[string]interface{}{}
	installedVersions := h.getInstalledPHPVersionsList()

	for _, ext := range phpExtensionList {
		installed := false
		for _, v := range installedVersions {
			if h.isExtensionInstalled(v, ext.Name) {
				installed = true
				break
			}
		}
		if installed {
			extensions = append(extensions, map[string]interface{}{
				"name":         ext.Name,
				"display_name": ext.DisplayName,
				"description":  ext.Description,
			})
		}
	}

	return extensions
}

// getApacheModulesSimple returns simplified Apache modules list
func (h *Handler) getApacheModulesSimple() []map[string]interface{} {
	modules := []map[string]interface{}{}
	enabledModules := h.getEnabledApacheModules()

	for _, mod := range apacheModuleList {
		if enabledModules[mod.Name] {
			modules = append(modules, map[string]interface{}{
				"name":         mod.Name,
				"display_name": mod.DisplayName,
				"description":  mod.Description,
			})
		}
	}

	return modules
}

// getAdditionalSoftwareSimple returns simplified additional software list
func (h *Handler) getAdditionalSoftwareSimple() []map[string]interface{} {
	software := []map[string]interface{}{}

	for _, sw := range additionalSoftwareList {
		pkg := h.getSoftwareStatus(sw.Name, sw.CheckCmd)
		if pkg["installed"].(bool) {
			software = append(software, map[string]interface{}{
				"name":        sw.DisplayName,
				"description": sw.Description,
				"version":     pkg["version"],
			})
		}
	}

	return software
}

// getSoftwareStatus checks if software is installed
func (h *Handler) getSoftwareStatus(name, checkCmd string) map[string]interface{} {
	result := map[string]interface{}{
		"installed": false,
		"version":   "",
	}

	if h.isCommandAvailable(checkCmd) {
		result["installed"] = true
		result["version"] = h.getSoftwareVersion(checkCmd)
	}

	return result
}

// isCommandAvailable checks if a command exists
func (h *Handler) isCommandAvailable(cmd string) bool {
	checkCmd := exec.Command("which", cmd)
	return checkCmd.Run() == nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
