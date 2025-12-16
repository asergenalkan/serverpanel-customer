package api

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// NodejsApp represents a Node.js application
type NodejsApp struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	DomainID    *int64 `json:"domain_id"`
	Name        string `json:"name"`
	AppRoot     string `json:"app_root"`
	StartupFile string `json:"startup_file"`
	NodeVersion string `json:"node_version"`
	Port        int    `json:"port"`
	AppURL      string `json:"app_url"`
	Mode        string `json:"mode"`
	Environment string `json:"environment"`
	AutoRestart bool   `json:"auto_restart"`
	Status      string `json:"status"`
	PM2ID       *int   `json:"pm2_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	// Extra fields for display
	DomainName string `json:"domain_name,omitempty"`
	Username   string `json:"username,omitempty"`
}

// ListNodejsApps returns all Node.js apps for the user
func (h *Handler) ListNodejsApps(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == "admin" {
		query = `
			SELECT na.id, na.user_id, na.domain_id, na.name, na.app_root, na.startup_file,
				   na.node_version, na.port, na.app_url, na.mode, na.environment,
				   na.auto_restart, na.status, na.pm2_id, na.created_at, na.updated_at,
				   COALESCE(d.name, '') as domain_name, u.username
			FROM nodejs_apps na
			LEFT JOIN domains d ON na.domain_id = d.id
			LEFT JOIN users u ON na.user_id = u.id
			ORDER BY na.created_at DESC
		`
	} else {
		query = `
			SELECT na.id, na.user_id, na.domain_id, na.name, na.app_root, na.startup_file,
				   na.node_version, na.port, na.app_url, na.mode, na.environment,
				   na.auto_restart, na.status, na.pm2_id, na.created_at, na.updated_at,
				   COALESCE(d.name, '') as domain_name, u.username
			FROM nodejs_apps na
			LEFT JOIN domains d ON na.domain_id = d.id
			LEFT JOIN users u ON na.user_id = u.id
			WHERE na.user_id = ?
			ORDER BY na.created_at DESC
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulamalar alınamadı",
		})
	}
	defer rows.Close()

	apps := []NodejsApp{}
	for rows.Next() {
		var app NodejsApp
		var autoRestart int
		err := rows.Scan(
			&app.ID, &app.UserID, &app.DomainID, &app.Name, &app.AppRoot, &app.StartupFile,
			&app.NodeVersion, &app.Port, &app.AppURL, &app.Mode, &app.Environment,
			&autoRestart, &app.Status, &app.PM2ID, &app.CreatedAt, &app.UpdatedAt,
			&app.DomainName, &app.Username,
		)
		if err != nil {
			continue
		}
		app.AutoRestart = autoRestart == 1

		// Get real-time status from PM2
		if app.PM2ID != nil {
			app.Status = h.getPM2AppStatus(*app.PM2ID)
		}

		apps = append(apps, app)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    apps,
	})
}

// CreateNodejsApp creates a new Node.js application
func (h *Handler) CreateNodejsApp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		Name        string `json:"name"`
		DomainID    *int64 `json:"domain_id"`
		AppRoot     string `json:"app_root"`
		StartupFile string `json:"startup_file"`
		NodeVersion string `json:"node_version"`
		AppURL      string `json:"app_url"`
		Mode        string `json:"mode"`
		Environment string `json:"environment"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate name
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(req.Name) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz uygulama adı (sadece harf, rakam, - ve _ kullanılabilir)",
		})
	}

	// Get username for path validation
	var username string
	h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)

	// Validate app root
	homeDir := "/home/" + username
	if role != "admin" && !strings.HasPrefix(req.AppRoot, homeDir) {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama dizini home dizininiz içinde olmalıdır",
		})
	}

	// Set defaults
	if req.StartupFile == "" {
		req.StartupFile = "app.js"
	}
	if req.NodeVersion == "" {
		req.NodeVersion = "lts"
	}
	if req.Mode == "" {
		req.Mode = "production"
	}

	// Find available port
	port := h.findAvailablePort()

	// Create app directory if not exists
	if _, err := os.Stat(req.AppRoot); os.IsNotExist(err) {
		if err := os.MkdirAll(req.AppRoot, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Uygulama dizini oluşturulamadı: " + err.Error(),
			})
		}
		// Set ownership to user
		exec.Command("chown", "-R", username+":"+username, req.AppRoot).Run()
	}

	// Create starter app.js if it doesn't exist
	startupPath := filepath.Join(req.AppRoot, req.StartupFile)
	if _, err := os.Stat(startupPath); os.IsNotExist(err) {
		starterApp := fmt.Sprintf(`// Node.js Starter Application
