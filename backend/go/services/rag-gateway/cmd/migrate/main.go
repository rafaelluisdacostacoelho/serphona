package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/config"
)

func setPGOptionsForLists(lists int) (string, error) {
	prev := os.Getenv("PGOPTIONS")
	if lists <= 0 {
		return prev, nil
	}
	val := fmt.Sprintf("-c PGVECTOR_LISTS=%d", lists)
	if err := os.Setenv("PGOPTIONS", val); err != nil {
		return prev, err
	}
	return prev, nil
}

func restorePGOptions(prev string) {
	if prev == "" {
		_ = os.Unsetenv("PGOPTIONS")
		return
	}
	_ = os.Setenv("PGOPTIONS", prev)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	prevPGOptions, err := setPGOptionsForLists(cfg.PGVectorLists)
	if err != nil {
		log.Fatalf("failed to set PGOPTIONS: %v", err)
	}
	defer restorePGOptions(prevPGOptions)

	sourceURL := "file://migrations"
	m, err := migrate.New(sourceURL, cfg.PGVectorURL)
	if err != nil {
		log.Fatalf("migration init failed: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration apply failed: %v", err)
	}

	log.Printf("migrations applied (PGVECTOR_TABLE=%s, DIM=%d, LISTS=%d)", cfg.PGVectorTable, cfg.PGVectorDim, cfg.PGVectorLists)
}
