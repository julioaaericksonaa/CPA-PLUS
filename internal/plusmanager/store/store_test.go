package store

import (
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
