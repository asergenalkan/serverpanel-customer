package api

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/models"
	dnsService "github.com/asergenalkan/serverpanel/internal/services/dns"
	"github.com/gofiber/fiber/v2"
)

// Domain represents a domain with extended info
type Domain struct {
	ID             int64          `json:"id"`
	UserID         int64          `json:"user_id"`
	Username       string         `json:"username,omitempty"`
	Name           string         `json:"name"`
	DomainType     string         `json:"domain_type"`
	ParentDomainID sql.NullInt64  `json:"-"`
	ParentDomain   string         `json:"parent_domain,omitempty"`
	DocumentRoot   string         `json:"document_root"`
	PHPVersion     string         `json:"php_version"`
	SSLEnabled     bool           `json:"ssl_enabled"`
	SSLExpiry      sql.NullString `json:"ssl_expiry,omitempty"`
	Active         bool           `json:"active"`
	CreatedAt      string         `json:"created_at"`
	SubdomainCount int            `json:"subdomain_count,omitempty"`
}

// Subdomain represents a subdomain
type Subdomain struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	DomainID     int64  `json:"domain_id"`
	DomainName   string `json:"domain_name,omitempty"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	DocumentRoot string `json:"document_root"`
	RedirectURL  string `json:"redirect_url,omitempty"`
	RedirectType string `json:"redirect_type,omitempty"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at"`
}

// UserLimits represents user's package limits and current usage
type UserLimits struct {
	MaxDomains        int `json:"max_domains"`
	CurrentDomains    int `json:"current_domains"`
	MaxSubdomains     int `json:"max_subdomains"`
	CurrentSubdomains int `json:"current_subdomains"`
}

// ========== DOMAIN HANDLERS ==========

func (h *Handler) ListDomains(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT d.id, d.user_id, u.username, d.name, d.domain_type, d.parent_domain_id, 
			       d.document_root, d.php_version, d.ssl_enabled, d.ssl_expiry, d.active, d.created_at,
			       (SELECT COUNT(*) FROM subdomains WHERE domain_id = d.id) as subdomain_count
			FROM domains d
			JOIN users u ON d.user_id = u.id
			ORDER BY d.domain_type, d.name`
	} else {
		query = `
			SELECT d.id, d.user_id, u.username, d.name, d.domain_type, d.parent_domain_id,
			       d.document_root, d.php_version, d.ssl_enabled, d.ssl_expiry, d.active, d.created_at,
			       (SELECT COUNT(*) FROM subdomains WHERE domain_id = d.id) as subdomain_count
			FROM domains d
			JOIN users u ON d.user_id = u.id
			WHERE d.user_id = ?
			ORDER BY d.domain_type, d.name`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch domains",
		})
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.UserID, &d.Username, &d.Name, &d.DomainType, &d.ParentDomainID,
			&d.DocumentRoot, &d.PHPVersion, &d.SSLEnabled, &d.SSLExpiry, &d.Active, &d.CreatedAt, &d.SubdomainCount); err != nil {
			continue
		}
		// Get parent domain name if exists
		if d.ParentDomainID.Valid {
			h.db.QueryRow("SELECT name FROM domains WHERE id = ?", d.ParentDomainID.Int64).Scan(&d.ParentDomain)
		}
		domains = append(domains, d)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    domains,
	})
}

func (h *Handler) GetUserLimits(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	limits, err := h.getUserLimits(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to get user limits",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    limits,
	})
}

func (h *Handler) getUserLimits(userID int64) (*UserLimits, error) {
	var limits UserLimits

	// Get package limits
	err := h.db.QueryRow(`
		SELECT COALESCE(p.max_domains, 1)
		FROM users u
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE u.id = ?
	`, userID).Scan(&limits.MaxDomains)
	if err != nil {
		limits.MaxDomains = 1 // Default
	}

	// Subdomain limit (10x domain limit as default)
	limits.MaxSubdomains = limits.MaxDomains * 10

	// Get current usage
	h.db.QueryRow("SELECT COUNT(*) FROM domains WHERE user_id = ?", userID).Scan(&limits.CurrentDomains)
	h.db.QueryRow("SELECT COUNT(*) FROM subdomains WHERE user_id = ?", userID).Scan(&limits.CurrentSubdomains)

	return &limits, nil
}

func (h *Handler) CreateDomain(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	username := c.Locals("username").(string)
	role := c.Locals("role").(string)

	var req struct {
		Name         string `json:"name"`
		DomainType   string `json:"domain_type"`
		DocumentRoot string `json:"document_root"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate domain name
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if !isValidDomain(req.Name) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain adı",
		})
	}

	// Set default type
	if req.DomainType == "" {
		req.DomainType = "addon"
	}

	// Check limits (unless admin)
	if role != models.RoleAdmin {
		limits, _ := h.getUserLimits(userID)
		if limits.CurrentDomains >= limits.MaxDomains {
			return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Domain limitinize ulaştınız (%d/%d)", limits.CurrentDomains, limits.MaxDomains),
			})
		}
	}

	// Check if domain exists
	var count int
	h.db.QueryRow("SELECT COUNT(*) FROM domains WHERE name = ?", req.Name).Scan(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain zaten kayıtlı",
		})
	}

	// Set document root
	if req.DocumentRoot == "" {
		req.DocumentRoot = fmt.Sprintf("/home/%s/public_html/%s", username, req.Name)
	}

	// Insert domain
	result, err := h.db.Exec(`
		INSERT INTO domains (user_id, name, domain_type, document_root, active)
		VALUES (?, ?, ?, ?, 1)
	`, userID, req.Name, req.DomainType, req.DocumentRoot)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain eklenemedi",
		})
	}

	domainID, _ := result.LastInsertId()

	// Create system resources (vhost, DNS zone, directory)
	go h.createDomainResources(username, req.Name, req.DocumentRoot)

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Domain başarıyla eklendi",
		Data:    map[string]int64{"id": domainID},
	})
}

