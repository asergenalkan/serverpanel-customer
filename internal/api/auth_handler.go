package api

import (
	"database/sql"
	"time"

	"github.com/asergenalkan/serverpanel/internal/auth"
	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Username and password are required",
		})
	}

	// Find user
	var user models.User
	var password string
	err := h.db.QueryRow(`
		SELECT id, username, email, password, role, parent_id, active, created_at, updated_at
		FROM users WHERE username = ? AND active = 1
	`, req.Username).Scan(
		&user.ID, &user.Username, &user.Email, &password,
		&user.Role, &user.ParentID, &user.Active, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Database error",
		})
	}

	// Verify password
	if !auth.CheckPassword(req.Password, password) {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	// Generate token
	token, err := auth.GenerateToken(&user, h.cfg.JWTSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
	}

	// Log activity
	h.logActivity(user.ID, "login", "User logged in", c.IP())

	return c.JSON(models.APIResponse{
		Success: true,
		Data: models.LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var user models.User
	err := h.db.QueryRow(`
		SELECT id, username, email, role, parent_id, active, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.Role, &user.ParentID, &user.Active, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "User not found",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    user,
	})
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	h.logActivity(userID, "logout", "User logged out", c.IP())

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

func (h *Handler) Health(c *fiber.Ctx) error {
	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		},
	})
}

func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Mevcut şifre ve yeni şifre gereklidir",
		})
	}

	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Yeni şifre en az 6 karakter olmalıdır",
		})
	}

	// Get current password hash
	var currentHash string
	err := h.db.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Kullanıcı bulunamadı",
		})
	}

	// Verify current password
	if !auth.CheckPassword(req.CurrentPassword, currentHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Mevcut şifre yanlış",
		})
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Şifre oluşturma hatası",
		})
	}

	// Update password
	_, err = h.db.Exec("UPDATE users SET password = ?, updated_at = NOW() WHERE id = ?", newHash, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Şifre güncellenemedi",
		})
	}

	h.logActivity(userID, "password_change", "User changed password", c.IP())

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Şifre başarıyla değiştirildi",
	})
}

func (h *Handler) logActivity(userID int64, action, details, ip string) {
	h.db.Exec(`
		INSERT INTO activity_logs (user_id, action, details, ip_address)
		VALUES (?, ?, ?, ?)
	`, userID, action, details, ip)
}
