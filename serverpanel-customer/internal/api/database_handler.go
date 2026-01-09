package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// Token store for phpMyAdmin SSO
var (
	pmaTokens = make(map[string]pmaTokenData)
	pmaMutex  sync.RWMutex
)

type pmaTokenData struct {
	User     string
	Password string
	DB       string
	Expires  time.Time
}

// MySQL connection for real database operations
func (h *Handler) getMySQLConnection() (*sql.DB, error) {
	// Read MySQL root password from config or environment
	mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
	if mysqlPassword == "" {
		mysqlPassword = "root" // Default for development
	}

	dsn := fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/", mysqlPassword)
	return sql.Open("mysql", dsn)
}

// Generate random password
func generatePassword(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// Validate database/user name (only alphanumeric and underscore)
func isValidDBName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]*$`, name)
	return matched && len(name) <= 64
}

// Get username from user_id
func (h *Handler) getUsernameByID(userID int64) string {
	var username string
	h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	return username
}

func (h *Handler) ListDatabases(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT d.id, d.user_id, u.username, d.name, d.type, d.size, d.created_at 
			FROM databases d 
			LEFT JOIN users u ON d.user_id = u.id 
			ORDER BY d.name
		`
	} else {
		query = `
			SELECT d.id, d.user_id, u.username, d.name, d.type, d.size, d.created_at 
			FROM databases d 
			LEFT JOIN users u ON d.user_id = u.id 
			WHERE d.user_id = ? 
			ORDER BY d.name
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch databases",
		})
	}
	defer rows.Close()

	var databases []models.Database
	for rows.Next() {
		var db models.Database
		var username sql.NullString
		if err := rows.Scan(&db.ID, &db.UserID, &username, &db.Name, &db.Type, &db.Size, &db.CreatedAt); err != nil {
			continue
		}
		if username.Valid {
			db.Username = username.String
		}
		databases = append(databases, db)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    databases,
	})
}

func (h *Handler) CreateDatabase(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	username := c.Locals("username").(string)

	var req struct {
		Name     string `json:"name"`
		Password string `json:"password"` // Optional, will be generated if empty
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Database name is required",
		})
	}

	// Validate name
	if !isValidDBName(req.Name) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid database name. Use only letters, numbers and underscores",
		})
	}

	// Create full database name with username prefix for isolation
	fullDBName := fmt.Sprintf("%s_%s", username, req.Name)
	dbUser := fmt.Sprintf("%s_%s", username, req.Name)

	// Limit length
	if len(fullDBName) > 64 {
		fullDBName = fullDBName[:64]
	}
	if len(dbUser) > 32 {
		dbUser = dbUser[:32]
	}

	// Generate password if not provided
	password := req.Password
	if password == "" {
		password = generatePassword(16)
	}

	// Check if running in production mode
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	if isProduction {
		// Create real MySQL database
		mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
		if mysqlPassword == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "MySQL root password not configured",
			})
		}

		// Create database using mysql command
		createDBCmd := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", fullDBName)
		cmd := exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", createDBCmd)
		if output, err := cmd.CombinedOutput(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to create database: %s", string(output)),
			})
		}

		// Create user and grant privileges
		createUserCmd := fmt.Sprintf(
			"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; FLUSH PRIVILEGES;",
			dbUser, password, fullDBName, dbUser,
		)
		cmd = exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", createUserCmd)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Rollback: drop database
			dropCmd := exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", fullDBName))
			dropCmd.Run()

			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to create database user: %s", string(output)),
			})
		}
	}

	// Save to panel database
	result, err := h.db.Exec(`
		INSERT INTO databases (user_id, name, type, size)
		VALUES (?, ?, 'mysql', 0)
	`, userID, fullDBName)

	if err != nil {
		// Rollback MySQL changes if production
		if isProduction {
			mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
			exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e",
				fmt.Sprintf("DROP DATABASE IF EXISTS `%s`; DROP USER IF EXISTS '%s'@'localhost';", fullDBName, dbUser)).Run()
		}

		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Database name already exists",
		})
	}

	id, _ := result.LastInsertId()

	// Save database user info with password
	h.db.Exec(`
		INSERT INTO database_users (user_id, database_id, db_username, password, host)
		VALUES (?, ?, ?, ?, 'localhost')
	`, userID, id, dbUser, password)

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Database created successfully",
		Data: map[string]interface{}{
			"id":       id,
			"name":     fullDBName,
			"username": dbUser,
			"password": password,
			"host":     "localhost",
		},
	})
}

func (h *Handler) DeleteDatabase(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid database ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get database info
	var dbName string
	var dbUserID int64
	err = h.db.QueryRow("SELECT user_id, name FROM databases WHERE id = ?", id).Scan(&dbUserID, &dbName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Database not found",
		})
	}

	// Check ownership unless admin
	if role != models.RoleAdmin && dbUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Permission denied",
		})
	}

	// Get database user
	var dbUser string
	h.db.QueryRow("SELECT db_username FROM database_users WHERE database_id = ?", id).Scan(&dbUser)

	isProduction := os.Getenv("ENVIRONMENT") == "production"

	if isProduction {
		mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
		if mysqlPassword == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "MySQL root password not configured",
			})
		}

		// Drop user first
		if dbUser != "" {
			dropUserCmd := fmt.Sprintf("DROP USER IF EXISTS '%s'@'localhost';", dbUser)
			exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", dropUserCmd).Run()
		}

		// Drop database
		dropDBCmd := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName)
		cmd := exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", dropDBCmd)
		if output, err := cmd.CombinedOutput(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to drop database: %s", string(output)),
			})
		}
	}

	// Delete from database_users
	h.db.Exec("DELETE FROM database_users WHERE database_id = ?", id)

	// Delete from databases
	_, err = h.db.Exec("DELETE FROM databases WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to delete database record",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Database deleted successfully",
	})
}

// ListDatabaseUsers returns all database users for a database
func (h *Handler) ListDatabaseUsers(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	var query string
	var args []interface{}

	if role == models.RoleAdmin {
		query = `
			SELECT du.id, du.user_id, du.database_id, du.db_username, du.host, du.created_at, d.name as db_name
			FROM database_users du
			LEFT JOIN databases d ON du.database_id = d.id
			ORDER BY du.db_username
		`
	} else {
		query = `
			SELECT du.id, du.user_id, du.database_id, du.db_username, du.host, du.created_at, d.name as db_name
			FROM database_users du
			LEFT JOIN databases d ON du.database_id = d.id
			WHERE du.user_id = ?
			ORDER BY du.db_username
		`
		args = append(args, userID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to fetch database users",
		})
	}
	defer rows.Close()

	type DBUserWithName struct {
		models.DatabaseUser
		DatabaseName string `json:"database_name"`
	}

	var users []DBUserWithName
	for rows.Next() {
		var u DBUserWithName
		var dbName sql.NullString
		if err := rows.Scan(&u.ID, &u.UserID, &u.DatabaseID, &u.DBUsername, &u.Host, &u.CreatedAt, &dbName); err != nil {
			continue
		}
		if dbName.Valid {
			u.DatabaseName = dbName.String
		}
		users = append(users, u)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    users,
	})
}

// CreateDatabaseUser creates a new database user
func (h *Handler) CreateDatabaseUser(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	username := c.Locals("username").(string)
	role := c.Locals("role").(string)

	var req struct {
		DatabaseID int64  `json:"database_id"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}

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

	if !isValidDBName(req.Username) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid username. Use only letters, numbers and underscores",
		})
	}

	// Get database info and verify ownership
	var dbName string
	var dbUserID int64
	err := h.db.QueryRow("SELECT user_id, name FROM databases WHERE id = ?", req.DatabaseID).Scan(&dbUserID, &dbName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Database not found",
		})
	}

	if role != models.RoleAdmin && dbUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Permission denied",
		})
	}

	// Create full username with prefix
	fullUsername := fmt.Sprintf("%s_%s", username, req.Username)
	if len(fullUsername) > 32 {
		fullUsername = fullUsername[:32]
	}

	isProduction := os.Getenv("ENVIRONMENT") == "production"

	if isProduction {
		mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
		if mysqlPassword == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "MySQL root password not configured",
			})
		}

		createUserCmd := fmt.Sprintf(
			"CREATE USER '%s'@'localhost' IDENTIFIED BY '%s'; GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; FLUSH PRIVILEGES;",
			fullUsername, req.Password, dbName, fullUsername,
		)
		cmd := exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", createUserCmd)
		if output, err := cmd.CombinedOutput(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to create user: %s", string(output)),
			})
		}
	}

	result, err := h.db.Exec(`
		INSERT INTO database_users (user_id, database_id, db_username, password, host)
		VALUES (?, ?, ?, ?, 'localhost')
	`, userID, req.DatabaseID, fullUsername, req.Password)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Username already exists",
		})
	}

	id, _ := result.LastInsertId()

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Database user created successfully",
		Data: map[string]interface{}{
			"id":       id,
			"username": fullUsername,
			"password": req.Password,
			"host":     "localhost",
		},
	})
}

