package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	*sql.DB
}

func Initialize(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	wrapper := &DB{db}

	// Run migrations
	if err := wrapper.migrate(); err != nil {
		return nil, err
	}

	log.Println("✅ Database initialized successfully")
	return wrapper, nil
}

func (db *DB) migrate() error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			parent_id INTEGER,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES users(id) ON DELETE SET NULL
		)`,

		// Packages table
		`CREATE TABLE IF NOT EXISTS packages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			disk_quota INTEGER DEFAULT 1024,
			bandwidth_quota INTEGER DEFAULT 10240,
			max_domains INTEGER DEFAULT 1,
			max_databases INTEGER DEFAULT 1,
			max_emails INTEGER DEFAULT 5,
			max_ftp INTEGER DEFAULT 1,
			max_php_memory TEXT DEFAULT '256M',
			max_php_upload TEXT DEFAULT '64M',
			max_php_execution_time INTEGER DEFAULT 300,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// User packages (assignment)
		`CREATE TABLE IF NOT EXISTS user_packages (
			user_id INTEGER NOT NULL,
			package_id INTEGER NOT NULL,
			PRIMARY KEY (user_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE CASCADE
		)`,

		// Domains table
		// domain_type: 'primary' (ana domain), 'addon' (ek domain), 'alias' (parked domain)
		`CREATE TABLE IF NOT EXISTS domains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT UNIQUE NOT NULL,
			domain_type TEXT DEFAULT 'primary',
			parent_domain_id INTEGER,
			document_root TEXT,
			php_version TEXT DEFAULT '8.1',
			ssl_enabled INTEGER DEFAULT 0,
			ssl_expiry DATETIME,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// Subdomains table
		`CREATE TABLE IF NOT EXISTS subdomains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			domain_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			full_name TEXT UNIQUE NOT NULL,
			document_root TEXT,
			redirect_url TEXT,
			redirect_type TEXT,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// PHP Settings table (per domain)
		`CREATE TABLE IF NOT EXISTS php_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id INTEGER NOT NULL UNIQUE,
			memory_limit TEXT DEFAULT '256M',
			max_execution_time INTEGER DEFAULT 300,
			max_input_time INTEGER DEFAULT 300,
			post_max_size TEXT DEFAULT '64M',
			upload_max_filesize TEXT DEFAULT '64M',
			max_file_uploads INTEGER DEFAULT 20,
			display_errors INTEGER DEFAULT 0,
			error_reporting TEXT DEFAULT 'E_ALL & ~E_DEPRECATED & ~E_STRICT',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// Databases table
		`CREATE TABLE IF NOT EXISTS databases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT UNIQUE NOT NULL,
			type TEXT DEFAULT 'mysql',
			size INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Database users table
		`CREATE TABLE IF NOT EXISTS database_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			database_id INTEGER NOT NULL,
			db_username TEXT UNIQUE NOT NULL,
			password TEXT,
			host TEXT DEFAULT 'localhost',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
		)`,

		// Email accounts table
		`CREATE TABLE IF NOT EXISTS email_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			domain_id INTEGER NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			quota_mb INTEGER DEFAULT 1024,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// Email forwarders table
		`CREATE TABLE IF NOT EXISTS email_forwarders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			domain_id INTEGER NOT NULL,
			source TEXT NOT NULL,
			destination TEXT NOT NULL,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// Email autoresponders table
		`CREATE TABLE IF NOT EXISTS email_autoresponders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			domain_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			start_date TEXT,
			end_date TEXT,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// FTP accounts table
		`CREATE TABLE IF NOT EXISTS ftp_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			home_directory TEXT NOT NULL,
			quota_mb INTEGER DEFAULT 0,
			upload_bandwidth INTEGER DEFAULT 0,
			download_bandwidth INTEGER DEFAULT 0,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// FTP server settings (admin config)
		`CREATE TABLE IF NOT EXISTS ftp_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			setting_key TEXT UNIQUE NOT NULL,
			setting_value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Activity logs
		`CREATE TABLE IF NOT EXISTS activity_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			action TEXT NOT NULL,
			details TEXT,
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		)`,

		// User-Package relationship
		`CREATE TABLE IF NOT EXISTS user_packages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			package_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (package_id) REFERENCES packages(id)
		)`,

		// DNS Records table
		`CREATE TABLE IF NOT EXISTS dns_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			ttl INTEGER DEFAULT 3600,
			priority INTEGER DEFAULT 0,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// Email settings (rate limits, DKIM, etc.) - Domain bazlı DKIM ayarları
		`CREATE TABLE IF NOT EXISTS email_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id INTEGER NOT NULL UNIQUE,
			dkim_enabled INTEGER DEFAULT 0,
			dkim_selector TEXT DEFAULT 'default',
			dkim_private_key TEXT,
			dkim_public_key TEXT,
			spf_record TEXT,
			dmarc_record TEXT,
			catch_all_email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,

		// Email send log (for rate limiting) - Her gönderilen mail kaydedilir
		`CREATE TABLE IF NOT EXISTS email_send_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			sender TEXT NOT NULL,
			recipient TEXT NOT NULL,
			subject TEXT,
			message_id TEXT,
			size_bytes INTEGER DEFAULT 0,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'sent',
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Mail queue - Rate limit aşıldığında mailler buraya eklenir
		`CREATE TABLE IF NOT EXISTS mail_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			sender TEXT NOT NULL,
			recipient TEXT NOT NULL,
			subject TEXT,
			body TEXT,
			headers TEXT,
			priority INTEGER DEFAULT 5,
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			scheduled_at DATETIME,
			status TEXT DEFAULT 'pending',
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Cron Jobs table
		`CREATE TABLE IF NOT EXISTS cron_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			command TEXT NOT NULL,
			schedule TEXT NOT NULL,
			minute TEXT DEFAULT '*',
			hour TEXT DEFAULT '*',
			day TEXT DEFAULT '*',
			month TEXT DEFAULT '*',
			weekday TEXT DEFAULT '*',
			active INTEGER DEFAULT 1,
			last_run DATETIME,
			next_run DATETIME,
			last_status TEXT,
			last_output TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_cron_jobs_user_id ON cron_jobs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dns_records_domain_id ON dns_records(domain_id)`,
		`CREATE INDEX IF NOT EXISTS idx_domains_user_id ON domains(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_databases_user_id ON databases(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_database_users_user_id ON database_users(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_database_users_database_id ON database_users(database_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_accounts_user_id ON email_accounts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_user_id ON activity_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_send_log_user_id ON email_send_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_send_log_sent_at ON email_send_log(sent_at)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_queue_user_id ON mail_queue(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_queue_status ON mail_queue(status)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_queue_scheduled_at ON mail_queue(scheduled_at)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	// Add password column to database_users if not exists
	db.Exec(`ALTER TABLE database_users ADD COLUMN password TEXT`)

	// Add php_version column to domains if not exists
	db.Exec(`ALTER TABLE domains ADD COLUMN php_version TEXT DEFAULT '8.1'`)

	// Add PHP limit columns to packages if not exists
	db.Exec(`ALTER TABLE packages ADD COLUMN max_php_memory TEXT DEFAULT '256M'`)
	db.Exec(`ALTER TABLE packages ADD COLUMN max_php_upload TEXT DEFAULT '64M'`)
	db.Exec(`ALTER TABLE packages ADD COLUMN max_php_execution_time INTEGER DEFAULT 300`)

	// Add mail rate limit columns to packages
	db.Exec(`ALTER TABLE packages ADD COLUMN max_emails_per_hour INTEGER DEFAULT 100`)
	db.Exec(`ALTER TABLE packages ADD COLUMN max_emails_per_day INTEGER DEFAULT 500`)

	// Create server_settings table for admin configuration
	db.Exec(`CREATE TABLE IF NOT EXISTS server_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Insert default server settings
	db.Exec(`INSERT OR IGNORE INTO server_settings (key, value) VALUES 
		('multiphp_enabled', 'true'),
		('default_php_version', '8.1'),
		('allowed_php_versions', '7.4,8.0,8.1,8.2,8.3'),
		('domain_based_php', 'true')
	`)

	// Create default admin user if not exists
	if err := db.createDefaultAdmin(); err != nil {
		log.Printf("Warning: Could not create default admin: %v", err)
	}

	// Create default package if not exists
	if err := db.createDefaultPackage(); err != nil {
		log.Printf("Warning: Could not create default package: %v", err)
	}

	return nil
}

func (db *DB) createDefaultAdmin() error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Generate bcrypt hash at runtime
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
		if err != nil {
			return err
		}
		_, err = db.Exec(`
			INSERT INTO users (username, email, password, role, active)
			VALUES ('admin', 'admin@localhost', ?, 'admin', 1)
		`, string(hashedBytes))
		if err != nil {
			return err
		}
		log.Println("✅ Default admin user created (username: admin, password: admin123)")
	}
	return nil
}

func (db *DB) createDefaultPackage() error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM packages").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO packages (name, disk_quota, bandwidth_quota, max_domains, max_databases, max_emails, max_ftp)
			VALUES 
				('Starter', 1024, 10240, 1, 1, 5, 1),
				('Professional', 5120, 51200, 5, 5, 25, 5),
				('Business', 20480, 204800, 20, 20, 100, 20)
		`)
		if err != nil {
			return err
		}
		log.Println("✅ Default packages created")
	}
	return nil
}
