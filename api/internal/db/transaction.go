package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// WithTransaction executes a function within a database transaction
// The transaction is automatically committed if the function returns nil
// or rolled back if the function returns an error or panics
func (db *DB) WithTransaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return fmt.Errorf("transaction error: %w", err)
		}
		return nil
	})
}
