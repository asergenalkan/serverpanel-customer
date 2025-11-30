package account

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/services/dns"
	"github.com/asergenalkan/serverpanel/internal/webserver"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidUsername = errors.New("invalid username format")
	ErrUserExists      = errors.New("username already exists")
	ErrInvalidDomain   = errors.New("invalid domain format")
	ErrDomainExists    = errors.New("domain already exists")
	ErrPackageNotFound = errors.New("package not found")
	ErrQuotaExceeded   = errors.New("quota exceeded")
)

// DB interface for database operations
type DB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Begin() (*sql.Tx, error)
}

type Service struct {
	db  DB
	cfg *config.Config
}

type CreateAccountRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	PackageID int64  `json:"package_id"`
}

type Account struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Domain      string `json:"domain"`
	HomeDir     string `json:"home_dir"`
	PackageID   int64  `json:"package_id"`
	PackageName string `json:"package_name"`
	DiskUsed    int64  `json:"disk_used"`
	DiskQuota   int64  `json:"disk_quota"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
}

func NewService(db DB) *Service {
	return &Service{
		db:  db,
		cfg: config.Get(),
	}
}

// ValidateUsername checks if username is valid
func (s *Service) ValidateUsername(username string) error {
	// Must be 3-32 characters, lowercase, alphanumeric, can contain underscore
	if len(username) < 3 || len(username) > 32 {
		return ErrInvalidUsername
	}

	matched, _ := regexp.MatchString(`^[a-z][a-z0-9_]*$`, username)
	if !matched {
		return ErrInvalidUsername
	}

	// Check reserved usernames
	reserved := []string{"root", "admin", "administrator", "www-data", "nginx", "mysql", "postgres", "mail", "ftp"}
	for _, r := range reserved {
		if username == r {
			return ErrInvalidUsername
		}
	}

	return nil
}

// ValidateDomain checks if domain format is valid
func (s *Service) ValidateDomain(domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Basic domain validation
	matched, _ := regexp.MatchString(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}$`, domain)
	if !matched {
		return ErrInvalidDomain
	}

	return nil
}

