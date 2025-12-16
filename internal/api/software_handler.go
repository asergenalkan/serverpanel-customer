package api

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// SoftwarePackage represents an installable software package
type SoftwarePackage struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Installed   bool   `json:"installed"`
	Active      bool   `json:"active"`
	Category    string `json:"category"`
}

// PHPExtension represents a PHP extension
type PHPExtension struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Installed   bool     `json:"installed"`
	PHPVersions []string `json:"php_versions"` // Which PHP versions have this extension
}

// ApacheModule represents an Apache module
type ApacheModule struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Available   bool   `json:"available"`
}

// SoftwareOverview represents the complete software status
type SoftwareOverview struct {
	PHPVersions        []SoftwarePackage `json:"php_versions"`
	PHPExtensions      []PHPExtension    `json:"php_extensions"`
	ApacheModules      []ApacheModule    `json:"apache_modules"`
	AdditionalSoftware []SoftwarePackage `json:"additional_software"`
}

// Common PHP extensions with descriptions
var phpExtensionList = []struct {
	Name        string
	DisplayName string
	Description string
}{
	{"mysqli", "MySQLi", "MySQL veritabanı bağlantısı"},
	{"pdo_mysql", "PDO MySQL", "PDO ile MySQL bağlantısı"},
	{"gd", "GD", "Resim işleme kütüphanesi"},
	{"curl", "cURL", "URL transfer kütüphanesi"},
	{"mbstring", "Multibyte String", "Çok baytlı karakter desteği"},
	{"xml", "XML", "XML işleme"},
	{"zip", "Zip", "ZIP arşiv desteği"},
	{"intl", "Intl", "Uluslararasılaştırma"},
	{"bcmath", "BCMath", "Keyfi hassasiyetli matematik"},
	{"soap", "SOAP", "SOAP protokolü desteği"},
	{"imagick", "ImageMagick", "Gelişmiş resim işleme"},
	{"redis", "Redis", "Redis önbellek desteği"},
	{"memcached", "Memcached", "Memcached önbellek desteği"},
	{"opcache", "OPcache", "PHP opcode önbelleği"},
	{"apcu", "APCu", "Kullanıcı önbelleği"},
	{"imap", "IMAP", "E-posta protokolü desteği"},
	{"ldap", "LDAP", "LDAP dizin desteği"},
	{"pgsql", "PostgreSQL", "PostgreSQL veritabanı desteği"},
	{"sqlite3", "SQLite3", "SQLite veritabanı desteği"},
	{"exif", "EXIF", "Resim meta verileri"},
	{"fileinfo", "Fileinfo", "Dosya tipi algılama"},
	{"json", "JSON", "JSON işleme"},
	{"tokenizer", "Tokenizer", "PHP tokenizer"},
	{"ctype", "Ctype", "Karakter tipi kontrolü"},
	{"dom", "DOM", "DOM XML işleme"},
	{"simplexml", "SimpleXML", "Basit XML işleme"},
	{"xsl", "XSL", "XSL dönüşümleri"},
	{"bz2", "Bzip2", "Bzip2 sıkıştırma"},
	{"calendar", "Calendar", "Takvim fonksiyonları"},
	{"gettext", "Gettext", "Çoklu dil desteği"},
	{"sockets", "Sockets", "Soket programlama"},
	{"ftp", "FTP", "FTP protokolü desteği"},
}

