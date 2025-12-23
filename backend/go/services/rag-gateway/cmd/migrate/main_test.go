package main

import (
	"os"
	"testing"
)

func TestSetPGOptionsForLists(t *testing.T) {
	prev := ""
	var err error
	prev, err = setPGOptionsForLists(250)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := os.Getenv("PGOPTIONS"); got != "-c PGVECTOR_LISTS=250" {
		t.Fatalf("expected PGOPTIONS set, got %s", got)
	}

	restorePGOptions(prev)
	if got := os.Getenv("PGOPTIONS"); got != prev {
		t.Fatalf("expected PGOPTIONS restored to %s, got %s", prev, got)
	}
}
