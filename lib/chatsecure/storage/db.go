package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db     *sql.DB
	dbPath string
	isRAM  bool
}

// InitDB initializes SQLite storage in RAM (/dev/shm) or in-memory fallback.
func InitDB(instanceName string, debug bool) (*Storage, error) {
	var dsn string
	var dbPath string
	var isRAM bool

	shmDir := "/dev/shm"
	useSHM := false
	if fi, err := os.Stat(shmDir); err == nil && fi.IsDir() {
		// Test write permissions
		testFile := filepath.Join(shmDir, fmt.Sprintf(".tcs_test_%s", instanceName))
		if f, err := os.Create(testFile); err == nil {
			_ = f.Close()
			_ = os.Remove(testFile)
			useSHM = true
		}
	}

	if useSHM {
		dbPath = filepath.Join(shmDir, fmt.Sprintf("tcs_%s.sqlite3", instanceName))
		// Clean any previous stale RAM db for this instance
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-shm")
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-journal")

		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)", dbPath)
		isRAM = true
		if debug {
			log.Printf("[storage] Using RAM storage in %s", dbPath)
		}
	} else {
		// In-memory shared cache fallback (e.g. on macOS or systems without /dev/shm)
		dbPath = ""
		dsn = fmt.Sprintf("file:tcs_%s?mode=memory&cache=shared&_pragma=foreign_keys(ON)", instanceName)
		isRAM = true
		if debug {
			log.Printf("[storage] /dev/shm unavailable; using SQLite shared in-memory DB: %s", dsn)
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize connection pooling for SQLite
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	s := &Storage{
		db:     db,
		dbPath: dbPath,
		isRAM:  isRAM,
	}

	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Storage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(24) NOT NULL,
		pubkey TEXT UNIQUE NOT NULL,
		fp VARCHAR(48) UNIQUE NOT NULL,
		token VARCHAR(48) UNIQUE NOT NULL,
		sid VARCHAR(32) UNIQUE,
		ts_join DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_update DATETIME DEFAULT CURRENT_TIMESTAMP,
		active BOOLEAN DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(24) NOT NULL,
		last_update DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sender_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		recipient_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		msg TEXT NOT NULL,
		ts DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS group_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sender_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		msg TEXT NOT NULL,
		ts DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS message_sequence (
		id INTEGER PRIMARY KEY AUTOINCREMENT
	);

	CREATE INDEX IF NOT EXISTS idx_users_token ON users(token);
	CREATE INDEX IF NOT EXISTS idx_users_fp ON users(fp);
	CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
	CREATE INDEX IF NOT EXISTS idx_user_messages_sync ON user_messages(id, sender_id, recipient_id);
	CREATE INDEX IF NOT EXISTS idx_group_messages_sync ON group_messages(id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Ensure default group exists (id = 1, name = 'default')
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM groups WHERE id = 1 OR name = 'default'").Scan(&count)
	if count == 0 {
		_, err := s.db.Exec("INSERT INTO groups (id, name, last_update) VALUES (1, 'default', CURRENT_TIMESTAMP)")
		if err != nil {
			// Try without id if autoincrement restriction
			_, _ = s.db.Exec("INSERT INTO groups (name, last_update) VALUES ('default', CURRENT_TIMESTAMP)")
		}
	}

	return nil
}

// Close and clean up any physical files in RAM
func (s *Storage) Close() error {
	if s.db != nil {
		_ = s.db.Close()
	}
	if s.dbPath != "" {
		_ = os.Remove(s.dbPath)
		_ = os.Remove(s.dbPath + "-shm")
		_ = os.Remove(s.dbPath + "-wal")
		_ = os.Remove(s.dbPath + "-journal")
	}
	return nil
}

// DB returns the underlying sql.DB
func (s *Storage) DB() *sql.DB {
	return s.db
}