// Common Apache modules with descriptions
var apacheModuleList = []struct {
	Name        string
	DisplayName string
	Description string
}{
	{"rewrite", "mod_rewrite", "URL yeniden yazma"},
	{"ssl", "mod_ssl", "SSL/TLS desteği"},
	{"headers", "mod_headers", "HTTP başlık kontrolü"},
	{"expires", "mod_expires", "Önbellek süreleri"},
	{"deflate", "mod_deflate", "Gzip sıkıştırma"},
	{"proxy", "mod_proxy", "Proxy desteği"},
	{"proxy_http", "mod_proxy_http", "HTTP proxy"},
	{"proxy_fcgi", "mod_proxy_fcgi", "FastCGI proxy"},
	{"http2", "mod_http2", "HTTP/2 protokolü"},
	{"security2", "mod_security2", "Web uygulama güvenlik duvarı"},
	{"evasive", "mod_evasive", "DDoS koruması"},
	{"pagespeed", "mod_pagespeed", "Sayfa optimizasyonu"},
	{"cache", "mod_cache", "Önbellek modülü"},
	{"cache_disk", "mod_cache_disk", "Disk önbelleği"},
	{"status", "mod_status", "Sunucu durumu"},
	{"info", "mod_info", "Sunucu bilgisi"},
	{"userdir", "mod_userdir", "Kullanıcı dizinleri"},
	{"alias", "mod_alias", "URL takma adları"},
	{"dir", "mod_dir", "Dizin listeleme"},
	{"autoindex", "mod_autoindex", "Otomatik indeks"},
	{"mime", "mod_mime", "MIME tipleri"},
	{"setenvif", "mod_setenvif", "Ortam değişkenleri"},
	{"env", "mod_env", "Ortam modülü"},
	{"auth_basic", "mod_auth_basic", "Temel kimlik doğrulama"},
	{"authz_core", "mod_authz_core", "Yetkilendirme çekirdeği"},
	{"authz_host", "mod_authz_host", "Host bazlı yetkilendirme"},
	{"authz_user", "mod_authz_user", "Kullanıcı yetkilendirme"},
	{"access_compat", "mod_access_compat", "Erişim uyumluluğu"},
	{"filter", "mod_filter", "İçerik filtreleme"},
	{"reqtimeout", "mod_reqtimeout", "İstek zaman aşımı"},
}

// Additional software packages
var additionalSoftwareList = []struct {
	Name        string
	DisplayName string
	Description string
	CheckCmd    string
}{
	{"imagemagick", "ImageMagick", "Gelişmiş resim işleme aracı", "convert"},
	{"ffmpeg", "FFmpeg", "Video/ses dönüştürme aracı", "ffmpeg"},
	{"redis-server", "Redis Server", "Bellek içi veri deposu", "redis-server"},
	{"memcached", "Memcached", "Dağıtık önbellek sistemi", "memcached"},
	{"git", "Git", "Versiyon kontrol sistemi", "git"},
	{"composer", "Composer", "PHP paket yöneticisi", "composer"},
	{"nodejs", "Node.js", "JavaScript çalışma ortamı", "node"},
	{"npm", "NPM", "Node.js paket yöneticisi", "npm"},
	{"certbot", "Certbot", "Let's Encrypt SSL aracı", "certbot"},
	{"wp-cli", "WP-CLI", "WordPress komut satırı aracı", "wp"},
	{"fail2ban", "Fail2ban", "Brute-force koruması", "fail2ban-client"},
	{"postfix", "Postfix", "Mail transfer agent", "postfix"},
	{"dovecot", "Dovecot", "IMAP/POP3 sunucusu", "dovecot"},
	{"opendkim", "OpenDKIM", "DKIM imzalama", "opendkim"},
	{"clamav", "ClamAV", "Antivirüs tarayıcı", "clamscan"},
	{"spamassassin", "SpamAssassin", "Spam filtresi", "spamassassin"},
	{"libapache2-mod-security2", "ModSecurity", "Web Application Firewall (WAF)", ""},
}

// GetSoftwareOverview returns complete software status
func (h *Handler) GetSoftwareOverview(c *fiber.Ctx) error {
	overview := SoftwareOverview{
		PHPVersions:        h.getPHPVersions(),
		PHPExtensions:      h.getPHPExtensions(),
		ApacheModules:      h.getApacheModules(),
		AdditionalSoftware: h.getAdditionalSoftware(),
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    overview,
	})
}

// getPHPVersions returns installed PHP versions
func (h *Handler) getPHPVersions() []SoftwarePackage {
	versions := []SoftwarePackage{}
	phpVersions := []string{"7.4", "8.0", "8.1", "8.2", "8.3"}

	for _, v := range phpVersions {
		pkg := SoftwarePackage{
			Name:        fmt.Sprintf("php%s", v),
			DisplayName: fmt.Sprintf("PHP %s", v),
			Description: fmt.Sprintf("PHP %s sürümü", v),
			Category:    "php",
		}

		// Check if installed
		fpmPath := fmt.Sprintf("/etc/php/%s/fpm/php-fpm.conf", v)
		if _, err := os.Stat(fpmPath); err == nil {
			pkg.Installed = true
			pkg.Version = h.getPHPVersionString(v)
			pkg.Active = h.isPHPFPMActive(v)
		}

		versions = append(versions, pkg)
	}

	return versions
}

