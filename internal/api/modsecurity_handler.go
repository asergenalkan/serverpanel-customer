package api

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ModSecurityStatus represents ModSecurity status
type ModSecurityStatus struct {
	Installed    bool   `json:"installed"`
	Enabled      bool   `json:"enabled"`
	Mode         string `json:"mode"` // DetectionOnly or On
	CRSInstalled bool   `json:"crs_installed"`
	CRSVersion   string `json:"crs_version"`
	RulesCount   int    `json:"rules_count"`
	AuditLogPath string `json:"audit_log_path"`
	AuditLogSize int64  `json:"audit_log_size"`
}

// ModSecurityRule represents a ModSecurity rule
type ModSecurityRule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Enabled     bool   `json:"enabled"`
	File        string `json:"file"`
}

// ModSecurityAuditLog represents an audit log entry
type ModSecurityAuditLog struct {
	Timestamp   string `json:"timestamp"`
	ClientIP    string `json:"client_ip"`
	RequestURI  string `json:"request_uri"`
	RuleID      string `json:"rule_id"`
	RuleMessage string `json:"rule_message"`
	Action      string `json:"action"`
	Severity    string `json:"severity"`
}

// GetModSecurityStatus returns ModSecurity status (admin only)
func (h *Handler) GetModSecurityStatus(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	status := ModSecurityStatus{
		AuditLogPath: "/var/log/modsecurity/modsec_audit.log",
	}

	// Check if ModSecurity is installed
	if _, err := os.Stat("/etc/modsecurity/modsecurity.conf"); err == nil {
		status.Installed = true
	}

	// Check if ModSecurity module is enabled
	if out, err := exec.Command("a2query", "-m", "security2").Output(); err == nil {
		if strings.Contains(string(out), "enabled") {
			status.Enabled = true
		}
	}

	// Check mode (DetectionOnly or On)
	if status.Installed {
		if content, err := os.ReadFile("/etc/modsecurity/modsecurity.conf"); err == nil {
			if strings.Contains(string(content), "SecRuleEngine On") {
				status.Mode = "On"
			} else if strings.Contains(string(content), "SecRuleEngine DetectionOnly") {
				status.Mode = "DetectionOnly"
			} else {
				status.Mode = "Off"
			}
		}
	}

	// Check OWASP CRS - check multiple possible locations
	crsPaths := []string{
		"/etc/modsecurity/crs",
		"/usr/share/modsecurity-crs",
	}

	for _, crsPath := range crsPaths {
		// Check for coreruleset-X.X.X directory structure
		if entries, err := os.ReadDir(crsPath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "coreruleset-") {
					status.CRSInstalled = true
					status.CRSVersion = strings.TrimPrefix(entry.Name(), "coreruleset-")

					rulesPath := filepath.Join(crsPath, entry.Name(), "rules")
					if ruleFiles, err := filepath.Glob(filepath.Join(rulesPath, "*.conf")); err == nil {
						status.RulesCount = len(ruleFiles)
					}
					break
				}
			}
		}

		// Check for Ubuntu's modsecurity-crs package structure
		rulesPath := filepath.Join(crsPath, "rules")
		if ruleFiles, err := filepath.Glob(filepath.Join(rulesPath, "*.conf")); err == nil && len(ruleFiles) > 0 {
			status.CRSInstalled = true
			status.CRSVersion = "3.3.2" // Ubuntu package version
			status.RulesCount = len(ruleFiles)
			break
		}
	}

	// Check audit log size
	if info, err := os.Stat(status.AuditLogPath); err == nil {
		status.AuditLogSize = info.Size()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    status,
	})
}

// ToggleModSecurity enables/disables ModSecurity (admin only)
func (h *Handler) ToggleModSecurity(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	var input struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	var cmd *exec.Cmd
	if input.Enabled {
		cmd = exec.Command("a2enmod", "security2")
	} else {
		cmd = exec.Command("a2dismod", "security2")
	}

	if err := cmd.Run(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "ModSecurity durumu değiştirilemedi"})
	}

	// Restart Apache
	if err := exec.Command("systemctl", "restart", "apache2").Run(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Apache yeniden başlatılamadı"})
	}

	action := "devre dışı bırakıldı"
	if input.Enabled {
		action = "etkinleştirildi"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ModSecurity " + action,
	})
}

