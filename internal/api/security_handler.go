package api

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ==================== FAIL2BAN ====================

// Fail2banStatus represents fail2ban service status
type Fail2banStatus struct {
	Running     bool       `json:"running"`
	Version     string     `json:"version"`
	Jails       []JailInfo `json:"jails"`
	TotalBanned int        `json:"total_banned"`
}

// JailInfo represents a fail2ban jail
type JailInfo struct {
	Name            string   `json:"name"`
	Enabled         bool     `json:"enabled"`
	CurrentlyBanned int      `json:"currently_banned"`
	TotalBanned     int      `json:"total_banned"`
	BannedIPs       []string `json:"banned_ips"`
	Filter          string   `json:"filter"`
	MaxRetry        int      `json:"max_retry"`
	BanTime         int      `json:"ban_time"`
	FindTime        int      `json:"find_time"`
}

// GetFail2banStatus returns fail2ban status and jail information
func (h *Handler) GetFail2banStatus(c *fiber.Ctx) error {
	status := Fail2banStatus{
		Running: false,
		Jails:   []JailInfo{},
	}

	// Check if fail2ban is running
	cmd := exec.Command("systemctl", "is-active", "fail2ban")
	if err := cmd.Run(); err == nil {
		status.Running = true
	}

	// Get version
	cmd = exec.Command("fail2ban-client", "--version")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			status.Version = strings.TrimSpace(lines[0])
		}
	}

	if !status.Running {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    status,
		})
	}

	// Get jail list
	cmd = exec.Command("fail2ban-client", "status")
	output, err := cmd.Output()
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    status,
		})
	}

	// Parse jail list
	jailLine := ""
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "Jail list:") {
			jailLine = strings.TrimSpace(strings.Split(line, ":")[1])
			break
		}
	}

	if jailLine == "" {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    status,
		})
	}

	jailNames := strings.Split(jailLine, ",")
	for _, jailName := range jailNames {
		jailName = strings.TrimSpace(jailName)
		if jailName == "" {
			continue
		}

		jail := JailInfo{
			Name:      jailName,
			Enabled:   true,
			BannedIPs: []string{},
		}

		// Get jail status
		cmd = exec.Command("fail2ban-client", "status", jailName)
		jailOutput, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(jailOutput), "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "Currently banned:") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						jail.CurrentlyBanned, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
					}
				} else if strings.Contains(line, "Total banned:") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						jail.TotalBanned, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
					}
				} else if strings.Contains(line, "Banned IP list:") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						ips := strings.TrimSpace(parts[1])
						if ips != "" {
							jail.BannedIPs = strings.Fields(ips)
						}
					}
				}
			}
		}

		// Get jail config
		cmd = exec.Command("fail2ban-client", "get", jailName, "maxretry")
		if out, err := cmd.Output(); err == nil {
			jail.MaxRetry, _ = strconv.Atoi(strings.TrimSpace(string(out)))
		}

		cmd = exec.Command("fail2ban-client", "get", jailName, "bantime")
		if out, err := cmd.Output(); err == nil {
			jail.BanTime, _ = strconv.Atoi(strings.TrimSpace(string(out)))
		}

		cmd = exec.Command("fail2ban-client", "get", jailName, "findtime")
		if out, err := cmd.Output(); err == nil {
			jail.FindTime, _ = strconv.Atoi(strings.TrimSpace(string(out)))
		}

		status.Jails = append(status.Jails, jail)
		status.TotalBanned += jail.CurrentlyBanned
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    status,
	})
}

// BanIP bans an IP address in a specific jail
func (h *Handler) BanIP(c *fiber.Ctx) error {
	var req struct {
		Jail string `json:"jail"`
		IP   string `json:"ip"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	// Validate IP
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipRegex.MatchString(req.IP) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz IP adresi",
		})
	}

	cmd := exec.Command("fail2ban-client", "set", req.Jail, "banip", req.IP)
	if err := cmd.Run(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "IP engellenemedi: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("%s IP adresi %s jail'inde engellendi", req.IP, req.Jail),
	})
}

// UnbanIP unbans an IP address from a specific jail
func (h *Handler) UnbanIP(c *fiber.Ctx) error {
	var req struct {
		Jail string `json:"jail"`
		IP   string `json:"ip"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	cmd := exec.Command("fail2ban-client", "set", req.Jail, "unbanip", req.IP)
	if err := cmd.Run(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "IP engeli kaldırılamadı: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("%s IP adresi %s jail'inden kaldırıldı", req.IP, req.Jail),
	})
}

