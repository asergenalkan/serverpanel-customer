package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// ═══════════════════════════════════════════════════════════════════════════════
// E-POSTA YÖNETİM SİSTEMİ
// ═══════════════════════════════════════════════════════════════════════════════
//
// Mail Server Mimarisi:
// - MTA (Mail Transfer Agent): Postfix - SMTP gönderim/alım
// - MDA (Mail Delivery Agent): Dovecot - IMAP/POP3 erişim
// - Webmail: Roundcube - Tarayıcı üzerinden e-posta
//
// Standart Portlar:
// ┌─────────────┬──────────┬────────────┬─────────────────────────────────┐
// │ Protokol    │ Port     │ Güvenlik   │ Açıklama                        │
// ├─────────────┼──────────┼────────────┼─────────────────────────────────┤
// │ SMTP        │ 25       │ Yok/STARTTLS│ Sunucular arası mail transferi │
// │ SMTP        │ 587      │ STARTTLS   │ Mail gönderimi (submission)     │
// │ SMTPS       │ 465      │ SSL/TLS    │ Güvenli mail gönderimi          │
// │ IMAP        │ 143      │ STARTTLS   │ Mail okuma                      │
// │ IMAPS       │ 993      │ SSL/TLS    │ Güvenli mail okuma              │
// │ POP3        │ 110      │ STARTTLS   │ Mail indirme                    │
// │ POP3S       │ 995      │ SSL/TLS    │ Güvenli mail indirme            │
// └─────────────┴──────────┴────────────┴─────────────────────────────────┘
//
// Roundcube Webmail:
// - Lisans: GPL-3.0 (Ücretsiz ve Açık Kaynak)
// - URL: https://roundcube.net
// - Erişim: http://server/webmail veya http://webmail.domain.com
// ═══════════════════════════════════════════════════════════════════════════════

// EmailAccount represents an email account
type EmailAccount struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	DomainID   int64  `json:"domain_id"`
	DomainName string `json:"domain_name"`
	Email      string `json:"email"`      // user@domain.com
	LocalPart  string `json:"local_part"` // user
	QuotaMB    int    `json:"quota_mb"`   // Mailbox boyutu (MB)
	UsedMB     int    `json:"used_mb"`    // Kullanılan alan
	Active     bool   `json:"active"`
	CreatedAt  string `json:"created_at"`
	// İstatistikler
	MessageCount int `json:"message_count,omitempty"`
}

// EmailForwarder represents an email forwarder
type EmailForwarder struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	DomainID    int64  `json:"domain_id"`
	DomainName  string `json:"domain_name"`
	Source      string `json:"source"`      // user@domain.com
	Destination string `json:"destination"` // target@external.com
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
}

// EmailAutoresponder represents an autoresponder
type EmailAutoresponder struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	DomainID  int64  `json:"domain_id"`
	Email     string `json:"email"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

// EmailStats represents email statistics for a domain
type EmailStats struct {
	TotalAccounts   int `json:"total_accounts"`
	TotalForwarders int `json:"total_forwarders"`
	TotalQuotaMB    int `json:"total_quota_mb"`
	UsedQuotaMB     int `json:"used_quota_mb"`
	MaxAccounts     int `json:"max_accounts"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// E-POSTA HESAPLARI
// ═══════════════════════════════════════════════════════════════════════════════

// ListEmailAccounts returns all email accounts for the user
func (h *Handler) ListEmailAccounts(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var accounts []EmailAccount
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT e.id, e.user_id, e.domain_id, d.name as domain_name,
			       e.email, e.quota_mb, e.active, e.created_at
			FROM email_accounts e
			JOIN domains d ON e.domain_id = d.id
			ORDER BY e.email
		`
	} else {
		query = `
			SELECT e.id, e.user_id, e.domain_id, d.name as domain_name,
			       e.email, e.quota_mb, e.active, e.created_at
			FROM email_accounts e
			JOIN domains d ON e.domain_id = d.id
			WHERE e.user_id = ?
			ORDER BY e.email
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesapları alınamadı",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var acc EmailAccount
		var activeInt int
		err := rows.Scan(&acc.ID, &acc.UserID, &acc.DomainID, &acc.DomainName,
			&acc.Email, &acc.QuotaMB, &activeInt, &acc.CreatedAt)
		if err != nil {
			continue
		}
		acc.Active = activeInt == 1
		// Email'den local part'ı çıkar
		parts := strings.Split(acc.Email, "@")
		if len(parts) > 0 {
			acc.LocalPart = parts[0]
		}
		// Mailbox boyutunu hesapla
		acc.UsedMB = h.getMailboxSize(acc.Email)
		accounts = append(accounts, acc)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    accounts,
	})
}

