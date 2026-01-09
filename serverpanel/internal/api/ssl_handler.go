package api

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// SSLCertificate represents an SSL certificate
type SSLCertificate struct {
	ID           int64     `json:"id"`
	DomainID     int64     `json:"domain_id"`
	SubdomainID  int64     `json:"subdomain_id,omitempty"`
	Domain       string    `json:"domain"`
	DomainType   string    `json:"domain_type"` // domain, subdomain, mail, www
	ParentDomain string    `json:"parent_domain,omitempty"`
	Issuer       string    `json:"issuer"`
	Status       string    `json:"status"` // active, expired, pending, none, error
	StatusDetail string    `json:"status_detail,omitempty"`
	ValidFrom    time.Time `json:"valid_from"`
	ValidUntil   time.Time `json:"valid_until"`
	AutoRenew    bool      `json:"auto_renew"`
	CertPath     string    `json:"cert_path,omitempty"`
	KeyPath      string    `json:"key_path,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListSSLCertificates returns all SSL certificates for the user
// Includes domains, subdomains, and standard subdomains (www, mail)
func (h *Handler) ListSSLCertificates(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var certificates []SSLCertificate

	// 1. Get all domains for this user
	var domainQuery string
	var domainArgs []interface{}

	if role == models.RoleAdmin {
		domainQuery = `SELECT d.id, d.name, d.user_id FROM domains d ORDER BY d.name`
	} else {
		domainQuery = `SELECT d.id, d.name, d.user_id FROM domains d WHERE d.user_id = ? ORDER BY d.name`
		domainArgs = append(domainArgs, userID)
	}

	domainRows, err := h.db.Query(domainQuery, domainArgs...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch domains",
		})
	}
	defer domainRows.Close()

	type domainInfo struct {
		ID     int64
		Name   string
		UserID int64
	}
	var domains []domainInfo

	for domainRows.Next() {
		var d domainInfo
		if err := domainRows.Scan(&d.ID, &d.Name, &d.UserID); err != nil {
			continue
		}
		domains = append(domains, d)
	}

	// 2. For each domain, add the domain itself and standard subdomains
	for _, domain := range domains {
		// Main domain
		cert := h.buildSSLCertificate(domain.ID, 0, domain.Name, "domain", "")
		certificates = append(certificates, cert)

		// www subdomain
		wwwCert := h.buildSSLCertificate(domain.ID, 0, "www."+domain.Name, "www", domain.Name)
		certificates = append(certificates, wwwCert)

		// mail subdomain
		mailCert := h.buildSSLCertificate(domain.ID, 0, "mail."+domain.Name, "mail", domain.Name)
		certificates = append(certificates, mailCert)

		// webmail subdomain
		webmailCert := h.buildSSLCertificate(domain.ID, 0, "webmail."+domain.Name, "webmail", domain.Name)
		certificates = append(certificates, webmailCert)

		// ftp subdomain
		ftpCert := h.buildSSLCertificate(domain.ID, 0, "ftp."+domain.Name, "ftp", domain.Name)
		certificates = append(certificates, ftpCert)
	}

	// 3. Get all subdomains for this user
	var subdomainQuery string
	var subdomainArgs []interface{}

	if role == models.RoleAdmin {
		subdomainQuery = `SELECT s.id, s.domain_id, s.full_name, d.name 
			FROM subdomains s 
			JOIN domains d ON s.domain_id = d.id 
			ORDER BY s.full_name`
	} else {
		subdomainQuery = `SELECT s.id, s.domain_id, s.full_name, d.name 
			FROM subdomains s 
			JOIN domains d ON s.domain_id = d.id 
			WHERE s.user_id = ? 
			ORDER BY s.full_name`
		subdomainArgs = append(subdomainArgs, userID)
	}

	subdomainRows, err := h.db.Query(subdomainQuery, subdomainArgs...)
	if err == nil {
		defer subdomainRows.Close()

		for subdomainRows.Next() {
			var subdomainID, domainID int64
			var fullName, parentDomain string
			if err := subdomainRows.Scan(&subdomainID, &domainID, &fullName, &parentDomain); err != nil {
				continue
			}

			// Subdomain
			cert := h.buildSSLCertificate(domainID, subdomainID, fullName, "subdomain", parentDomain)
			certificates = append(certificates, cert)

			// www for subdomain (optional, but cPanel shows it)
			wwwCert := h.buildSSLCertificate(domainID, subdomainID, "www."+fullName, "www", fullName)
			certificates = append(certificates, wwwCert)
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    certificates,
	})
}

// buildSSLCertificate creates an SSLCertificate with status info
func (h *Handler) buildSSLCertificate(domainID, subdomainID int64, fqdn, domainType, parentDomain string) SSLCertificate {
	cert := SSLCertificate{
		DomainID:     domainID,
		SubdomainID:  subdomainID,
		Domain:       fqdn,
		DomainType:   domainType,
		ParentDomain: parentDomain,
		AutoRenew:    true,
	}

	// Check if certificate exists for this FQDN
	certInfo := h.getCertificateInfo(fqdn)
	if certInfo != nil {
		cert.Issuer = certInfo.Issuer
		cert.ValidFrom = certInfo.ValidFrom
		cert.ValidUntil = certInfo.ValidUntil
		cert.CertPath = certInfo.CertPath
		cert.KeyPath = certInfo.KeyPath

		if time.Now().After(certInfo.ValidUntil) {
			cert.Status = "expired"
			cert.StatusDetail = "Sertifika süresi dolmuş"
		} else if time.Now().Before(certInfo.ValidFrom) {
			cert.Status = "pending"
			cert.StatusDetail = "Sertifika henüz aktif değil"
		} else {
			cert.Status = "active"
			daysLeft := int(time.Until(certInfo.ValidUntil).Hours() / 24)
			cert.StatusDetail = fmt.Sprintf("Sertifika geçerli, %d gün kaldı", daysLeft)
		}
	} else {
		// Check if covered by parent domain's wildcard or SAN certificate
		if parentDomain != "" {
			parentCertInfo := h.getCertificateInfo(parentDomain)
			if parentCertInfo != nil && h.isCoveredByCert(fqdn, parentDomain) {
				cert.Issuer = parentCertInfo.Issuer
				cert.ValidFrom = parentCertInfo.ValidFrom
				cert.ValidUntil = parentCertInfo.ValidUntil
				cert.Status = "active"
				cert.StatusDetail = fmt.Sprintf("Ana domain sertifikası ile korunuyor (%s)", parentDomain)
			} else {
				cert.Status = "none"
				cert.StatusDetail = "SSL sertifikası yok"
			}
		} else {
			cert.Status = "none"
			cert.StatusDetail = "SSL sertifikası yok"
		}
	}

	return cert
}

// isCoveredByCert checks if a subdomain is covered by parent's certificate
func (h *Handler) isCoveredByCert(subdomain, parentDomain string) bool {
	certPath := filepath.Join("/etc/letsencrypt/live", parentDomain, "fullchain.pem")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	// Check if subdomain matches any SAN
	for _, san := range cert.DNSNames {
		if san == subdomain {
			return true
		}
		// Check wildcard
		if strings.HasPrefix(san, "*.") {
			wildcardDomain := san[2:]
			if strings.HasSuffix(subdomain, "."+wildcardDomain) || subdomain == wildcardDomain {
				return true
			}
		}
	}

	return false
}

// GetSSLCertificate returns SSL certificate details for a domain
func (h *Handler) GetSSLCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	currentUserID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain
	var domain string
	var ownerID int64
	err = h.db.QueryRow("SELECT name, user_id FROM domains WHERE id = ?", domainID).Scan(&domain, &ownerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && currentUserID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	cert := SSLCertificate{
		DomainID:  domainID,
		Domain:    domain,
		AutoRenew: true,
	}

	certInfo := h.getCertificateInfo(domain)
	if certInfo != nil {
		cert.Issuer = certInfo.Issuer
		cert.ValidFrom = certInfo.ValidFrom
		cert.ValidUntil = certInfo.ValidUntil
		cert.CertPath = certInfo.CertPath
		cert.KeyPath = certInfo.KeyPath

		if time.Now().After(certInfo.ValidUntil) {
			cert.Status = "expired"
		} else {
			cert.Status = "active"
		}
	} else {
		cert.Status = "none"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    cert,
	})
}

// IssueSSLCertificate issues a new Let's Encrypt certificate
func (h *Handler) IssueSSLCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	currentUserID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info
	var domain, username string
	var ownerID int64
	err = h.db.QueryRow(`
		SELECT d.name, d.user_id, u.username 
		FROM domains d 
		JOIN users u ON d.user_id = u.id 
		WHERE d.id = ?`, domainID).Scan(&domain, &ownerID, &username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && currentUserID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Determine webroot
	webRoot := filepath.Join("/home", username, "public_html")
	if _, err := os.Stat(webRoot); os.IsNotExist(err) {
		webRoot = "/var/www/html"
	}

	// Get email
	var email string
	h.db.QueryRow("SELECT email FROM users WHERE id = ?", ownerID).Scan(&email)
	if email == "" {
		email = "admin@" + domain
	}

	// Issue certificate using certbot
	certInfo, err := h.issueCertificate(domain, webRoot, email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to issue certificate: " + err.Error(),
		})
	}

	// Update Apache/Nginx config to use SSL
	if err := h.configureSSLVhost(domain, username, certInfo); err != nil {
		// Log but don't fail - certificate was issued
		// log.Printf("Warning: Failed to configure SSL vhost: %v", err)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "SSL certificate issued successfully",
		Data: SSLCertificate{
			DomainID:   domainID,
			Domain:     domain,
			Issuer:     certInfo.Issuer,
			Status:     "active",
			ValidFrom:  certInfo.ValidFrom,
			ValidUntil: certInfo.ValidUntil,
			AutoRenew:  true,
			CertPath:   certInfo.CertPath,
			KeyPath:    certInfo.KeyPath,
		},
	})
}

// IssueSSLForFQDN issues SSL certificate for any FQDN (subdomain, mail, www, etc.)
func (h *Handler) IssueSSLForFQDN(c *fiber.Ctx) error {
	var req struct {
		FQDN       string `json:"fqdn"`
		DomainID   int64  `json:"domain_id"`
		DomainType string `json:"domain_type"` // subdomain, www, mail
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.FQDN == "" || req.DomainID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "FQDN and domain_id are required",
		})
	}

	currentUserID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get parent domain info for permission check
	var parentDomain, username string
	var ownerID int64
	err := h.db.QueryRow(`
		SELECT d.name, d.user_id, u.username 
		FROM domains d 
		JOIN users u ON d.user_id = u.id 
		WHERE d.id = ?`, req.DomainID).Scan(&parentDomain, &ownerID, &username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Parent domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && currentUserID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Determine webroot based on domain type
	var webRoot string
	switch req.DomainType {
	case "webmail", "mail", "ftp":
		// System subdomains always use /var/www/html for ACME challenge
		webRoot = "/var/www/html"
	case "subdomain":
		// Check if subdomain exists in database
		var subdomainDocRoot string
		err := h.db.QueryRow("SELECT document_root FROM subdomains WHERE full_name = ?", req.FQDN).Scan(&subdomainDocRoot)
		if err == nil && subdomainDocRoot != "" {
			webRoot = subdomainDocRoot
		} else {
			webRoot = filepath.Join("/home", username, "public_html", req.FQDN)
		}
		if _, err := os.Stat(webRoot); os.IsNotExist(err) {
			webRoot = "/var/www/html"
		}
	default:
		// www - use parent domain's webroot
		webRoot = filepath.Join("/home", username, "public_html")
		if _, err := os.Stat(webRoot); os.IsNotExist(err) {
			webRoot = "/var/www/html"
		}
	}

	// Get email
	var email string
	h.db.QueryRow("SELECT email FROM users WHERE id = ?", ownerID).Scan(&email)
	if email == "" {
		email = "admin@" + parentDomain
	}

	// Issue certificate for the FQDN
	certInfo, err := h.issueCertificateForFQDN(req.FQDN, webRoot, email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to issue certificate: " + err.Error(),
		})
	}

	// Configure SSL vhost based on domain type
	switch req.DomainType {
	case "subdomain":
		h.configureSSLVhostForFQDN(req.FQDN, username, webRoot, certInfo)
	case "webmail":
		h.configureSSLVhostForWebmail(req.FQDN, certInfo)
	case "mail":
		h.configureSSLVhostForMail(req.FQDN, certInfo)
	case "ftp":
		h.configureSSLVhostForFTP(req.FQDN, certInfo)
	case "www":
		// www uses the same vhost as main domain, just update SSL
		h.configureSSLVhostForWWW(req.FQDN, username, certInfo)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("SSL certificate issued for %s", req.FQDN),
		Data: SSLCertificate{
			DomainID:   req.DomainID,
			Domain:     req.FQDN,
			DomainType: req.DomainType,
			Issuer:     certInfo.Issuer,
			Status:     "active",
			ValidFrom:  certInfo.ValidFrom,
			ValidUntil: certInfo.ValidUntil,
			AutoRenew:  true,
			CertPath:   certInfo.CertPath,
			KeyPath:    certInfo.KeyPath,
		},
	})
}

// RenewSSLCertificate renews an existing certificate
func (h *Handler) RenewSSLCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	currentUserID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain
	var domain string
	var ownerID int64
	err = h.db.QueryRow("SELECT name, user_id FROM domains WHERE id = ?", domainID).Scan(&domain, &ownerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && currentUserID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Renew certificate
	if err := h.renewCertificate(domain); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to renew certificate: " + err.Error(),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "SSL certificate renewed successfully",
	})
}

// RevokeSSLCertificate revokes and removes a certificate
func (h *Handler) RevokeSSLCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid domain ID",
		})
	}

	currentUserID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain
	var domain, username string
	var ownerID int64
	err = h.db.QueryRow(`
		SELECT d.name, d.user_id, u.username 
		FROM domains d 
		JOIN users u ON d.user_id = u.id 
		WHERE d.id = ?`, domainID).Scan(&domain, &ownerID, &username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain not found",
		})
	}

	// Check permission
	if role != models.RoleAdmin && currentUserID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Access denied",
		})
	}

	// Revoke certificate
	if err := h.revokeCertificate(domain); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to revoke certificate: " + err.Error(),
		})
	}

	// Remove SSL from vhost
	h.removeSSLVhost(domain, username)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "SSL certificate revoked successfully",
	})
}

// Helper functions

type certInfo struct {
	Issuer     string
	ValidFrom  time.Time
	ValidUntil time.Time
	CertPath   string
	KeyPath    string
}

func (h *Handler) getCertificateInfo(domain string) *certInfo {
	certPath := filepath.Join("/etc/letsencrypt/live", domain, "fullchain.pem")
	keyPath := filepath.Join("/etc/letsencrypt/live", domain, "privkey.pem")

	// Check if certificate exists
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil
	}

	// Read and parse certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		// Return basic info even if parsing fails
		return &certInfo{
			Issuer:     "Let's Encrypt",
			ValidFrom:  time.Now().AddDate(0, -1, 0),
			ValidUntil: time.Now().AddDate(0, 2, 0),
			CertPath:   certPath,
			KeyPath:    keyPath,
		}
	}

	issuer := "Let's Encrypt"
	if len(cert.Issuer.Organization) > 0 {
		issuer = cert.Issuer.Organization[0]
	}

	return &certInfo{
		Issuer:     issuer,
		ValidFrom:  cert.NotBefore,
		ValidUntil: cert.NotAfter,
		CertPath:   certPath,
		KeyPath:    keyPath,
	}
}

func (h *Handler) issueCertificate(domain, webRoot, email string) (*certInfo, error) {
	// Build certbot command
	args := []string{
		"certonly",
		"--webroot",
		"-w", webRoot,
		"-d", domain,
		"-d", "www." + domain,
		"--email", email,
		"--agree-tos",
		"--non-interactive",
	}

	cmd := exec.Command("certbot", args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		// If www subdomain fails, try without it
		args = []string{
			"certonly",
			"--webroot",
			"-w", webRoot,
			"-d", domain,
			"--email", email,
			"--agree-tos",
			"--non-interactive",
		}
		cmd = exec.Command("certbot", args...)
		output2, err2 := cmd.CombinedOutput()
		if err2 != nil {
			return nil, fmt.Errorf("%s: %w", strings.TrimSpace(string(output2)), err2)
		}
	}

	certPath := filepath.Join("/etc/letsencrypt/live", domain, "fullchain.pem")
	keyPath := filepath.Join("/etc/letsencrypt/live", domain, "privkey.pem")

	return &certInfo{
		Issuer:     "Let's Encrypt",
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().AddDate(0, 3, 0), // 90 days
		CertPath:   certPath,
		KeyPath:    keyPath,
	}, nil
}

func (h *Handler) renewCertificate(domain string) error {
	cmd := exec.Command("certbot", "renew", "--cert-name", domain, "--non-interactive")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (h *Handler) revokeCertificate(domain string) error {
	cmd := exec.Command("certbot", "revoke", "--cert-name", domain, "--delete-after-revoke", "--non-interactive")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (h *Handler) configureSSLVhost(domain, username string, cert *certInfo) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", domain+"-ssl.conf")

	docRoot := filepath.Join("/home", username, "public_html")
	phpVersion := "8.1" // Default PHP version

	vhostContent := fmt.Sprintf(`<VirtualHost *:443>
    ServerName %s
    ServerAlias www.%s
    DocumentRoot %s

    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s

    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm-%s.sock|fcgi://localhost"
    </FilesMatch>

    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, domain, domain, docRoot, cert.CertPath, cert.KeyPath, docRoot, phpVersion, username, domain, domain)

	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		return err
	}

	// Enable site
	exec.Command("a2ensite", domain+"-ssl").Run()

	// Enable SSL module
	exec.Command("a2enmod", "ssl").Run()

	// Reload Apache
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}