// UpdateJailSettings updates jail settings
func (h *Handler) UpdateJailSettings(c *fiber.Ctx) error {
	var req struct {
		Jail     string `json:"jail"`
		MaxRetry int    `json:"max_retry"`
		BanTime  int    `json:"ban_time"`
		FindTime int    `json:"find_time"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	// Update settings
	if req.MaxRetry > 0 {
		exec.Command("fail2ban-client", "set", req.Jail, "maxretry", strconv.Itoa(req.MaxRetry)).Run()
	}
	if req.BanTime > 0 {
		exec.Command("fail2ban-client", "set", req.Jail, "bantime", strconv.Itoa(req.BanTime)).Run()
	}
	if req.FindTime > 0 {
		exec.Command("fail2ban-client", "set", req.Jail, "findtime", strconv.Itoa(req.FindTime)).Run()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Jail ayarları güncellendi",
	})
}

// ToggleFail2ban starts or stops fail2ban service
func (h *Handler) ToggleFail2ban(c *fiber.Ctx) error {
	var req struct {
		Action string `json:"action"` // start, stop, restart
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	var cmd *exec.Cmd
	switch req.Action {
	case "start":
		cmd = exec.Command("systemctl", "start", "fail2ban")
	case "stop":
		cmd = exec.Command("systemctl", "stop", "fail2ban")
	case "restart":
		cmd = exec.Command("systemctl", "restart", "fail2ban")
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz aksiyon",
		})
	}

	if err := cmd.Run(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "İşlem başarısız: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Fail2ban " + req.Action + " edildi",
	})
}

// GetFail2banWhitelist returns whitelisted IPs
func (h *Handler) GetFail2banWhitelist(c *fiber.Ctx) error {
	whitelist := []string{}

	// Read from jail.local
	file, err := os.Open("/etc/fail2ban/jail.local")
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    whitelist,
		})
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ignoreip") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) > 1 {
				ips := strings.Fields(strings.TrimSpace(parts[1]))
				whitelist = append(whitelist, ips...)
			}
			break
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    whitelist,
	})
}

// UpdateFail2banWhitelist updates whitelisted IPs
func (h *Handler) UpdateFail2banWhitelist(c *fiber.Ctx) error {
	var req struct {
		IPs []string `json:"ips"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	// Read current jail.local
	content, err := os.ReadFile("/etc/fail2ban/jail.local")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Yapılandırma dosyası okunamadı",
		})
	}

	// Update ignoreip line
	lines := strings.Split(string(content), "\n")
	newLines := []string{}
	ignoreipFound := false

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "ignoreip") {
			newLines = append(newLines, "ignoreip = "+strings.Join(req.IPs, " "))
			ignoreipFound = true
		} else {
			newLines = append(newLines, line)
		}
	}

	if !ignoreipFound {
		// Add after [DEFAULT] section
		for i, line := range newLines {
			if strings.TrimSpace(line) == "[DEFAULT]" {
				newLines = append(newLines[:i+1], append([]string{"ignoreip = " + strings.Join(req.IPs, " ")}, newLines[i+1:]...)...)
				break
			}
		}
	}

	// Write back
	if err := os.WriteFile("/etc/fail2ban/jail.local", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Yapılandırma dosyası yazılamadı",
		})
	}

	// Reload fail2ban
	exec.Command("fail2ban-client", "reload").Run()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Whitelist güncellendi",
	})
}

// ==================== UFW FIREWALL ====================

// FirewallStatus represents UFW status
type FirewallStatus struct {
	Active  bool           `json:"active"`
	Rules   []FirewallRule `json:"rules"`
	Default struct {
		Incoming string `json:"incoming"`
		Outgoing string `json:"outgoing"`
	} `json:"default"`
}

// FirewallRule represents a UFW rule
type FirewallRule struct {
	ID        int    `json:"id"`
	To        string `json:"to"`
	Action    string `json:"action"`
	From      string `json:"from"`
	Direction string `json:"direction"`
}

