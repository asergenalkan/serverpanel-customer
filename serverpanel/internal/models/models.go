package models

import "time"

// User roles
const (
	RoleAdmin    = "admin"
	RoleReseller = "reseller"
	RoleUser     = "user"
)

// User represents a panel user
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never expose password
	Role      string    `json:"role"`
	ParentID  *int64    `json:"parent_id,omitempty"` // For reseller hierarchy
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Package represents a hosting package
type Package struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	DiskQuota      int64     `json:"disk_quota"`      // MB
	BandwidthQuota int64     `json:"bandwidth_quota"` // MB per month
	MaxDomains     int       `json:"max_domains"`
	MaxDatabases   int       `json:"max_databases"`
	MaxEmails      int       `json:"max_emails"`
	MaxFTP         int       `json:"max_ftp"`
	CreatedAt      time.Time `json:"created_at"`
}

// Domain represents a hosted domain
type Domain struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Name         string     `json:"name"`
	DocumentRoot string     `json:"document_root"`
	SSLEnabled   bool       `json:"ssl_enabled"`
	SSLExpiry    *time.Time `json:"ssl_expiry,omitempty"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Database represents a MySQL/PostgreSQL database
type Database struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"` // Panel username for display
	Name      string    `json:"name"`
	Type      string    `json:"type"` // mysql, postgresql
	Size      int64     `json:"size"` // bytes
	CreatedAt time.Time `json:"created_at"`
}

// DatabaseUser represents a database user
type DatabaseUser struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`     // Panel user ID
	DatabaseID int64     `json:"database_id"` // Associated database
	DBUsername string    `json:"db_username"` // MySQL username
	Host       string    `json:"host"`        // Usually localhost
	CreatedAt  time.Time `json:"created_at"`
}

// EmailAccount represents an email account
type EmailAccount struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	DomainID  int64     `json:"domain_id"`
	Email     string    `json:"email"`
	Quota     int64     `json:"quota"` // MB
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// SystemStats represents server statistics
type SystemStats struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryTotal int64     `json:"memory_total"`
	MemoryUsed  int64     `json:"memory_used"`
	DiskTotal   int64     `json:"disk_total"`
	DiskUsed    int64     `json:"disk_used"`
	Uptime      int64     `json:"uptime"`
	LoadAverage []float64 `json:"load_average"`
}

// LoginRequest represents login payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
