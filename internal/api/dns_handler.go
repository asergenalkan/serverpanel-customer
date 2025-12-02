package api

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asergenalkan/serverpanel/internal/services/dns"
	"github.com/gofiber/fiber/v2"
)

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID        int64     `json:"id"`
	DomainID  int64     `json:"domain_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	TTL       int       `json:"ttl"`
	Priority  int       `json:"priority"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DNSZone represents a domain's DNS zone with all records
type DNSZone struct {
	DomainID   int64       `json:"domain_id"`
	DomainName string      `json:"domain_name"`
	Records    []DNSRecord `json:"records"`
	ServerIP   string      `json:"server_ip"`
}

// Supported DNS record types
var supportedRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"}

// ListDNSZones returns all DNS zones for the user
func (h *Handler) ListDNSZones(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == "admin" {
		query = `
			SELECT d.id, d.name, u.username
			FROM domains d
			JOIN users u ON d.user_id = u.id
			WHERE d.active = 1
			ORDER BY d.name`
	} else {
		query = `
			SELECT d.id, d.name, u.username
			FROM domains d
			JOIN users u ON d.user_id = u.id
			WHERE d.user_id = ? AND d.active = 1
			ORDER BY d.name`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}
	defer rows.Close()

	type ZoneInfo struct {
		DomainID   int64  `json:"domain_id"`
		DomainName string `json:"domain_name"`
		Username   string `json:"username"`
	}

	var zones []ZoneInfo
	for rows.Next() {
		var z ZoneInfo
		if err := rows.Scan(&z.DomainID, &z.DomainName, &z.Username); err != nil {
			continue
		}
		zones = append(zones, z)
	}

	return c.JSON(zones)
}

// GetDNSZone returns all DNS records for a domain
func (h *Handler) GetDNSZone(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz domain ID"})
	}

	// Check domain ownership
	var domain struct {
		ID     int64
		Name   string
		UserID int64
	}

	err = h.db.QueryRow(`
		SELECT id, name, user_id FROM domains WHERE id = ?
	`, domainID).Scan(&domain.ID, &domain.Name, &domain.UserID)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Domain bulunamadı"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}

	// Check permission
	if role != "admin" && domain.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Bu domain'e erişim yetkiniz yok"})
	}

	// Get DNS records
	rows, err := h.db.Query(`
		SELECT id, domain_id, name, type, content, ttl, priority, active, created_at, updated_at
		FROM dns_records
		WHERE domain_id = ?
		ORDER BY type, name
	`, domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}
	defer rows.Close()

	var records []DNSRecord
	for rows.Next() {
		var r DNSRecord
		var active int
		if err := rows.Scan(&r.ID, &r.DomainID, &r.Name, &r.Type, &r.Content, &r.TTL, &r.Priority, &active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		r.Active = active == 1
		records = append(records, r)
	}

	// If no records exist, create default records
	if len(records) == 0 {
		records, err = h.createDefaultDNSRecords(domainID, domain.Name)
		if err != nil {
			log.Printf("Warning: Could not create default DNS records: %v", err)
		}
	}

	serverIP := os.Getenv("SERVER_IP")
	if serverIP == "" {
		serverIP = "127.0.0.1"
	}

	return c.JSON(DNSZone{
		DomainID:   domain.ID,
		DomainName: domain.Name,
		Records:    records,
		ServerIP:   serverIP,
	})
}

