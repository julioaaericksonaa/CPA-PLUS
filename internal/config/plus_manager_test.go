package config

import "testing"

func TestParseConfigBytesPlusManagerDefaults(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`remote-management:
  secret-key: test-key
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if !cfg.PlusManager.Enabled {
		t.Fatalf("PlusManager.Enabled = false, want true")
	}
	if cfg.PlusManager.DataDir != "./data" {
		t.Fatalf("PlusManager.DataDir = %q, want ./data", cfg.PlusManager.DataDir)
	}
	if cfg.PlusManager.DBPath != "./data/usage.sqlite" {
		t.Fatalf("PlusManager.DBPath = %q, want ./data/usage.sqlite", cfg.PlusManager.DBPath)
	}
	if !cfg.PlusManager.CollectorEnabled {
		t.Fatalf("PlusManager.CollectorEnabled = false, want true")
	}
	if cfg.PlusManager.CollectorMode != "auto" {
		t.Fatalf("PlusManager.CollectorMode = %q, want auto", cfg.PlusManager.CollectorMode)
	}
	if cfg.PlusManager.PollIntervalMs != 1000 {
		t.Fatalf("PlusManager.PollIntervalMs = %d, want 1000", cfg.PlusManager.PollIntervalMs)
	}
}

func TestParseConfigBytesPlusManagerOverrides(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`plus-manager:
  enabled: false
  data-dir: /var/lib/cpa-plus
  db-path: /var/lib/cpa-plus/custom.sqlite
  collector-enabled: false
  collector-mode: http
  poll-interval-ms: 2500
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if cfg.PlusManager.Enabled {
		t.Fatalf("PlusManager.Enabled = true, want false")
	}
	if cfg.PlusManager.DataDir != "/var/lib/cpa-plus" {
		t.Fatalf("PlusManager.DataDir = %q", cfg.PlusManager.DataDir)
	}
	if cfg.PlusManager.DBPath != "/var/lib/cpa-plus/custom.sqlite" {
		t.Fatalf("PlusManager.DBPath = %q", cfg.PlusManager.DBPath)
	}
	if cfg.PlusManager.CollectorEnabled {
		t.Fatalf("PlusManager.CollectorEnabled = true, want false")
	}
	if cfg.PlusManager.CollectorMode != "http" {
		t.Fatalf("PlusManager.CollectorMode = %q, want http", cfg.PlusManager.CollectorMode)
	}
	if cfg.PlusManager.PollIntervalMs != 2500 {
		t.Fatalf("PlusManager.PollIntervalMs = %d, want 2500", cfg.PlusManager.PollIntervalMs)
	}
}

func TestParseConfigBytesPlusManagerDBPathFallsBackToDataDir(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`plus-manager:
  data-dir: /tmp/cpa-plus
  db-path: ""
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if cfg.PlusManager.DBPath != "/tmp/cpa-plus/usage.sqlite" {
		t.Fatalf("PlusManager.DBPath = %q, want /tmp/cpa-plus/usage.sqlite", cfg.PlusManager.DBPath)
	}
}
