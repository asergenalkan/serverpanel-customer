package api

import (
	"bufio"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/database"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// TaskStatus represents the status of a running task
type TaskStatus struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // running, completed, failed
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Logs      []string  `json:"logs"`
}

// TaskManager manages running tasks
type TaskManager struct {
	tasks map[string]*TaskStatus
	mu    sync.RWMutex
	subs  map[string][]chan string // task_id -> subscribers
}

var taskManager = &TaskManager{
	tasks: make(map[string]*TaskStatus),
	subs:  make(map[string][]chan string),
}

// WebSocketUpgrade middleware for WebSocket connections
func WebSocketUpgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleTaskWebSocketDirect returns a WebSocket handler for direct use in main.go
func HandleTaskWebSocketDirect(db *database.DB, cfg *config.Config) func(*websocket.Conn) {
	h := &Handler{db: db, cfg: cfg}
	return h.HandleTaskWebSocket
}

// validateToken validates JWT token and returns claims
func (h *Handler) validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// HandleTaskWebSocket handles WebSocket connections for task logs
func (h *Handler) HandleTaskWebSocket(c *websocket.Conn) {
	// Verify token from query parameter
	token := c.Query("token")
	if token == "" {
		c.WriteJSON(map[string]string{"error": "token required"})
		c.Close()
		return
	}

	// Validate token and check admin role
	claims, err := h.validateToken(token)
	if err != nil || claims["role"] != "admin" {
		c.WriteJSON(map[string]string{"error": "unauthorized"})
		c.Close()
		return
	}

	taskID := c.Params("task_id")
	if taskID == "" {
		c.WriteJSON(map[string]string{"error": "task_id required"})
		c.Close()
		return
	}

	// Create subscriber channel
	logChan := make(chan string, 100)
	taskManager.subscribe(taskID, logChan)
	defer taskManager.unsubscribe(taskID, logChan)

	// Send existing logs
	taskManager.mu.RLock()
	task, exists := taskManager.tasks[taskID]
	if exists {
		for _, log := range task.Logs {
			c.WriteJSON(map[string]interface{}{
				"type": "log",
				"data": log,
			})
		}
		if task.Status != "running" {
			c.WriteJSON(map[string]interface{}{
				"type":   "status",
				"status": task.Status,
			})
		}
	}
	taskManager.mu.RUnlock()

	// Listen for new logs
	done := make(chan struct{})
	go func() {
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				close(done)
				return
			}
		}
	}()

	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				return
			}
			if err := c.WriteJSON(map[string]interface{}{
				"type": "log",
				"data": log,
			}); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (tm *TaskManager) subscribe(taskID string, ch chan string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.subs[taskID] = append(tm.subs[taskID], ch)
}

func (tm *TaskManager) unsubscribe(taskID string, ch chan string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	subs := tm.subs[taskID]
	for i, sub := range subs {
		if sub == ch {
			tm.subs[taskID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	close(ch)
}

func (tm *TaskManager) broadcast(taskID, message string) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, ch := range tm.subs[taskID] {
		select {
		case ch <- message:
		default:
		}
	}
}

func (tm *TaskManager) createTask(id, taskType, name string) *TaskStatus {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task := &TaskStatus{
		ID:        id,
		Type:      taskType,
		Name:      name,
		Status:    "running",
		StartTime: time.Now(),
		Logs:      []string{},
	}
	tm.tasks[id] = task
	return task
}

func (tm *TaskManager) addLog(taskID, log string) {
	tm.mu.Lock()
	if task, exists := tm.tasks[taskID]; exists {
		task.Logs = append(task.Logs, log)
	}
	tm.mu.Unlock()
	tm.broadcast(taskID, log)
}

func (tm *TaskManager) completeTask(taskID string, success bool) {
	tm.mu.Lock()
	if task, exists := tm.tasks[taskID]; exists {
		task.EndTime = time.Now()
		if success {
			task.Status = "completed"
		} else {
			task.Status = "failed"
		}
	}
	tm.mu.Unlock()

	status := "completed"
	if !success {
		status = "failed"
	}
	tm.broadcast(taskID, fmt.Sprintf("__STATUS__%s", status))
}

func (tm *TaskManager) getTask(taskID string) *TaskStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tasks[taskID]
}

// RunCommandWithLogs runs a command and streams output to task
func RunCommandWithLogs(taskID string, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			taskManager.addLog(taskID, scanner.Text())
		}
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			taskManager.addLog(taskID, scanner.Text())
		}
	}()

	return cmd.Wait()
}

