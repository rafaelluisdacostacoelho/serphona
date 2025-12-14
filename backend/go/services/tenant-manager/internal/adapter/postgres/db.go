// Package postgres provides PostgreSQL database functionality.
package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgxpool.Pool for easy access.
type DB struct {
	Pool *pgxpool.Pool
}

// Close closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
