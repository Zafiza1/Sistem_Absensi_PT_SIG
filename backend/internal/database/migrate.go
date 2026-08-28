package database

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres database driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // file:// source driver
)

// migrator builds a *migrate.Migrate instance pointed at the given
// migrations directory and the given database.
func migrator(databaseURL, migrationsPath string) (*migrate.Migrate, error) {
	m, err := migrate.New(fileSourceURL(migrationsPath), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: failed to init migrator: %w", err)
	}
	return m, nil
}

// fileSourceURL turns an OS-native absolute path into a "file://" URL the
// migrate library accepts on every platform. A naive "file://" + path
// breaks on Windows: filepath.Abs returns backslashes
// ("D:\Sistem_Absensi\..."), which the URL parser doesn't understand and
// whose drive-letter colon gets misread as a port separator —
// filepath.ToSlash fixes both by giving "D:/Sistem_Absensi/...", which
// golang-migrate's file source driver accepts directly (unlike a strict
// RFC 8089 parser, it does NOT want a third leading slash before a
// Windows drive letter). This had simply never been exercised outside a
// Linux container until Phase 5's native-Windows backend fallback.
func fileSourceURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

// MigrateUp applies all pending "up" migrations. It is a no-op (returns nil)
// when the schema is already at the latest version.
func MigrateUp(databaseURL, migrationsPath string) error {
	m, err := migrator(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("database: migrate up failed: %w", err)
	}
	return nil
}

// MigrateDown rolls back exactly one migration step. Intended for local
// development and CI rollback testing, not for routine production use.
func MigrateDown(databaseURL, migrationsPath string) error {
	m, err := migrator(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("database: migrate down failed: %w", err)
	}
	return nil
}
