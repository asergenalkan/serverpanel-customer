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
	ID         int64     `json:"id"`
	DomainID   int64     `json:"domain_id"`
	Domain     string    `json:"domain"`
	Issuer     string    `json:"issuer"`
	Status     string    `json:"status"` // active, expired, pending, none
	ValidFrom  time.Time `json:"valid_from"`
	ValidUntil time.Time `json:"valid_until"`
	AutoRenew  bool      `json:"auto_renew"`
	CertPath   string    `json:"cert_path,omitempty"`
	KeyPath    string    `json:"key_path,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListSSLCertificates returns all SSL certificates for the user
func (h *Handler) ListSSLCertificates(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var certificates []SSLCertificate

	// Get domains for this user
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `SELECT d.id, d.name FROM domains d ORDER BY d.name`
	} else {
		query = `SELECT d.id, d.name FROM domains d WHERE d.user_id = ? ORDER BY d.name`
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

	for rows.Next() {
		var domainID int64
		var domain string
		if err := rows.Scan(&domainID, &domain); err != nil {
			continue
		}

		cert := SSLCertificate{
			DomainID:  domainID,
			Domain:    domain,
			AutoRenew: true,
		}

		// Check if certificate exists
		certInfo := h.getCertificateInfo(domain)
		if certInfo != nil {
			cert.Issuer = certInfo.Issuer
			cert.ValidFrom = certInfo.ValidFrom
			cert.ValidUntil = certInfo.ValidUntil
			cert.CertPath = certInfo.CertPath
			cert.KeyPath = certInfo.KeyPath

			if time.Now().After(certInfo.ValidUntil) {
				cert.Status = "expired"
			} else if time.Now().Before(certInfo.ValidFrom) {
				cert.Status = "pending"
			} else {
				cert.Status = "active"
			}
		} else {
			cert.Status = "none"
		}

		certificates = append(certificates, cert)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    certificates,
	})
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
