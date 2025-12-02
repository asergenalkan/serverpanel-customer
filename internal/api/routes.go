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
	router.Get("/internal/pma-credentials", h.GetPhpMyAdminCredentials)

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
	protected.Get("/packages/:id", admin, h.GetPackage)
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
	protected.Get("/databases/:id/size", h.GetDatabaseSize)
	protected.Get("/databases/phpmyadmin", h.GetPhpMyAdminURL)

	// Database Users (all authenticated users)
	protected.Get("/database-users", h.ListDatabaseUsers)
	protected.Post("/database-users", h.CreateDatabaseUser)
	protected.Delete("/database-users/:id", h.DeleteDatabaseUser)

	// System (admin only)
	protected.Get("/system/stats", admin, h.GetSystemStats)
	protected.Get("/system/services", admin, h.GetServices)
	protected.Post("/system/services/:name/restart", admin, h.RestartService)

	// SSL Certificates (all authenticated users)
	protected.Get("/ssl", h.ListSSLCertificates)
	protected.Get("/ssl/:id", h.GetSSLCertificate)
	protected.Post("/ssl/:id/issue", h.IssueSSLCertificate)
	protected.Post("/ssl/:id/renew", h.RenewSSLCertificate)
	protected.Delete("/ssl/:id", h.RevokeSSLCertificate)

	// PHP Management (all authenticated users)
	protected.Get("/php/versions", h.GetInstalledPHPVersions)
	protected.Get("/php/domains/:id", h.GetDomainPHPSettings)
	protected.Put("/php/domains/:id/version", h.UpdateDomainPHPVersion)
	protected.Put("/php/domains/:id/settings", h.UpdateDomainPHPSettings)

	// FTP Management (all authenticated users)
	protected.Get("/ftp/accounts", h.ListFTPAccounts)
	protected.Post("/ftp/accounts", h.CreateFTPAccount)
	protected.Put("/ftp/accounts/:id", h.UpdateFTPAccount)
	protected.Delete("/ftp/accounts/:id", h.DeleteFTPAccount)
	protected.Post("/ftp/accounts/:id/toggle", h.ToggleFTPAccount)

	// FTP Server Settings (admin only)
	protected.Get("/ftp/settings", admin, h.GetFTPSettings)
	protected.Put("/ftp/settings", admin, h.UpdateFTPSettings)
	protected.Get("/ftp/status", admin, h.GetFTPServerStatus)
	protected.Post("/ftp/restart", admin, h.RestartFTPServer)

	// DNS Management (all authenticated users)
	protected.Get("/dns/zones", h.ListDNSZones)
	protected.Get("/dns/zones/:id", h.GetDNSZone)
	protected.Post("/dns/records", h.CreateDNSRecord)
	protected.Put("/dns/records/:id", h.UpdateDNSRecord)
	protected.Delete("/dns/records/:id", h.DeleteDNSRecord)
	protected.Post("/dns/zones/:id/reset", h.ResetDNSZone)

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