// DeleteDatabaseUser deletes a database user
func (h *Handler) DeleteDatabaseUser(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get user info
	var dbUser string
	var dbUserID int64
	err = h.db.QueryRow("SELECT user_id, db_username FROM database_users WHERE id = ?", id).Scan(&dbUserID, &dbUser)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Database user not found",
		})
	}

	if role != models.RoleAdmin && dbUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Permission denied",
		})
	}

	isProduction := os.Getenv("ENVIRONMENT") == "production"

	if isProduction {
		mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
		if mysqlPassword == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "MySQL root password not configured",
			})
		}

		dropUserCmd := fmt.Sprintf("DROP USER IF EXISTS '%s'@'localhost'; FLUSH PRIVILEGES;", dbUser)
		cmd := exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-e", dropUserCmd)
		if output, err := cmd.CombinedOutput(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to drop user: %s", string(output)),
			})
		}
	}

	_, err = h.db.Exec("DELETE FROM database_users WHERE id = ?", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to delete database user",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Database user deleted successfully",
	})
}

// GetPhpMyAdminURL returns phpMyAdmin URL with auto-login token
func (h *Handler) GetPhpMyAdminURL(c *fiber.Ctx) error {
	dbID, err := strconv.ParseInt(c.Query("database_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Database ID is required",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get database info
	var dbName string
	var dbUserID int64
	err = h.db.QueryRow("SELECT user_id, name FROM databases WHERE id = ?", dbID).Scan(&dbUserID, &dbName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Database not found",
		})
	}

	if role != models.RoleAdmin && dbUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Permission denied",
		})
	}

	// Get database user and password
	var dbUser, dbPassword string
	err = h.db.QueryRow("SELECT db_username, password FROM database_users WHERE database_id = ? LIMIT 1", dbID).Scan(&dbUser, &dbPassword)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Database user not found",
		})
	}

	// Generate random token
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	// Store token in memory
	// Token süresini 5 dakika yapıyoruz çünkü SignonScript her request'te çağrılıyor
	pmaMutex.Lock()
	pmaTokens[token] = pmaTokenData{
		User:     dbUser,
		Password: dbPassword,
		DB:       dbName,
		Expires:  time.Now().Add(5 * time.Minute), // 5 minutes valid (SignonScript her request'te çağrılıyor)
	}
	pmaMutex.Unlock()

	// Get server IP/host for URL
	// Try to get from request header first (for reverse proxy scenarios)
	serverHost := c.Get("Host")
	if serverHost == "" {
		// Fallback to environment variable
		serverHost = os.Getenv("SERVER_IP")
		if serverHost == "" {
			serverHost = "localhost"
		}
	}

	// Determine protocol (http or https)
	protocol := "http"
	if c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" {
		protocol = "https"
	}

	// Remove port from host if present (Apache serves on port 80/443)
	if idx := strings.Index(serverHost, ":"); idx != -1 {
		serverHost = serverHost[:idx]
	}

	// Return signon URL (pma-signon.php üzerinden geçecek)
	pmaURL := fmt.Sprintf("%s://%s/pma-signon.php?token=%s", protocol, serverHost, token)

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"url":      pmaURL,
			"database": dbName,
			"username": dbUser,
		},
	})
}