// CreateEmailAccount creates a new email account
func (h *Handler) CreateEmailAccount(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		DomainID int64  `json:"domain_id"`
		Username string `json:"username"` // local part (before @)
		Password string `json:"password"`
		QuotaMB  int    `json:"quota_mb"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate
	if req.DomainID == 0 || req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain, kullanıcı adı ve şifre gerekli",
		})
	}

	// Username validation
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	if !isValidEmailUsername(req.Username) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz kullanıcı adı. Sadece harf, rakam, nokta, tire ve alt çizgi kullanılabilir",
		})
	}

	// Default quota
	if req.QuotaMB <= 0 {
		req.QuotaMB = 1024 // 1GB default
	}

	// Get domain info and check ownership
	var domainName string
	var domainUserID int64
	err := h.db.QueryRow("SELECT name, user_id FROM domains WHERE id = ?", req.DomainID).
		Scan(&domainName, &domainUserID)
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
			Error:   "Bu domain'e e-posta ekleme yetkiniz yok",
		})
	}

	// Check email limit from package
	var maxEmails int
	err = h.db.QueryRow(`
		SELECT COALESCE(p.max_email_accounts, 10)
		FROM users u
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE u.id = ?
	`, domainUserID).Scan(&maxEmails)
	if err != nil {
		maxEmails = 10
	}

	// Count existing emails
	var currentCount int
	h.db.QueryRow("SELECT COUNT(*) FROM email_accounts WHERE user_id = ?", domainUserID).Scan(&currentCount)
	if currentCount >= maxEmails && maxEmails != -1 {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("E-posta hesap limitine ulaşıldı (%d/%d)", currentCount, maxEmails),
		})
	}

	// Create full email address
	email := fmt.Sprintf("%s@%s", req.Username, domainName)

	// Check if email already exists
	var exists int
	h.db.QueryRow("SELECT COUNT(*) FROM email_accounts WHERE email = ?", email).Scan(&exists)
	if exists > 0 {
		return c.Status(fiber.StatusConflict).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu e-posta adresi zaten mevcut",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Şifre işlenemedi",
		})
	}

	// Insert into database
	result, err := h.db.Exec(`
		INSERT INTO email_accounts (user_id, domain_id, email, password_hash, quota_mb, active)
		VALUES (?, ?, ?, ?, ?, 1)
	`, domainUserID, req.DomainID, email, string(hashedPassword), req.QuotaMB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı oluşturulamadı",
		})
	}

	accountID, _ := result.LastInsertId()

	// Create mailbox on system (pass plain password for doveadm to hash)
	go h.createMailbox(email, req.Password, req.QuotaMB, domainName)

	log.Printf("📧 E-posta hesabı oluşturuldu: %s", email)

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "E-posta hesabı oluşturuldu",
		Data: EmailAccount{
			ID:         accountID,
			UserID:     domainUserID,
			DomainID:   req.DomainID,
			DomainName: domainName,
			Email:      email,
			LocalPart:  req.Username,
			QuotaMB:    req.QuotaMB,
			Active:     true,
		},
	})
}

// UpdateEmailAccount updates an email account
func (h *Handler) UpdateEmailAccount(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		Password string `json:"password,omitempty"`
		QuotaMB  int    `json:"quota_mb,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Get account info
	var accountUserID int64
	var email string
	err = h.db.QueryRow("SELECT user_id, email FROM email_accounts WHERE id = ?", id).
		Scan(&accountUserID, &email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && accountUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu hesabı düzenleme yetkiniz yok",
		})
	}

	// Update password if provided
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Şifre işlenemedi",
			})
		}
		h.db.Exec("UPDATE email_accounts SET password_hash = ? WHERE id = ?", string(hashedPassword), id)
		go h.updateMailboxPassword(email, string(hashedPassword))
	}

	// Update quota if provided
	if req.QuotaMB > 0 {
		h.db.Exec("UPDATE email_accounts SET quota_mb = ? WHERE id = ?", req.QuotaMB, id)
		go h.updateMailboxQuota(email, req.QuotaMB)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "E-posta hesabı güncellendi",
	})
}

// DeleteEmailAccount deletes an email account
func (h *Handler) DeleteEmailAccount(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get account info
	var accountUserID int64
	var email string
	err = h.db.QueryRow("SELECT user_id, email FROM email_accounts WHERE id = ?", id).
		Scan(&accountUserID, &email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && accountUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu hesabı silme yetkiniz yok",
		})
	}

	// Delete from database
	_, err = h.db.Exec("DELETE FROM email_accounts WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı silinemedi",
		})
	}

	// Delete mailbox from system
	go h.deleteMailbox(email)

	log.Printf("📧 E-posta hesabı silindi: %s", email)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "E-posta hesabı silindi",
	})
}

