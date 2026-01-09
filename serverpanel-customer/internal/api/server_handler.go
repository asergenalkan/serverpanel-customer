package api

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// ServerInfo represents server information
type ServerInfo struct {
	Hostname    string      `json:"hostname"`
	OS          string      `json:"os"`
	Kernel      string      `json:"kernel"`
	Uptime      string      `json:"uptime"`
	LoadAverage string      `json:"load_average"`
	CPU         CPUInfo     `json:"cpu"`
	Memory      MemoryInfo  `json:"memory"`
	Disk        DiskInfo    `json:"disk"`
	Network     NetworkInfo `json:"network"`
}

type CPUInfo struct {
	Model string  `json:"model"`
	Cores int     `json:"cores"`
	Usage float64 `json:"usage"`
}

type MemoryInfo struct {
	Total int64   `json:"total"`
	Used  int64   `json:"used"`
	Free  int64   `json:"free"`
	Usage float64 `json:"usage"`
}

type DiskInfo struct {
	Total int64   `json:"total"`
	Used  int64   `json:"used"`
	Free  int64   `json:"free"`
	Usage float64 `json:"usage"`
}

type NetworkInfo struct {
	IP         string   `json:"ip"`
	Interfaces []string `json:"interfaces"`
}

// DailyLogEntry represents a user's daily resource usage
type DailyLogEntry struct {
	User        string  `json:"user"`
	Domain      string  `json:"domain"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	DBProcesses float64 `json:"db_processes"`
}

// ProcessInfo represents a running process
type ProcessInfo struct {
	User       string  `json:"user"`
	Domain     string  `json:"domain"`
	CPUPercent float64 `json:"cpu_percent"`
	Command    string  `json:"command"`
	PID        int     `json:"pid"`
	Memory     string  `json:"memory"`
}

// QueueData represents task queue information
type QueueData struct {
	MailQueue      []MailQueueItem `json:"mail_queue"`
	MailQueueCount int             `json:"mail_queue_count"`
	CronJobs       []CronJob       `json:"cron_jobs"`
	PendingTasks   int             `json:"pending_tasks"`
}

type MailQueueItem struct {
	ID        string `json:"id"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Size      string `json:"size"`
	Time      string `json:"time"`
	Status    string `json:"status"`
}