// SetModSecurityMode sets ModSecurity mode (admin only)
func (h *Handler) SetModSecurityMode(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	var input struct {
		Mode string `json:"mode"` // On, DetectionOnly, Off
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	if input.Mode != "On" && input.Mode != "DetectionOnly" && input.Mode != "Off" {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz mod. On, DetectionOnly veya Off olmalı"})
	}

	configPath := "/etc/modsecurity/modsecurity.conf"
	content, err := os.ReadFile(configPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Config dosyası okunamadı"})
	}

	// Replace SecRuleEngine directive
	re := regexp.MustCompile(`SecRuleEngine\s+(On|DetectionOnly|Off)`)
	newContent := re.ReplaceAllString(string(content), "SecRuleEngine "+input.Mode)

	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Config dosyası yazılamadı"})
	}

	// Restart Apache
	if err := exec.Command("systemctl", "restart", "apache2").Run(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Apache yeniden başlatılamadı"})
	}

	modeText := map[string]string{
		"On":            "Engelleme Modu",
		"DetectionOnly": "Sadece Tespit Modu",
		"Off":           "Kapalı",
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ModSecurity modu değiştirildi: " + modeText[input.Mode],
	})
}

// GetModSecurityRules returns ModSecurity rules (admin only)
func (h *Handler) GetModSecurityRules(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	// Find CRS directory - check multiple locations
	crsPaths := []string{
		"/etc/modsecurity/crs",
		"/usr/share/modsecurity-crs",
	}

	var rulesPath string

	for _, crsPath := range crsPaths {
		// Check for coreruleset-X.X.X directory structure
		if entries, err := os.ReadDir(crsPath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "coreruleset-") {
					rulesPath = filepath.Join(crsPath, entry.Name(), "rules")
					break
				}
			}
		}

		// Check for Ubuntu's modsecurity-crs package structure
		if rulesPath == "" {
			testPath := filepath.Join(crsPath, "rules")
			if _, err := os.Stat(testPath); err == nil {
				rulesPath = testPath
			}
		}

		if rulesPath != "" {
			break
		}
	}

	if rulesPath == "" {
		return c.Status(404).JSON(fiber.Map{"error": "OWASP CRS kuralları bulunamadı"})
	}

	// List rule files
	ruleFiles, err := filepath.Glob(filepath.Join(rulesPath, "*.conf"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kural dosyaları okunamadı"})
	}

	var rules []fiber.Map
	for _, file := range ruleFiles {
		fileName := filepath.Base(file)

		// Skip data files
		if strings.Contains(fileName, "-data") {
			continue
		}

		// Parse rule file name
		// Format: REQUEST-901-INITIALIZATION.conf
		parts := strings.Split(strings.TrimSuffix(fileName, ".conf"), "-")

		category := ""
		if len(parts) >= 2 {
			category = parts[0] // REQUEST, RESPONSE, etc.
		}

		description := strings.Join(parts[1:], " ")

		// Check if rule is enabled (not commented out in security2.conf)
		enabled := true

		rules = append(rules, fiber.Map{
			"file":        fileName,
			"path":        file,
			"category":    category,
			"description": description,
			"enabled":     enabled,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    rules,
	})
}

// GetModSecurityAuditLog returns recent audit log entries (admin only)
func (h *Handler) GetModSecurityAuditLog(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	limitStr := c.Query("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit > 200 {
		limit = 200
	}

	logPath := "/var/log/modsecurity/modsec_audit.log"

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []ModSecurityAuditLog{},
		})
	}

	file, err := os.Open(logPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Log dosyası açılamadı"})
	}
	defer file.Close()

	var logs []ModSecurityAuditLog
	scanner := bufio.NewScanner(file)

	// ModSecurity audit log format is complex, parse simplified version
	var currentLog ModSecurityAuditLog
	inSection := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Section markers
		if strings.HasPrefix(line, "--") && strings.Contains(line, "-A--") {
			// New log entry
			if currentLog.Timestamp != "" {
				logs = append(logs, currentLog)
			}
			currentLog = ModSecurityAuditLog{}
			inSection = "A"
		} else if strings.HasPrefix(line, "--") && strings.Contains(line, "-B--") {
			inSection = "B"
		} else if strings.HasPrefix(line, "--") && strings.Contains(line, "-H--") {
			inSection = "H"
		} else if strings.HasPrefix(line, "--") && strings.Contains(line, "-Z--") {
			inSection = ""
		}

		// Parse sections
		switch inSection {
		case "A":
			// Timestamp and transaction info
			if strings.Contains(line, "[") && strings.Contains(line, "]") {
				// Extract timestamp
				if idx := strings.Index(line, "["); idx != -1 {
					if endIdx := strings.Index(line[idx:], "]"); endIdx != -1 {
						currentLog.Timestamp = line[idx+1 : idx+endIdx]
					}
				}
				// Extract client IP
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.Count(part, ".") == 3 {
						currentLog.ClientIP = part
						break
					}
				}
			}
		case "B":
			// Request line
			if strings.HasPrefix(line, "GET ") || strings.HasPrefix(line, "POST ") ||
				strings.HasPrefix(line, "PUT ") || strings.HasPrefix(line, "DELETE ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					currentLog.RequestURI = parts[1]
				}
			}
		case "H":
			// Audit log trailer (contains rule info)
			if strings.Contains(line, "id \"") {
				re := regexp.MustCompile(`id "(\d+)"`)
				if matches := re.FindStringSubmatch(line); len(matches) > 1 {
					currentLog.RuleID = matches[1]
				}
			}
			if strings.Contains(line, "msg \"") {
				re := regexp.MustCompile(`msg "([^"]+)"`)
				if matches := re.FindStringSubmatch(line); len(matches) > 1 {
					currentLog.RuleMessage = matches[1]
				}
			}
			if strings.Contains(line, "severity \"") {
				re := regexp.MustCompile(`severity "([^"]+)"`)
				if matches := re.FindStringSubmatch(line); len(matches) > 1 {
					currentLog.Severity = matches[1]
				}
			}
			if strings.Contains(line, "Action:") {
				if strings.Contains(line, "Intercepted") {
					currentLog.Action = "blocked"
				} else {
					currentLog.Action = "logged"
				}
			}
		}
	}

	// Add last log entry
	if currentLog.Timestamp != "" {
		logs = append(logs, currentLog)
	}

	// Reverse to get newest first and limit
	var result []ModSecurityAuditLog
	for i := len(logs) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, logs[i])
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetModSecurityStats returns ModSecurity statistics (admin only)
func (h *Handler) GetModSecurityStats(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	stats := fiber.Map{
		"total_requests":   0,
		"blocked_requests": 0,
		"logged_requests":  0,
		"top_rules":        []fiber.Map{},
		"top_ips":          []fiber.Map{},
	}

	logPath := "/var/log/modsecurity/modsec_audit.log"

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    stats,
		})
	}

	// Parse log for stats
	file, err := os.Open(logPath)
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    stats,
		})
	}
	defer file.Close()

	ruleCount := make(map[string]int)
	ipCount := make(map[string]int)
	totalRequests := 0
	blockedRequests := 0

	scanner := bufio.NewScanner(file)
	currentIP := ""
	currentRuleID := ""
	isBlocked := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "--") && strings.Contains(line, "-A--") {
			// New entry - save previous
			if currentRuleID != "" {
				ruleCount[currentRuleID]++
				totalRequests++
				if isBlocked {
					blockedRequests++
				}
			}
			if currentIP != "" {
				ipCount[currentIP]++
			}
			currentIP = ""
			currentRuleID = ""
			isBlocked = false
		}

		// Extract IP
		if strings.Count(line, ".") == 3 && currentIP == "" {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Count(part, ".") == 3 && !strings.Contains(part, "/") {
					currentIP = part
					break
				}
			}
		}

		// Extract rule ID
		if strings.Contains(line, "id \"") {
			re := regexp.MustCompile(`id "(\d+)"`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				currentRuleID = matches[1]
			}
		}

		// Check if blocked
		if strings.Contains(line, "Action: Intercepted") {
			isBlocked = true
		}
	}

	// Convert maps to sorted slices
	type countItem struct {
		Key   string
		Count int
	}

	var topRules []fiber.Map
	for id, count := range ruleCount {
		topRules = append(topRules, fiber.Map{"rule_id": id, "count": count})
	}
	// Sort by count (simple bubble sort for small data)
	for i := 0; i < len(topRules)-1; i++ {
		for j := 0; j < len(topRules)-i-1; j++ {
			if topRules[j]["count"].(int) < topRules[j+1]["count"].(int) {
				topRules[j], topRules[j+1] = topRules[j+1], topRules[j]
			}
		}
	}
	if len(topRules) > 10 {
		topRules = topRules[:10]
	}

	var topIPs []fiber.Map
	for ip, count := range ipCount {
		topIPs = append(topIPs, fiber.Map{"ip": ip, "count": count})
	}
	for i := 0; i < len(topIPs)-1; i++ {
		for j := 0; j < len(topIPs)-i-1; j++ {
			if topIPs[j]["count"].(int) < topIPs[j+1]["count"].(int) {
				topIPs[j], topIPs[j+1] = topIPs[j+1], topIPs[j]
			}
		}
	}
	if len(topIPs) > 10 {
		topIPs = topIPs[:10]
	}

	stats["total_requests"] = totalRequests
	stats["blocked_requests"] = blockedRequests
	stats["logged_requests"] = totalRequests - blockedRequests
	stats["top_rules"] = topRules
	stats["top_ips"] = topIPs

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// ClearModSecurityAuditLog clears the audit log (admin only)
func (h *Handler) ClearModSecurityAuditLog(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	logPath := "/var/log/modsecurity/modsec_audit.log"

	// Backup old log
	backupPath := logPath + "." + time.Now().Format("20060102-150405")
	if err := os.Rename(logPath, backupPath); err != nil {
		// If rename fails, just truncate
		if err := os.Truncate(logPath, 0); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Log dosyası temizlenemedi"})
		}
	} else {
		// Create new empty log file
		if _, err := os.Create(logPath); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Yeni log dosyası oluşturulamadı"})
		}
		os.Chown(logPath, 33, 33) // www-data
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Audit log temizlendi",
	})
}

