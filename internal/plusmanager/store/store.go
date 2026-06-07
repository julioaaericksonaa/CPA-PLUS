package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/model"
	_ "modernc.org/sqlite"
)

type ModelPrice = model.ModelPrice
type APIKeyAlias = model.APIKeyAlias

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	dbDir := filepath.Dir(dbPath)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS model_prices (
		model TEXT PRIMARY KEY,
		input_per_mtok REAL NOT NULL,
		output_per_mtok REAL NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS api_key_aliases (
		api_key_hash TEXT PRIMARY KEY,
		alias TEXT NOT NULL,
		updated_at_ms INTEGER NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) ListModelPrices() ([]ModelPrice, error) {
	rows, err := s.db.Query(`SELECT model, input_per_mtok, output_per_mtok FROM model_prices ORDER BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := []ModelPrice{}
	for rows.Next() {
		var price ModelPrice
		if err := rows.Scan(&price.Model, &price.InputPerMTok, &price.OutputPerMTok); err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prices, nil
}

func (s *Store) ReplaceModelPrices(prices []ModelPrice) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM model_prices`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO model_prices (model, input_per_mtok, output_per_mtok) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, price := range prices {
		if _, err := stmt.Exec(price.Model, price.InputPerMTok, price.OutputPerMTok); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListAPIKeyAliases() ([]APIKeyAlias, error) {
	rows, err := s.db.Query(`SELECT api_key_hash, alias, updated_at_ms FROM api_key_aliases ORDER BY alias COLLATE NOCASE, api_key_hash`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aliases := []APIKeyAlias{}
	for rows.Next() {
		var alias APIKeyAlias
		if err := rows.Scan(&alias.APIKeyHash, &alias.Alias, &alias.UpdatedAtMS); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return aliases, nil
}

func (s *Store) UpsertAPIKeyAliases(aliases []APIKeyAlias) error {
	return s.UpsertAPIKeyAliasesWithActiveHashes(aliases, nil, false)
}

func (s *Store) UpsertAPIKeyAliasesWithActiveHashes(aliases []APIKeyAlias, activeHashes []string, allowOrphanCleanup bool) error {
	if len(aliases) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	normalizedAliases := make([]APIKeyAlias, 0, len(aliases))
	seenAliases := map[string]string{}
	for _, alias := range aliases {
		normalized, err := normalizeAPIKeyAlias(alias, now)
		if err != nil {
			return err
		}
		aliasKey := normalizeAPIKeyAliasUniqueKey(normalized.Alias)
		if existingHash, ok := seenAliases[aliasKey]; ok && existingHash != normalized.APIKeyHash {
			return errors.New("api key alias already exists")
		}
		seenAliases[aliasKey] = normalized.APIKeyHash
		normalizedAliases = append(normalizedAliases, normalized)
	}

	var activeSet map[string]struct{}
	if len(activeHashes) > 0 {
		activeSet = make(map[string]struct{}, len(activeHashes)+len(normalizedAliases))
		for _, h := range activeHashes {
			hash := strings.ToLower(strings.TrimSpace(h))
			if validAPIKeyHash(hash) {
				activeSet[hash] = struct{}{}
			}
		}
		for _, normalized := range normalizedAliases {
			activeSet[normalized.APIKeyHash] = struct{}{}
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO api_key_aliases (api_key_hash, alias, updated_at_ms)
		VALUES (?, ?, ?)
		ON CONFLICT(api_key_hash) DO UPDATE SET
			alias = excluded.alias,
			updated_at_ms = excluded.updated_at_ms`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	deleteStmt, err := tx.Prepare(`DELETE FROM api_key_aliases WHERE api_key_hash = ?`)
	if err != nil {
		return err
	}
	defer deleteStmt.Close()

	existingRows, err := tx.Query(`SELECT api_key_hash, alias FROM api_key_aliases`)
	if err != nil {
		return err
	}
	existingAliases := map[string]string{}
	for existingRows.Next() {
		var apiKeyHash string
		var alias string
		if err := existingRows.Scan(&apiKeyHash, &alias); err != nil {
			_ = existingRows.Close()
			return err
		}
		existingAliases[normalizeAPIKeyAliasUniqueKey(alias)] = apiKeyHash
	}
	if err := existingRows.Close(); err != nil {
		return err
	}
	if err := existingRows.Err(); err != nil {
		return err
	}

	for _, normalized := range normalizedAliases {
		aliasKey := normalizeAPIKeyAliasUniqueKey(normalized.Alias)
		if existingHash, ok := existingAliases[aliasKey]; ok && existingHash != normalized.APIKeyHash {
			if activeSet == nil {
				return errors.New("api key alias already exists")
			}
			if _, isActive := activeSet[existingHash]; isActive {
				return errors.New("api key alias already exists")
			}
			if !allowOrphanCleanup {
				return errors.New("api key alias already exists")
			}
			if _, err := deleteStmt.Exec(existingHash); err != nil {
				return err
			}
			delete(existingAliases, aliasKey)
		}
		if _, err := stmt.Exec(normalized.APIKeyHash, normalized.Alias, normalized.UpdatedAtMS); err != nil {
			return err
		}
		existingAliases[aliasKey] = normalized.APIKeyHash
	}
	return tx.Commit()
}

func (s *Store) DeleteAPIKeyAlias(apiKeyHash string) error {
	hash := strings.ToLower(strings.TrimSpace(apiKeyHash))
	if !validAPIKeyHash(hash) {
		return errors.New("valid apiKeyHash is required")
	}
	_, err := s.db.Exec(`DELETE FROM api_key_aliases WHERE api_key_hash = ?`, hash)
	return err
}

func normalizeAPIKeyAlias(alias APIKeyAlias, now int64) (APIKeyAlias, error) {
	hash := strings.ToLower(strings.TrimSpace(alias.APIKeyHash))
	if !validAPIKeyHash(hash) {
		return APIKeyAlias{}, errors.New("valid apiKeyHash is required")
	}
	label := strings.TrimSpace(alias.Alias)
	if label == "" {
		return APIKeyAlias{}, errors.New("alias is required")
	}
	if len([]rune(label)) > 120 {
		return APIKeyAlias{}, errors.New("alias must be 120 characters or less")
	}
	if alias.UpdatedAtMS <= 0 {
		alias.UpdatedAtMS = now
	}
	alias.APIKeyHash = hash
	alias.Alias = label
	return alias, nil
}

func normalizeAPIKeyAliasUniqueKey(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

func validAPIKeyHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}