type CronJob struct {
	User     string `json:"user"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	NextRun  string `json:"next_run"`
}

// GetServerInfo returns server information
func (h *Handler) GetServerInfo(c *fiber.Ctx) error {
	info := ServerInfo{}

	// Hostname
	hostname, _ := os.Hostname()
	info.Hostname = hostname

	// OS Info
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	// Kernel
	if output, err := exec.Command("uname", "-r").Output(); err == nil {
		info.Kernel = strings.TrimSpace(string(output))
	}

	// Uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) > 0 {
			seconds, _ := strconv.ParseFloat(parts[0], 64)
			duration := time.Duration(seconds) * time.Second
			days := int(duration.Hours() / 24)
			hours := int(duration.Hours()) % 24
			minutes := int(duration.Minutes()) % 60
			info.Uptime = fmt.Sprintf("%d gün, %d saat, %d dakika", days, hours, minutes)
		}
	}

	// Load Average
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			info.LoadAverage = fmt.Sprintf("%s, %s, %s", parts[0], parts[1], parts[2])
		}
	}

	// CPU Info
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		cores := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) > 1 {
					info.CPU.Model = strings.TrimSpace(parts[1])
				}
			}
			if strings.HasPrefix(line, "processor") {
				cores++
			}
		}
		info.CPU.Cores = cores
	}

	// CPU Usage
	info.CPU.Usage = getCPUUsage()

	// Memory Info
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		var total, free, available, buffers, cached int64
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseInt(fields[1], 10, 64)
				value *= 1024 // Convert from KB to bytes
				switch fields[0] {
				case "MemTotal:":
					total = value
				case "MemFree:":
					free = value
				case "MemAvailable:":
					available = value
				case "Buffers:":
					buffers = value
				case "Cached:":
					cached = value
				}
			}
		}
		info.Memory.Total = total
		if available > 0 {
			info.Memory.Free = available
		} else {
			info.Memory.Free = free + buffers + cached
		}
		info.Memory.Used = total - info.Memory.Free
		if total > 0 {
			info.Memory.Usage = float64(info.Memory.Used) / float64(total) * 100
		}
	}

	// Disk Info
	if output, err := exec.Command("df", "-B1", "/").Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				info.Disk.Total, _ = strconv.ParseInt(fields[1], 10, 64)
				info.Disk.Used, _ = strconv.ParseInt(fields[2], 10, 64)
				info.Disk.Free, _ = strconv.ParseInt(fields[3], 10, 64)
				if info.Disk.Total > 0 {
					info.Disk.Usage = float64(info.Disk.Used) / float64(info.Disk.Total) * 100
				}
			}
		}
	}

	// Network Info
	if output, err := exec.Command("hostname", "-I").Output(); err == nil {
		ips := strings.Fields(string(output))
		if len(ips) > 0 {
			info.Network.IP = ips[0]
		}
	}

	// Network Interfaces
	if output, err := exec.Command("ls", "/sys/class/net").Output(); err == nil {
		info.Network.Interfaces = strings.Fields(string(output))
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    info,
	})
}

// getCPUUsage calculates CPU usage from /proc/stat
func getCPUUsage() float64 {
	readCPUStat := func() (idle, total uint64) {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0, 0
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "cpu ") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					var values []uint64
					for _, f := range fields[1:] {
						v, _ := strconv.ParseUint(f, 10, 64)
						values = append(values, v)
					}
					if len(values) >= 4 {
						idle = values[3]
						for _, v := range values {
							total += v
						}
					}
				}
				break
			}
		}
		return
	}

	idle1, total1 := readCPUStat()
	time.Sleep(100 * time.Millisecond)
	idle2, total2 := readCPUStat()

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)

	if totalDelta == 0 {
		return 0
	}

	return (1 - idleDelta/totalDelta) * 100
}

// GetDailyLog returns daily resource usage log
func (h *Handler) GetDailyLog(c *fiber.Ctx) error {
	// Get users and their domains from database
	rows, err := h.db.Query(`
		SELECT u.username, d.name 
		FROM users u 
		LEFT JOIN domains d ON u.id = d.user_id 
		WHERE u.role = 'user'
		ORDER BY u.username
	`)
	if err != nil {
		return c.JSON(models.APIResponse{
			Success: true,
			Data:    []DailyLogEntry{},
		})
	}
	defer rows.Close()

	var entries []DailyLogEntry
	userDomains := make(map[string]string)

	for rows.Next() {
		var username, domain string
		rows.Scan(&username, &domain)
		if domain != "" {
			userDomains[username] = domain
		}
	}

	// Get process stats for each user
	for user, domain := range userDomains {
		entry := DailyLogEntry{
			User:   user,
			Domain: domain,
		}

		// Get CPU and memory usage for user's processes
		cmd := exec.Command("ps", "-u", user, "-o", "%cpu,%mem", "--no-headers")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					cpu, _ := strconv.ParseFloat(fields[0], 64)
					mem, _ := strconv.ParseFloat(fields[1], 64)
					entry.CPUPercent += cpu
					entry.MemPercent += mem
				}
			}
		}

		// Get MySQL process count for user
		cmd = exec.Command("mysql", "-N", "-e",
			fmt.Sprintf("SELECT COUNT(*) FROM information_schema.processlist WHERE user LIKE '%s%%'", user))
		if output, err := cmd.Output(); err == nil {
			count, _ := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
			entry.DBProcesses = count
		}

		entries = append(entries, entry)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    entries,
	})
}

// GetTopProcesses returns top CPU consuming processes
func (h *Handler) GetTopProcesses(c *fiber.Ctx) error {
	cmd := exec.Command("ps", "aux", "--sort=-%cpu")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Process bilgileri alınamadı",
		})
	}

	var processes []ProcessInfo
	lines := strings.Split(string(output), "\n")

	// Get user-domain mapping
	userDomains := make(map[string]string)
	rows, _ := h.db.Query("SELECT u.username, d.name FROM users u LEFT JOIN domains d ON u.id = d.user_id")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var user, domain string
			rows.Scan(&user, &domain)
			if domain != "" {
				userDomains[user] = domain
			}
		}
	}

	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}
		if i > 50 {
			break // Limit to 50 processes
		}

		fields := strings.Fields(line)
		if len(fields) >= 11 {
			user := fields[0]
			pid, _ := strconv.Atoi(fields[1])
			cpu, _ := strconv.ParseFloat(fields[2], 64)
			mem := fields[5]
			command := strings.Join(fields[10:], " ")

			// Skip system processes with 0 CPU
			if cpu == 0 && i > 20 {
				continue
			}

			processes = append(processes, ProcessInfo{
				User:       user,
				Domain:     userDomains[user],
				CPUPercent: cpu,
				Command:    command,
				PID:        pid,
				Memory:     mem,
			})
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    processes,
	})
}

// GetTaskQueue returns mail queue and cron jobs
func (h *Handler) GetTaskQueue(c *fiber.Ctx) error {
	data := QueueData{}

	// Get mail queue
	cmd := exec.Command("mailq")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		queueRegex := regexp.MustCompile(`^([A-F0-9]+)\s+(\d+)\s+(\w+\s+\w+\s+\d+\s+\d+:\d+:\d+)\s+(.+)$`)

		for _, line := range lines {
			if matches := queueRegex.FindStringSubmatch(line); matches != nil {
				item := MailQueueItem{
					ID:     matches[1],
					Size:   matches[2] + " bytes",
					Time:   matches[3],
					Sender: matches[4],
					Status: "queued",
				}
				data.MailQueue = append(data.MailQueue, item)
			}
			if strings.Contains(line, "(deferred") {
				if len(data.MailQueue) > 0 {
					data.MailQueue[len(data.MailQueue)-1].Status = "deferred"
				}
			}
		}
		data.MailQueueCount = len(data.MailQueue)
	}

	// Get cron jobs
	cronDir := "/var/spool/cron/crontabs"
	if files, err := os.ReadDir(cronDir); err == nil {
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			user := file.Name()
			cronFile := cronDir + "/" + user

			if f, err := os.Open(cronFile); err == nil {
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "#") || line == "" {
						continue
					}
					fields := strings.Fields(line)
					if len(fields) >= 6 {
						schedule := strings.Join(fields[:5], " ")
						command := strings.Join(fields[5:], " ")
						data.CronJobs = append(data.CronJobs, CronJob{
							User:     user,
							Schedule: schedule,
							Command:  command,
							NextRun:  "-",
						})
					}
				}
				f.Close()
			}
		}
	}

	// Count pending tasks (simplified)
	data.PendingTasks = data.MailQueueCount

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    data,
	})
}

// FlushMailQueue flushes the mail queue
func (h *Handler) FlushMailQueue(c *fiber.Ctx) error {
	cmd := exec.Command("postqueue", "-f")
	if err := cmd.Run(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Mail kuyruğu temizlenemedi",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Mail kuyruğu yeniden işleme alındı",
	})
}
