package webserver

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ApacheDriver implements the Driver interface for Apache
type ApacheDriver struct {
	simulateMode bool
	basePath     string // For simulation: ~/.serverpanel/simulate
}

// NewApacheDriver creates a new Apache driver
func NewApacheDriver(simulateMode bool, basePath string) *ApacheDriver {
	return &ApacheDriver{
		simulateMode: simulateMode,
		basePath:     basePath,
	}
}

func (d *ApacheDriver) Name() string {
	return "Apache"
}

func (d *ApacheDriver) SupportsHtaccess() bool {
	return true
}

func (d *ApacheDriver) GetConfigPath() string {
	if d.simulateMode {
		return filepath.Join(d.basePath, "apache", "sites-available")
	}
	return "/etc/apache2/sites-available"
}

func (d *ApacheDriver) CreateVhost(config VhostConfig) error {
	vhostConfig := d.generateConfig(config)

	configPath := d.GetConfigPath()
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configPath, config.Domain+".conf")
	if err := os.WriteFile(configFile, []byte(vhostConfig), 0644); err != nil {
		return fmt.Errorf("failed to write vhost config: %w", err)
	}

	log.Printf("📝 Apache config created: %s", configFile)

	// Enable site
	if err := d.EnableSite(config.Domain); err != nil {
		return err
	}

	return d.Reload()
}

func (d *ApacheDriver) generateConfig(config VhostConfig) string {
	serverAliases := ""
	if len(config.Aliases) > 0 {
		serverAliases = "ServerAlias " + strings.Join(config.Aliases, " ")
	} else {
		serverAliases = fmt.Sprintf("ServerAlias www.%s", config.Domain)
	}

	phpVersion := config.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.1" // Default to 8.1 for Ubuntu 22.04
	}

	// PHP-FPM socket path
	phpFpmSocket := fmt.Sprintf("/run/php/php%s-fpm-%s.sock", phpVersion, config.Username)
	if d.simulateMode {
		phpFpmSocket = filepath.Join(d.basePath, "php-fpm", config.Username+".sock")
	}

	vhost := fmt.Sprintf(`# Virtual Host for %s
# User: %s
# Web Server: Apache
# .htaccess: ENABLED
<VirtualHost *:80>
    ServerName %s
    %s
    
    DocumentRoot %s
    
    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
    
    # PHP-FPM Configuration
    <FilesMatch \.php$>
        SetHandler "proxy:unix:%s|fcgi://localhost"
    </FilesMatch>
    
    # Logging
    ErrorLog %s/error.log
    CustomLog %s/access.log combined
    
    # Security Headers
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-XSS-Protection "1; mode=block"
</VirtualHost>
`,
		config.Domain,
		config.Username,
		config.Domain,
		serverAliases,
		config.DocumentRoot,
		config.DocumentRoot,
		phpFpmSocket,
		filepath.Join(config.HomeDir, "logs"),
		filepath.Join(config.HomeDir, "logs"),
	)

	// Add SSL configuration if enabled
	if config.SSLEnabled && config.SSLCertPath != "" && config.SSLKeyPath != "" {
		vhost += fmt.Sprintf(`
<VirtualHost *:443>
    ServerName %s
    %s
    
    DocumentRoot %s
    
    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
    
    # PHP-FPM Configuration
    <FilesMatch \.php$>
        SetHandler "proxy:unix:%s|fcgi://localhost"
    </FilesMatch>
    
    # SSL Configuration
    SSLEngine on
    SSLCertificateFile %s
    SSLCertificateKeyFile %s
    
    # Logging
    ErrorLog %s/error.log
    CustomLog %s/access.log combined
    
    # Security Headers
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-XSS-Protection "1; mode=block"
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
</VirtualHost>
`,
			config.Domain,
			serverAliases,
			config.DocumentRoot,
			config.DocumentRoot,
			phpFpmSocket,
			config.SSLCertPath,
			config.SSLKeyPath,
			filepath.Join(config.HomeDir, "logs"),
			filepath.Join(config.HomeDir, "logs"),
		)
	}

	return vhost
}

func (d *ApacheDriver) DeleteVhost(domain string) error {
	if err := d.DisableSite(domain); err != nil {
		log.Printf("Warning: failed to disable site: %v", err)
	}

	configFile := filepath.Join(d.GetConfigPath(), domain+".conf")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	log.Printf("🗑️ Apache config deleted: %s", configFile)
	return d.Reload()
}

func (d *ApacheDriver) EnableSite(domain string) error {
	if d.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] a2ensite %s.conf", domain)
		// Create symlink in sites-enabled for simulation
		enabledPath := filepath.Join(d.basePath, "apache", "sites-enabled")
		os.MkdirAll(enabledPath, 0755)
		src := filepath.Join(d.GetConfigPath(), domain+".conf")
		dst := filepath.Join(enabledPath, domain+".conf")
		os.Symlink(src, dst)
		return nil
	}

	cmd := exec.Command("a2ensite", domain+".conf")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable site: %s - %w", string(output), err)
	}
	return nil
}

func (d *ApacheDriver) DisableSite(domain string) error {
	if d.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] a2dissite %s.conf", domain)
		enabledPath := filepath.Join(d.basePath, "apache", "sites-enabled", domain+".conf")
		os.Remove(enabledPath)
		return nil
	}

	cmd := exec.Command("a2dissite", domain+".conf")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable site: %s - %w", string(output), err)
	}
	return nil
}

func (d *ApacheDriver) Reload() error {
	if d.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] apachectl configtest && systemctl reload apache2")
		return nil
	}

	// Test config first
	if err := d.TestConfig(); err != nil {
		return err
	}

	cmd := exec.Command("systemctl", "reload", "apache2")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload apache: %s - %w", string(output), err)
	}

	log.Printf("✅ Apache reloaded successfully")
	return nil
}

func (d *ApacheDriver) TestConfig() error {
	if d.simulateMode {
		return nil
	}

	cmd := exec.Command("apachectl", "configtest")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apache config test failed: %s - %w", string(output), err)
	}
	return nil
}

// CreateWebmailVhost creates a webmail subdomain vhost that proxies to Roundcube
func (d *ApacheDriver) CreateWebmailVhost(domain string) error {
	webmailDomain := "webmail." + domain

	vhostConfig := fmt.Sprintf(`# Webmail Virtual Host for %s
# Auto-generated by ServerPanel
<VirtualHost *:80>
    ServerName %s
    
    DocumentRoot /usr/share/roundcube
    
    <Directory /usr/share/roundcube>
        Options +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
    
    <Directory /usr/share/roundcube/config>
        Require all denied
    </Directory>
    
    # Logging
    ErrorLog /var/log/apache2/%s-error.log
    CustomLog /var/log/apache2/%s-access.log combined
</VirtualHost>
`, domain, webmailDomain, webmailDomain, webmailDomain)

	configPath := d.GetConfigPath()
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configPath, webmailDomain+".conf")
	if err := os.WriteFile(configFile, []byte(vhostConfig), 0644); err != nil {
		return fmt.Errorf("failed to write webmail vhost config: %w", err)
	}

	log.Printf("📝 Webmail vhost created: %s", configFile)

	// Enable site
	if err := d.EnableSite(webmailDomain); err != nil {
		return err
	}

	return d.Reload()
}

// DeleteWebmailVhost removes the webmail subdomain vhost
func (d *ApacheDriver) DeleteWebmailVhost(domain string) error {
	webmailDomain := "webmail." + domain
	return d.DeleteVhost(webmailDomain)
}