// Node.js Sürümü: %s
// Port: %d (otomatik atandı)
// Oluşturulma: ServerPanel tarafından otomatik oluşturuldu

const http = require('http');

const PORT = process.env.PORT || %d;

const html = '<html><head><title>Node.js</title>' +
  '<style>body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%)}' +
  '.container{background:white;padding:40px 60px;border-radius:16px;text-align:center;box-shadow:0 20px 60px rgba(0,0,0,0.3)}' +
  'h1{color:#333;margin:0 0 10px 0}.status{color:#22c55e;font-size:24px;margin-bottom:20px}.info{color:#666;font-size:14px}' +
  '.version{background:#f0f0f0;padding:10px 20px;border-radius:8px;margin-top:20px;font-family:monospace}</style></head>' +
  '<body><div class="container"><div class="status">✓ Çalışıyor</div><h1>Node.js Uygulaması</h1>' +
  '<p class="info">Uygulama başarıyla çalışıyor!</p>' +
  '<div class="version">Node.js: ' + process.version + '<br>Port: ' + PORT + '<br>Mode: %s</div>' +
  '</div></body></html>';

const server = http.createServer((req, res) => {
  res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
  res.end(html);
});

server.listen(PORT, () => {
  console.log('Server running at http://localhost:' + PORT);
  console.log('Node.js Version: ' + process.version);
});
`, req.NodeVersion, port, port, req.Mode)

		if err := os.WriteFile(startupPath, []byte(starterApp), 0644); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Başlangıç dosyası oluşturulamadı: " + err.Error(),
			})
		}
		// Set ownership
		exec.Command("chown", username+":"+username, startupPath).Run()
	}

	// Create package.json if it doesn't exist
	packagePath := filepath.Join(req.AppRoot, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Node.js application managed by ServerPanel",
  "main": "%s",
  "scripts": {
    "start": "node %s"
  },
  "engines": {
    "node": ">=%s"
  }
}
`, req.Name, req.StartupFile, req.StartupFile, req.NodeVersion)

		os.WriteFile(packagePath, []byte(packageJSON), 0644)
		exec.Command("chown", username+":"+username, packagePath).Run()
	}

	// Insert into database
	result, err := h.db.Exec(`
		INSERT INTO nodejs_apps (user_id, domain_id, name, app_root, startup_file, node_version, port, app_url, mode, environment)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.DomainID, req.Name, req.AppRoot, req.StartupFile, req.NodeVersion, port, req.AppURL, req.Mode, req.Environment)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama oluşturulamadı: " + err.Error(),
		})
	}

	appID, _ := result.LastInsertId()

	// Create Apache proxy config if domain/subdomain URL is specified
	if req.AppURL != "" {
		h.createApacheProxyConfig(appID, port, req.AppURL)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Uygulama oluşturuldu",
		Data:    map[string]interface{}{"id": appID, "port": port},
	})
}

// StartNodejsApp starts a Node.js application
func (h *Handler) StartNodejsApp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var app NodejsApp
	var autoRestart int
	query := "SELECT id, user_id, name, app_root, startup_file, node_version, port, mode, environment, auto_restart FROM nodejs_apps WHERE id = ?"
	err := h.db.QueryRow(query, appID).Scan(&app.ID, &app.UserID, &app.Name, &app.AppRoot, &app.StartupFile, &app.NodeVersion, &app.Port, &app.Mode, &app.Environment, &autoRestart)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}
	app.AutoRestart = autoRestart == 1

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamayı başlatma yetkiniz yok",
		})
	}

	// Build PM2 start command
	envVars := fmt.Sprintf("PORT=%d NODE_ENV=%s", app.Port, app.Mode)
	if app.Environment != "" {
		envVars += " " + app.Environment
	}

	pm2Name := fmt.Sprintf("app-%d-%s", app.ID, app.Name)
	startupPath := filepath.Join(app.AppRoot, app.StartupFile)

	startCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		cd %s
		%s pm2 start %s --name %s --cwd %s
	`, app.AppRoot, envVars, startupPath, pm2Name, app.AppRoot)

	cmd := exec.Command("bash", "-c", startCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Uygulama başlatılamadı: %s", string(output)),
		})
	}

	// Get PM2 ID and update database
	pm2ID := h.getPM2AppID(pm2Name)
	h.db.Exec("UPDATE nodejs_apps SET status = 'running', pm2_id = ? WHERE id = ?", pm2ID, appID)

	// Save PM2 process list
	exec.Command("bash", "-c", `export HOME=/root && export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && pm2 save`).Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Uygulama başlatıldı",
	})
}

