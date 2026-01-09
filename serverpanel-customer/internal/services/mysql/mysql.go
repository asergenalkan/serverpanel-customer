package mysql

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
)

// Manager handles MySQL database and user operations
type Manager struct {
	simulateMode bool
	basePath     string
	rootUser     string
	rootPass     string
	host         string
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Name     string
	Username string
	Password string
	Charset  string
}

// NewManager creates a new MySQL manager
func NewManager(simulateMode bool, basePath string) *Manager {
	return &Manager{
		simulateMode: simulateMode,
		basePath:     basePath,
		rootUser:     "root",
		rootPass:     os.Getenv("MYSQL_ROOT_PASSWORD"),
		host:         "localhost",
	}
}

// CreateDatabase creates a MySQL database and user for a hosting account
func (m *Manager) CreateDatabase(config DatabaseConfig) error {
	if config.Charset == "" {
		config.Charset = "utf8mb4"
	}

	if m.simulateMode {
		return m.simulateCreateDatabase(config)
	}

	return m.realCreateDatabase(config)
}

func (m *Manager) simulateCreateDatabase(config DatabaseConfig) error {
	// Create simulation log
	simPath := filepath.Join(m.basePath, "mysql")
	os.MkdirAll(simPath, 0755)

	logFile := filepath.Join(simPath, config.Name+".sql")
	sql := fmt.Sprintf(`-- Database: %s
-- User: %s
-- Created by ServerPanel

CREATE DATABASE IF NOT EXISTS %s CHARACTER SET %s COLLATE %s_unicode_ci;
CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';
GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';
FLUSH PRIVILEGES;
`,
		config.Name, config.Username,
		config.Name, config.Charset, config.Charset,
		config.Username, config.Password,
		config.Name, config.Username,
	)

	if err := os.WriteFile(logFile, []byte(sql), 0600); err != nil {
		return err
	}

	log.Printf("🔧 [SIMÜLASYON] MySQL database created: %s", config.Name)
	log.Printf("🔧 [SIMÜLASYON] MySQL user created: %s", config.Username)
	log.Printf("📝 SQL file: %s", logFile)
	return nil
}

func (m *Manager) realCreateDatabase(config DatabaseConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/", m.rootUser, m.rootPass, m.host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer db.Close()

	// Create database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s_unicode_ci",
		config.Name, config.Charset, config.Charset))
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Create user
	_, err = db.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'",
		config.Username, config.Password))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Grant privileges
	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'",
		config.Name, config.Username))
	if err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	// Flush privileges
	_, err = db.Exec("FLUSH PRIVILEGES")
	if err != nil {
		return fmt.Errorf("failed to flush privileges: %w", err)
	}

	log.Printf("✅ MySQL database created: %s", config.Name)
	log.Printf("✅ MySQL user created: %s", config.Username)
	return nil
}

// DeleteDatabase removes a MySQL database and user
func (m *Manager) DeleteDatabase(name, username string) error {
	if m.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] DROP DATABASE %s", name)
		log.Printf("🔧 [SIMÜLASYON] DROP USER '%s'@'localhost'", username)

		// Remove simulation file
		simPath := filepath.Join(m.basePath, "mysql", name+".sql")
		os.Remove(simPath)
		return nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/", m.rootUser, m.rootPass, m.host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer db.Close()

	// Drop database
	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))

	// Drop user
	db.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'localhost'", username))

	log.Printf("🗑️ MySQL database deleted: %s", name)
	return nil
}

// GeneratePassword generates a random password
func GeneratePassword(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// GetDatabaseSize returns the size of a database in bytes
func (m *Manager) GetDatabaseSize(name string) (int64, error) {
	if m.simulateMode {
		return 0, nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/information_schema", m.rootUser, m.rootPass, m.host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var size int64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(data_length + index_length), 0)
		FROM tables WHERE table_schema = ?
	`, name).Scan(&size)

	return size, err
}

// ListDatabases returns all databases for a user prefix
func (m *Manager) ListDatabases(prefix string) ([]string, error) {
	if m.simulateMode {
		// List from simulation directory
		simPath := filepath.Join(m.basePath, "mysql")
		files, _ := filepath.Glob(filepath.Join(simPath, prefix+"*.sql"))
		var dbs []string
		for _, f := range files {
			name := filepath.Base(f)
			name = name[:len(name)-4] // Remove .sql
			dbs = append(dbs, name)
		}
		return dbs, nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/", m.rootUser, m.rootPass, m.host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			databases = append(databases, name)
		}
	}

	return databases, nil
}