func (h *Handler) GetDomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var d Domain
	err = h.db.QueryRow(`
		SELECT d.id, d.user_id, u.username, d.name, d.domain_type, d.parent_domain_id,
		       d.document_root, d.php_version, d.ssl_enabled, d.ssl_expiry, d.active, d.created_at
		FROM domains d
		JOIN users u ON d.user_id = u.id
		WHERE d.id = ?
	`, id).Scan(&d.ID, &d.UserID, &d.Username, &d.Name, &d.DomainType, &d.ParentDomainID,
		&d.DocumentRoot, &d.PHPVersion, &d.SSLEnabled, &d.SSLExpiry, &d.Active, &d.CreatedAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && d.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain'e erişim yetkiniz yok",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    d,
	})
}

func (h *Handler) UpdateDomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Check ownership
	var domainUserID int64
	h.db.QueryRow("SELECT user_id FROM domains WHERE id = ?", id).Scan(&domainUserID)
	if role != models.RoleAdmin && domainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain'i düzenleme yetkiniz yok",
		})
	}

	var req struct {
		DocumentRoot string `json:"document_root"`
		PHPVersion   string `json:"php_version"`
		Active       *bool  `json:"active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}

	if req.DocumentRoot != "" {
		updates = append(updates, "document_root = ?")
		args = append(args, req.DocumentRoot)
	}
	if req.PHPVersion != "" {
		updates = append(updates, "php_version = ?")
		args = append(args, req.PHPVersion)
	}
	if req.Active != nil {
		updates = append(updates, "active = ?")
		args = append(args, *req.Active)
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Güncellenecek alan belirtilmedi",
		})
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE domains SET %s WHERE id = ?", strings.Join(updates, ", "))

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain güncellenemedi",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Domain güncellendi",
	})
}

func (h *Handler) DeleteDomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	// Check if files should be deleted
	deleteFiles := c.Query("delete_files") == "true"

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info
	var domainUserID int64
	var domainName, domainType, username, documentRoot string
	err = h.db.QueryRow(`
		SELECT d.user_id, d.name, d.domain_type, u.username, COALESCE(d.document_root, '')
		FROM domains d 
		JOIN users u ON d.user_id = u.id 
		WHERE d.id = ?
	`, id).Scan(&domainUserID, &domainName, &domainType, &username, &documentRoot)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && domainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain'i silme yetkiniz yok",
		})
	}

	// Don't allow deleting primary domain
	if domainType == "primary" {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Ana domain silinemez. Hesabı silmeniz gerekir.",
		})
	}

	// Delete from database (subdomains will be cascade deleted)
	_, err = h.db.Exec("DELETE FROM domains WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain silinemedi",
		})
	}

	// Remove system resources
	go h.removeDomainResources(username, domainName)

	// Delete document root if requested
	if deleteFiles && documentRoot != "" {
		go func() {
			if !config.IsDevelopment() {
				// Safety check: only delete if path is under /home
				if strings.HasPrefix(documentRoot, "/home/") {
					if err := os.RemoveAll(documentRoot); err != nil {
						log.Printf("❌ Domain dizini silinemedi: %s - %v", documentRoot, err)
					} else {
						log.Printf("✅ Domain dizini silindi: %s", documentRoot)
					}
				}
			} else {
				log.Printf("🔧 [DEV] Domain dizini silinecek: %s", documentRoot)
			}
		}()
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Domain silindi",
	})
}

// ========== SUBDOMAIN HANDLERS ==========

func (h *Handler) ListSubdomains(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT s.id, s.user_id, s.domain_id, d.name as domain_name, s.name, s.full_name,
			       s.document_root, s.redirect_url, s.redirect_type, s.active, s.created_at
			FROM subdomains s
			JOIN domains d ON s.domain_id = d.id
			ORDER BY s.full_name`
	} else {
		query = `
			SELECT s.id, s.user_id, s.domain_id, d.name as domain_name, s.name, s.full_name,
			       s.document_root, s.redirect_url, s.redirect_type, s.active, s.created_at
			FROM subdomains s
			JOIN domains d ON s.domain_id = d.id
			WHERE s.user_id = ?
			ORDER BY s.full_name`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch subdomains",
		})
	}
	defer rows.Close()

	var subdomains []Subdomain
	for rows.Next() {
		var s Subdomain
		var redirectURL, redirectType sql.NullString
		if err := rows.Scan(&s.ID, &s.UserID, &s.DomainID, &s.DomainName, &s.Name, &s.FullName,
			&s.DocumentRoot, &redirectURL, &redirectType, &s.Active, &s.CreatedAt); err != nil {
			continue
		}
		if redirectURL.Valid {
			s.RedirectURL = redirectURL.String
		}
		if redirectType.Valid {
			s.RedirectType = redirectType.String
		}
		subdomains = append(subdomains, s)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    subdomains,
	})
}

