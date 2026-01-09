package config

import (
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Port             string
	DatabasePath     string
	JWTSecret        string
	Environment      string // "development" or "production"
	DataDir          string
	HomeBaseDir      string // /home on Linux, simulated on Mac
	SimulateBasePath string // Base path for simulation files
	WebServer        string // "apache" or "nginx" - default: apache
	PHPVersion       string // e.g., "8.2"
	ServerIP         string // Server IP address
	IsLinux          bool
	SimulateMode     bool // true if running in simulation mode
}

var cfg *Config

func Load() *Config {
	if cfg != nil {
		return cfg
	}

	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".serverpanel")

	// Create data directory if not exists
	os.MkdirAll(dataDir, 0755)

	isLinux := runtime.GOOS == "linux"
	env := getEnv("ENVIRONMENT", "development")
	simulateMode := !(env == "production" && isLinux)

	// Determine paths based on mode
	var homeBaseDir, simulateBasePath string
	if simulateMode {
		// Development mode - use local simulation
		simulateBasePath = filepath.Join(dataDir, "simulate")
		homeBaseDir = filepath.Join(simulateBasePath, "home")
		os.MkdirAll(homeBaseDir, 0755)
		os.MkdirAll(filepath.Join(simulateBasePath, "apache"), 0755)
		os.MkdirAll(filepath.Join(simulateBasePath, "nginx"), 0755)
		os.MkdirAll(filepath.Join(simulateBasePath, "php-fpm"), 0755)
	} else {
		// Production mode - use real paths
		homeBaseDir = "/home"
		simulateBasePath = ""
	}

	// Web server: default to Apache (supports .htaccess)
	webServer := getEnv("WEB_SERVER", "apache")

	cfg = &Config{
		Port:             getEnv("PORT", "8443"),
		DatabasePath:     filepath.Join(dataDir, "panel.db"),
		JWTSecret:        getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		Environment:      env,
		DataDir:          dataDir,
		HomeBaseDir:      homeBaseDir,
		SimulateBasePath: simulateBasePath,
		WebServer:        webServer,
		PHPVersion:       getEnv("PHP_VERSION", "8.2"),
		ServerIP:         getEnv("SERVER_IP", "127.0.0.1"),
		IsLinux:          isLinux,
		SimulateMode:     simulateMode,
	}

	return cfg
}

func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

func IsDevelopment() bool {
	return Get().Environment == "development"
}

func IsProduction() bool {
	return Get().Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
