package webserver

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// PHPFPMManager manages PHP-FPM pools for users
type PHPFPMManager struct {
	simulateMode bool
	basePath     string
	phpVersion   string
}

// PHPFPMConfig contains configuration for a PHP-FPM pool
type PHPFPMConfig struct {
	Username   string
	HomeDir    string
	PHPVersion string
}

// NewPHPFPMManager creates a new PHP-FPM manager
func NewPHPFPMManager(simulateMode bool, basePath string, phpVersion string) *PHPFPMManager {
	if phpVersion == "" {
		phpVersion = "8.1" // Default to 8.1 for Ubuntu 22.04
	}
	return &PHPFPMManager{
		simulateMode: simulateMode,
		basePath:     basePath,
		phpVersion:   phpVersion,
	}
}

func (m *PHPFPMManager) GetPoolPath() string {
	if m.simulateMode {
		return filepath.Join(m.basePath, "php-fpm", "pool.d")
	}
	return fmt.Sprintf("/etc/php/%s/fpm/pool.d", m.phpVersion)
}

// CreatePool creates a PHP-FPM pool for a user
func (m *PHPFPMManager) CreatePool(config PHPFPMConfig) error {
	phpVersion := config.PHPVersion
	if phpVersion == "" {
		phpVersion = m.phpVersion
	}

	poolConfig := m.generatePoolConfig(config, phpVersion)

	poolPath := m.GetPoolPath()
	if err := os.MkdirAll(poolPath, 0755); err != nil {
		return fmt.Errorf("failed to create pool directory: %w", err)
	}

	poolFile := filepath.Join(poolPath, config.Username+".conf")
	if err := os.WriteFile(poolFile, []byte(poolConfig), 0644); err != nil {
		return fmt.Errorf("failed to write pool config: %w", err)
	}

	log.Printf("📝 PHP-FPM pool created: %s", poolFile)

	return m.Reload()
}

func (m *PHPFPMManager) generatePoolConfig(config PHPFPMConfig, phpVersion string) string {
	socketPath := fmt.Sprintf("/run/php/php%s-fpm-%s.sock", phpVersion, config.Username)
	if m.simulateMode {
		socketPath = filepath.Join(m.basePath, "php-fpm", config.Username+".sock")
	}

	return fmt.Sprintf(`[%s]
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
php_admin_value[disable_functions] = exec,passthru,shell_exec,system,proc_open,popen
php_admin_value[upload_tmp_dir] = %s/tmp
php_admin_value[session.save_path] = %s/tmp

; Limits
php_admin_value[memory_limit] = 256M
php_admin_value[max_execution_time] = 300
php_admin_value[max_input_time] = 300
php_admin_value[post_max_size] = 64M
php_admin_value[upload_max_filesize] = 64M
`,
		config.Username,
		config.Username,
		config.Username,
		config.Username,
		socketPath,
		config.HomeDir,
		config.HomeDir,
		config.HomeDir,
		config.HomeDir,
	)
}

// DeletePool removes a PHP-FPM pool
func (m *PHPFPMManager) DeletePool(username string) error {
	poolFile := filepath.Join(m.GetPoolPath(), username+".conf")
	if err := os.Remove(poolFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pool config: %w", err)
	}

	log.Printf("🗑️ PHP-FPM pool deleted: %s", poolFile)
	return m.Reload()
}

// Reload reloads or restarts PHP-FPM depending on its state
func (m *PHPFPMManager) Reload() error {
	if m.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] systemctl reload php%s-fpm", m.phpVersion)
		return nil
	}

	service := fmt.Sprintf("php%s-fpm", m.phpVersion)

	// Check if service is active
	checkCmd := exec.Command("systemctl", "is-active", "--quiet", service)
	isActive := checkCmd.Run() == nil

	var cmd *exec.Cmd
	if isActive {
		// Service is active, reload it
		cmd = exec.Command("systemctl", "reload", service)
	} else {
		// Service is not active, start it
		log.Printf("⚠️ PHP-FPM was not active, starting it...")
		cmd = exec.Command("systemctl", "start", service)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		// If reload/start failed, try restart as fallback
		log.Printf("⚠️ PHP-FPM reload/start failed, trying restart...")
		restartCmd := exec.Command("systemctl", "restart", service)
		if restartOutput, restartErr := restartCmd.CombinedOutput(); restartErr != nil {
			return fmt.Errorf("failed to reload/restart php-fpm: %s - %w", string(restartOutput), restartErr)
		}
	} else {
		_ = output // suppress unused warning
	}

	log.Printf("✅ PHP-FPM reloaded successfully")
	return nil
}