// GetFirewallStatus returns UFW status and rules
func (h *Handler) GetFirewallStatus(c *fiber.Ctx) error {
	status := FirewallStatus{
		Active: false,
		Rules:  []FirewallRule{},
	}

	// Check if UFW is active
	cmd := exec.Command("ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    status,
		})
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "Status: active") {
		status.Active = true
	}

	// Get numbered rules
	cmd = exec.Command("ufw", "status", "numbered")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		ruleRegex := regexp.MustCompile(`\[\s*(\d+)\]\s+(.+?)\s+(ALLOW|DENY|REJECT|LIMIT)\s+(IN|OUT)?\s*(.*)`)

		for _, line := range lines {
			matches := ruleRegex.FindStringSubmatch(line)
			if len(matches) >= 4 {
				id, _ := strconv.Atoi(matches[1])
				rule := FirewallRule{
					ID:     id,
					To:     strings.TrimSpace(matches[2]),
					Action: matches[3],
				}
				if len(matches) >= 5 {
					rule.Direction = matches[4]
				}
				if len(matches) >= 6 {
					rule.From = strings.TrimSpace(matches[5])
				}
				status.Rules = append(status.Rules, rule)
			}
		}
	}

	// Get default policies
	cmd = exec.Command("ufw", "status", "verbose")
	output, err = cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, "Default:") {
				if strings.Contains(line, "incoming") {
					if strings.Contains(line, "deny") {
						status.Default.Incoming = "deny"
					} else if strings.Contains(line, "allow") {
						status.Default.Incoming = "allow"
					}
				}
				if strings.Contains(line, "outgoing") {
					if strings.Contains(line, "deny") {
						status.Default.Outgoing = "deny"
					} else if strings.Contains(line, "allow") {
						status.Default.Outgoing = "allow"
					}
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    status,
	})
}

// AddFirewallRule adds a new UFW rule
func (h *Handler) AddFirewallRule(c *fiber.Ctx) error {
	var req struct {
		Port     string `json:"port"`
		Protocol string `json:"protocol"` // tcp, udp, or empty for both
		Action   string `json:"action"`   // allow, deny, limit
		From     string `json:"from"`     // IP or "any"
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	// Build command
	args := []string{}

	if req.Action == "" {
		req.Action = "allow"
	}
	args = append(args, req.Action)

	if req.From != "" && req.From != "any" {
		args = append(args, "from", req.From)
	}

	if req.Protocol != "" {
		args = append(args, "proto", req.Protocol)
	}

	args = append(args, "to", "any", "port", req.Port)

	cmd := exec.Command("ufw", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Kural eklenemedi: " + string(output),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Firewall kuralı eklendi",
	})
}

// DeleteFirewallRule deletes a UFW rule by number
func (h *Handler) DeleteFirewallRule(c *fiber.Ctx) error {
	ruleID := c.Params("id")

	cmd := exec.Command("ufw", "--force", "delete", ruleID)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Kural silinemedi: " + string(output),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Firewall kuralı silindi",
	})
}

