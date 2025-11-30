package api

import (
	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/database"
	"github.com/asergenalkan/serverpanel/internal/middleware"
	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	db  *database.DB
	cfg *config.Config
}

func SetupRoutes(router fiber.Router, db *database.DB) {
	cfg := config.Load()
	h := &Handler{db: db, cfg: cfg}

	// Public routes
	router.Post("/auth/login", h.Login)
	router.Get("/health", h.Health)

	// Protected routes
	protected := router.Group("/", middleware.AuthMiddleware(cfg.JWTSecret))

	// Auth
	protected.Get("/auth/me", h.GetCurrentUser)
	protected.Post("/auth/logout", h.Logout)

	// Dashboard
	protected.Get("/dashboard/stats", h.GetDashboardStats)

	// Admin middleware for admin-only routes
	admin := middleware.RoleMiddleware(models.RoleAdmin)

	// Users (admin only)
	protected.Get("/users", admin, h.ListUsers)
	protected.Post("/users", admin, h.CreateUser)
	protected.Get("/users/:id", admin, h.GetUser)
	protected.Put("/users/:id", admin, h.UpdateUser)
	protected.Delete("/users/:id", admin, h.DeleteUser)

	// Packages (admin only)
	protected.Get("/packages", admin, h.ListPackages)
	protected.Post("/packages", admin, h.CreatePackage)
	protected.Put("/packages/:id", admin, h.UpdatePackage)
	protected.Delete("/packages/:id", admin, h.DeletePackage)

	// Accounts - Hosting hesapları (admin only)
	protected.Get("/accounts", admin, h.ListAccounts)
	protected.Post("/accounts", admin, h.CreateAccount)
	protected.Get("/accounts/:id", admin, h.GetAccount)
	protected.Delete("/accounts/:id", admin, h.DeleteAccount)
	protected.Post("/accounts/:id/suspend", admin, h.SuspendAccount)
	protected.Post("/accounts/:id/unsuspend", admin, h.UnsuspendAccount)

	// Domains (all authenticated users)
	protected.Get("/domains", h.ListDomains)
	protected.Post("/domains", h.CreateDomain)
	protected.Get("/domains/:id", h.GetDomain)
	protected.Delete("/domains/:id", h.DeleteDomain)

	// Databases (all authenticated users)
	protected.Get("/databases", h.ListDatabases)
	protected.Post("/databases", h.CreateDatabase)
	protected.Delete("/databases/:id", h.DeleteDatabase)

	// System (admin only)
	protected.Get("/system/stats", admin, h.GetSystemStats)
	protected.Get("/system/services", admin, h.GetServices)
	protected.Post("/system/services/:name/restart", admin, h.RestartService)

	// File Manager (all authenticated users)
	protected.Get("/files/list", h.ListFiles)
	protected.Get("/files/read", h.ReadFile)
	protected.Post("/files/write", h.WriteFile)
	protected.Post("/files/mkdir", h.CreateDirectory)
	protected.Post("/files/delete", h.DeleteFiles)
	protected.Post("/files/rename", h.RenameFile)
	protected.Post("/files/copy", h.CopyFiles)
	protected.Post("/files/move", h.MoveFiles)
	protected.Post("/files/upload", h.UploadFiles)
	protected.Get("/files/download", h.DownloadFile)
	protected.Post("/files/compress", h.CompressFiles)
	protected.Post("/files/extract", h.ExtractFiles)
	protected.Get("/files/info", h.GetFileInfo)
}
