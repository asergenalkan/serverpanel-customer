package api

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// DetailedProcessInfo represents detailed process information for process manager
type DetailedProcessInfo struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	User       string  `json:"user"`
	Priority   int     `json:"priority"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	Command    string  `json:"command"`
	File       string  `json:"file"`
	CWD        string  `json:"cwd"`
}

// DiskUsageInfo represents disk usage information
type DiskUsageInfo struct {
	Device      string  `json:"device"`
	Size        string  `json:"size"`
	Used        string  `json:"used"`
	Available   string  `json:"available"`
	PercentUsed float64 `json:"percent_used"`
	MountPoint  string  `json:"mount_point"`
}

// IOStats represents disk I/O statistics
type IOStats struct {
	Device            string `json:"device"`
	TransPerSec       string `json:"trans_per_sec"`
	BlocksReadPerSec  string `json:"blocks_read_per_sec"`
	BlocksWritePerSec string `json:"blocks_write_per_sec"`
	TotalBlocksRead   string `json:"total_blocks_read"`
	TotalBlocksWrite  string `json:"total_blocks_write"`
}

// BackgroundKillerSettings represents settings for background process killer
type BackgroundKillerSettings struct {
	Processes    []string `json:"processes"`
	TrustedUsers []string `json:"trusted_users"`
}

// Default dangerous processes to monitor
var defaultDangerousProcesses = []string{
	"BitchX",
	"bnc",
	"eggdrop",
	"generic-sniffers",
	"guardservices",
	"ircd",
	"psyBNC",
	"ptlink",
	"services",
}

// GetProcessManager returns all running processes with details (for Process Manager page)
func (h *Handler) GetProcessManager(c *fiber.Ctx) error {
	// Get all processes with detailed info
	cmd := exec.Command("ps", "aux", "--sort=-pcpu")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "İşlem listesi alınamadı",
		})
	}

	var processes []DetailedProcessInfo
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		priority, _ := strconv.Atoi(fields[4])

		// Get command line
		command := strings.Join(fields[10:], " ")

		// Get executable path
		exePath := ""
		if pid > 0 {
			if link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
				exePath = link
			}
		}

		// Get current working directory
		cwd := ""
		if pid > 0 {
			if link, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
				cwd = link
			}
		}

		processes = append(processes, DetailedProcessInfo{
			PID:        pid,
			Name:       filepath.Base(fields[10]),
			User:       fields[0],
			Priority:   priority,
			CPUPercent: cpu,
			MemPercent: mem,
			Command:    command,
			File:       exePath,
			CWD:        cwd,
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    processes,
	})
}

// GetRunningProcesses returns all running processes (simpler view)
func (h *Handler) GetRunningProcesses(c *fiber.Ctx) error {
	cmd := exec.Command("ps", "-eo", "pid,comm,args,cwd", "--sort=pid")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "İşlem listesi alınamadı",
		})
	}

	type SimpleProcess struct {
		PID     int    `json:"pid"`
		Name    string `json:"name"`
		File    string `json:"file"`
		CWD     string `json:"cwd"`
		Command string `json:"command"`
	}

	var processes []SimpleProcess
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		name := fields[1]
		command := ""
		if len(fields) > 2 {
			command = strings.Join(fields[2:], " ")
		}

		// Get executable path
		exePath := ""
		if pid > 0 {
			if link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
				exePath = link
			}
		}

		// Get current working directory
		cwd := ""
		if pid > 0 {
			if link, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
				cwd = link
			}
		}

		processes = append(processes, SimpleProcess{
			PID:     pid,
			Name:    name,
			File:    exePath,
			CWD:     cwd,
			Command: command,
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    processes,
	})
}

// GetDiskUsage returns current disk usage information
func (h *Handler) GetDiskUsage(c *fiber.Ctx) error {
	// Get disk usage
	cmd := exec.Command("df", "-h", "--output=source,size,used,avail,pcent,target")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Disk bilgisi alınamadı",
		})
	}

	var disks []DiskUsageInfo
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Skip non-physical devices
		if !strings.HasPrefix(fields[0], "/dev/") {
			continue
		}

		percentStr := strings.TrimSuffix(fields[4], "%")
		percent, _ := strconv.ParseFloat(percentStr, 64)

		disks = append(disks, DiskUsageInfo{
			Device:      fields[0],
			Size:        fields[1],
			Used:        fields[2],
			Available:   fields[3],
			PercentUsed: percent,
			MountPoint:  fields[5],
		})
	}

	// Get I/O statistics
	var ioStats []IOStats
	iostatCmd := exec.Command("iostat", "-d")
	iostatOutput, err := iostatCmd.Output()
	if err == nil {
		iostatLines := strings.Split(string(iostatOutput), "\n")
		headerFound := false
		for _, line := range iostatLines {
			if strings.HasPrefix(line, "Device") {
				headerFound = true
				continue
			}
			if headerFound && line != "" {
				fields := strings.Fields(line)
				if len(fields) >= 6 {
					ioStats = append(ioStats, IOStats{
						Device:            fields[0],
						TransPerSec:       fields[1],
						BlocksReadPerSec:  fields[2],
						BlocksWritePerSec: fields[3],
						TotalBlocksRead:   fields[4],
						TotalBlocksWrite:  fields[5],
					})
				}
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"disks":    disks,
			"io_stats": ioStats,
		},
	})
}

// KillProcess kills a specific process by PID
func (h *Handler) KillProcess(c *fiber.Ctx) error {
	var req struct {
		PID    int    `json:"pid"`
		Signal string `json:"signal"` // TERM, KILL, etc.
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	if req.PID <= 1 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz PID",
		})
	}

	// Default to SIGTERM
	signal := syscall.SIGTERM
	if req.Signal == "KILL" {
		signal = syscall.SIGKILL
	}

	process, err := os.FindProcess(req.PID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "İşlem bulunamadı",
		})
	}

	if err := process.Signal(signal); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("İşlem sonlandırılamadı: %v", err),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("İşlem %d sonlandırıldı", req.PID),
	})
}

// KillUserProcesses kills all processes for a specific user
func (h *Handler) KillUserProcesses(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	if req.Username == "" || req.Username == "root" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz kullanıcı adı",
		})
	}

	// Kill all processes for user
	cmd := exec.Command("pkill", "-u", req.Username)
	if err := cmd.Run(); err != nil {
		// pkill returns 1 if no processes matched, which is ok
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 1 {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "İşlemler sonlandırılamadı",
			})
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s kullanıcısının tüm işlemleri sonlandırıldı", req.Username),
	})
}

// GetBackgroundKillerSettings returns background process killer settings
func (h *Handler) GetBackgroundKillerSettings(c *fiber.Ctx) error {
	// Read settings from file or return defaults
	settingsFile := "/opt/serverpanel/config/background_killer.conf"

	settings := BackgroundKillerSettings{
		Processes:    defaultDangerousProcesses,
		TrustedUsers: []string{},
	}

	// Try to read existing settings
	if file, err := os.Open(settingsFile); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		section := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if line == "[processes]" {
				section = "processes"
				settings.Processes = []string{}
				continue
			}
			if line == "[trusted_users]" {
				section = "trusted_users"
				continue
			}
			if section == "processes" {
				settings.Processes = append(settings.Processes, line)
			} else if section == "trusted_users" {
				settings.TrustedUsers = append(settings.TrustedUsers, line)
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    settings,
	})
}

// SaveBackgroundKillerSettings saves background process killer settings
func (h *Handler) SaveBackgroundKillerSettings(c *fiber.Ctx) error {
	var settings BackgroundKillerSettings

	if err := c.BodyParser(&settings); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Create config directory if not exists
	configDir := "/opt/serverpanel/config"
	os.MkdirAll(configDir, 0755)

	// Write settings to file
	settingsFile := filepath.Join(configDir, "background_killer.conf")
	var content strings.Builder
	content.WriteString("# ServerPanel Background Process Killer Settings\n\n")
	content.WriteString("[processes]\n")
	for _, p := range settings.Processes {
		content.WriteString(p + "\n")
	}
	content.WriteString("\n[trusted_users]\n")
	for _, u := range settings.TrustedUsers {
		content.WriteString(u + "\n")
	}

	if err := os.WriteFile(settingsFile, []byte(content.String()), 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Ayarlar kaydedilemedi",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Ayarlar kaydedildi",
	})
}

// GetSystemUsers returns list of system users for filtering
func (h *Handler) GetSystemUsers(c *fiber.Ctx) error {
	cmd := exec.Command("awk", "-F:", "$3 >= 1000 {print $1}", "/etc/passwd")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Kullanıcı listesi alınamadı",
		})
	}

	users := strings.Split(strings.TrimSpace(string(output)), "\n")

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    users,
	})
}