// CreateDNSRecord creates a new DNS record
func (h *Handler) CreateDNSRecord(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var req struct {
		DomainID int64  `json:"domain_id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Content  string `json:"content"`
		TTL      int    `json:"ttl"`
		Priority int    `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz istek"})
	}

	// Validate record type
	if !isValidRecordType(req.Type) {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz kayıt tipi"})
	}

	// Check domain ownership
	var domain struct {
		ID     int64
		Name   string
		UserID int64
	}

	err := h.db.QueryRow(`
		SELECT id, name, user_id FROM domains WHERE id = ?
	`, req.DomainID).Scan(&domain.ID, &domain.Name, &domain.UserID)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Domain bulunamadı"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}

	// Check permission
	if role != "admin" && domain.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Bu domain'e erişim yetkiniz yok"})
	}

	// Validate and sanitize input
	req.Name = sanitizeDNSName(req.Name)
	if err := validateDNSRecord(req.Type, req.Name, req.Content); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Set default TTL
	if req.TTL <= 0 {
		req.TTL = 3600
	}

	// Insert record
	result, err := h.db.Exec(`
		INSERT INTO dns_records (domain_id, name, type, content, ttl, priority, active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, req.DomainID, req.Name, req.Type, req.Content, req.TTL, req.Priority)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kayıt oluşturulamadı"})
	}

	recordID, _ := result.LastInsertId()

	// Update zone file
	if err := h.updateZoneFile(domain.Name); err != nil {
		log.Printf("Warning: Could not update zone file: %v", err)
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "DNS kaydı oluşturuldu",
		"id":      recordID,
	})
}

// UpdateDNSRecord updates an existing DNS record
func (h *Handler) UpdateDNSRecord(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	recordID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz kayıt ID"})
	}

	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Content  string `json:"content"`
		TTL      int    `json:"ttl"`
		Priority int    `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz istek"})
	}

	// Get record and check ownership
	var record struct {
		DomainID   int64
		DomainName string
		UserID     int64
	}

	err = h.db.QueryRow(`
		SELECT r.domain_id, d.name, d.user_id
		FROM dns_records r
		JOIN domains d ON r.domain_id = d.id
		WHERE r.id = ?
	`, recordID).Scan(&record.DomainID, &record.DomainName, &record.UserID)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Kayıt bulunamadı"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}

	// Check permission
	if role != "admin" && record.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Bu kayda erişim yetkiniz yok"})
	}

	// Validate record type
	if !isValidRecordType(req.Type) {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz kayıt tipi"})
	}

	// Validate and sanitize input
	req.Name = sanitizeDNSName(req.Name)
	if err := validateDNSRecord(req.Type, req.Name, req.Content); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Set default TTL
	if req.TTL <= 0 {
		req.TTL = 3600
	}

	// Update record
	_, err = h.db.Exec(`
		UPDATE dns_records
		SET name = ?, type = ?, content = ?, ttl = ?, priority = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, req.Name, req.Type, req.Content, req.TTL, req.Priority, recordID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kayıt güncellenemedi"})
	}

	// Update zone file
	if err := h.updateZoneFile(record.DomainName); err != nil {
		log.Printf("Warning: Could not update zone file: %v", err)
	}

	return c.JSON(fiber.Map{"message": "DNS kaydı güncellendi"})
}

// DeleteDNSRecord deletes a DNS record
func (h *Handler) DeleteDNSRecord(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	recordID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz kayıt ID"})
	}

	// Get record and check ownership
	var record struct {
		DomainID   int64
		DomainName string
		UserID     int64
		RecordType string
		RecordName string
	}

	err = h.db.QueryRow(`
		SELECT r.domain_id, d.name, d.user_id, r.type, r.name
		FROM dns_records r
		JOIN domains d ON r.domain_id = d.id
		WHERE r.id = ?
	`, recordID).Scan(&record.DomainID, &record.DomainName, &record.UserID, &record.RecordType, &record.RecordName)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Kayıt bulunamadı"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}

	// Check permission
	if role != "admin" && record.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Bu kayda erişim yetkiniz yok"})
	}

	// Prevent deletion of critical records (SOA, root NS)
	if record.RecordType == "SOA" || (record.RecordType == "NS" && record.RecordName == "@") {
		return c.Status(400).JSON(fiber.Map{"error": "Bu kayıt silinemez (kritik kayıt)"})
	}

	// Delete record
	_, err = h.db.Exec(`DELETE FROM dns_records WHERE id = ?`, recordID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kayıt silinemedi"})
	}

	// Update zone file
	if err := h.updateZoneFile(record.DomainName); err != nil {
		log.Printf("Warning: Could not update zone file: %v", err)
	}

	return c.JSON(fiber.Map{"message": "DNS kaydı silindi"})
}

// ResetDNSZone resets DNS zone to default records
func (h *Handler) ResetDNSZone(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Geçersiz domain ID"})
	}

	// Check domain ownership
	var domain struct {
		ID     int64
		Name   string
		UserID int64
	}

	err = h.db.QueryRow(`
		SELECT id, name, user_id FROM domains WHERE id = ?
	`, domainID).Scan(&domain.ID, &domain.Name, &domain.UserID)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Domain bulunamadı"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası"})
	}

	// Check permission
	if role != "admin" && domain.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Bu domain'e erişim yetkiniz yok"})
	}

	// Delete all existing records
	_, err = h.db.Exec(`DELETE FROM dns_records WHERE domain_id = ?`, domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Kayıtlar silinemedi"})
	}

	// Create default records
	records, err := h.createDefaultDNSRecords(domainID, domain.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Varsayılan kayıtlar oluşturulamadı"})
	}

	// Update zone file
	if err := h.updateZoneFile(domain.Name); err != nil {
		log.Printf("Warning: Could not update zone file: %v", err)
	}

	return c.JSON(fiber.Map{
		"message": "DNS zone sıfırlandı",
		"records": records,
	})
}

// Helper functions

func (h *Handler) createDefaultDNSRecords(domainID int64, domainName string) ([]DNSRecord, error) {
	serverIP := os.Getenv("SERVER_IP")
	if serverIP == "" {
		serverIP = "127.0.0.1"
	}

	defaultRecords := []struct {
		Name     string
		Type     string
		Content  string
		TTL      int
		Priority int
	}{
		// A Records
		{"@", "A", serverIP, 3600, 0},
		{"www", "A", serverIP, 3600, 0},
		{"mail", "A", serverIP, 3600, 0},
		{"ftp", "A", serverIP, 3600, 0},
		// CNAME Records
		{"webmail", "CNAME", "mail." + domainName + ".", 3600, 0},
		// MX Records
		{"@", "MX", "mail." + domainName + ".", 3600, 10},
		// TXT Records (SPF)
		{"@", "TXT", "v=spf1 a mx ip4:" + serverIP + " ~all", 3600, 0},
		// NS Records
		{"@", "NS", "ns1.serverpanel.local.", 3600, 0},
		{"@", "NS", "ns2.serverpanel.local.", 3600, 0},
	}

	var records []DNSRecord
	for _, r := range defaultRecords {
		result, err := h.db.Exec(`
			INSERT INTO dns_records (domain_id, name, type, content, ttl, priority, active)
			VALUES (?, ?, ?, ?, ?, ?, 1)
		`, domainID, r.Name, r.Type, r.Content, r.TTL, r.Priority)

		if err != nil {
			log.Printf("Warning: Could not create default record %s %s: %v", r.Type, r.Name, err)
			continue
		}

		id, _ := result.LastInsertId()
		records = append(records, DNSRecord{
			ID:       id,
			DomainID: domainID,
			Name:     r.Name,
			Type:     r.Type,
			Content:  r.Content,
			TTL:      r.TTL,
			Priority: r.Priority,
			Active:   true,
		})
	}

	return records, nil
}

func (h *Handler) updateZoneFile(domainName string) error {
	// Get all records for this domain
	rows, err := h.db.Query(`
		SELECT name, type, content, ttl, priority
		FROM dns_records
		WHERE domain_id = (SELECT id FROM domains WHERE name = ?)
		AND active = 1
		ORDER BY type, name
	`, domainName)
	if err != nil {
		return err
	}
	defer rows.Close()

	var records []struct {
		Name     string
		Type     string
		Content  string
		TTL      int
		Priority int
	}

	for rows.Next() {
		var r struct {
			Name     string
			Type     string
			Content  string
			TTL      int
			Priority int
		}
		if err := rows.Scan(&r.Name, &r.Type, &r.Content, &r.TTL, &r.Priority); err != nil {
			continue
		}
		records = append(records, r)
	}

	// Generate zone file content
	zoneContent := h.generateZoneFileContent(domainName, records)

	// Write zone file using DNS manager
	dnsManager := dns.NewManager(h.cfg.SimulateMode, h.cfg.SimulateBasePath)
	zonePath := dnsManager.GetZonePath()
	zoneFile := zonePath + "/db." + domainName

	if err := os.WriteFile(zoneFile, []byte(zoneContent), 0644); err != nil {
		return fmt.Errorf("failed to write zone file: %w", err)
	}

	// Reload BIND
	return dnsManager.Reload()
}

func (h *Handler) generateZoneFileContent(domain string, records []struct {
	Name     string
	Type     string
	Content  string
	TTL      int
	Priority int
}) string {
	serial := time.Now().Format("2006010215")

	zone := fmt.Sprintf(`; Zone file for %s
; Generated by ServerPanel
; Last updated: %s
$TTL 3600
@       IN      SOA     ns1.serverpanel.local. hostmaster.%s. (
                        %s      ; Serial
                        3600            ; Refresh
                        1800            ; Retry
                        604800          ; Expire
                        86400 )         ; Minimum TTL

`, domain, time.Now().Format("2006-01-02 15:04:05"), domain, serial)

	// Group records by type
	recordsByType := make(map[string][]struct {
		Name     string
		Type     string
		Content  string
		TTL      int
		Priority int
	})

	for _, r := range records {
		recordsByType[r.Type] = append(recordsByType[r.Type], r)
	}

	// Write records in order: NS, A, AAAA, CNAME, MX, TXT, SRV, CAA
	typeOrder := []string{"NS", "A", "AAAA", "CNAME", "MX", "TXT", "SRV", "CAA"}

	for _, recordType := range typeOrder {
		recs, ok := recordsByType[recordType]
		if !ok || len(recs) == 0 {
			continue
		}

		zone += fmt.Sprintf("; %s Records\n", recordType)
		for _, r := range recs {
			name := r.Name
			if name == "" {
				name = "@"
			}

			switch r.Type {
			case "MX":
				zone += fmt.Sprintf("%-8s%d\tIN\t%s\t%d\t%s\n", name, r.TTL, r.Type, r.Priority, r.Content)
			case "SRV":
				zone += fmt.Sprintf("%-8s%d\tIN\t%s\t%d\t%s\n", name, r.TTL, r.Type, r.Priority, r.Content)
			case "TXT":
				// Ensure TXT content is quoted
				content := r.Content
				if !strings.HasPrefix(content, "\"") {
					content = "\"" + content + "\""
				}
				zone += fmt.Sprintf("%-8s%d\tIN\t%s\t%s\n", name, r.TTL, r.Type, content)
			default:
				zone += fmt.Sprintf("%-8s%d\tIN\t%s\t%s\n", name, r.TTL, r.Type, r.Content)
			}
		}
		zone += "\n"
	}

	return zone
}

func isValidRecordType(recordType string) bool {
	for _, t := range supportedRecordTypes {
		if t == recordType {
			return true
		}
	}
	return false
}

func sanitizeDNSName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	if name == "" || name == "@" {
		return "@"
	}
	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")
	return name
}

func validateDNSRecord(recordType, name, content string) error {
	if content == "" {
		return fmt.Errorf("içerik boş olamaz")
	}

	switch recordType {
	case "A":
		ip := net.ParseIP(content)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("geçersiz IPv4 adresi")
		}
	case "AAAA":
		ip := net.ParseIP(content)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("geçersiz IPv6 adresi")
		}
	case "CNAME", "NS", "MX":
		// Should be a valid hostname
		if !isValidHostname(content) {
			return fmt.Errorf("geçersiz hostname")
		}
	case "TXT":
		// TXT records can contain almost anything
		if len(content) > 255 {
			return fmt.Errorf("TXT kaydı 255 karakterden uzun olamaz")
		}
	case "SRV":
		// Format: weight port target
		parts := strings.Fields(content)
		if len(parts) < 3 {
			return fmt.Errorf("SRV kaydı formatı: weight port target")
		}
	case "CAA":
		// Format: flags tag value
		parts := strings.Fields(content)
		if len(parts) < 3 {
			return fmt.Errorf("CAA kaydı formatı: flags tag value")
		}
	}

	return nil
}

func isValidHostname(hostname string) bool {
	hostname = strings.TrimSuffix(hostname, ".")
	if len(hostname) > 253 {
		return false
	}
	// Simple hostname validation
	pattern := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`
	matched, _ := regexp.MatchString(pattern, hostname)
	return matched
}