// StopNodejsApp stops a Node.js application
func (h *Handler) StopNodejsApp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var app NodejsApp
	err := h.db.QueryRow("SELECT id, user_id, name, pm2_id FROM nodejs_apps WHERE id = ?", appID).Scan(&app.ID, &app.UserID, &app.Name, &app.PM2ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamayı durdurma yetkiniz yok",
		})
	}

	pm2Name := fmt.Sprintf("app-%d-%s", app.ID, app.Name)
	stopCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 stop %s 2>/dev/null || true
	`, pm2Name)

	cmd := exec.Command("bash", "-c", stopCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	cmd.Run()

	h.db.Exec("UPDATE nodejs_apps SET status = 'stopped' WHERE id = ?", appID)

	// Save PM2 process list (so resurrect knows this app should be stopped)
	exec.Command("bash", "-c", `export HOME=/root && export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && pm2 save`).Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Uygulama durduruldu",
	})
}

// RestartNodejsApp restarts a Node.js application
func (h *Handler) RestartNodejsApp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var app NodejsApp
	err := h.db.QueryRow("SELECT id, user_id, name FROM nodejs_apps WHERE id = ?", appID).Scan(&app.ID, &app.UserID, &app.Name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamayı yeniden başlatma yetkiniz yok",
		})
	}

	pm2Name := fmt.Sprintf("app-%d-%s", app.ID, app.Name)
	restartCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 restart %s
	`, pm2Name)

	cmd := exec.Command("bash", "-c", restartCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Uygulama yeniden başlatılamadı: %s", string(output)),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Uygulama yeniden başlatıldı",
	})
}

// DeleteNodejsApp deletes a Node.js application
func (h *Handler) DeleteNodejsApp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var app NodejsApp
	err := h.db.QueryRow("SELECT id, user_id, name, port FROM nodejs_apps WHERE id = ?", appID).Scan(&app.ID, &app.UserID, &app.Name, &app.Port)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamayı silme yetkiniz yok",
		})
	}

	// Stop and delete from PM2
	pm2Name := fmt.Sprintf("app-%d-%s", app.ID, app.Name)
	deleteCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 delete %s 2>/dev/null || true
		pm2 save
	`, pm2Name)

	cmd := exec.Command("bash", "-c", deleteCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	cmd.Run()

	// Remove Apache proxy config
	h.removeApacheProxyConfig(app.ID)

	// Delete from database
	h.db.Exec("DELETE FROM nodejs_apps WHERE id = ?", appID)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Uygulama silindi",
	})
}

// GetNodejsAppLogs returns logs for a Node.js application
func (h *Handler) GetNodejsAppLogs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")
	lines := c.QueryInt("lines", 100)

	// Get app
	var app NodejsApp
	err := h.db.QueryRow("SELECT id, user_id, name FROM nodejs_apps WHERE id = ?", appID).Scan(&app.ID, &app.UserID, &app.Name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamanın loglarını görme yetkiniz yok",
		})
	}

	pm2Name := fmt.Sprintf("app-%d-%s", app.ID, app.Name)
	logsCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 logs %s --lines %d --nostream 2>&1
	`, pm2Name, lines)

	cmd := exec.Command("bash", "-c", logsCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, _ := cmd.CombinedOutput()

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    string(output),
	})
}

// RunNpmCommand runs npm commands for a Node.js application
func (h *Handler) RunNpmCommand(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	var req struct {
		Command string `json:"command"` // install, build, start, run <script>
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate command - only allow safe npm commands
	allowedCommands := map[string]bool{
		"install": true,
		"ci":      true,
		"build":   true,
		"start":   true,
		"test":    true,
		"audit":   true,
	}

	// Check if it's a "run <script>" command
	cmdParts := strings.Fields(req.Command)
	if len(cmdParts) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Komut belirtilmedi",
		})
	}

	baseCmd := cmdParts[0]
	if baseCmd == "run" && len(cmdParts) >= 2 {
		// Allow npm run <script>
		baseCmd = "run"
	}

	if !allowedCommands[baseCmd] && baseCmd != "run" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu komut izin verilmiyor: " + req.Command,
		})
	}

	// Get app
	var app NodejsApp
	err := h.db.QueryRow("SELECT id, user_id, app_root, node_version FROM nodejs_apps WHERE id = ?", appID).Scan(&app.ID, &app.UserID, &app.AppRoot, &app.NodeVersion)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamada komut çalıştırma yetkiniz yok",
		})
	}

	// Run npm command
	npmCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		cd %s
		npm %s 2>&1
	`, app.AppRoot, req.Command)

	cmd := exec.Command("bash", "-c", npmCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return c.JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Komut başarısız: %s", string(output)),
			Data:    string(output),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Komut başarıyla çalıştırıldı",
		Data:    string(output),
	})
}

// Helper functions

func (h *Handler) findAvailablePort() int {
	// Start from port 3001 and find the next available port
	var maxPort int
	h.db.QueryRow("SELECT COALESCE(MAX(port), 3000) FROM nodejs_apps").Scan(&maxPort)
	return maxPort + 1
}

func (h *Handler) getPM2AppStatus(pm2ID int) string {
	statusCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 show %d 2>/dev/null | grep "│ status" | head -1 | sed 's/.*│ //' | sed 's/ *│.*//' | tr -d ' '
	`, pm2ID)

	cmd := exec.Command("bash", "-c", statusCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	status := strings.TrimSpace(string(output))
	if status == "" {
		return "stopped"
	}
	return status
}