// getPHPVersionString gets the full version string
func (h *Handler) getPHPVersionString(version string) string {
	cmd := exec.Command(fmt.Sprintf("/usr/bin/php%s", version), "-v")
	output, err := cmd.Output()
	if err != nil {
		return version
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return version
}

// isPHPFPMActive checks if PHP-FPM service is active
func (h *Handler) isPHPFPMActive(version string) bool {
	cmd := exec.Command("systemctl", "is-active", fmt.Sprintf("php%s-fpm", version))
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output)) == "active"
}

// getPHPExtensions returns PHP extensions status
func (h *Handler) getPHPExtensions() []PHPExtension {
	extensions := []PHPExtension{}
	installedVersions := h.getInstalledPHPVersionsList()

	for _, ext := range phpExtensionList {
		phpExt := PHPExtension{
			Name:        ext.Name,
			DisplayName: ext.DisplayName,
			Description: ext.Description,
			PHPVersions: []string{},
		}

		// Check which PHP versions have this extension
		for _, v := range installedVersions {
			if h.isExtensionInstalled(v, ext.Name) {
				phpExt.PHPVersions = append(phpExt.PHPVersions, v)
				phpExt.Installed = true
			}
		}

		extensions = append(extensions, phpExt)
	}

	return extensions
}

// getInstalledPHPVersionsList returns list of installed PHP versions
func (h *Handler) getInstalledPHPVersionsList() []string {
	versions := []string{}
	phpVersions := []string{"7.4", "8.0", "8.1", "8.2", "8.3"}

	for _, v := range phpVersions {
		fpmPath := fmt.Sprintf("/etc/php/%s/fpm/php-fpm.conf", v)
		if _, err := os.Stat(fpmPath); err == nil {
			versions = append(versions, v)
		}
	}

	return versions
}

// isExtensionInstalled checks if a PHP extension is installed for a version
func (h *Handler) isExtensionInstalled(phpVersion, extension string) bool {
	// Check if package is installed
	cmd := exec.Command("dpkg", "-l", fmt.Sprintf("php%s-%s", phpVersion, extension))
	if err := cmd.Run(); err == nil {
		return true
	}

	// Also check via php -m
	cmd = exec.Command(fmt.Sprintf("/usr/bin/php%s", phpVersion), "-m")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	modules := strings.ToLower(string(output))
	return strings.Contains(modules, strings.ToLower(extension))
}

// getApacheModules returns Apache modules status
func (h *Handler) getApacheModules() []ApacheModule {
	modules := []ApacheModule{}

	// Get enabled modules
	enabledModules := h.getEnabledApacheModules()

	// Get available modules
	availableModules := h.getAvailableApacheModules()

	for _, mod := range apacheModuleList {
		apacheMod := ApacheModule{
			Name:        mod.Name,
			DisplayName: mod.DisplayName,
			Description: mod.Description,
			Enabled:     enabledModules[mod.Name],
			Available:   availableModules[mod.Name],
		}
		modules = append(modules, apacheMod)
	}

	return modules
}

// getEnabledApacheModules returns map of enabled modules
func (h *Handler) getEnabledApacheModules() map[string]bool {
	enabled := make(map[string]bool)

	cmd := exec.Command("apache2ctl", "-M")
	output, err := cmd.Output()
	if err != nil {
		return enabled
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "_module") {
			// Extract module name (e.g., "rewrite_module (shared)" -> "rewrite")
			parts := strings.Fields(line)
			if len(parts) > 0 {
				modName := strings.TrimSuffix(parts[0], "_module")
				enabled[modName] = true
			}
		}
	}

	return enabled
}

// getAvailableApacheModules returns map of available modules
func (h *Handler) getAvailableApacheModules() map[string]bool {
	available := make(map[string]bool)

	files, err := os.ReadDir("/etc/apache2/mods-available")
	if err != nil {
		return available
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".load") {
			modName := strings.TrimSuffix(file.Name(), ".load")
			available[modName] = true
		}
	}

	return available
}