// StartInstallTask starts an installation task with real-time logs
func (h *Handler) StartInstallTask(c *fiber.Ctx) error {
	var req struct {
		Type       string `json:"type"`   // php, extension, apache, software
		Action     string `json:"action"` // install, uninstall, enable, disable
		Target     string `json:"target"` // version, extension name, module name, package name
		PHPVersion string `json:"php_version,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Geçersiz istek",
		})
	}

	// Generate task ID
	taskID := fmt.Sprintf("%s-%s-%s-%d", req.Type, req.Action, req.Target, time.Now().UnixNano())

	// Create task
	taskName := getTaskName(req.Type, req.Action, req.Target)
	taskManager.createTask(taskID, req.Type, taskName)

	// Run task in background
	go func() {
		var success bool
		var err error

		taskManager.addLog(taskID, fmt.Sprintf("🚀 %s başlatılıyor...", taskName))
		taskManager.addLog(taskID, "")

		switch req.Type {
		case "php":
			if req.Action == "install" {
				success, err = h.installPHPWithLogs(taskID, req.Target)
			} else if req.Action == "uninstall" {
				success, err = h.uninstallPHPWithLogs(taskID, req.Target)
			}
		case "extension":
			if req.Action == "install" {
				success, err = h.installExtensionWithLogs(taskID, req.PHPVersion, req.Target)
			} else if req.Action == "uninstall" {
				success, err = h.uninstallExtensionWithLogs(taskID, req.PHPVersion, req.Target)
			}
		case "apache":
			if req.Action == "enable" {
				success, err = h.enableApacheModuleWithLogs(taskID, req.Target)
			} else if req.Action == "disable" {
				success, err = h.disableApacheModuleWithLogs(taskID, req.Target)
			}
		case "software":
			if req.Action == "install" {
				success, err = h.installSoftwareWithLogs(taskID, req.Target)
			} else if req.Action == "uninstall" {
				success, err = h.uninstallSoftwareWithLogs(taskID, req.Target)
			}
		}

		if err != nil {
			taskManager.addLog(taskID, "")
			taskManager.addLog(taskID, fmt.Sprintf("❌ Hata: %s", err.Error()))
		} else if success {
			taskManager.addLog(taskID, "")
			taskManager.addLog(taskID, fmt.Sprintf("✅ %s başarıyla tamamlandı!", taskName))
		}

		taskManager.completeTask(taskID, success && err == nil)
	}()

	return c.JSON(fiber.Map{
		"success": true,
		"task_id": taskID,
		"message": "İşlem başlatıldı",
	})
}

func getTaskName(taskType, action, target string) string {
	actionNames := map[string]string{
		"install":   "kurulumu",
		"uninstall": "kaldırılması",
		"enable":    "etkinleştirilmesi",
		"disable":   "devre dışı bırakılması",
	}
	return fmt.Sprintf("%s %s", target, actionNames[action])
}

func (h *Handler) installPHPWithLogs(taskID, version string) (bool, error) {
	// First, ensure ondrej/php PPA is added (required for PHP 7.x and multiple PHP versions)
	taskManager.addLog(taskID, "📋 Ondrej PHP PPA kontrol ediliyor...")

	// Check if PPA is already added
	checkPPA := exec.Command("bash", "-c", "grep -r 'ondrej/php' /etc/apt/sources.list.d/ 2>/dev/null || true")
	ppaOutput, _ := checkPPA.Output()

	if len(ppaOutput) == 0 {
		taskManager.addLog(taskID, "➕ Ondrej PHP PPA ekleniyor...")
		taskManager.addLog(taskID, "")

		// Add PPA
		err := RunCommandWithLogs(taskID, "bash", "-c",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y software-properties-common && add-apt-repository -y ppa:ondrej/php")
		if err != nil {
			taskManager.addLog(taskID, "⚠️ PPA eklenemedi, devam ediliyor...")
		}
		taskManager.addLog(taskID, "")
	} else {
		taskManager.addLog(taskID, "✓ Ondrej PHP PPA zaten mevcut")
	}

	packages := fmt.Sprintf("php%s-fpm php%s-cli php%s-common php%s-mysql php%s-gd php%s-curl php%s-mbstring php%s-xml php%s-zip",
		version, version, version, version, version, version, version, version, version)

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, fmt.Sprintf("📦 Paketler: %s", packages))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "bash", "-c",
		fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y %s", packages))

	if err != nil {
		return false, err
	}

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, "🔧 PHP-FPM servisi etkinleştiriliyor...")

	exec.Command("systemctl", "enable", fmt.Sprintf("php%s-fpm", version)).Run()
	exec.Command("systemctl", "start", fmt.Sprintf("php%s-fpm", version)).Run()

	return true, nil
}

func (h *Handler) uninstallPHPWithLogs(taskID, version string) (bool, error) {
	taskManager.addLog(taskID, "🛑 PHP-FPM servisi durduruluyor...")

	exec.Command("systemctl", "stop", fmt.Sprintf("php%s-fpm", version)).Run()
	exec.Command("systemctl", "disable", fmt.Sprintf("php%s-fpm", version)).Run()

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, fmt.Sprintf("🗑️ PHP %s paketleri kaldırılıyor...", version))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "bash", "-c",
		fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get remove -y php%s-*", version))

	return err == nil, err
}

func (h *Handler) installExtensionWithLogs(taskID, phpVersion, extension string) (bool, error) {
	packageName := fmt.Sprintf("php%s-%s", phpVersion, extension)

	taskManager.addLog(taskID, fmt.Sprintf("📦 Paket: %s", packageName))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "bash", "-c",
		fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y %s", packageName))

	if err != nil {
		return false, err
	}

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, "🔄 PHP-FPM yeniden başlatılıyor...")

	exec.Command("systemctl", "restart", fmt.Sprintf("php%s-fpm", phpVersion)).Run()

	return true, nil
}

func (h *Handler) uninstallExtensionWithLogs(taskID, phpVersion, extension string) (bool, error) {
	packageName := fmt.Sprintf("php%s-%s", phpVersion, extension)

	taskManager.addLog(taskID, fmt.Sprintf("🗑️ Paket kaldırılıyor: %s", packageName))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "bash", "-c",
		fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get remove -y %s", packageName))

	if err != nil {
		return false, err
	}

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, "🔄 PHP-FPM yeniden başlatılıyor...")

	exec.Command("systemctl", "restart", fmt.Sprintf("php%s-fpm", phpVersion)).Run()

	return true, nil
}

func (h *Handler) enableApacheModuleWithLogs(taskID, module string) (bool, error) {
	taskManager.addLog(taskID, fmt.Sprintf("🔧 Apache modülü etkinleştiriliyor: %s", module))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "a2enmod", module)

	if err != nil {
		return false, err
	}

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, "🔄 Apache yeniden yükleniyor...")

	exec.Command("systemctl", "reload", "apache2").Run()

	return true, nil
}

func (h *Handler) disableApacheModuleWithLogs(taskID, module string) (bool, error) {
	taskManager.addLog(taskID, fmt.Sprintf("🔧 Apache modülü devre dışı bırakılıyor: %s", module))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "a2dismod", module)

	if err != nil {
		return false, err
	}

	taskManager.addLog(taskID, "")
	taskManager.addLog(taskID, "🔄 Apache yeniden yükleniyor...")

	exec.Command("systemctl", "reload", "apache2").Run()

	return true, nil
}

func (h *Handler) installSoftwareWithLogs(taskID, packageName string) (bool, error) {
	taskManager.addLog(taskID, fmt.Sprintf("📦 Paket: %s", packageName))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "bash", "-c",
		fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y %s", packageName))

	return err == nil, err
}

func (h *Handler) uninstallSoftwareWithLogs(taskID, packageName string) (bool, error) {
	taskManager.addLog(taskID, fmt.Sprintf("🗑️ Paket kaldırılıyor: %s", packageName))
	taskManager.addLog(taskID, "")

	err := RunCommandWithLogs(taskID, "bash", "-c",
		fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get remove -y %s", packageName))

	return err == nil, err
}

// GetTaskStatus returns the status of a task
func (h *Handler) GetTaskStatus(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	task := taskManager.getTask(taskID)

	if task == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Task bulunamadı",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    task,
	})
}
