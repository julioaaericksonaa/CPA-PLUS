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