// getAdditionalSoftware returns additional software status
func (h *Handler) getAdditionalSoftware() []SoftwarePackage {
	software := []SoftwarePackage{}

	for _, sw := range additionalSoftwareList {
		pkg := SoftwarePackage{
			Name:        sw.Name,
			DisplayName: sw.DisplayName,
			Description: sw.Description,
			Category:    "additional",
		}

		// Special handling for packages without command
		if sw.CheckCmd == "" {
			// Check via dpkg
			cmd := exec.Command("dpkg", "-l", sw.Name)
			if output, err := cmd.Output(); err == nil && strings.Contains(string(output), "ii") {
				pkg.Installed = true
				// Extract version from dpkg output
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "ii") {
						fields := strings.Fields(line)
						if len(fields) >= 3 {
							pkg.Version = fields[2]
						}
						break
					}
				}
				// Check if ModSecurity module is enabled
				if sw.Name == "libapache2-mod-security2" {
					checkCmd := exec.Command("a2query", "-m", "security2")
					if checkOutput, err := checkCmd.Output(); err == nil && strings.Contains(string(checkOutput), "enabled") {
						pkg.Active = true
					}
				}
			}
		} else {
			// Check if command exists
			cmd := exec.Command("which", sw.CheckCmd)
			if err := cmd.Run(); err == nil {
				pkg.Installed = true
				pkg.Version = h.getSoftwareVersion(sw.CheckCmd)
				pkg.Active = h.isSoftwareActive(sw.Name)
			}
		}

		software = append(software, pkg)
	}

	return software
}

// getSoftwareVersion gets version of installed software
func (h *Handler) getSoftwareVersion(command string) string {
	cmd := exec.Command(command, "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Extract version from first line
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		// Try to extract version number
		re := regexp.MustCompile(`\d+\.\d+(\.\d+)?`)
		if match := re.FindString(lines[0]); match != "" {
			return match
		}
		return strings.TrimSpace(lines[0])
	}
	return ""
}

// isSoftwareActive checks if software service is active
func (h *Handler) isSoftwareActive(name string) bool {
	// Map package names to service names
	serviceMap := map[string]string{
		"redis-server": "redis-server",
		"memcached":    "memcached",
		"postfix":      "postfix",
		"dovecot":      "dovecot",
		"opendkim":     "opendkim",
		"fail2ban":     "fail2ban",
	}

	serviceName, ok := serviceMap[name]
	if !ok {
		return false
	}

	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output)) == "active"
}

// InstallPHPVersion installs a PHP version
func (h *Handler) InstallPHPVersion(c *fiber.Ctx) error {
	var req struct {
		Version string `json:"version"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate version format
	validVersions := map[string]bool{"7.4": true, "8.0": true, "8.1": true, "8.2": true, "8.3": true}
	if !validVersions[req.Version] {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz PHP sürümü",
		})
	}

	// Install PHP version
	packages := fmt.Sprintf("php%s-fpm php%s-cli php%s-common php%s-mysql php%s-gd php%s-curl php%s-mbstring php%s-xml php%s-zip",
		req.Version, req.Version, req.Version, req.Version, req.Version, req.Version, req.Version, req.Version, req.Version)

	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y %s", packages))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("PHP kurulumu başarısız: %s", string(output)),
		})
	}

	// Enable and start PHP-FPM
	exec.Command("systemctl", "enable", fmt.Sprintf("php%s-fpm", req.Version)).Run()
	exec.Command("systemctl", "start", fmt.Sprintf("php%s-fpm", req.Version)).Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("PHP %s başarıyla kuruldu", req.Version),
	})
}

// UninstallPHPVersion removes a PHP version
func (h *Handler) UninstallPHPVersion(c *fiber.Ctx) error {
	var req struct {
		Version string `json:"version"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Don't allow removing the default/only PHP version
	installedVersions := h.getInstalledPHPVersionsList()
	if len(installedVersions) <= 1 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "En az bir PHP sürümü kurulu olmalıdır",
		})
	}

	// Stop and disable PHP-FPM
	exec.Command("systemctl", "stop", fmt.Sprintf("php%s-fpm", req.Version)).Run()
	exec.Command("systemctl", "disable", fmt.Sprintf("php%s-fpm", req.Version)).Run()

	// Remove PHP packages
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get remove -y php%s-*", req.Version))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("PHP kaldırma başarısız: %s", string(output)),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("PHP %s başarıyla kaldırıldı", req.Version),
	})
}