// CreateAccount creates a new hosting account
func (s *Service) CreateAccount(req CreateAccountRequest) (*Account, error) {
	// Validate username
	if err := s.ValidateUsername(req.Username); err != nil {
		return nil, err
	}

	// Validate domain
	if err := s.ValidateDomain(req.Domain); err != nil {
		return nil, err
	}

	// Check if username exists in database
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}

	// Check if domain exists
	err = s.db.QueryRow("SELECT COUNT(*) FROM domains WHERE name = ?", req.Domain).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrDomainExists
	}

	// Get package info
	var packageName string
	var diskQuota int64
	err = s.db.QueryRow("SELECT name, disk_quota FROM packages WHERE id = ?", req.PackageID).Scan(&packageName, &diskQuota)
	if err != nil {
		return nil, ErrPackageNotFound
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, err
	}

	homeDir := filepath.Join(s.cfg.HomeBaseDir, req.Username)

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create user in database
	result, err := tx.Exec(`
		INSERT INTO users (username, email, password, role, active, created_at, updated_at)
		VALUES (?, ?, ?, 'user', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, req.Username, req.Email, string(hashedPassword))
	if err != nil {
		return nil, err
	}

	userID, _ := result.LastInsertId()

	// Assign package to user
	_, err = tx.Exec(`
		INSERT INTO user_packages (user_id, package_id)
		VALUES (?, ?)
	`, userID, req.PackageID)
	if err != nil {
		return nil, err
	}

	// Create domain entry
	documentRoot := filepath.Join(homeDir, "public_html")
	_, err = tx.Exec(`
		INSERT INTO domains (user_id, name, document_root, active)
		VALUES (?, ?, ?, 1)
	`, userID, req.Domain, documentRoot)
	if err != nil {
		return nil, err
	}

	// Create system resources
	if err := s.createSystemUser(req.Username, homeDir); err != nil {
		return nil, fmt.Errorf("failed to create system user: %w", err)
	}

	if err := s.createDirectoryStructure(req.Username, homeDir); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	if err := s.createWebServerVhost(req.Username, req.Domain, homeDir, documentRoot); err != nil {
		return nil, fmt.Errorf("failed to create web server config: %w", err)
	}

	if err := s.createPHPFPMPool(req.Username, homeDir); err != nil {
		return nil, fmt.Errorf("failed to create PHP-FPM pool: %w", err)
	}

	// Create DNS zone
	if err := s.createDNSZone(req.Domain); err != nil {
		log.Printf("Warning: failed to create DNS zone: %v", err)
		// Don't fail account creation if DNS fails
	}

	if err := s.createWelcomePage(req.Username, req.Domain, documentRoot); err != nil {
		log.Printf("Warning: failed to create welcome page: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	log.Printf("✅ Account created: %s (%s) - %s", req.Username, req.Email, req.Domain)

	return &Account{
		ID:          userID,
		Username:    req.Username,
		Email:       req.Email,
		Domain:      req.Domain,
		HomeDir:     homeDir,
		PackageID:   req.PackageID,
		PackageName: packageName,
		DiskQuota:   diskQuota,
		Active:      true,
	}, nil
}

// createSystemUser creates a Linux user (or simulates it)
func (s *Service) createSystemUser(username, homeDir string) error {
	if config.IsDevelopment() {
		log.Printf("🔧 [SIMÜLASYON] useradd -m -d %s -s /bin/bash %s", homeDir, username)
		return nil
	}

	// Production mode - create real Linux user
	if !s.cfg.IsLinux {
		return errors.New("system user creation only supported on Linux")
	}

	cmd := exec.Command("useradd", "-m", "-d", homeDir, "-s", "/bin/bash", username)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd failed: %s - %w", string(output), err)
	}

	return nil
}

// createDirectoryStructure creates the home directory structure
func (s *Service) createDirectoryStructure(username, homeDir string) error {
	dirs := []string{
		filepath.Join(homeDir, "public_html"),
		filepath.Join(homeDir, "logs"),
		filepath.Join(homeDir, "tmp"),
		filepath.Join(homeDir, "mail"),
		filepath.Join(homeDir, "ssl"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		log.Printf("📁 Directory created: %s", dir)
	}

	// Set home directory permissions (711) - Apache needs execute to traverse
	os.Chmod(homeDir, 0711)
	// Set public_html permissions (755) - Apache needs read access
	os.Chmod(filepath.Join(homeDir, "public_html"), 0755)

	// Create .htaccess for security
	htaccess := filepath.Join(homeDir, "public_html", ".htaccess")
	htaccessContent := `# Security headers
Options -Indexes
`
	os.WriteFile(htaccess, []byte(htaccessContent), 0644)

	if config.IsDevelopment() {
		log.Printf("🔧 [SIMÜLASYON] chown -R %s:%s %s", username, username, homeDir)
	} else if s.cfg.IsLinux {
		// Set ownership
		cmd := exec.Command("chown", "-R", username+":"+username, homeDir)
		cmd.Run()
	}

	return nil
}

// createWebServerVhost creates virtual host configuration using the configured web server driver
func (s *Service) createWebServerVhost(username, domain, homeDir, documentRoot string) error {
	// Get the appropriate web server driver
	driverType := webserver.DriverApache // Default: Apache (supports .htaccess)
	if s.cfg.WebServer == "nginx" {
		driverType = webserver.DriverNginx
	}

	driver := webserver.NewDriver(driverType, s.cfg.SimulateMode, s.cfg.SimulateBasePath)

	vhostConfig := webserver.VhostConfig{
		Domain:       domain,
		Aliases:      []string{fmt.Sprintf("www.%s", domain)},
		Username:     username,
		DocumentRoot: documentRoot,
		HomeDir:      homeDir,
		PHPVersion:   s.cfg.PHPVersion,
	}

	if err := driver.CreateVhost(vhostConfig); err != nil {
		return err
	}

	log.Printf("✅ %s vhost created for: %s", driver.Name(), domain)
	return nil
}

// createPHPFPMPool creates a PHP-FPM pool for the user
func (s *Service) createPHPFPMPool(username, homeDir string) error {
	manager := webserver.NewPHPFPMManager(s.cfg.SimulateMode, s.cfg.SimulateBasePath, s.cfg.PHPVersion)

	poolConfig := webserver.PHPFPMConfig{
		Username:   username,
		HomeDir:    homeDir,
		PHPVersion: s.cfg.PHPVersion,
	}

	if err := manager.CreatePool(poolConfig); err != nil {
		return err
	}

	log.Printf("✅ PHP-FPM pool created for: %s", username)
	return nil
}

// createDNSZone creates a DNS zone for the domain
func (s *Service) createDNSZone(domain string) error {
	dnsManager := dns.NewManager(s.cfg.SimulateMode, s.cfg.SimulateBasePath)

	// Get server IP (in production, this would be the actual server IP)
	serverIP := os.Getenv("SERVER_IP")
	if serverIP == "" {
		serverIP = "127.0.0.1" // Default for development
	}

	zoneConfig := dns.ZoneConfig{
		Domain:    domain,
		IPAddress: serverIP,
		TTL:       3600,
	}

	if err := dnsManager.CreateZone(zoneConfig); err != nil {
		return err
	}

	log.Printf("✅ DNS zone created for: %s", domain)
	return nil
}

// createWelcomePage creates a default index.html
func (s *Service) createWelcomePage(username, domain, documentRoot string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Hoş Geldiniz</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        h1 { font-size: 3rem; margin-bottom: 1rem; }
        p { font-size: 1.2rem; opacity: 0.9; }
        .domain { 
            font-family: monospace; 
            background: rgba(255,255,255,0.2); 
            padding: 0.5rem 1rem; 
            border-radius: 8px;
            margin-top: 1rem;
            display: inline-block;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎉 Tebrikler!</h1>
        <p>Website başarıyla oluşturuldu.</p>
        <div class="domain">%s</div>
        <p style="margin-top: 2rem; font-size: 0.9rem; opacity: 0.7;">
            Bu sayfayı değiştirmek için public_html klasörüne dosyalarınızı yükleyin.
        </p>
    </div>
</body>
</html>
`, domain, domain)

	indexPath := filepath.Join(documentRoot, "index.html")
	if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
		return err
	}

	// Set ownership to the user (not root)
	if !config.IsDevelopment() && s.cfg.IsLinux {
		exec.Command("chown", username+":"+username, indexPath).Run()
	}

	log.Printf("📝 Welcome page created: %s", indexPath)
	return nil
}