func (h *Handler) removeSSLVhost(domain, username string) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", domain+"-ssl.conf")

	// Disable site
	exec.Command("a2dissite", domain+"-ssl").Run()

	// Remove config file
	os.Remove(vhostPath)

	// Reload Apache
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}

// issueCertificateForFQDN issues certificate for any FQDN (subdomain, www, mail)
func (h *Handler) issueCertificateForFQDN(fqdn, webRoot, email string) (*certInfo, error) {
	args := []string{
		"certonly",
		"--webroot",
		"-w", webRoot,
		"-d", fqdn,
		"--email", email,
		"--agree-tos",
		"--non-interactive",
	}

	cmd := exec.Command("certbot", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}

	certPath := filepath.Join("/etc/letsencrypt/live", fqdn, "fullchain.pem")
	keyPath := filepath.Join("/etc/letsencrypt/live", fqdn, "privkey.pem")

	return &certInfo{
		Issuer:     "Let's Encrypt",
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().AddDate(0, 3, 0), // 90 days
		CertPath:   certPath,
		KeyPath:    keyPath,
	}, nil
}

// configureSSLVhostForFQDN configures SSL vhost for subdomain
func (h *Handler) configureSSLVhostForFQDN(fqdn, username, docRoot string, cert *certInfo) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", fqdn+"-ssl.conf")
	phpVersion := "8.1"

	vhostContent := fmt.Sprintf(`<VirtualHost *:443>
    ServerName %s
    DocumentRoot %s

    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s

    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm-%s.sock|fcgi://localhost"
    </FilesMatch>

    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, fqdn, docRoot, cert.CertPath, cert.KeyPath, docRoot, phpVersion, username, fqdn, fqdn)

	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		return err
	}

	// Enable site
	exec.Command("a2ensite", fqdn+"-ssl").Run()

	// Reload Apache
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}

// configureSSLVhostForWebmail configures SSL vhost for webmail subdomain
func (h *Handler) configureSSLVhostForWebmail(fqdn string, cert *certInfo) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", fqdn+"-ssl.conf")

	vhostContent := fmt.Sprintf(`<VirtualHost *:443>
    ServerName %s
    
    DocumentRoot /usr/share/roundcube
    
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    <Directory /usr/share/roundcube>
        Options +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
    
    <Directory /usr/share/roundcube/config>
        Require all denied
    </Directory>
    
    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, fqdn, cert.CertPath, cert.KeyPath, fqdn, fqdn)

	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		return err
	}

	exec.Command("a2ensite", fqdn+"-ssl").Run()
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}