// GetPhpMyAdminCredentials retrieves credentials for a given token (Internal use)
// Token'ı consume etmiyoruz çünkü SignonScript her request'te çağrılıyor
func (h *Handler) GetPhpMyAdminCredentials(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token required"})
	}

	pmaMutex.RLock()
	data, exists := pmaTokens[token]
	if exists {
		// Check expiration
		if time.Now().After(data.Expires) {
			exists = false
		}
	}
	pmaMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	// Token'ı consume etmiyoruz - SignonScript her request'te çağrıldığı için
	// Token süresi dolduğunda otomatik olarak geçersiz olacak
	return c.JSON(fiber.Map{
		"user":     data.User,
		"password": data.Password,
		"host":     "localhost",
		"db":       data.DB,
	})
}

// GetDatabaseSize returns the size of a specific database
func (h *Handler) GetDatabaseSize(c *fiber.Ctx) error {
	dbID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid database ID",
		})
	}

	userID := c.Locals("user_id").(int64)
	role := c.Locals("role").(string)

	// Get database info
	var dbName string
	var dbUserID int64
	err = h.db.QueryRow("SELECT user_id, name FROM databases WHERE id = ?", dbID).Scan(&dbUserID, &dbName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Database not found",
		})
	}

	if role != models.RoleAdmin && dbUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "Permission denied",
		})
	}

	isProduction := os.Getenv("ENVIRONMENT") == "production"
	var size int64 = 0

	if isProduction {
		mysqlPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
		if mysqlPassword != "" {
			// Get database size from MySQL
			sizeCmd := fmt.Sprintf(
				"SELECT COALESCE(SUM(data_length + index_length), 0) FROM information_schema.tables WHERE table_schema = '%s';",
				dbName,
			)
			cmd := exec.Command("mysql", "-u", "root", fmt.Sprintf("-p%s", mysqlPassword), "-N", "-e", sizeCmd)
			output, err := cmd.Output()
			if err == nil {
				fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &size)

				// Update size in panel database
				h.db.Exec("UPDATE databases SET size = ? WHERE id = ?", size, dbID)
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"size":       size,
			"size_human": formatBytes(size),
		},
	})
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