// ToggleEmailAccount enables/disables an email account
func (h *Handler) ToggleEmailAccount(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get account info
	var accountUserID int64
	var active int
	err = h.db.QueryRow("SELECT user_id, active FROM email_accounts WHERE id = ?", id).
		Scan(&accountUserID, &active)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && accountUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu hesabı değiştirme yetkiniz yok",
		})
	}

	// Toggle active status
	newActive := 0
	if active == 0 {
		newActive = 1
	}

	_, err = h.db.Exec("UPDATE email_accounts SET active = ? WHERE id = ?", newActive, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Durum güncellenemedi",
		})
	}

	status := "devre dışı bırakıldı"
	if newActive == 1 {
		status = "aktifleştirildi"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("E-posta hesabı %s", status),
		Data:    map[string]bool{"active": newActive == 1},
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// E-POSTA YÖNLENDİRMELERİ (FORWARDERS)
// ═══════════════════════════════════════════════════════════════════════════════

// ListEmailForwarders returns all email forwarders
func (h *Handler) ListEmailForwarders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var forwarders []EmailForwarder
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT f.id, f.user_id, f.domain_id, d.name as domain_name,
			       f.source, f.destination, f.active, f.created_at
			FROM email_forwarders f
			JOIN domains d ON f.domain_id = d.id
			ORDER BY f.source
		`
	} else {
		query = `
			SELECT f.id, f.user_id, f.domain_id, d.name as domain_name,
			       f.source, f.destination, f.active, f.created_at
			FROM email_forwarders f
			JOIN domains d ON f.domain_id = d.id
			WHERE f.user_id = ?
			ORDER BY f.source
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Yönlendirmeler alınamadı",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var f EmailForwarder
		var activeInt int
		err := rows.Scan(&f.ID, &f.UserID, &f.DomainID, &f.DomainName,
			&f.Source, &f.Destination, &activeInt, &f.CreatedAt)
		if err != nil {
			continue
		}
		f.Active = activeInt == 1
		forwarders = append(forwarders, f)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    forwarders,
	})
}

// CreateEmailForwarder creates a new email forwarder
func (h *Handler) CreateEmailForwarder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		DomainID    int64  `json:"domain_id"`
		Source      string `json:"source"`      // user@domain.com or just user
		Destination string `json:"destination"` // target@external.com
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	if req.DomainID == 0 || req.Source == "" || req.Destination == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain, kaynak ve hedef gerekli",
		})
	}

	// Get domain info
	var domainName string
	var domainUserID int64
	err := h.db.QueryRow("SELECT name, user_id FROM domains WHERE id = ?", req.DomainID).
		Scan(&domainName, &domainUserID)
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
			Error:   "Bu domain'e yönlendirme ekleme yetkiniz yok",
		})
	}

	// Format source email
	source := strings.ToLower(strings.TrimSpace(req.Source))
	if !strings.Contains(source, "@") {
		source = fmt.Sprintf("%s@%s", source, domainName)
	}

	// Validate destination email
	destination := strings.ToLower(strings.TrimSpace(req.Destination))
	if !strings.Contains(destination, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz hedef e-posta adresi",
		})
	}

	// Insert into database
	result, err := h.db.Exec(`
		INSERT INTO email_forwarders (user_id, domain_id, source, destination, active)
		VALUES (?, ?, ?, ?, 1)
	`, domainUserID, req.DomainID, source, destination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Yönlendirme oluşturulamadı",
		})
	}

	forwarderID, _ := result.LastInsertId()

	// Create forwarder on system
	go h.createForwarder(source, destination)

	log.Printf("📧 E-posta yönlendirmesi oluşturuldu: %s -> %s", source, destination)

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "E-posta yönlendirmesi oluşturuldu",
		Data: EmailForwarder{
			ID:          forwarderID,
			UserID:      domainUserID,
			DomainID:    req.DomainID,
			DomainName:  domainName,
			Source:      source,
			Destination: destination,
			Active:      true,
		},
	})
}

// DeleteEmailForwarder deletes an email forwarder
func (h *Handler) DeleteEmailForwarder(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get forwarder info
	var forwarderUserID int64
	var source string
	err = h.db.QueryRow("SELECT user_id, source FROM email_forwarders WHERE id = ?", id).
		Scan(&forwarderUserID, &source)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Yönlendirme bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && forwarderUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu yönlendirmeyi silme yetkiniz yok",
		})
	}

	// Delete from database
	_, err = h.db.Exec("DELETE FROM email_forwarders WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Yönlendirme silinemedi",
		})
	}

	// Delete forwarder from system
	go h.deleteForwarder(source)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "E-posta yönlendirmesi silindi",
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// OTOMATİK YANITLAYICI (AUTORESPONDER)
// ═══════════════════════════════════════════════════════════════════════════════

