package api

import (
	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/asergenalkan/serverpanel/internal/system"
	"github.com/gofiber/fiber/v2"
)

type DashboardStats struct {
	TotalUsers     int                 `json:"total_users"`
	TotalDomains   int                 `json:"total_domains"`
	TotalDatabases int                 `json:"total_databases"`
	TotalEmails    int                 `json:"total_emails"`
	SystemStats    *models.SystemStats `json:"system_stats"`
}

func (h *Handler) GetDashboardStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var stats DashboardStats

	// Get counts based on role
	if role == models.RoleAdmin {
		h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
		h.db.QueryRow("SELECT COUNT(*) FROM domains").Scan(&stats.TotalDomains)
		h.db.QueryRow("SELECT COUNT(*) FROM databases").Scan(&stats.TotalDatabases)
		h.db.QueryRow("SELECT COUNT(*) FROM email_accounts").Scan(&stats.TotalEmails)
	} else if role == models.RoleReseller {
		// Get counts for reseller's clients
		h.db.QueryRow("SELECT COUNT(*) FROM users WHERE parent_id = ?", userID).Scan(&stats.TotalUsers)
		h.db.QueryRow(`
			SELECT COUNT(*) FROM domains d 
			JOIN users u ON d.user_id = u.id 
			WHERE u.parent_id = ? OR u.id = ?
		`, userID, userID).Scan(&stats.TotalDomains)
		h.db.QueryRow(`
			SELECT COUNT(*) FROM databases db 
			JOIN users u ON db.user_id = u.id 
			WHERE u.parent_id = ? OR u.id = ?
		`, userID, userID).Scan(&stats.TotalDatabases)
		h.db.QueryRow(`
			SELECT COUNT(*) FROM email_accounts e 
			JOIN users u ON e.user_id = u.id 
			WHERE u.parent_id = ? OR u.id = ?
		`, userID, userID).Scan(&stats.TotalEmails)
	} else {
		// Regular user - only their own resources
		h.db.QueryRow("SELECT COUNT(*) FROM domains WHERE user_id = ?", userID).Scan(&stats.TotalDomains)
		h.db.QueryRow("SELECT COUNT(*) FROM databases WHERE user_id = ?", userID).Scan(&stats.TotalDatabases)
		h.db.QueryRow("SELECT COUNT(*) FROM email_accounts WHERE user_id = ?", userID).Scan(&stats.TotalEmails)
		stats.TotalUsers = 1
	}

	// Get system stats (admin only for detailed view)
	if role == models.RoleAdmin {
		stats.SystemStats = system.GetSystemStats()
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    stats,
	})
}