// InstallPHPExtension installs a PHP extension
func (h *Handler) InstallPHPExtension(c *fiber.Ctx) error {
	var req struct {
		PHPVersion string `json:"php_version"`
		Extension  string `json:"extension"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate extension name (security: prevent command injection)
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(req.Extension) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz eklenti adı",
		})
	}

	// Install extension
	packageName := fmt.Sprintf("php%s-%s", req.PHPVersion, req.Extension)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y %s", packageName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Eklenti kurulumu başarısız: %s", string(output)),
		})
	}

	// Restart PHP-FPM
	exec.Command("systemctl", "restart", fmt.Sprintf("php%s-fpm", req.PHPVersion)).Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s eklentisi PHP %s için kuruldu", req.Extension, req.PHPVersion),
	})
}

// UninstallPHPExtension removes a PHP extension
func (h *Handler) UninstallPHPExtension(c *fiber.Ctx) error {
	var req struct {
		PHPVersion string `json:"php_version"`
		Extension  string `json:"extension"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate extension name
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(req.Extension) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz eklenti adı",
		})
	}

	// Remove extension
	packageName := fmt.Sprintf("php%s-%s", req.PHPVersion, req.Extension)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get remove -y %s", packageName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Eklenti kaldırma başarısız: %s", string(output)),
		})
	}

	// Restart PHP-FPM
	exec.Command("systemctl", "restart", fmt.Sprintf("php%s-fpm", req.PHPVersion)).Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s eklentisi PHP %s için kaldırıldı", req.Extension, req.PHPVersion),
	})
}

// EnableApacheModule enables an Apache module
func (h *Handler) EnableApacheModule(c *fiber.Ctx) error {
	var req struct {
		Module string `json:"module"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate module name
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(req.Module) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz modül adı",
		})
	}

	// Enable module
	cmd := exec.Command("a2enmod", req.Module)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Modül etkinleştirme başarısız: %s", string(output)),
		})
	}

	// Reload Apache
	exec.Command("systemctl", "reload", "apache2").Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s modülü etkinleştirildi", req.Module),
	})
}

// DisableApacheModule disables an Apache module
func (h *Handler) DisableApacheModule(c *fiber.Ctx) error {
	var req struct {
		Module string `json:"module"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate module name
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(req.Module) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz modül adı",
		})
	}

	// Critical modules that shouldn't be disabled
	criticalModules := map[string]bool{
		"mpm_prefork": true, "mpm_worker": true, "mpm_event": true,
		"authz_core": true, "authz_host": true, "dir": true, "mime": true,
	}
	if criticalModules[req.Module] {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu modül devre dışı bırakılamaz (kritik modül)",
		})
	}

	// Disable module
	cmd := exec.Command("a2dismod", req.Module)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Modül devre dışı bırakma başarısız: %s", string(output)),
		})
	}

	// Reload Apache
	exec.Command("systemctl", "reload", "apache2").Run()

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s modülü devre dışı bırakıldı", req.Module),
	})
}