// ListAutoresponders returns all autoresponders
func (h *Handler) ListAutoresponders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var autoresponders []EmailAutoresponder
	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT a.id, a.user_id, a.domain_id, a.email, a.subject, a.body,
			       a.start_date, a.end_date, a.active, a.created_at
			FROM email_autoresponders a
			ORDER BY a.email
		`
	} else {
		query = `
			SELECT a.id, a.user_id, a.domain_id, a.email, a.subject, a.body,
			       a.start_date, a.end_date, a.active, a.created_at
			FROM email_autoresponders a
			WHERE a.user_id = ?
			ORDER BY a.email
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Otomatik yanıtlayıcılar alınamadı",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var a EmailAutoresponder
		var activeInt int
		var startDate, endDate *string
		err := rows.Scan(&a.ID, &a.UserID, &a.DomainID, &a.Email, &a.Subject, &a.Body,
			&startDate, &endDate, &activeInt, &a.CreatedAt)
		if err != nil {
			continue
		}
		a.Active = activeInt == 1
		if startDate != nil {
			a.StartDate = *startDate
		}
		if endDate != nil {
			a.EndDate = *endDate
		}
		autoresponders = append(autoresponders, a)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    autoresponders,
	})
}

// CreateAutoresponder creates a new autoresponder
func (h *Handler) CreateAutoresponder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		EmailAccountID int64  `json:"email_account_id"`
		Subject        string `json:"subject"`
		Body           string `json:"body"`
		StartDate      string `json:"start_date,omitempty"`
		EndDate        string `json:"end_date,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	if req.EmailAccountID == 0 || req.Subject == "" || req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı, konu ve mesaj gerekli",
		})
	}

	// Get email account info
	var email string
	var domainID, accountUserID int64
	err := h.db.QueryRow("SELECT email, domain_id, user_id FROM email_accounts WHERE id = ?", req.EmailAccountID).
		Scan(&email, &domainID, &accountUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "E-posta hesabı bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && accountUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu hesaba otomatik yanıtlayıcı ekleme yetkiniz yok",
		})
	}

	// Insert into database
	result, err := h.db.Exec(`
		INSERT INTO email_autoresponders (user_id, domain_id, email, subject, body, start_date, end_date, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`, accountUserID, domainID, email, req.Subject, req.Body, nullString(req.StartDate), nullString(req.EndDate))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Otomatik yanıtlayıcı oluşturulamadı",
		})
	}

	autoresponderID, _ := result.LastInsertId()

	// Create autoresponder on system
	go h.createAutoresponder(email, req.Subject, req.Body)

	log.Printf("📧 Otomatik yanıtlayıcı oluşturuldu: %s", email)

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Otomatik yanıtlayıcı oluşturuldu",
		Data: EmailAutoresponder{
			ID:        autoresponderID,
			UserID:    accountUserID,
			DomainID:  domainID,
			Email:     email,
			Subject:   req.Subject,
			Body:      req.Body,
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
			Active:    true,
		},
	})
}

// DeleteAutoresponder deletes an autoresponder
func (h *Handler) DeleteAutoresponder(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get autoresponder info
	var autoresponderUserID int64
	var email string
	err = h.db.QueryRow("SELECT user_id, email FROM email_autoresponders WHERE id = ?", id).
		Scan(&autoresponderUserID, &email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Otomatik yanıtlayıcı bulunamadı",
		})
	}

	// Check ownership
	if role != models.RoleAdmin && autoresponderUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu otomatik yanıtlayıcıyı silme yetkiniz yok",
		})
	}

	// Delete from database
	_, err = h.db.Exec("DELETE FROM email_autoresponders WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Otomatik yanıtlayıcı silinemedi",
		})
	}

	// Delete autoresponder from system
	go h.deleteAutoresponder(email)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Otomatik yanıtlayıcı silindi",
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// WEBMAIL (ROUNDCUBE)
// ═══════════════════════════════════════════════════════════════════════════════

// GetWebmailURL returns the webmail URL for the user
func (h *Handler) GetWebmailURL(c *fiber.Ctx) error {
	cfg := config.Get()

	// Roundcube URL
	webmailURL := fmt.Sprintf("http://%s/webmail", cfg.ServerIP)

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]string{
			"url":         webmailURL,
			"description": "Roundcube Webmail",
		},
	})
}

// GetEmailStats returns email statistics for the user
func (h *Handler) GetEmailStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var stats EmailStats

	// Get max accounts from package
	var maxEmails int
	err := h.db.QueryRow(`
		SELECT COALESCE(p.max_email_accounts, 10)
		FROM users u
		LEFT JOIN user_packages up ON u.id = up.user_id
		LEFT JOIN packages p ON up.package_id = p.id
		WHERE u.id = ?
	`, userID).Scan(&maxEmails)
	if err != nil {
		maxEmails = 10
	}
	stats.MaxAccounts = maxEmails

	// Count accounts
	if role == models.RoleAdmin {
		h.db.QueryRow("SELECT COUNT(*) FROM email_accounts").Scan(&stats.TotalAccounts)
		h.db.QueryRow("SELECT COUNT(*) FROM email_forwarders").Scan(&stats.TotalForwarders)
		h.db.QueryRow("SELECT COALESCE(SUM(quota_mb), 0) FROM email_accounts").Scan(&stats.TotalQuotaMB)
	} else {
		h.db.QueryRow("SELECT COUNT(*) FROM email_accounts WHERE user_id = ?", userID).Scan(&stats.TotalAccounts)
		h.db.QueryRow("SELECT COUNT(*) FROM email_forwarders WHERE user_id = ?", userID).Scan(&stats.TotalForwarders)
		h.db.QueryRow("SELECT COALESCE(SUM(quota_mb), 0) FROM email_accounts WHERE user_id = ?", userID).Scan(&stats.TotalQuotaMB)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// YARDIMCI FONKSİYONLAR
// ═══════════════════════════════════════════════════════════════════════════════

func isValidEmailUsername(username string) bool {
	if len(username) < 1 || len(username) > 64 {
		return false
	}
	// Allow letters, numbers, dots, hyphens, underscores
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func (h *Handler) getMailboxSize(email string) int {
	// Get mailbox size from Dovecot/Maildir
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return 0
	}

	maildir := filepath.Join("/var/mail/vhosts", parts[1], parts[0])

	var totalSize int64
	filepath.Walk(maildir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return int(totalSize / 1024 / 1024) // Convert to MB
}

// ═══════════════════════════════════════════════════════════════════════════════
// SİSTEM KOMUTLARI (Postfix/Dovecot)
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) createMailbox(email, password string, quotaMB int, domain string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Mailbox oluşturulacak: %s", email)
		return
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return
	}
	localPart := parts[0]

	// Create mail directory structure
	maildir := filepath.Join("/var/mail/vhosts", domain, localPart)
	os.MkdirAll(filepath.Join(maildir, "cur"), 0700)
	os.MkdirAll(filepath.Join(maildir, "new"), 0700)
	os.MkdirAll(filepath.Join(maildir, "tmp"), 0700)

	// Set ownership to vmail user
	exec.Command("chown", "-R", "vmail:vmail", maildir).Run()

	// Add domain to Postfix virtual domains if not exists
	vdomainsFile := "/etc/postfix/vdomains"
	vdomainsContent, _ := os.ReadFile(vdomainsFile)
	if !strings.Contains(string(vdomainsContent), domain) {
		f, err := os.OpenFile(vdomainsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(domain + "\n")
			f.Close()
		}
	}

	// Add to Postfix virtual mailbox maps
	virtualMailboxFile := "/etc/postfix/vmailbox"
	f, err := os.OpenFile(virtualMailboxFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(fmt.Sprintf("%s %s/%s/\n", email, domain, localPart))
		f.Close()
		exec.Command("postmap", virtualMailboxFile).Run()
	}

	// Generate Dovecot-compatible password hash using doveadm
	dovecotHash := ""
	out, err := exec.Command("doveadm", "pw", "-s", "BLF-CRYPT", "-p", password).Output()
	if err == nil {
		dovecotHash = strings.TrimSpace(string(out))
	} else {
		log.Printf("⚠️ doveadm pw hatası: %v", err)
		return
	}

	// Add to Dovecot passwd file
	passwdFile := "/etc/dovecot/users"
	f, err = os.OpenFile(passwdFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err == nil {
		// Format: user@domain:{SCHEME}password
		f.WriteString(fmt.Sprintf("%s:%s\n", email, dovecotHash))
		f.Close()
		// Set proper ownership
		exec.Command("chown", "root:dovecot", passwdFile).Run()
	}

	// Reload services
	exec.Command("postfix", "reload").Run()

	log.Printf("✅ Mailbox oluşturuldu: %s", email)
}

func (h *Handler) updateMailboxPassword(email, passwordHash string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Mailbox şifresi güncellenecek: %s", email)
		return
	}

	// Update Dovecot passwd file
	passwdFile := "/etc/dovecot/users"
	content, err := os.ReadFile(passwdFile)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, email+":") {
			// Update password
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 3 {
				line = fmt.Sprintf("%s:%s:%s", parts[0], passwordHash, parts[2])
			}
		}
		newLines = append(newLines, line)
	}

	os.WriteFile(passwdFile, []byte(strings.Join(newLines, "\n")), 0600)
	exec.Command("doveadm", "reload").Run()

	log.Printf("✅ Mailbox şifresi güncellendi: %s", email)
}

func (h *Handler) updateMailboxQuota(email string, quotaMB int) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Mailbox kotası güncellenecek: %s -> %dMB", email, quotaMB)
		return
	}

	// Update quota in Dovecot
	exec.Command("doveadm", "quota", "recalc", "-u", email).Run()
	log.Printf("✅ Mailbox kotası güncellendi: %s -> %dMB", email, quotaMB)
}

func (h *Handler) deleteMailbox(email string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Mailbox silinecek: %s", email)
		return
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return
	}

	// Remove from Postfix virtual mailbox
	virtualMailboxFile := "/etc/postfix/vmailbox"
	content, err := os.ReadFile(virtualMailboxFile)
	if err == nil {
		lines := strings.Split(string(content), "\n")
		var newLines []string
		for _, line := range lines {
			if !strings.HasPrefix(line, email+" ") {
				newLines = append(newLines, line)
			}
		}
		os.WriteFile(virtualMailboxFile, []byte(strings.Join(newLines, "\n")), 0644)
		exec.Command("postmap", virtualMailboxFile).Run()
	}

	// Remove from Dovecot passwd
	passwdFile := "/etc/dovecot/users"
	content, err = os.ReadFile(passwdFile)
	if err == nil {
		lines := strings.Split(string(content), "\n")
		var newLines []string
		for _, line := range lines {
			if !strings.HasPrefix(line, email+":") {
				newLines = append(newLines, line)
			}
		}
		os.WriteFile(passwdFile, []byte(strings.Join(newLines, "\n")), 0600)
	}

	// Delete maildir
	maildir := filepath.Join("/var/mail/vhosts", parts[1], parts[0])
	os.RemoveAll(maildir)

	// Reload services
	exec.Command("postfix", "reload").Run()
	exec.Command("doveadm", "reload").Run()

	log.Printf("✅ Mailbox silindi: %s", email)
}

func (h *Handler) createForwarder(source, destination string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Forwarder oluşturulacak: %s -> %s", source, destination)
		return
	}

	// Add to Postfix virtual alias maps
	virtualAliasFile := "/etc/postfix/virtual"
	f, err := os.OpenFile(virtualAliasFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(fmt.Sprintf("%s %s\n", source, destination))
		f.Close()
		exec.Command("postmap", virtualAliasFile).Run()
		exec.Command("postfix", "reload").Run()
	}

	log.Printf("✅ Forwarder oluşturuldu: %s -> %s", source, destination)
}

func (h *Handler) deleteForwarder(source string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Forwarder silinecek: %s", source)
		return
	}

	virtualAliasFile := "/etc/postfix/virtual"
	content, err := os.ReadFile(virtualAliasFile)
	if err == nil {
		lines := strings.Split(string(content), "\n")
		var newLines []string
		for _, line := range lines {
			if !strings.HasPrefix(line, source+" ") {
				newLines = append(newLines, line)
			}
		}
		os.WriteFile(virtualAliasFile, []byte(strings.Join(newLines, "\n")), 0644)
		exec.Command("postmap", virtualAliasFile).Run()
		exec.Command("postfix", "reload").Run()
	}

	log.Printf("✅ Forwarder silindi: %s", source)
}

func (h *Handler) createAutoresponder(email, subject, body string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Autoresponder oluşturulacak: %s", email)
		return
	}

	// Create Sieve script for autoresponder
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return
	}

	sieveDir := filepath.Join("/var/mail/vhosts", parts[1], parts[0], "sieve")
	os.MkdirAll(sieveDir, 0700)

	sieveScript := fmt.Sprintf(`require ["vacation"];
vacation
  :days 1
  :subject "%s"
  "%s";
`, subject, body)

	sievePath := filepath.Join(sieveDir, "vacation.sieve")
	os.WriteFile(sievePath, []byte(sieveScript), 0600)
	exec.Command("chown", "-R", "vmail:vmail", sieveDir).Run()

	// Compile sieve script
	exec.Command("sievec", sievePath).Run()

	log.Printf("✅ Autoresponder oluşturuldu: %s", email)
}

func (h *Handler) deleteAutoresponder(email string) {
	if config.IsDevelopment() {
		log.Printf("🔧 [DEV] Autoresponder silinecek: %s", email)
		return
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return
	}

	sievePath := filepath.Join("/var/mail/vhosts", parts[1], parts[0], "sieve", "vacation.sieve")
	os.Remove(sievePath)
	os.Remove(sievePath + "c") // Compiled version

	log.Printf("✅ Autoresponder silindi: %s", email)
}

// generateRandomPassword generates a random password
func generateRandomPassword(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// ═══════════════════════════════════════════════════════════════════════════════
// DKIM VE MAİL GÜVENLİK
// ═══════════════════════════════════════════════════════════════════════════════

// EmailSettings represents email settings for a domain
type EmailSettings struct {
	ID            int64  `json:"id"`
	DomainID      int64  `json:"domain_id"`
	DomainName    string `json:"domain_name"`
	HourlyLimit   int    `json:"hourly_limit"`
	DailyLimit    int    `json:"daily_limit"`
	DKIMEnabled   bool   `json:"dkim_enabled"`
	DKIMSelector  string `json:"dkim_selector"`
	DKIMPublicKey string `json:"dkim_public_key,omitempty"`
	SPFRecord     string `json:"spf_record"`
	DMARCRecord   string `json:"dmarc_record"`
	CatchAllEmail string `json:"catch_all_email"`
}

// GetEmailSettings returns email settings for a domain
func (h *Handler) GetEmailSettings(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("domain_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info and check ownership
	var domainName string
	var domainUserID int64
	err = h.db.QueryRow("SELECT name, user_id FROM domains WHERE id = ?", domainID).
		Scan(&domainName, &domainUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	if role != models.RoleAdmin && domainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain'e erişim yetkiniz yok",
		})
	}

	// Get or create settings
	var settings EmailSettings
	settings.DomainID = domainID
	settings.DomainName = domainName

	err = h.db.QueryRow(`
		SELECT id, hourly_limit, daily_limit, dkim_enabled, dkim_selector, 
		       COALESCE(dkim_public_key, ''), COALESCE(spf_record, ''), 
		       COALESCE(dmarc_record, ''), COALESCE(catch_all_email, '')
		FROM email_settings WHERE domain_id = ?
	`, domainID).Scan(
		&settings.ID, &settings.HourlyLimit, &settings.DailyLimit,
		&settings.DKIMEnabled, &settings.DKIMSelector, &settings.DKIMPublicKey,
		&settings.SPFRecord, &settings.DMARCRecord, &settings.CatchAllEmail,
	)

	if err != nil {
		// Create default settings
		cfg := config.Get()
		settings.HourlyLimit = 100
		settings.DailyLimit = 500
		settings.DKIMSelector = "default"
		settings.SPFRecord = fmt.Sprintf("v=spf1 ip4:%s ~all", cfg.ServerIP)
		settings.DMARCRecord = fmt.Sprintf("v=DMARC1; p=none; rua=mailto:postmaster@%s", domainName)

		h.db.Exec(`
			INSERT INTO email_settings (domain_id, hourly_limit, daily_limit, dkim_selector, spf_record, dmarc_record)
			VALUES (?, ?, ?, ?, ?, ?)
		`, domainID, settings.HourlyLimit, settings.DailyLimit, settings.DKIMSelector, settings.SPFRecord, settings.DMARCRecord)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    settings,
	})
}

// UpdateEmailSettings updates email settings for a domain
func (h *Handler) UpdateEmailSettings(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("domain_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		HourlyLimit   int    `json:"hourly_limit"`
		DailyLimit    int    `json:"daily_limit"`
		CatchAllEmail string `json:"catch_all_email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Only admin can change rate limits
	if role != models.RoleAdmin {
		// Get current limits
		var currentHourly, currentDaily int
		h.db.QueryRow("SELECT hourly_limit, daily_limit FROM email_settings WHERE domain_id = ?", domainID).
			Scan(&currentHourly, &currentDaily)
		req.HourlyLimit = currentHourly
		req.DailyLimit = currentDaily
	}

	// Get domain info
	var domainUserID int64
	err = h.db.QueryRow("SELECT user_id FROM domains WHERE id = ?", domainID).Scan(&domainUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	if role != models.RoleAdmin && domainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain'i düzenleme yetkiniz yok",
		})
	}

	// Update settings
	_, err = h.db.Exec(`
		UPDATE email_settings 
		SET hourly_limit = ?, daily_limit = ?, catch_all_email = ?, updated_at = CURRENT_TIMESTAMP
		WHERE domain_id = ?
	`, req.HourlyLimit, req.DailyLimit, req.CatchAllEmail, domainID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Ayarlar güncellenemedi",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "E-posta ayarları güncellendi",
	})
}

