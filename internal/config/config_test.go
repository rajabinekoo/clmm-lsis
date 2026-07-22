package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/config"
)

func TestLoadExampleConfiguration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(
		"..",
		"..",
		"configs",
		"study.example.json",
	)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if cfg.SchemaVersion != 1 {
		t.Fatalf(
			"SchemaVersion = %d, want 1",
			cfg.SchemaVersion,
		)
	}

	if len(cfg.Pools) != 5 {
		t.Fatalf(
			"len(Pools) = %d, want 5",
			len(cfg.Pools),
		)
	}

	if cfg.StructuralStudy.PrimaryPositionLimit != 20 {
		t.Fatalf(
			"PrimaryPositionLimit = %d, want 20",
			cfg.StructuralStudy.PrimaryPositionLimit,
		)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "study.json")

	payload := `{
		"schema_version": 1,
		"chain_id": 1,
		"unexpected_field": true
	}`

	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("config.Load() expected error")
	}

	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf(
			"config.Load() error = %q, want unknown field error",
			err,
		)
	}
}