func (h *Handler) getPM2AppID(name string) int {
	idCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 jlist 2>/dev/null | grep -o '"name":"%s"[^}]*"pm_id":[0-9]*' | grep -o '"pm_id":[0-9]*' | cut -d':' -f2
	`, name)

	cmd := exec.Command("bash", "-c", idCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	id, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return id
}

func (h *Handler) createApacheProxyConfig(appID int64, port int, appURL string) {
	// appURL is the domain/subdomain name (e.g., "demonode.sergenalkan.com")
	domain := strings.TrimSpace(appURL)
	if domain == "" {
		return
	}

	// Check if vhost exists for this domain
	vhostPath := fmt.Sprintf("/etc/apache2/sites-available/%s.conf", domain)
	if _, err := os.Stat(vhostPath); os.IsNotExist(err) {
		return
	}

	// Create Node.js optimized vhost (replaces PHP vhost)
	vhostContent := fmt.Sprintf(`# Node.js Application - App ID: %d
# Auto-generated by ServerPanel
<VirtualHost *:80>
    ServerName %s
    
    # Node.js Reverse Proxy
    ProxyPreserveHost On
    ProxyPass / http://127.0.0.1:%d/
    ProxyPassReverse / http://127.0.0.1:%d/
    
    # WebSocket support
    RewriteEngine On
    RewriteCond %%{HTTP:Upgrade} websocket [NC]
    RewriteCond %%{HTTP:Connection} upgrade [NC]
    RewriteRule ^/?(.*) "ws://127.0.0.1:%d/$1" [P,L]
    
    ErrorLog ${APACHE_LOG_DIR}/%s-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-access.log combined
</VirtualHost>
`, appID, domain, port, port, port, domain, domain)

	// Backup original vhost
	backupPath := vhostPath + ".php-backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		content, _ := os.ReadFile(vhostPath)
		os.WriteFile(backupPath, content, 0644)
	}

	// Write new Node.js vhost
	os.WriteFile(vhostPath, []byte(vhostContent), 0644)

	// Check for SSL vhost and update it too
	sslVhostPath := fmt.Sprintf("/etc/apache2/sites-available/%s-ssl.conf", domain)
	if _, err := os.Stat(sslVhostPath); err == nil {
		// Read SSL cert paths from existing config
		sslContent, _ := os.ReadFile(sslVhostPath)
		sslContentStr := string(sslContent)

		// Extract SSL certificate paths
		certFile := "/etc/letsencrypt/live/" + domain + "/fullchain.pem"
		keyFile := "/etc/letsencrypt/live/" + domain + "/privkey.pem"

		// Check if cert paths exist in original
		if strings.Contains(sslContentStr, "SSLCertificateFile") {
			// Create SSL Node.js vhost
			sslVhostContent := fmt.Sprintf(`# Node.js Application SSL - App ID: %d
# Auto-generated by ServerPanel
<VirtualHost *:443>
    ServerName %s
    
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    # Node.js Reverse Proxy
    ProxyPreserveHost On
    ProxyPass / http://127.0.0.1:%d/
    ProxyPassReverse / http://127.0.0.1:%d/
    
    # WebSocket support
    RewriteEngine On
    RewriteCond %%{HTTP:Upgrade} websocket [NC]
    RewriteCond %%{HTTP:Connection} upgrade [NC]
    RewriteRule ^/?(.*) "ws://127.0.0.1:%d/$1" [P,L]
    
    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, appID, domain, certFile, keyFile, port, port, port, domain, domain)

			// Backup original SSL vhost
			sslBackupPath := sslVhostPath + ".php-backup"
			if _, err := os.Stat(sslBackupPath); os.IsNotExist(err) {
				os.WriteFile(sslBackupPath, sslContent, 0644)
			}

			os.WriteFile(sslVhostPath, []byte(sslVhostContent), 0644)
		}
	}

	// Reload Apache
	exec.Command("systemctl", "reload", "apache2").Run()
}