// GenerateDKIM generates DKIM keys for a domain
func (h *Handler) GenerateDKIM(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("domain_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get domain info
	var domainName string
	var domainUserID int64
	err = h.db.QueryRow("SELECT name, user_id FROM domains WHERE id = ?", domainID).
		Scan(&domainName, &domainUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	if role != models.RoleAdmin && domainUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu domain için DKIM oluşturma yetkiniz yok",
		})
	}

	// Generate DKIM keys
	selector := "default"
	keyDir := fmt.Sprintf("/etc/opendkim/keys/%s", domainName)

	if config.IsDevelopment() {
		return c.JSON(models.APIResponse{
			Success: true,
			Message: "DKIM anahtarları oluşturuldu (DEV)",
			Data: map[string]string{
				"selector":   selector,
				"dns_record": fmt.Sprintf("%s._domainkey.%s", selector, domainName),
				"public_key": "v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...",
			},
		})
	}

	// Create key directory
	os.MkdirAll(keyDir, 0700)
	exec.Command("chown", "opendkim:opendkim", keyDir).Run()

	// Generate keys using opendkim-genkey
	cmd := exec.Command("opendkim-genkey", "-b", "2048", "-d", domainName, "-D", keyDir, "-s", selector, "-v")
	cmd.Run()

	// Read public key
	publicKeyPath := filepath.Join(keyDir, selector+".txt")
	publicKeyContent, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "DKIM anahtarı oluşturulamadı",
		})
	}

	// Read private key
	privateKeyPath := filepath.Join(keyDir, selector+".private")
	privateKeyContent, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "DKIM özel anahtarı okunamadı",
		})
	}

	// Set permissions
	exec.Command("chown", "opendkim:opendkim", privateKeyPath).Run()
	exec.Command("chmod", "600", privateKeyPath).Run()

	// Update OpenDKIM KeyTable
	keyTableFile := "/etc/opendkim/KeyTable"
	keyTableEntry := fmt.Sprintf("%s._domainkey.%s %s:%s:%s\n", selector, domainName, domainName, selector, privateKeyPath)

	// Remove old entry if exists
	content, _ := os.ReadFile(keyTableFile)
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, domainName) {
			newLines = append(newLines, line)
		}
	}
	newLines = append(newLines, strings.TrimSpace(keyTableEntry))
	os.WriteFile(keyTableFile, []byte(strings.Join(newLines, "\n")), 0644)

	// Update OpenDKIM SigningTable
	signingTableFile := "/etc/opendkim/SigningTable"
	signingTableEntry := fmt.Sprintf("*@%s %s._domainkey.%s\n", domainName, selector, domainName)

	content, _ = os.ReadFile(signingTableFile)
	lines = strings.Split(string(content), "\n")
	newLines = []string{}
	for _, line := range lines {
		if !strings.Contains(line, domainName) {
			newLines = append(newLines, line)
		}
	}
	newLines = append(newLines, strings.TrimSpace(signingTableEntry))
	os.WriteFile(signingTableFile, []byte(strings.Join(newLines, "\n")), 0644)

	// Add domain to TrustedHosts
	trustedHostsFile := "/etc/opendkim/TrustedHosts"
	content, _ = os.ReadFile(trustedHostsFile)
	if !strings.Contains(string(content), domainName) {
		f, _ := os.OpenFile(trustedHostsFile, os.O_APPEND|os.O_WRONLY, 0644)
		f.WriteString(domainName + "\n")
		f.Close()
	}

	// Restart OpenDKIM
	exec.Command("systemctl", "restart", "opendkim").Run()

	// Save to database
	h.db.Exec(`
		UPDATE email_settings 
		SET dkim_enabled = 1, dkim_selector = ?, dkim_private_key = ?, dkim_public_key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE domain_id = ?
	`, selector, string(privateKeyContent), string(publicKeyContent), domainID)

	log.Printf("🔐 DKIM anahtarları oluşturuldu: %s", domainName)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "DKIM anahtarları oluşturuldu",
		Data: map[string]string{
			"selector":   selector,
			"dns_record": fmt.Sprintf("%s._domainkey.%s", selector, domainName),
			"dns_value":  string(publicKeyContent),
		},
	})
}