// ToggleFirewall enables or disables UFW
func (h *Handler) ToggleFirewall(c *fiber.Ctx) error {
	var req struct {
		Enable bool `json:"enable"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	if req.Enable {
		// Etkinleştirmeden önce varsayılan portları aç
		defaultPorts := []string{
			"22/tcp",   // SSH
			"80/tcp",   // HTTP
			"443/tcp",  // HTTPS
			"8443/tcp", // ServerPanel
			"21/tcp",   // FTP
			"25/tcp",   // SMTP
			"465/tcp",  // SMTPS
			"587/tcp",  // Submission
			"110/tcp",  // POP3
			"995/tcp",  // POP3S
			"143/tcp",  // IMAP
			"993/tcp",  // IMAPS
			"53/tcp",   // DNS
			"53/udp",   // DNS
			"3306/tcp", // MySQL
		}

		// Varsayılan politikaları ayarla
		exec.Command("ufw", "default", "deny", "incoming").Run()
		exec.Command("ufw", "default", "allow", "outgoing").Run()

		// Varsayılan portları aç
		for _, port := range defaultPorts {
			exec.Command("ufw", "allow", port).Run()
		}

		// Passive FTP port aralığı
		exec.Command("ufw", "allow", "30000:31000/tcp").Run()

		// Şimdi etkinleştir
		cmd := exec.Command("ufw", "--force", "enable")
		if output, err := cmd.CombinedOutput(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Firewall etkinleştirilemedi: " + string(output),
			})
		}
	} else {
		cmd := exec.Command("ufw", "--force", "disable")
		if output, err := cmd.CombinedOutput(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Firewall devre dışı bırakılamadı: " + string(output),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Firewall durumu güncellendi",
	})
}

// ==================== SSH SECURITY ====================

// SSHConfig represents SSH configuration
type SSHConfig struct {
	Port                   int    `json:"port"`
	PermitRootLogin        string `json:"permit_root_login"`
	PasswordAuthentication string `json:"password_authentication"`
	PubkeyAuthentication   string `json:"pubkey_authentication"`
	MaxAuthTries           int    `json:"max_auth_tries"`
	LoginGraceTime         int    `json:"login_grace_time"`
}

// GetSSHConfig returns current SSH configuration
func (h *Handler) GetSSHConfig(c *fiber.Ctx) error {
	config := SSHConfig{
		Port:                   22,
		PermitRootLogin:        "yes",
		PasswordAuthentication: "yes",
		PubkeyAuthentication:   "yes",
		MaxAuthTries:           6,
		LoginGraceTime:         120,
	}

	file, err := os.Open("/etc/ssh/sshd_config")
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    config,
		})
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "port":
			config.Port, _ = strconv.Atoi(value)
		case "permitrootlogin":
			config.PermitRootLogin = value
		case "passwordauthentication":
			config.PasswordAuthentication = value
		case "pubkeyauthentication":
			config.PubkeyAuthentication = value
		case "maxauthtries":
			config.MaxAuthTries, _ = strconv.Atoi(value)
		case "logingracetime":
			config.LoginGraceTime, _ = strconv.Atoi(value)
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    config,
	})
}

// UpdateSSHConfig updates SSH configuration
func (h *Handler) UpdateSSHConfig(c *fiber.Ctx) error {
	var req SSHConfig

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	// Read current config
	content, err := os.ReadFile("/etc/ssh/sshd_config")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "SSH yapılandırması okunamadı",
		})
	}

	// Update settings
	settings := map[string]string{
		"Port":                   strconv.Itoa(req.Port),
		"PermitRootLogin":        req.PermitRootLogin,
		"PasswordAuthentication": req.PasswordAuthentication,
		"PubkeyAuthentication":   req.PubkeyAuthentication,
		"MaxAuthTries":           strconv.Itoa(req.MaxAuthTries),
		"LoginGraceTime":         strconv.Itoa(req.LoginGraceTime),
	}

	lines := strings.Split(string(content), "\n")
	newLines := []string{}
	foundSettings := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		modified := false

		for key, value := range settings {
			// Check both commented and uncommented lines
			if strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, "#"+key+" ") || strings.HasPrefix(trimmed, "# "+key+" ") {
				newLines = append(newLines, key+" "+value)
				foundSettings[key] = true
				modified = true
				break
			}
		}

		if !modified {
			newLines = append(newLines, line)
		}
	}

	// Add missing settings
	for key, value := range settings {
		if !foundSettings[key] {
			newLines = append(newLines, key+" "+value)
		}
	}

	// Write back
	if err := os.WriteFile("/etc/ssh/sshd_config", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "SSH yapılandırması yazılamadı",
		})
	}

	// If port changed, update UFW
	if req.Port != 22 {
		exec.Command("ufw", "allow", strconv.Itoa(req.Port)+"/tcp").Run()
	}

	// Restart SSH
	exec.Command("systemctl", "restart", "sshd").Run()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "SSH yapılandırması güncellendi",
	})
}

// ==================== SECURITY OVERVIEW ====================

// SecurityOverview represents overall security status
type SecurityOverview struct {
	Fail2ban struct {
		Installed bool `json:"installed"`
		Running   bool `json:"running"`
		Banned    int  `json:"banned"`
	} `json:"fail2ban"`
	Firewall struct {
		Installed bool `json:"installed"`
		Active    bool `json:"active"`
		Rules     int  `json:"rules"`
	} `json:"firewall"`
	SSH struct {
		Port      int    `json:"port"`
		RootLogin string `json:"root_login"`
	} `json:"ssh"`
	RecentAttacks []AttackInfo `json:"recent_attacks"`
}

// AttackInfo represents a recent attack
type AttackInfo struct {
	IP        string `json:"ip"`
	Service   string `json:"service"`
	Attempts  int    `json:"attempts"`
	Timestamp string `json:"timestamp"`
}

// GetSecurityOverview returns overall security status
func (h *Handler) GetSecurityOverview(c *fiber.Ctx) error {
	overview := SecurityOverview{}

	// Fail2ban status
	if _, err := exec.LookPath("fail2ban-client"); err == nil {
		overview.Fail2ban.Installed = true
		cmd := exec.Command("systemctl", "is-active", "fail2ban")
		if err := cmd.Run(); err == nil {
			overview.Fail2ban.Running = true
		}

		// Count banned IPs
		cmd = exec.Command("fail2ban-client", "status")
		if output, err := cmd.Output(); err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				if strings.Contains(line, "Jail list:") {
					jailLine := strings.TrimSpace(strings.Split(line, ":")[1])
					jails := strings.Split(jailLine, ",")
					for _, jail := range jails {
						jail = strings.TrimSpace(jail)
						if jail == "" {
							continue
						}
						cmd = exec.Command("fail2ban-client", "status", jail)
						if jailOutput, err := cmd.Output(); err == nil {
							for _, jailLine := range strings.Split(string(jailOutput), "\n") {
								if strings.Contains(jailLine, "Currently banned:") {
									parts := strings.Split(jailLine, ":")
									if len(parts) > 1 {
										count, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
										overview.Fail2ban.Banned += count
									}
								}
							}
						}
					}
					break
				}
			}
		}
	}

	// Firewall status
	if _, err := exec.LookPath("ufw"); err == nil {
		overview.Firewall.Installed = true
		cmd := exec.Command("ufw", "status")
		if output, err := cmd.Output(); err == nil {
			if strings.Contains(string(output), "Status: active") {
				overview.Firewall.Active = true
			}
		}

		cmd = exec.Command("ufw", "status", "numbered")
		if output, err := cmd.Output(); err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "[") {
					overview.Firewall.Rules++
				}
			}
		}
	}

	// SSH status
	file, err := os.Open("/etc/ssh/sshd_config")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				switch strings.ToLower(parts[0]) {
				case "port":
					overview.SSH.Port, _ = strconv.Atoi(parts[1])
				case "permitrootlogin":
					overview.SSH.RootLogin = parts[1]
				}
			}
		}
	}
	if overview.SSH.Port == 0 {
		overview.SSH.Port = 22
	}
	if overview.SSH.RootLogin == "" {
		overview.SSH.RootLogin = "yes"
	}

	// Recent attacks from auth.log
	overview.RecentAttacks = []AttackInfo{}
	authLog, err := os.Open("/var/log/auth.log")
	if err == nil {
		defer authLog.Close()
		scanner := bufio.NewScanner(authLog)
		attackMap := make(map[string]*AttackInfo)
		ipRegex := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Failed password") || strings.Contains(line, "Invalid user") {
				matches := ipRegex.FindStringSubmatch(line)
				if len(matches) > 0 {
					ip := matches[1]
					if ip == "127.0.0.1" {
						continue
					}
					if attack, exists := attackMap[ip]; exists {
						attack.Attempts++
					} else {
						attackMap[ip] = &AttackInfo{
							IP:       ip,
							Service:  "SSH",
							Attempts: 1,
						}
					}
				}
			}
		}

		// Get top 10 attackers
		for _, attack := range attackMap {
			if attack.Attempts >= 3 {
				overview.RecentAttacks = append(overview.RecentAttacks, *attack)
			}
		}

		// Sort by attempts (simple bubble sort for small list)
		for i := 0; i < len(overview.RecentAttacks); i++ {
			for j := i + 1; j < len(overview.RecentAttacks); j++ {
				if overview.RecentAttacks[j].Attempts > overview.RecentAttacks[i].Attempts {
					overview.RecentAttacks[i], overview.RecentAttacks[j] = overview.RecentAttacks[j], overview.RecentAttacks[i]
				}
			}
		}

		// Limit to 10
		if len(overview.RecentAttacks) > 10 {
			overview.RecentAttacks = overview.RecentAttacks[:10]
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    overview,
	})
}