// InstallSoftware installs additional software
func (h *Handler) InstallSoftware(c *fiber.Ctx) error {
	var req struct {
		Package string `json:"package"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate package name against whitelist
	validPackages := make(map[string]bool)
	for _, sw := range additionalSoftwareList {
		validPackages[sw.Name] = true
	}
	if !validPackages[req.Package] {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz paket adı",
		})
	}

	// Special handling for certain packages
	packageToInstall := req.Package
	postInstallCmd := ""

	switch req.Package {
	case "clamav":
		// ClamAV needs daemon package too
		packageToInstall = "clamav clamav-daemon"
		postInstallCmd = "systemctl enable clamav-daemon && systemctl start clamav-daemon"
	case "spamassassin":
		postInstallCmd = "systemctl enable spamassassin && systemctl start spamassassin"
	case "fail2ban":
		postInstallCmd = "systemctl enable fail2ban && systemctl start fail2ban"
	}

	// Install package
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y %s", packageToInstall))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Paket kurulumu başarısız: %s", string(output)),
		})
	}

	// Run post-install commands if any
	if postInstallCmd != "" {
		exec.Command("bash", "-c", postInstallCmd).Run()
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s başarıyla kuruldu", req.Package),
	})
}

// UninstallSoftware removes additional software
func (h *Handler) UninstallSoftware(c *fiber.Ctx) error {
	var req struct {
		Package string `json:"package"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate package name against whitelist
	validPackages := make(map[string]bool)
	for _, sw := range additionalSoftwareList {
		validPackages[sw.Name] = true
	}
	if !validPackages[req.Package] {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz paket adı",
		})
	}

	// Critical packages that shouldn't be removed
	criticalPackages := map[string]bool{
		"postfix": true, "dovecot": true, "git": true,
	}
	if criticalPackages[req.Package] {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Bu paket kaldırılamaz (kritik paket)",
		})
	}

	// Special handling for certain packages
	packageToRemove := req.Package
	preRemoveCmd := ""

	switch req.Package {
	case "clamav":
		// ClamAV has daemon package too
		packageToRemove = "clamav clamav-daemon clamav-freshclam"
		preRemoveCmd = "systemctl stop clamav-daemon clamav-freshclam 2>/dev/null; systemctl disable clamav-daemon 2>/dev/null"
	case "spamassassin":
		preRemoveCmd = "systemctl stop spamassassin 2>/dev/null; systemctl disable spamassassin 2>/dev/null"
	case "fail2ban":
		preRemoveCmd = "systemctl stop fail2ban 2>/dev/null; systemctl disable fail2ban 2>/dev/null"
	case "imagemagick":
		// Also purge common package to avoid rc state
		packageToRemove = "imagemagick imagemagick-6-common"
	}

	// Run pre-remove commands if any
	if preRemoveCmd != "" {
		exec.Command("bash", "-c", preRemoveCmd).Run()
	}

	// Remove package with purge (removes config files too)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get purge -y %s && apt-get autoremove -y", packageToRemove))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Paket kaldırma başarısız: %s", string(output)),
		})
	}

	// Clean up data directories for certain packages
	switch req.Package {
	case "clamav":
		exec.Command("rm", "-rf", "/var/lib/clamav").Run()
		exec.Command("rm", "-rf", "/var/log/clamav").Run()
		exec.Command("rm", "-rf", "/var/run/clamav").Run()
		exec.Command("bash", "-c", "systemctl stop clamav-daemon.socket 2>/dev/null; systemctl disable clamav-daemon.socket 2>/dev/null; systemctl daemon-reload").Run()
		exec.Command("userdel", "clamav").Run()
		exec.Command("groupdel", "clamav").Run()
	case "imagemagick":
		exec.Command("rm", "-rf", "/etc/ImageMagick-6").Run()
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s başarıyla kaldırıldı", req.Package),
	})
}

// GetNodejsStatus returns Node.js/NVM/PM2 installation status
func (h *Handler) GetNodejsStatus(c *fiber.Ctx) error {
	status := map[string]interface{}{
		"nvm_installed":  h.isNVMInstalled(),
		"pm2_installed":  h.isPM2Installed(),
		"node_versions":  h.getInstalledNodeVersions(),
		"active_version": h.getActiveNodeVersion(),
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data:    status,
	})
}

// InstallNodejsSupport installs NVM and PM2
func (h *Handler) InstallNodejsSupport(c *fiber.Ctx) error {
	// Check if already installed
	if h.isNVMInstalled() && h.isPM2Installed() {
		return c.JSON(models.APIResponse{
			Success: true,
			Message: "Node.js desteği zaten kurulu",
		})
	}

	// Install NVM
	if !h.isNVMInstalled() {
		nvmInstallCmd := `
			export HOME=/root
			curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
			export NVM_DIR="$HOME/.nvm"
			[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
			nvm install --lts
			nvm use --lts
			nvm alias default lts/*
		`
		cmd := exec.Command("bash", "-c", nvmInstallCmd)
		cmd.Env = append(os.Environ(), "HOME=/root")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   fmt.Sprintf("NVM kurulumu başarısız: %s", string(output)),
			})
		}
	}

	// Install PM2 globally and configure startup
	pm2InstallCmd := `
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		npm install -g pm2
		# Configure PM2 to start on boot
		pm2 startup systemd -u root --hp /root --no-daemon 2>&1 | grep -E "sudo|env" | bash 2>/dev/null || true
		# Enable PM2 service
		systemctl enable pm2-root 2>/dev/null || true
		# Save empty process list initially
		pm2 save
	`
	cmd := exec.Command("bash", "-c", pm2InstallCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("PM2 kurulumu başarısız: %s", string(output)),
		})
	}

	// Enable Apache proxy modules for Node.js apps
	exec.Command("a2enmod", "proxy").Run()
	exec.Command("a2enmod", "proxy_http").Run()
	exec.Command("a2enmod", "proxy_wstunnel").Run()
	exec.Command("systemctl", "reload", "apache2").Run()

	// Update server setting
	h.db.Exec(`INSERT INTO server_settings (key, value) VALUES ('nodejs_enabled', 'true') ON CONFLICT(key) DO UPDATE SET value = 'true'`)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Node.js desteği başarıyla kuruldu (NVM + PM2)",
	})
}