// AddModSecurityWhitelist adds an IP to ModSecurity whitelist (admin only)
func (h *Handler) AddModSecurityWhitelist(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	var input struct {
		IP      string `json:"ip"`
		Comment string `json:"comment"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	if input.IP == "" {
		return c.Status(400).JSON(fiber.Map{"error": "IP adresi gerekli"})
	}

	// Create whitelist file if not exists
	whitelistPath := "/etc/modsecurity/whitelist.conf"

	// Check if IP already exists
	if content, err := os.ReadFile(whitelistPath); err == nil {
		if strings.Contains(string(content), input.IP) {
			return c.Status(400).JSON(fiber.Map{"error": "Bu IP zaten whitelist'te"})
		}
	}

	// Add IP to whitelist
	f, err := os.OpenFile(whitelistPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Whitelist dosyası açılamadı"})
	}
	defer f.Close()

	comment := input.Comment
	if comment == "" {
		comment = "Added via panel"
	}

	rule := "\n# " + comment + " - " + time.Now().Format("2006-01-02 15:04:05") + "\n"
	rule += "SecRule REMOTE_ADDR \"@ipMatch " + input.IP + "\" \"id:1000000,phase:1,allow,nolog\"\n"

	if _, err := f.WriteString(rule); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Whitelist yazılamadı"})
	}

	// Restart Apache
	exec.Command("systemctl", "restart", "apache2").Run()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "IP whitelist'e eklendi: " + input.IP,
	})
}

// GetModSecurityWhitelist returns whitelisted IPs (admin only)
func (h *Handler) GetModSecurityWhitelist(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	whitelistPath := "/etc/modsecurity/whitelist.conf"

	content, err := os.ReadFile(whitelistPath)
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []fiber.Map{},
		})
	}

	var whitelist []fiber.Map
	lines := strings.Split(string(content), "\n")

	var currentComment string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") {
			currentComment = strings.TrimPrefix(line, "# ")
		} else if strings.Contains(line, "@ipMatch") {
			// Extract IP
			re := regexp.MustCompile(`@ipMatch\s+([^\s"]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				whitelist = append(whitelist, fiber.Map{
					"ip":      matches[1],
					"comment": currentComment,
				})
			}
			currentComment = ""
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    whitelist,
	})
}

// RemoveModSecurityWhitelist removes an IP from whitelist (admin only)
func (h *Handler) RemoveModSecurityWhitelist(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Yetkiniz yok"})
	}

	var input struct {
		IP string `json:"ip"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz veri"})
	}

	whitelistPath := "/etc/modsecurity/whitelist.conf"

	content, err := os.ReadFile(whitelistPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Whitelist dosyası okunamadı"})
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		if skipNext {
			skipNext = false
			continue
		}

		if strings.Contains(line, "@ipMatch "+input.IP) {
			// Skip this line and the comment before it
			if len(newLines) > 0 && strings.HasPrefix(newLines[len(newLines)-1], "#") {
				newLines = newLines[:len(newLines)-1]
			}
			continue
		}

		newLines = append(newLines, line)
	}

	if err := os.WriteFile(whitelistPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Whitelist yazılamadı"})
	}

	// Restart Apache
	exec.Command("systemctl", "restart", "apache2").Run()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "IP whitelist'ten kaldırıldı: " + input.IP,
	})
}