// ListAccounts returns all hosting accounts
func (s *Service) ListAccounts() ([]Account, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.username, u.email, u.active, u.created_at,
			   COALESCE(d.name, '') as domain,
			   COALESCE(p.id, 0) as package_id,
			   COALESCE(p.name, 'No Package') as package_name,
			   COALESCE(p.disk_quota, 0) as disk_quota
		FROM users u
		LEFT JOIN domains d ON d.user_id = u.id
		LEFT JOIN user_packages up ON up.user_id = u.id
		LEFT JOIN packages p ON p.id = up.package_id
		WHERE u.role = 'user'
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Username, &a.Email, &a.Active, &a.CreatedAt,
			&a.Domain, &a.PackageID, &a.PackageName, &a.DiskQuota); err != nil {
			continue
		}
		a.HomeDir = filepath.Join(s.cfg.HomeBaseDir, a.Username)
		accounts = append(accounts, a)
	}

	return accounts, nil
}

// DeleteAccount deletes a hosting account
func (s *Service) DeleteAccount(userID int64) error {
	// Get username first
	var username string
	err := s.db.QueryRow("SELECT username FROM users WHERE id = ? AND role = 'user'", userID).Scan(&username)
	if err != nil {
		return err
	}

	homeDir := filepath.Join(s.cfg.HomeBaseDir, username)

	// Get all domains BEFORE deleting from database
	var domains []string
	rows, err := s.db.Query("SELECT name FROM domains WHERE user_id = ?", userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var domainName string
			if rows.Scan(&domainName) == nil {
				domains = append(domains, domainName)
			}
		}
	}

	// Delete web server configs for all domains
	driverType := webserver.DriverApache
	if s.cfg.WebServer == "nginx" {
		driverType = webserver.DriverNginx
	}
	driver := webserver.NewDriver(driverType, s.cfg.SimulateMode, s.cfg.SimulateBasePath)

	for _, domainName := range domains {
		if err := driver.DeleteVhost(domainName); err != nil {
			log.Printf("Warning: failed to delete vhost for %s: %v", domainName, err)
		}
		// Delete DNS zone
		dnsManager := dns.NewManager(s.cfg.SimulateMode, s.cfg.SimulateBasePath)
		if err := dnsManager.DeleteZone(domainName); err != nil {
			log.Printf("Warning: failed to delete DNS zone for %s: %v", domainName, err)
		}
	}

	// Delete PHP-FPM pool
	phpfpm := webserver.NewPHPFPMManager(s.cfg.SimulateMode, s.cfg.SimulateBasePath, s.cfg.PHPVersion)
	if err := phpfpm.DeletePool(username); err != nil {
		log.Printf("Warning: failed to delete PHP-FPM pool for %s: %v", username, err)
	}

	// Delete from database
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM user_packages WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM domains WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM databases WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM email_accounts WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM users WHERE id = ?", userID)

	if err := tx.Commit(); err != nil {
		return err
	}

	// Delete system resources (Linux user)
	if config.IsDevelopment() {
		log.Printf(" [SIMÜLASYON] userdel -r %s", username)
		log.Printf(" [SIMÜLASYON] rm -rf %s", homeDir)
	} else if s.cfg.IsLinux {
		exec.Command("userdel", "-r", username).Run()
	}

	// Always try to delete the home directory
	os.RemoveAll(homeDir)

	log.Printf(" Account deleted: %s", username)
	return nil
}

// SuspendAccount suspends an account
func (s *Service) SuspendAccount(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET active = 0 WHERE id = ?", userID)
	return err
}

// UnsuspendAccount unsuspends an account
func (s *Service) UnsuspendAccount(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET active = 1 WHERE id = ?", userID)
	return err
}