// UninstallNodejsSupport removes NVM and PM2
func (h *Handler) UninstallNodejsSupport(c *fiber.Ctx) error {
	// Stop all PM2 processes
	stopCmd := `
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		pm2 kill 2>/dev/null || true
		pm2 unstartup systemd 2>/dev/null || true
	`
	exec.Command("bash", "-c", stopCmd).Run()

	// Remove NVM directory
	exec.Command("rm", "-rf", "/root/.nvm").Run()

	// Remove NVM lines from bashrc
	exec.Command("bash", "-c", `sed -i '/NVM_DIR/d' /root/.bashrc`).Run()

	// Update server setting
	h.db.Exec(`UPDATE server_settings SET value = 'false' WHERE key = 'nodejs_enabled'`)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Node.js desteği kaldırıldı",
	})
}

// InstallNodeVersion installs a specific Node.js version
func (h *Handler) InstallNodeVersion(c *fiber.Ctx) error {
	var req struct {
		Version string `json:"version"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	// Validate version format (e.g., "18", "20", "lts")
	if !regexp.MustCompile(`^(lts|\d+)$`).MatchString(req.Version) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz Node.js sürümü",
		})
	}

	installCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		nvm install %s
	`, req.Version)

	cmd := exec.Command("bash", "-c", installCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Node.js kurulumu başarısız: %s", string(output)),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node.js %s başarıyla kuruldu", req.Version),
	})
}

// SetActiveNodeVersion sets the active Node.js version
func (h *Handler) SetActiveNodeVersion(c *fiber.Ctx) error {
	var req struct {
		Version string `json:"version"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Geçersiz istek",
		})
	}

	setCmd := fmt.Sprintf(`
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		nvm use %s
		nvm alias default %s
	`, req.Version, req.Version)

	cmd := exec.Command("bash", "-c", setCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Node.js sürümü değiştirilemedi: %s", string(output)),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node.js %s aktif edildi", req.Version),
	})
}

// Helper functions for Node.js
func (h *Handler) isNVMInstalled() bool {
	_, err := os.Stat("/root/.nvm/nvm.sh")
	return err == nil
}

func (h *Handler) isPM2Installed() bool {
	checkCmd := `
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		which pm2
	`
	cmd := exec.Command("bash", "-c", checkCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	err := cmd.Run()
	return err == nil
}

func (h *Handler) getInstalledNodeVersions() []map[string]interface{} {
	versions := []map[string]interface{}{}

	if !h.isNVMInstalled() {
		return versions
	}

	listCmd := `
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		nvm ls --no-colors 2>/dev/null | grep -oP 'v\d+\.\d+\.\d+' | sort -V | uniq
	`
	cmd := exec.Command("bash", "-c", listCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.Output()
	if err != nil {
		return versions
	}

	activeVersion := h.getActiveNodeVersion()
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			versions = append(versions, map[string]interface{}{
				"version": line,
				"active":  line == activeVersion,
			})
		}
	}

	return versions
}

func (h *Handler) getActiveNodeVersion() string {
	if !h.isNVMInstalled() {
		return ""
	}

	versionCmd := `
		export HOME=/root
		export NVM_DIR="$HOME/.nvm"
		[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
		node --version 2>/dev/null
	`
	cmd := exec.Command("bash", "-c", versionCmd)
	cmd.Env = append(os.Environ(), "HOME=/root")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
