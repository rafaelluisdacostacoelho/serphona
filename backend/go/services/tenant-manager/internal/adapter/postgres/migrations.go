// Package postgres provides PostgreSQL repository implementations.
package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes *.up.sql files in lexicographical order.
// It keeps track of applied files in schema_migrations table to make reruns idempotent.
func RunMigrations(databaseURL, migrationsPath string) error {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migration connection: %w", err)
	}
	defer pool.Close()

	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var files []fs.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, f := range files {
		name := f.Name()
		applied, err := isApplied(ctx, pool, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := filepath.Join(migrationsPath, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", name, err)
		}

		if err := markApplied(ctx, pool, name); err != nil {
			return fmt.Errorf("failed to mark migration %s: %w", name, err)
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations: %w", err)
	}
	return nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, filename string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, filename).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check migration %s: %w", filename, err)
	}
	return exists, nil
}

func markApplied(ctx context.Context, pool *pgxpool.Pool, filename string) error {
	_, err := pool.Exec(ctx, `INSERT INTO schema_migrations (filename, applied_at) VALUES ($1, $2)`, filename, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to record migration %s: %w", filename, err)
	}
	return nil
}
