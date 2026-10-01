package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens (or creates) the SQLite database and runs any pending migrations.
func Open() (*sql.DB, error) {
	path, err := dataPath()
	if err != nil {
		return nil, err
	}
	return openAt(path)
}

// OpenAt opens a database at a specific path. Used in tests.
func OpenAt(path string) (*sql.DB, error) {
	return openAt(path)
}

func openAt(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// SQLite performs best with a single writer connection.
	conn.SetMaxOpenConns(1)

	if err := runMigrations(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func dataPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}
	dir := filepath.Join(base, "ztime")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create data directory: %w", err)
	}
	return filepath.Join(dir, "ztime.db"), nil
}

func runMigrations(conn *sql.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