func (h *Handler) CreateSubdomain(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	username := c.Locals("username").(string)
	role := c.Locals("role").(string)

	var req struct {
		DomainID     int64  `json:"domain_id"`
		Name         string `json:"name"`
		DocumentRoot string `json:"document_root"`
		RedirectURL  string `json:"redirect_url"`
		RedirectType string `json:"redirect_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate subdomain name
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if !isValidSubdomain(req.Name) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz subdomain adı (sadece harf, rakam ve tire kullanılabilir)",
		})
	}

	// Get domain info and check ownership
	var domainUserID int64
	var domainName string
	err := h.db.QueryRow("SELECT user_id, name FROM domains WHERE id = ?", req.DomainID).Scan(&domainUserID, &domainName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	if role != models.RoleAdmin && domainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain'e subdomain ekleme yetkiniz yok",
		})
	}

	// Check limits
	if role != models.RoleAdmin {
		limits, _ := h.getUserLimits(userID)
		if limits.CurrentSubdomains >= limits.MaxSubdomains {
			return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Subdomain limitinize ulaştınız (%d/%d)", limits.CurrentSubdomains, limits.MaxSubdomains),
			})
		}
	}

	// Build full name
	fullName := fmt.Sprintf("%s.%s", req.Name, domainName)

	// Check if subdomain exists
	var count int
	h.db.QueryRow("SELECT COUNT(*) FROM subdomains WHERE full_name = ?", fullName).Scan(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu subdomain zaten kayıtlı",
		})
	}

	// Set document root
	if req.DocumentRoot == "" && req.RedirectURL == "" {
		req.DocumentRoot = fmt.Sprintf("/home/%s/public_html/%s", username, fullName)
	}

	// Insert subdomain
	result, err := h.db.Exec(`
		INSERT INTO subdomains (user_id, domain_id, name, full_name, document_root, redirect_url, redirect_type, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`, userID, req.DomainID, req.Name, fullName, req.DocumentRoot, nullString(req.RedirectURL), nullString(req.RedirectType))

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Subdomain eklenemedi",
		})
	}

	subdomainID, _ := result.LastInsertId()

	// Create system resources
	go h.createSubdomainResources(username, fullName, req.DocumentRoot, req.RedirectURL, req.RedirectType)

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Subdomain başarıyla eklendi",
		Data:    map[string]interface{}{"id": subdomainID, "full_name": fullName},
	})
}

func (h *Handler) DeleteSubdomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz subdomain ID",
		})
	}

	// Check if files should be deleted
	deleteFiles := c.Query("delete_files") == "true"

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get subdomain info
	var subdomainUserID int64
	var fullName, username, documentRoot string
	err = h.db.QueryRow(`
		SELECT s.user_id, s.full_name, u.username, COALESCE(s.document_root, '')
		FROM subdomains s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = ?
	`, id).Scan(&subdomainUserID, &fullName, &username, &documentRoot)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Subdomain bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && subdomainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu subdomain'i silme yetkiniz yok",
		})
	}

	// Delete from database
	_, err = h.db.Exec("DELETE FROM subdomains WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Subdomain silinemedi",
		})
	}

	// Remove system resources
	go h.removeSubdomainResources(username, fullName)

	// Delete document root if requested
	if deleteFiles && documentRoot != "" {
		go func() {
			if !config.IsDevelopment() {
				// Safety check: only delete if path is under /home
				if strings.HasPrefix(documentRoot, "/home/") {
					if err := os.RemoveAll(documentRoot); err != nil {
						log.Printf("❌ Subdomain dizini silinemedi: %s - %v", documentRoot, err)
					} else {
						log.Printf("✅ Subdomain dizini silindi: %s", documentRoot)
					}
				}
			} else {
				log.Printf("🔧 [DEV] Subdomain dizini silinecek: %s", documentRoot)
			}
		}()
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Subdomain silindi",
	})
}

// ========== HELPER FUNCTIONS ==========

func isValidDomain(domain string) bool {
	matched, _ := regexp.MatchString(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}$`, domain)
	return matched
}

func isValidSubdomain(name string) bool {
	if len(name) < 1 || len(name) > 63 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, name)
	return matched
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// createDomainResources creates Apache vhost, DNS zone, and directory
func (h *Handler) createDomainResources(username, domain, documentRoot string) {
	cfg := config.Get()
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Domain kaynakları oluşturulacak: %s -> %s", domain, documentRoot)
		return
	}

	// Create directory
	if err := os.MkdirAll(documentRoot, 0755); err != nil {
		log.Printf("❌ Dizin oluşturulamadı: %v", err)
	} else {
		// Set ownership
		exec.Command("chown", "-R", fmt.Sprintf("%s:%s", username, username), documentRoot).Run()
	}

	// Create self-signed SSL certificate for immediate HTTPS support (Cloudflare compatible)
	sslDir := fmt.Sprintf("/etc/ssl/serverpanel/%s", domain)
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		log.Printf("⚠️ SSL dizini oluşturulamadı: %v", err)
	}

	certPath := filepath.Join(sslDir, "cert.pem")
	keyPath := filepath.Join(sslDir, "key.pem")

	// Generate self-signed certificate (valid for 10 years)
	opensslCmd := exec.Command("openssl", "req", "-x509", "-nodes", "-days", "3650",
		"-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-subj", fmt.Sprintf("/CN=%s/O=ServerPanel/C=TR", domain),
		"-addext", fmt.Sprintf("subjectAltName=DNS:%s,DNS:www.%s", domain, domain),
	)
	if err := opensslCmd.Run(); err != nil {
		log.Printf("⚠️ Self-signed SSL oluşturulamadı: %v", err)
	} else {
		log.Printf("✅ Self-signed SSL oluşturuldu: %s", domain)
	}

	// Create Apache vhost (HTTP + HTTPS)
	vhostContent := fmt.Sprintf(`# HTTP VirtualHost
<VirtualHost *:80>
    ServerName %s
    ServerAlias www.%s
    DocumentRoot %s
    
    <Directory %s>
        AllowOverride All
        Require all granted
    </Directory>
    
    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm.sock|fcgi://localhost"
    </FilesMatch>
    
    ErrorLog ${APACHE_LOG_DIR}/%s-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-access.log combined
</VirtualHost>

# HTTPS VirtualHost (Self-Signed - Cloudflare Full SSL Compatible)
<VirtualHost *:443>
    ServerName %s
    ServerAlias www.%s
    DocumentRoot %s
    
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    <Directory %s>
        AllowOverride All
        Require all granted
    </Directory>
    
    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm.sock|fcgi://localhost"
    </FilesMatch>
    
    # Security Headers
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-Content-Type-Options "nosniff"
    
    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, domain, domain, documentRoot, documentRoot, cfg.PHPVersion, domain, domain,
		domain, domain, documentRoot, certPath, keyPath, documentRoot, cfg.PHPVersion, domain, domain)

	vhostPath := fmt.Sprintf("/etc/apache2/sites-available/%s.conf", domain)
	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		log.Printf("❌ Vhost oluşturulamadı: %v", err)
	} else {
		// Enable SSL module
		exec.Command("a2enmod", "ssl").Run()
		exec.Command("a2enmod", "headers").Run()
		exec.Command("a2ensite", domain+".conf").Run()
		exec.Command("systemctl", "reload", "apache2").Run()
		log.Printf("✅ Apache vhost oluşturuldu (HTTP+HTTPS): %s", domain)
	}

	// Create DNS zone
	dnsManager := dnsService.NewManager(cfg.SimulateMode, cfg.SimulateBasePath)
	zoneConfig := dnsService.ZoneConfig{
		Domain:    domain,
		IPAddress: cfg.ServerIP,
	}
	if err := dnsManager.CreateZone(zoneConfig); err != nil {
		log.Printf("⚠️ DNS zone oluşturulamadı: %v", err)
	} else {
		log.Printf("✅ DNS zone oluşturuldu: %s", domain)
	}

	// Create welcome page
	welcomeHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; background: #f5f5f5; }
        .container { background: white; padding: 40px; border-radius: 10px; max-width: 600px; margin: 0 auto; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <p>Bu domain başarıyla yapılandırıldı.</p>
        <p>Dosyalarınızı yükleyerek sitenizi yayınlayabilirsiniz.</p>
    </div>
</body>
</html>
`, domain, domain)
	indexPath := filepath.Join(documentRoot, "index.html")
	os.WriteFile(indexPath, []byte(welcomeHTML), 0644)
	exec.Command("chown", fmt.Sprintf("%s:%s", username, username), indexPath).Run()
}

