package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes database migrations from the specified path.
// It uses golang-migrate to apply migrations to the provided database connection.
//
// Parameters:
//   - db: An open database connection
//   - migrationsPath: Path to the directory containing migration files (e.g., "file://./migrations")
//
// Returns:
//   - error: nil if migrations succeed or if there are no changes to apply
//
// Example usage:
//
//	db, err := sql.Open("sqlite3", "./data/hermes.db")
//	if err != nil {
//	    return err
//	}
//	defer db.Close()
//
//	if err := RunMigrations(db, "file://./migrations"); err != nil {
//	    return fmt.Errorf("migration failed: %w", err)
//	}
func RunMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