// configureSSLVhostForMail configures SSL vhost for mail subdomain (redirects to webmail)
func (h *Handler) configureSSLVhostForMail(fqdn string, cert *certInfo) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", fqdn+"-ssl.conf")

	// Extract domain from fqdn (mail.example.com -> example.com)
	parts := strings.SplitN(fqdn, ".", 2)
	domain := fqdn
	if len(parts) > 1 {
		domain = parts[1]
	}

	vhostContent := fmt.Sprintf(`<VirtualHost *:443>
    ServerName %s
    
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    # Redirect to webmail
    RedirectPermanent / https://webmail.%s/
    
    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, fqdn, cert.CertPath, cert.KeyPath, domain, fqdn, fqdn)

	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		return err
	}

	exec.Command("a2ensite", fqdn+"-ssl").Run()
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}

// configureSSLVhostForFTP configures SSL vhost for ftp subdomain (info page)
func (h *Handler) configureSSLVhostForFTP(fqdn string, cert *certInfo) error {
	vhostPath := filepath.Join("/etc/apache2/sites-available", fqdn+"-ssl.conf")

	vhostContent := fmt.Sprintf(`<VirtualHost *:443>
    ServerName %s
    
    DocumentRoot /var/www/html
    
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    <Directory /var/www/html>
        Options -Indexes
        AllowOverride None
        Require all granted
    </Directory>
    
    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, fqdn, cert.CertPath, cert.KeyPath, fqdn, fqdn)

	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		return err
	}

	exec.Command("a2ensite", fqdn+"-ssl").Run()
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}