func (h *Handler) removeDomainResources(username, domain string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Domain kaynakları silinecek: %s", domain)
		return
	}

	// Disable and remove Apache vhost
	exec.Command("a2dissite", domain+".conf").Run()
	os.Remove(fmt.Sprintf("/etc/apache2/sites-available/%s.conf", domain))
	exec.Command("systemctl", "reload", "apache2").Run()

	// Remove self-signed SSL certificate
	sslDir := fmt.Sprintf("/etc/ssl/serverpanel/%s", domain)
	os.RemoveAll(sslDir)
	log.Printf("✅ SSL sertifikası silindi: %s", domain)

	// Remove DNS zone
	cfg := config.Get()
	dnsManager := dnsService.NewManager(cfg.SimulateMode, cfg.SimulateBasePath)
	dnsManager.DeleteZone(domain)

	log.Printf("✅ Domain kaynakları silindi: %s", domain)
}

func (h *Handler) createSubdomainResources(username, fullName, documentRoot, redirectURL, redirectType string) {
	cfg := config.Get()
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Subdomain kaynakları oluşturulacak: %s", fullName)
		return
	}

	var vhostContent string

	if redirectURL != "" {
		// Redirect subdomain
		redirectCode := "301"
		if redirectType == "302" {
			redirectCode = "302"
		}
		vhostContent = fmt.Sprintf(`<VirtualHost *:80>
    ServerName %s
    Redirect %s / %s
</VirtualHost>
`, fullName, redirectCode, redirectURL)
	} else {
		// Normal subdomain with document root
		if err := os.MkdirAll(documentRoot, 0755); err != nil {
			log.Printf("❌ Subdomain dizini oluşturulamadı: %v", err)
		} else {
			exec.Command("chown", "-R", fmt.Sprintf("%s:%s", username, username), documentRoot).Run()
		}

		// Create self-signed SSL for subdomain (Cloudflare compatible)
		sslDir := fmt.Sprintf("/etc/ssl/serverpanel/%s", fullName)
		os.MkdirAll(sslDir, 0755)
		certPath := filepath.Join(sslDir, "cert.pem")
		keyPath := filepath.Join(sslDir, "key.pem")

		opensslCmd := exec.Command("openssl", "req", "-x509", "-nodes", "-days", "3650",
			"-newkey", "rsa:2048",
			"-keyout", keyPath,
			"-out", certPath,
			"-subj", fmt.Sprintf("/CN=%s/O=ServerPanel/C=TR", fullName),
		)
		if err := opensslCmd.Run(); err != nil {
			log.Printf("⚠️ Subdomain SSL oluşturulamadı: %v", err)
		}

		vhostContent = fmt.Sprintf(`# HTTP VirtualHost
<VirtualHost *:80>
    ServerName %s
    DocumentRoot %s
    
    <Directory %s>
        AllowOverride All
        Require all granted
    </Directory>
    
    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm.sock|fcgi://localhost"
    </FilesMatch>
    
    ErrorLog ${APACHE_LOG_DIR}/%s-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-access.log combined
</VirtualHost>

# HTTPS VirtualHost (Self-Signed - Cloudflare Full SSL Compatible)
<VirtualHost *:443>
    ServerName %s
    DocumentRoot %s
    
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    <Directory %s>
        AllowOverride All
        Require all granted
    </Directory>
    
    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm.sock|fcgi://localhost"
    </FilesMatch>
    
    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, fullName, documentRoot, documentRoot, cfg.PHPVersion, fullName, fullName,
			fullName, documentRoot, certPath, keyPath, documentRoot, cfg.PHPVersion, fullName, fullName)

		// Create welcome page (same design as main domain)
		welcomeHTML := fmt.Sprintf(`<!DOCTYPE html>
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
            Bu sayfayı değiştirmek için dosyalarınızı yükleyin.
        </p>
    </div>
</body>
</html>
`, fullName, fullName)
		indexPath := filepath.Join(documentRoot, "index.html")
		os.WriteFile(indexPath, []byte(welcomeHTML), 0644)
		exec.Command("chown", fmt.Sprintf("%s:%s", username, username), indexPath).Run()
	}

	vhostPath := fmt.Sprintf("/etc/apache2/sites-available/%s.conf", fullName)
	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		log.Printf("❌ Subdomain vhost oluşturulamadı: %v", err)
	} else {
		exec.Command("a2ensite", fullName+".conf").Run()
		exec.Command("systemctl", "reload", "apache2").Run()
		log.Printf("✅ Subdomain vhost oluşturuldu: %s", fullName)
	}

	// Add DNS A record for subdomain
	dnsManager := dnsService.NewManager(cfg.SimulateMode, cfg.SimulateBasePath)
	// Extract main domain from fullName (e.g., "blog.example.com" -> "example.com")
	parts := strings.SplitN(fullName, ".", 2)
	if len(parts) == 2 {
		mainDomain := parts[1]
		subdomain := parts[0]
		dnsManager.AddRecord(mainDomain, "A", subdomain, cfg.ServerIP, 3600)
	}
}

func (h *Handler) removeSubdomainResources(username, fullName string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Subdomain kaynakları silinecek: %s", fullName)
		return
	}

	// Disable and remove Apache vhost
	exec.Command("a2dissite", fullName+".conf").Run()
	os.Remove(fmt.Sprintf("/etc/apache2/sites-available/%s.conf", fullName))
	exec.Command("systemctl", "reload", "apache2").Run()

	// Remove self-signed SSL certificate
	sslDir := fmt.Sprintf("/etc/ssl/serverpanel/%s", fullName)
	os.RemoveAll(sslDir)

	log.Printf("✅ Subdomain kaynakları silindi: %s", fullName)
}