// GetDNSRecordsForEmail returns required DNS records for email
func (h *Handler) GetDNSRecordsForEmail(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("domain_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz domain ID",
		})
	}

	// Get domain info
	var domainName string
	err = h.db.QueryRow("SELECT name FROM domains WHERE id = ?", domainID).Scan(&domainName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Domain bulunamadı",
		})
	}

	cfg := config.Get()

	// Get email settings
	var settings EmailSettings
	h.db.QueryRow(`
		SELECT COALESCE(spf_record, ''), COALESCE(dmarc_record, ''), dkim_enabled, dkim_selector, COALESCE(dkim_public_key, '')
		FROM email_settings WHERE domain_id = ?
	`, domainID).Scan(&settings.SPFRecord, &settings.DMARCRecord, &settings.DKIMEnabled, &settings.DKIMSelector, &settings.DKIMPublicKey)

	if settings.SPFRecord == "" {
		settings.SPFRecord = fmt.Sprintf("v=spf1 ip4:%s ~all", cfg.ServerIP)
	}
	if settings.DMARCRecord == "" {
		settings.DMARCRecord = fmt.Sprintf("v=DMARC1; p=none; rua=mailto:postmaster@%s", domainName)
	}

	records := []map[string]string{
		{
			"type":        "TXT",
			"name":        "@",
			"value":       settings.SPFRecord,
			"description": "SPF - Hangi sunucuların mail gönderebileceğini belirtir",
		},
		{
			"type":        "TXT",
			"name":        "_dmarc",
			"value":       settings.DMARCRecord,
			"description": "DMARC - Mail doğrulama politikası",
		},
		{
			"type":        "MX",
			"name":        "@",
			"value":       fmt.Sprintf("10 %s", cfg.ServerIP),
			"description": "MX - Mail sunucusu",
		},
		{
			"type":        "A",
			"name":        "mail",
			"value":       cfg.ServerIP,
			"description": "Mail sunucu IP adresi",
		},
		{
			"type":        "A",
			"name":        "webmail",
			"value":       cfg.ServerIP,
			"description": "Webmail erişimi için",
		},
	}

	if settings.DKIMEnabled && settings.DKIMPublicKey != "" {
		records = append(records, map[string]string{
			"type":        "TXT",
			"name":        fmt.Sprintf("%s._domainkey", settings.DKIMSelector),
			"value":       settings.DKIMPublicKey,
			"description": "DKIM - Mail imzalama anahtarı",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    records,
	})
}
