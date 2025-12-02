package api

import (
	"strconv"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// Package represents a hosting package with all limits
type Package struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	DiskQuota           int    `json:"disk_quota"`
	BandwidthQuota      int    `json:"bandwidth_quota"`
	MaxDomains          int    `json:"max_domains"`
	MaxDatabases        int    `json:"max_databases"`
	MaxEmails           int    `json:"max_emails"`
	MaxFTP              int    `json:"max_ftp"`
	MaxPHPMemory        string `json:"max_php_memory"`
	MaxPHPUpload        string `json:"max_php_upload"`
	MaxPHPExecutionTime int    `json:"max_php_execution_time"`
	CreatedAt           string `json:"created_at"`
	UserCount           int    `json:"user_count,omitempty"`
}

func (h *Handler) ListPackages(c *fiber.Ctx) error {
	rows, err := h.db.Query(`
		SELECT p.id, p.name, p.disk_quota, p.bandwidth_quota, p.max_domains, 
		       p.max_databases, p.max_emails, p.max_ftp, 
		       p.max_php_memory, p.max_php_upload, p.max_php_execution_time,
		       p.created_at,
		       (SELECT COUNT(*) FROM user_packages WHERE package_id = p.id) as user_count
		FROM packages p ORDER BY p.name
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch packages",
		})
	}
	defer rows.Close()

	var packages []Package
	for rows.Next() {
		var p Package
		if err := rows.Scan(&p.ID, &p.Name, &p.DiskQuota, &p.BandwidthQuota, &p.MaxDomains,
			&p.MaxDatabases, &p.MaxEmails, &p.MaxFTP,
			&p.MaxPHPMemory, &p.MaxPHPUpload, &p.MaxPHPExecutionTime,
			&p.CreatedAt, &p.UserCount); err != nil {
			continue
		}
		packages = append(packages, p)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    packages,
	})
}

func (h *Handler) GetPackage(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid package ID",
		})
	}

	var p Package
	err = h.db.QueryRow(`
		SELECT id, name, disk_quota, bandwidth_quota, max_domains, 
		       max_databases, max_emails, max_ftp,
		       max_php_memory, max_php_upload, max_php_execution_time, created_at
		FROM packages WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.DiskQuota, &p.BandwidthQuota, &p.MaxDomains,
		&p.MaxDatabases, &p.MaxEmails, &p.MaxFTP,
		&p.MaxPHPMemory, &p.MaxPHPUpload, &p.MaxPHPExecutionTime, &p.CreatedAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Package not found",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    p,
	})
}

func (h *Handler) CreatePackage(c *fiber.Ctx) error {
	var pkg Package
	if err := c.BodyParser(&pkg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if pkg.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Package name is required",
		})
	}

	// Set defaults
	if pkg.DiskQuota == 0 {
		pkg.DiskQuota = 1024
	}
	if pkg.BandwidthQuota == 0 {
		pkg.BandwidthQuota = 10240
	}
	if pkg.MaxDomains == 0 {
		pkg.MaxDomains = 1
	}
	if pkg.MaxDatabases == 0 {
		pkg.MaxDatabases = 1
	}
	if pkg.MaxEmails == 0 {
		pkg.MaxEmails = 5
	}
	if pkg.MaxFTP == 0 {
		pkg.MaxFTP = 1
	}
	if pkg.MaxPHPMemory == "" {
		pkg.MaxPHPMemory = "256M"
	}
	if pkg.MaxPHPUpload == "" {
		pkg.MaxPHPUpload = "64M"
	}
	if pkg.MaxPHPExecutionTime == 0 {
		pkg.MaxPHPExecutionTime = 300
	}

	result, err := h.db.Exec(`
		INSERT INTO packages (name, disk_quota, bandwidth_quota, max_domains, max_databases, max_emails, max_ftp, max_php_memory, max_php_upload, max_php_execution_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pkg.Name, pkg.DiskQuota, pkg.BandwidthQuota, pkg.MaxDomains, pkg.MaxDatabases, pkg.MaxEmails, pkg.MaxFTP, pkg.MaxPHPMemory, pkg.MaxPHPUpload, pkg.MaxPHPExecutionTime)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Package name already exists",
		})
	}

	id, _ := result.LastInsertId()

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Package created successfully",
		Data:    map[string]int64{"id": id},
	})
}

func (h *Handler) UpdatePackage(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid package ID",
		})
	}

	var pkg Package
	if err := c.BodyParser(&pkg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	_, err = h.db.Exec(`
		UPDATE packages SET name = ?, disk_quota = ?, bandwidth_quota = ?, 
		max_domains = ?, max_databases = ?, max_emails = ?, max_ftp = ?,
		max_php_memory = ?, max_php_upload = ?, max_php_execution_time = ?
		WHERE id = ?
	`, pkg.Name, pkg.DiskQuota, pkg.BandwidthQuota, pkg.MaxDomains, pkg.MaxDatabases, pkg.MaxEmails, pkg.MaxFTP, pkg.MaxPHPMemory, pkg.MaxPHPUpload, pkg.MaxPHPExecutionTime, id)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to update package",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Package updated successfully",
	})
}

func (h *Handler) DeletePackage(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid package ID",
		})
	}

	// Check if package is in use
	var count int
	h.db.QueryRow("SELECT COUNT(*) FROM user_packages WHERE package_id = ?", id).Scan(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Cannot delete package that is in use",
		})
	}

	_, err = h.db.Exec("DELETE FROM packages WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to delete package",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Package deleted successfully",
	})
}
