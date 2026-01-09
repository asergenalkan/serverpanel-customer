package webserver

// Driver interface for web server operations
type Driver interface {
	// Name returns the web server name
	Name() string

	// CreateVhost creates a virtual host configuration for a domain
	CreateVhost(config VhostConfig) error

	// DeleteVhost removes the virtual host configuration
	DeleteVhost(domain string) error

	// EnableSite enables a site (for Apache a2ensite style)
	EnableSite(domain string) error

	// DisableSite disables a site
	DisableSite(domain string) error

	// Reload reloads the web server configuration
	Reload() error

	// TestConfig tests if the configuration is valid
	TestConfig() error

	// GetConfigPath returns the path where vhost configs are stored
	GetConfigPath() string

	// SupportsHtaccess returns whether this driver supports .htaccess
	SupportsHtaccess() bool
}

// VhostConfig contains all configuration for a virtual host
type VhostConfig struct {
	Domain       string
	Aliases      []string // www.domain.com, etc.
	Username     string
	DocumentRoot string
	HomeDir      string
	PHPVersion   string // e.g., "8.2"
	SSLEnabled   bool
	SSLCertPath  string
	SSLKeyPath   string
}

// DriverType represents the type of web server
type DriverType string

const (
	DriverApache DriverType = "apache"
	DriverNginx  DriverType = "nginx"
)

// NewDriver creates a new web server driver based on type
func NewDriver(driverType DriverType, simulateMode bool, basePath string) Driver {
	switch driverType {
	case DriverNginx:
		return NewNginxDriver(simulateMode, basePath)
	case DriverApache:
		fallthrough
	default:
		return NewApacheDriver(simulateMode, basePath)
	}
}
