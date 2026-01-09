package system

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/models"
)

// GetSystemStats returns current system statistics
func GetSystemStats() *models.SystemStats {
	stats := &models.SystemStats{}

	// CPU Usage (simplified - works on Linux)
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("sh", "-c", "top -bn1 | grep 'Cpu(s)' | awk '{print $2}'").Output(); err == nil {
			if cpu, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
				stats.CPUUsage = cpu
			}
		}
	}

	// Memory info
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("sh", "-c", "free -b | grep Mem").Output(); err == nil {
			fields := strings.Fields(string(out))
			if len(fields) >= 3 {
				stats.MemoryTotal, _ = strconv.ParseInt(fields[1], 10, 64)
				stats.MemoryUsed, _ = strconv.ParseInt(fields[2], 10, 64)
			}
		}
	} else if runtime.GOOS == "darwin" {
		// macOS - simplified
		stats.MemoryTotal = 16 * 1024 * 1024 * 1024 // 16GB placeholder
		stats.MemoryUsed = 8 * 1024 * 1024 * 1024   // 8GB placeholder
	}

	// Disk info
	if out, err := exec.Command("df", "-B1", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				stats.DiskTotal, _ = strconv.ParseInt(fields[1], 10, 64)
				stats.DiskUsed, _ = strconv.ParseInt(fields[2], 10, 64)
			}
		}
	}

	// Uptime
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("cat", "/proc/uptime").Output(); err == nil {
			fields := strings.Fields(string(out))
			if len(fields) >= 1 {
				if uptime, err := strconv.ParseFloat(fields[0], 64); err == nil {
					stats.Uptime = int64(uptime)
				}
			}
		}
	}

	// Load average
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if out, err := exec.Command("sh", "-c", "uptime | awk -F'load average:' '{print $2}'").Output(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), ",")
			for _, p := range parts {
				if load, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil {
					stats.LoadAverage = append(stats.LoadAverage, load)
				}
			}
		}
	}

	return stats
}

// Service represents a system service
type Service struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // running, stopped, unknown
	Enabled bool   `json:"enabled"`
}

// GetServices returns list of hosting-related services
func GetServices() []Service {
	services := []string{"nginx", "apache2", "mysql", "mariadb", "postgresql", "php-fpm", "postfix", "dovecot", "named", "pure-ftpd"}
	result := make([]Service, 0)

	for _, name := range services {
		svc := Service{Name: name, Status: "unknown"}

		if runtime.GOOS == "linux" {
			// Check if service exists and its status
			if out, err := exec.Command("systemctl", "is-active", name).Output(); err == nil {
				status := strings.TrimSpace(string(out))
				if status == "active" {
					svc.Status = "running"
				} else {
					svc.Status = "stopped"
				}

				// Check if enabled
				if out, err := exec.Command("systemctl", "is-enabled", name).Output(); err == nil {
					svc.Enabled = strings.TrimSpace(string(out)) == "enabled"
				}

				result = append(result, svc)
			}
		}
	}

	return result
}

// RestartService restarts a system service
func RestartService(name string) error {
	if runtime.GOOS == "linux" {
		return exec.Command("systemctl", "restart", name).Run()
	}
	return nil
}

// ExecuteCommand runs a shell command and returns output
func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
