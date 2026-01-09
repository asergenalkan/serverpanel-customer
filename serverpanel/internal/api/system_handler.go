package api

import (
	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/asergenalkan/serverpanel/internal/system"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetSystemStats(c *fiber.Ctx) error {
	stats := system.GetSystemStats()
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (h *Handler) GetServices(c *fiber.Ctx) error {
	services := system.GetServices()
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    services,
	})
}

func (h *Handler) RestartService(c *fiber.Ctx) error {
	name := c.Params("name")

	// Whitelist allowed services
	allowedServices := map[string]bool{
		"nginx":      true,
		"apache2":    true,
		"mysql":      true,
		"mariadb":    true,
		"postgresql": true,
		"php-fpm":    true,
		"postfix":    true,
		"dovecot":    true,
		"named":      true,
		"pure-ftpd":  true,
	}

	if !allowedServices[name] {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid service name",
		})
	}

	if err := system.RestartService(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to restart service: " + err.Error(),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Service restarted successfully",
	})
}
