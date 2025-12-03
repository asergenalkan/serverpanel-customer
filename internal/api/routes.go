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
	protected.Get("/domains/limits", h.GetUserLimits)
	protected.Post("/domains", h.CreateDomain)
	protected.Get("/domains/:id", h.GetDomain)
	protected.Put("/domains/:id", h.UpdateDomain)
	protected.Delete("/domains/:id", h.DeleteDomain)

	// Subdomains (all authenticated users)
	protected.Get("/subdomains", h.ListSubdomains)
	protected.Post("/subdomains", h.CreateSubdomain)
	protected.Delete("/subdomains/:id", h.DeleteSubdomain)

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
	protected.Post("/ssl/issue-fqdn", h.IssueSSLForFQDN)
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

	// Email Management (all authenticated users)
	protected.Get("/email/accounts", h.ListEmailAccounts)
	protected.Post("/email/accounts", h.CreateEmailAccount)
	protected.Put("/email/accounts/:id", h.UpdateEmailAccount)
	protected.Delete("/email/accounts/:id", h.DeleteEmailAccount)
	protected.Post("/email/accounts/:id/toggle", h.ToggleEmailAccount)
	protected.Get("/email/forwarders", h.ListEmailForwarders)
	protected.Post("/email/forwarders", h.CreateEmailForwarder)
	protected.Delete("/email/forwarders/:id", h.DeleteEmailForwarder)
	protected.Get("/email/autoresponders", h.ListAutoresponders)
	protected.Post("/email/autoresponders", h.CreateAutoresponder)
	protected.Delete("/email/autoresponders/:id", h.DeleteAutoresponder)
	protected.Get("/email/webmail", h.GetWebmailURL)
	protected.Get("/email/stats", h.GetEmailStats)
	protected.Get("/email/settings/:domain_id", h.GetEmailSettings)
	protected.Put("/email/settings/:domain_id", h.UpdateEmailSettings)
	protected.Post("/email/dkim/:domain_id", h.GenerateDKIM)
	protected.Get("/email/dns-records/:domain_id", h.GetDNSRecordsForEmail)

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

	// Server Status (admin only)
	protected.Get("/server/info", admin, h.GetServerInfo)
	protected.Get("/server/daily-log", admin, h.GetDailyLog)
	protected.Get("/server/processes", admin, h.GetTopProcesses)
	protected.Get("/server/queue", admin, h.GetTaskQueue)
	protected.Post("/server/queue/flush", admin, h.FlushMailQueue)

	// Mail Queue Management (admin only)
	protected.Get("/mail-queue/stats", admin, h.GetMailQueueStats)
	protected.Get("/mail-queue", admin, h.GetMailQueue)
	protected.Delete("/mail-queue/:id", admin, h.DeleteMailQueueItem)
	protected.Post("/mail-queue/:id/retry", admin, h.RetryMailQueueItem)
	protected.Post("/mail-queue/clear", admin, h.ClearMailQueue)

	// User Mail Stats (all users)
	protected.Get("/email/my-stats", h.GetUserMailStats)

	// Software Manager (admin only)
	protected.Get("/software/overview", admin, h.GetSoftwareOverview)
	protected.Post("/software/php/install", admin, h.InstallPHPVersion)
	protected.Post("/software/php/uninstall", admin, h.UninstallPHPVersion)
	protected.Post("/software/php/extension/install", admin, h.InstallPHPExtension)
	protected.Post("/software/php/extension/uninstall", admin, h.UninstallPHPExtension)
	protected.Post("/software/apache/module/enable", admin, h.EnableApacheModule)
	protected.Post("/software/apache/module/disable", admin, h.DisableApacheModule)
	protected.Post("/software/install", admin, h.InstallSoftware)
	protected.Post("/software/uninstall", admin, h.UninstallSoftware)

	// Server Settings (admin only)
	protected.Get("/settings/server", admin, h.GetServerSettings)
	protected.Put("/settings/server", admin, h.UpdateServerSettings)

	// Server Features (all users - read only)
	protected.Get("/server/features", h.GetServerFeatures)
	protected.Get("/php/allowed-versions", h.GetAllowedPHPVersions)

	// Task Management (admin only)
	protected.Post("/tasks/start", admin, h.StartInstallTask)
	protected.Get("/tasks/:task_id", admin, h.GetTaskStatus)

	// Spam Filters (all authenticated users)
	protected.Get("/spam/settings", h.GetSpamSettings)
	protected.Put("/spam/settings", h.UpdateSpamSettings)
	protected.Post("/spam/update-clamav", admin, h.UpdateClamAV)
	protected.Get("/spam/global", admin, h.GetGlobalSpamSettings)
	protected.Post("/spam/toggle-service", admin, h.ToggleSpamService)

	// Note: WebSocket route is defined in main.go to avoid SPA fallback conflict
}
