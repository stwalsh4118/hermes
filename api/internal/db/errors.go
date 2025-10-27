package db

import (
	"errors"

	"gorm.io/gorm"
)

// Custom database errors
var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicate    = errors.New("duplicate record")
	ErrForeignKey   = errors.New("foreign key constraint violation")
	ErrInvalidInput = errors.New("invalid input")
)

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}

// IsDuplicate checks if error is a duplicate error
func IsDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicate)
}

// IsForeignKey checks if error is a foreign key constraint violation
func IsForeignKey(err error) bool {
	return errors.Is(err, ErrForeignKey)
}

// MapGormError maps GORM errors to custom domain errors
func MapGormError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	// Check for SQLite constraint errors
	errMsg := err.Error()
	if containsAny(errMsg, []string{"UNIQUE constraint", "unique constraint"}) {
		return ErrDuplicate
	}
	if containsAny(errMsg, []string{"FOREIGN KEY constraint", "foreign key constraint"}) {
		return ErrForeignKey
	}

	return err
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