// configureSSLVhostForWWW configures SSL for www subdomain (same as main domain)
func (h *Handler) configureSSLVhostForWWW(fqdn, username string, cert *certInfo) error {
	// www subdomain typically shares the main domain's vhost
	// Just ensure the certificate is properly configured

	// Extract main domain from www.example.com
	parts := strings.SplitN(fqdn, ".", 2)
	if len(parts) < 2 {
		return nil
	}
	mainDomain := parts[1]

	// Check if main domain SSL vhost exists and update it
	mainVhostPath := filepath.Join("/etc/apache2/sites-available", mainDomain+"-ssl.conf")
	if _, err := os.Stat(mainVhostPath); err == nil {
		// Main domain SSL vhost exists, www should be covered by ServerAlias
		return nil
	}

	// Create separate www SSL vhost if main doesn't exist
	docRoot := filepath.Join("/home", username, "public_html")
	phpVersion := "8.1"

	vhostPath := filepath.Join("/etc/apache2/sites-available", fqdn+"-ssl.conf")
	vhostContent := fmt.Sprintf(`<VirtualHost *:443>
    ServerName %s
    DocumentRoot %s

    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s

    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm-%s.sock|fcgi://localhost"
    </FilesMatch>

    ErrorLog ${APACHE_LOG_DIR}/%s-ssl-error.log
    CustomLog ${APACHE_LOG_DIR}/%s-ssl-access.log combined
</VirtualHost>
`, fqdn, docRoot, cert.CertPath, cert.KeyPath, docRoot, phpVersion, username, fqdn, fqdn)

	if err := os.WriteFile(vhostPath, []byte(vhostContent), 0644); err != nil {
		return err
	}

	exec.Command("a2ensite", fqdn+"-ssl").Run()
	exec.Command("systemctl", "reload", "apache2").Run()

	return nil
}
