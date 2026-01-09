package webserver

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NginxDriver implements the Driver interface for Nginx
type NginxDriver struct {
	simulateMode bool
	basePath     string
}

// NewNginxDriver creates a new Nginx driver
func NewNginxDriver(simulateMode bool, basePath string) *NginxDriver {
	return &NginxDriver{
		simulateMode: simulateMode,
		basePath:     basePath,
	}
}

func (d *NginxDriver) Name() string {
	return "Nginx"
}

func (d *NginxDriver) SupportsHtaccess() bool {
	return false // Nginx does NOT support .htaccess
}

func (d *NginxDriver) GetConfigPath() string {
	if d.simulateMode {
		return filepath.Join(d.basePath, "nginx", "sites-available")
	}
	return "/etc/nginx/sites-available"
}

func (d *NginxDriver) CreateVhost(config VhostConfig) error {
	vhostConfig := d.generateConfig(config)

	configPath := d.GetConfigPath()
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configPath, config.Domain+".conf")
	if err := os.WriteFile(configFile, []byte(vhostConfig), 0644); err != nil {
		return fmt.Errorf("failed to write vhost config: %w", err)
	}

	log.Printf("📝 Nginx config created: %s", configFile)

	// Enable site
	if err := d.EnableSite(config.Domain); err != nil {
		return err
	}

	return d.Reload()
}

func (d *NginxDriver) generateConfig(config VhostConfig) string {
	serverNames := config.Domain
	if len(config.Aliases) > 0 {
		serverNames += " " + strings.Join(config.Aliases, " ")
	} else {
		serverNames += fmt.Sprintf(" www.%s", config.Domain)
	}

	phpVersion := config.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.2"
	}

	// PHP-FPM socket path
	phpFpmSocket := fmt.Sprintf("/run/php/php%s-fpm-%s.sock", phpVersion, config.Username)
	if d.simulateMode {
		phpFpmSocket = filepath.Join(d.basePath, "php-fpm", config.Username+".sock")
	}

	vhost := fmt.Sprintf(`# Virtual Host for %s
# User: %s
# Web Server: Nginx
# .htaccess: NOT SUPPORTED
server {
    listen 80;
    server_name %s;
    
    root %s;
    index index.php index.html index.htm;
    
    access_log %s/access.log;
    error_log %s/error.log;
    
    # Main location
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    
    # PHP handling
    location ~ \.php$ {
        fastcgi_pass unix:%s;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
    
    # Deny access to hidden files
    location ~ /\.ht {
        deny all;
    }
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
}
`,
		config.Domain,
		config.Username,
		serverNames,
		config.DocumentRoot,
		filepath.Join(config.HomeDir, "logs"),
		filepath.Join(config.HomeDir, "logs"),
		phpFpmSocket,
	)

	// Add SSL configuration if enabled
	if config.SSLEnabled && config.SSLCertPath != "" && config.SSLKeyPath != "" {
		vhost += fmt.Sprintf(`
server {
    listen 443 ssl http2;
    server_name %s;
    
    root %s;
    index index.php index.html index.htm;
    
    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;
    
    access_log %s/access.log;
    error_log %s/error.log;
    
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    
    location ~ \.php$ {
        fastcgi_pass unix:%s;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
    
    location ~ /\.ht {
        deny all;
    }
    
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
}
`,
			serverNames,
			config.DocumentRoot,
			config.SSLCertPath,
			config.SSLKeyPath,
			filepath.Join(config.HomeDir, "logs"),
			filepath.Join(config.HomeDir, "logs"),
			phpFpmSocket,
		)
	}

	return vhost
}

func (d *NginxDriver) DeleteVhost(domain string) error {
	if err := d.DisableSite(domain); err != nil {
		log.Printf("Warning: failed to disable site: %v", err)
	}

	configFile := filepath.Join(d.GetConfigPath(), domain+".conf")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	log.Printf("🗑️ Nginx config deleted: %s", configFile)
	return d.Reload()
}

func (d *NginxDriver) EnableSite(domain string) error {
	if d.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] ln -s sites-available/%s.conf sites-enabled/", domain)
		enabledPath := filepath.Join(d.basePath, "nginx", "sites-enabled")
		os.MkdirAll(enabledPath, 0755)
		src := filepath.Join(d.GetConfigPath(), domain+".conf")
		dst := filepath.Join(enabledPath, domain+".conf")
		os.Symlink(src, dst)
		return nil
	}

	enabledPath := "/etc/nginx/sites-enabled"
	src := filepath.Join(d.GetConfigPath(), domain+".conf")
	dst := filepath.Join(enabledPath, domain+".conf")

	// Remove existing symlink if exists
	os.Remove(dst)

	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("failed to enable site: %w", err)
	}
	return nil
}

func (d *NginxDriver) DisableSite(domain string) error {
	if d.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] rm sites-enabled/%s.conf", domain)
		enabledPath := filepath.Join(d.basePath, "nginx", "sites-enabled", domain+".conf")
		os.Remove(enabledPath)
		return nil
	}

	enabledPath := filepath.Join("/etc/nginx/sites-enabled", domain+".conf")
	if err := os.Remove(enabledPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to disable site: %w", err)
	}
	return nil
}

func (d *NginxDriver) Reload() error {
	if d.simulateMode {
		log.Printf("🔧 [SIMÜLASYON] nginx -t && systemctl reload nginx")
		return nil
	}

	// Test config first
	if err := d.TestConfig(); err != nil {
		return err
	}

	cmd := exec.Command("systemctl", "reload", "nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload nginx: %s - %w", string(output), err)
	}

	log.Printf("✅ Nginx reloaded successfully")
	return nil
}

func (d *NginxDriver) TestConfig() error {
	if d.simulateMode {
		return nil
	}

	cmd := exec.Command("nginx", "-t")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nginx config test failed: %s - %w", string(output), err)
	}
	return nil
}
