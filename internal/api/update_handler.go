package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Version bilgisi - her build'de güncellenir
const CurrentVersion = "1.0.0"

// GitHub repository bilgileri
const (
	GitHubOwner = "asergenalkan"
	GitHubRepo  = "serverpanel"
)

// Update durumu
type UpdateStatus struct {
	IsRunning    bool      `json:"is_running"`
	Progress     string    `json:"progress"`
	Logs         []string  `json:"logs"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

var (
	updateStatus = &UpdateStatus{}
	updateMutex  = &sync.Mutex{}
)

// GitHub commit bilgisi
type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// Güncelleme kontrol response
type UpdateCheckResponse struct {
	CurrentVersion string    `json:"current_version"`
	LatestCommit   string    `json:"latest_commit"`
	LocalCommit    string    `json:"local_commit"`
	HasUpdate      bool      `json:"has_update"`
	CommitMessage  string    `json:"commit_message"`
	CommitAuthor   string    `json:"commit_author"`
	CommitDate     time.Time `json:"commit_date"`
	LastChecked    time.Time `json:"last_checked"`
}

// CheckForUpdates - GitHub'dan son commit'i kontrol eder
func (h *Handler) CheckForUpdates(c *fiber.Ctx) error {
	// GitHub API'den son commit'i al
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/main", GitHubOwner, GitHubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "GitHub API isteği oluşturulamadı"})
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ServerPanel-Update-Checker")

	resp, err := client.Do(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "GitHub'a bağlanılamadı"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Status(500).JSON(fiber.Map{"error": "GitHub API hatası"})
	}

	var commit GitHubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "GitHub yanıtı ayrıştırılamadı"})
	}

	// Yerel commit'i al
	localCommit := getLocalCommit()

	// Karşılaştır
	hasUpdate := localCommit != "" && commit.SHA != "" && !strings.HasPrefix(commit.SHA, localCommit) && !strings.HasPrefix(localCommit, commit.SHA[:7])

	return c.JSON(UpdateCheckResponse{
		CurrentVersion: CurrentVersion,
		LatestCommit:   commit.SHA[:7],
		LocalCommit:    localCommit,
		HasUpdate:      hasUpdate,
		CommitMessage:  commit.Commit.Message,
		CommitAuthor:   commit.Commit.Author.Name,
		CommitDate:     commit.Commit.Author.Date,
		LastChecked:    time.Now(),
	})
}

// getLocalCommit - Yerel git commit hash'ini alır
func getLocalCommit() string {
	// Önce /opt/serverpanel'de dene
	paths := []string{"/opt/serverpanel", "."}

	for _, path := range paths {
		cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
		cmd.Dir = path
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	}

	return ""
}

// GetUpdateStatus - Güncelleme durumunu döndürür
func (h *Handler) GetUpdateStatus(c *fiber.Ctx) error {
	updateMutex.Lock()
	defer updateMutex.Unlock()

	return c.JSON(updateStatus)
}

// RunUpdate - Güncelleme script'ini çalıştırır
func (h *Handler) RunUpdate(c *fiber.Ctx) error {
	updateMutex.Lock()
	if updateStatus.IsRunning {
		updateMutex.Unlock()
		return c.Status(400).JSON(fiber.Map{"error": "Güncelleme zaten çalışıyor"})
	}

	// Durumu sıfırla
	updateStatus.IsRunning = true
	updateStatus.Progress = "Başlatılıyor..."
	updateStatus.Logs = []string{}
	updateStatus.StartedAt = time.Now()
	updateStatus.CompletedAt = time.Time{}
	updateStatus.Success = false
	updateStatus.ErrorMessage = ""
	updateMutex.Unlock()

	// Arka planda çalıştır
	go runUpdateScript()

	return c.JSON(fiber.Map{
		"message": "Güncelleme başlatıldı",
		"status":  "running",
	})
}

// runUpdateScript - Update script'ini arka planda çalıştırır
func runUpdateScript() {
	scriptPath := "/opt/serverpanel/scripts/update-server.sh"

	// Script var mı kontrol et
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		updateMutex.Lock()
		updateStatus.IsRunning = false
		updateStatus.Success = false
		updateStatus.ErrorMessage = "Update script bulunamadı: " + scriptPath
		updateStatus.CompletedAt = time.Now()
		updateMutex.Unlock()
		return
	}

	addLog("Güncelleme başlatılıyor...")

	// setsid ile script'i tamamen yeni bir session'da başlat
	// Bu, parent process (backend) kapansa bile script'in devam etmesini sağlar
	// nohup + setsid + background (&) kombinasyonu en güvenilir yöntem
	cmd := exec.Command("setsid", "bash", "-c", fmt.Sprintf("nohup %s > /tmp/serverpanel-update.log 2>&1 &", scriptPath))
	cmd.Dir = "/opt/serverpanel"

	// Tüm bağlantıları kes
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// SysProcAttr ile yeni process group oluştur
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Run(); err != nil {
		updateMutex.Lock()
		updateStatus.IsRunning = false
		updateStatus.Success = false
		updateStatus.ErrorMessage = "Script başlatılamadı: " + err.Error()
		updateStatus.CompletedAt = time.Now()
		updateMutex.Unlock()
		return
	}

	addLog("Güncelleme script'i başlatıldı, servis yeniden başlatılacak...")
	addLog("Log dosyası: /tmp/serverpanel-update.log")
}

// addLog - Log ekler (mutex ile)
func addLog(log string) {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	addLogUnsafe(log)
}

// addLogUnsafe - Log ekler (mutex olmadan - caller mutex tutmalı)
func addLogUnsafe(log string) {
	// Son 100 log'u tut
	if len(updateStatus.Logs) >= 100 {
		updateStatus.Logs = updateStatus.Logs[1:]
	}
	updateStatus.Logs = append(updateStatus.Logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), log))
}

// Kullanılmayan import'u engellemek için
var _ = io.EOF
