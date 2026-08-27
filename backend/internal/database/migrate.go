package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres database driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // file:// source driver
)

// migrator builds a *migrate.Migrate instance pointed at the given
// migrations directory (a "file://" path) and the given database.
func migrator(databaseURL, migrationsPath string) (*migrate.Migrate, error) {
	m, err := migrate.New("file://"+migrationsPath, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: failed to init migrator: %w", err)
	}
	return m, nil
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