func (h *Handler) removeApacheProxyConfig(appID int64) {
	// Get app URL from database to find the vhost
	var appURL string
	h.db.QueryRow("SELECT app_url FROM nodejs_apps WHERE id = ?", appID).Scan(&appURL)

	if appURL != "" {
		domain := strings.TrimSpace(appURL)
		vhostPath := fmt.Sprintf("/etc/apache2/sites-available/%s.conf", domain)
		backupPath := vhostPath + ".php-backup"

		// Restore original PHP vhost if backup exists
		if _, err := os.Stat(backupPath); err == nil {
			content, _ := os.ReadFile(backupPath)
			os.WriteFile(vhostPath, content, 0644)
			os.Remove(backupPath)
		}

		// Restore SSL vhost too
		sslVhostPath := fmt.Sprintf("/etc/apache2/sites-available/%s-ssl.conf", domain)
		sslBackupPath := sslVhostPath + ".php-backup"
		if _, err := os.Stat(sslBackupPath); err == nil {
			content, _ := os.ReadFile(sslBackupPath)
			os.WriteFile(sslVhostPath, content, 0644)
			os.Remove(sslBackupPath)
		}
	}

	// Remove old style conf file if exists
	configPath := fmt.Sprintf("/etc/apache2/conf-available/nodejs-app-%d.conf", appID)
	exec.Command("a2disconf", fmt.Sprintf("nodejs-app-%d", appID)).Run()
	os.Remove(configPath)

	exec.Command("systemctl", "reload", "apache2").Run()
}

// UpdateNodejsApp updates a Node.js application
func (h *Handler) UpdateNodejsApp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var existingApp NodejsApp
	err := h.db.QueryRow("SELECT id, user_id FROM nodejs_apps WHERE id = ?", appID).Scan(&existingApp.ID, &existingApp.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && existingApp.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamayı düzenleme yetkiniz yok",
		})
	}

	var req struct {
		StartupFile string `json:"startup_file"`
		NodeVersion string `json:"node_version"`
		Mode        string `json:"mode"`
		Environment string `json:"environment"`
		AutoRestart bool   `json:"auto_restart"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	autoRestartInt := 0
	if req.AutoRestart {
		autoRestartInt = 1
	}

	_, err = h.db.Exec(`
		UPDATE nodejs_apps 
		SET startup_file = ?, node_version = ?, mode = ?, environment = ?, auto_restart = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, req.StartupFile, req.NodeVersion, req.Mode, req.Environment, autoRestartInt, appID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama güncellenemedi",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Uygulama güncellendi",
	})
}

// GetNodejsAppEnv returns environment variables for a Node.js application
func (h *Handler) GetNodejsAppEnv(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var app NodejsApp
	err := h.db.QueryRow("SELECT id, user_id, environment FROM nodejs_apps WHERE id = ?", appID).Scan(&app.ID, &app.UserID, &app.Environment)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && app.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamanın ortam değişkenlerini görme yetkiniz yok",
		})
	}

	// Parse environment string to key-value pairs
	envVars := []map[string]string{}
	if app.Environment != "" {
		var envMap map[string]string
		if err := json.Unmarshal([]byte(app.Environment), &envMap); err == nil {
			for k, v := range envMap {
				envVars = append(envVars, map[string]string{"key": k, "value": v})
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    envVars,
	})
}

// UpdateNodejsAppEnv updates environment variables for a Node.js application
func (h *Handler) UpdateNodejsAppEnv(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	appID := c.Params("id")

	// Get app
	var existingApp NodejsApp
	err := h.db.QueryRow("SELECT id, user_id FROM nodejs_apps WHERE id = ?", appID).Scan(&existingApp.ID, &existingApp.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Uygulama bulunamadı",
		})
	}

	// Check permission
	if role != "admin" && existingApp.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu uygulamanın ortam değişkenlerini düzenleme yetkiniz yok",
		})
	}

	var req struct {
		Environment map[string]string `json:"environment"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	envJSON, _ := json.Marshal(req.Environment)

	_, err = h.db.Exec("UPDATE nodejs_apps SET environment = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", string(envJSON), appID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Ortam değişkenleri güncellenemedi",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Ortam değişkenleri güncellendi",
	})
}
