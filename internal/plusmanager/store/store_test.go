package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModelPricesPersist(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	prices := []ModelPrice{{Model: "gpt-test", InputPerMTok: 1.25, OutputPerMTok: 5.5}}
	if err := s.ReplaceModelPrices(prices); err != nil {
		t.Fatalf("ReplaceModelPrices() error = %v", err)
	}
	got, err := s.ListModelPrices()
	if err != nil {
		t.Fatalf("ListModelPrices() error = %v", err)
	}
	if len(got) != 1 || got[0].Model != "gpt-test" || got[0].InputPerMTok != 1.25 || got[0].OutputPerMTok != 5.5 {
		t.Fatalf("ListModelPrices() = %#v", got)
	}
}

func TestOpenCreatesParentDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "data", "usage.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected database file to exist: %v", err)
	}
}

func TestAPIKeyAliasesPersistUpdateDelete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := s.UpsertAPIKeyAliases([]APIKeyAlias{{APIKeyHash: hash, Alias: " First Alias ", UpdatedAtMS: 111}}); err != nil {
		t.Fatalf("UpsertAPIKeyAliases() insert error = %v", err)
	}
	got, err := s.ListAPIKeyAliases()
	if err != nil {
		t.Fatalf("ListAPIKeyAliases() error = %v", err)
	}
	if len(got) != 1 || got[0].APIKeyHash != hash || got[0].Alias != "First Alias" || got[0].UpdatedAtMS != 111 {
		t.Fatalf("ListAPIKeyAliases() after insert = %#v", got)
	}

	if err := s.UpsertAPIKeyAliases([]APIKeyAlias{{APIKeyHash: hash, Alias: "Updated Alias", UpdatedAtMS: 222}}); err != nil {
		t.Fatalf("UpsertAPIKeyAliases() update error = %v", err)
	}
	got, err = s.ListAPIKeyAliases()
	if err != nil {
		t.Fatalf("ListAPIKeyAliases() after update error = %v", err)
	}
	if len(got) != 1 || got[0].Alias != "Updated Alias" || got[0].UpdatedAtMS != 222 {
		t.Fatalf("ListAPIKeyAliases() after update = %#v", got)
	}

	if err := s.DeleteAPIKeyAlias(hash); err != nil {
		t.Fatalf("DeleteAPIKeyAlias() error = %v", err)
	}
	got, err = s.ListAPIKeyAliases()
	if err != nil {
		t.Fatalf("ListAPIKeyAliases() after delete error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListAPIKeyAliases() after delete = %#v, want empty", got)
	}
}

func TestAPIKeyAliasesOrphanCleanup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	orphanHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	activeHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := s.UpsertAPIKeyAliases([]APIKeyAlias{{APIKeyHash: orphanHash, Alias: "Shared Alias", UpdatedAtMS: 111}}); err != nil {
		t.Fatalf("UpsertAPIKeyAliases() seed error = %v", err)
	}

	if err := s.UpsertAPIKeyAliasesWithActiveHashes(
		[]APIKeyAlias{{APIKeyHash: activeHash, Alias: "shared alias", UpdatedAtMS: 222}},
		[]string{activeHash},
		true,
	); err != nil {
		t.Fatalf("UpsertAPIKeyAliasesWithActiveHashes() cleanup error = %v", err)
	}

	got, err := s.ListAPIKeyAliases()
	if err != nil {
		t.Fatalf("ListAPIKeyAliases() error = %v", err)
	}
	if len(got) != 1 || got[0].APIKeyHash != activeHash || got[0].Alias != "shared alias" {
		t.Fatalf("ListAPIKeyAliases() after orphan cleanup = %#v", got)
	}
}

func TestUsageEventsPersistSummaryAndExport(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	result, err := s.ImportUsageEvents([]UsageEvent{
		{
			EventHash:    "event-success",
			TimestampMS:  1700000000000,
			Model:        "gpt-test",
			Endpoint:     "/v1/chat/completions",
			Method:       "POST",
			Path:         "/v1/chat/completions",
			AuthIndex:    "auth-1",
			Source:       "cli",
			SourceHash:   "source-1",
			APIKeyHash:   "key-1",
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
			LatencyMS:    123,
			RawJSON:      json.RawMessage(`{"event_hash":"event-success"}`),
		},
		{
			EventHash:      "event-failure",
			TimestampMS:    1700000001000,
			Model:          "gpt-test",
			Endpoint:       "/v1/responses",
			Method:         "POST",
			Path:           "/v1/responses",
			AuthIndex:      "auth-2",
			Source:         "worker",
			SourceHash:     "source-2",
			APIKeyHash:     "key-2",
			TotalTokens:    5,
			Failed:         true,
			FailStatusCode: 429,
			FailSummary:    "rate limited",
			RawJSON:        json.RawMessage(`{"event_hash":"event-failure","failed":true}`),
		},
	})
	if err != nil {
		t.Fatalf("ImportUsageEvents() error = %v", err)
	}
	if result.Added != 2 || result.Skipped != 0 || result.Total != 2 || result.Failed != 0 {
		t.Fatalf("ImportUsageEvents() = %#v", result)
	}

	summary, err := s.UsageSummary(UsageQuery{})
	if err != nil {
		t.Fatalf("UsageSummary() error = %v", err)
	}
	if summary.TotalRequests != 2 || summary.SuccessCount != 1 || summary.FailureCount != 1 || summary.TotalTokens != 35 {
		t.Fatalf("UsageSummary() = %#v", summary)
	}
	if len(summary.APIs) != 2 {
		t.Fatalf("UsageSummary().APIs = %#v", summary.APIs)
	}
	api := summary.APIs["POST /v1/chat/completions"]
	model := api.Models["gpt-test"]
	if api.Requests != 1 || model.Requests != 1 || len(model.Details) != 1 || model.Details[0].Tokens.TotalTokens != 30 {
		t.Fatalf("UsageSummary().APIs detail = %#v", summary.APIs)
	}

	rows, err := s.ExportUsageEvents(UsageQuery{})
	if err != nil {
		t.Fatalf("ExportUsageEvents() error = %v", err)
	}
	if len(rows) != 2 || rows[0].EventHash != "event-success" || rows[1].EventHash != "event-failure" {
		t.Fatalf("ExportUsageEvents() = %#v", rows)
	}
	if string(rows[0].RawJSON) == "" || string(rows[1].RawJSON) == "" {
		t.Fatalf("ExportUsageEvents() raw json missing: %#v", rows)
	}
}
