package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/model"
	_ "modernc.org/sqlite"
)

type ModelPrice = model.ModelPrice
type APIKeyAlias = model.APIKeyAlias
type UsageEvent = model.UsageEvent
type UsageImportResult = model.UsageImportResult
type UsagePayload = model.UsagePayload
type UsageQuery = model.UsageQuery

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
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS usage_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_hash TEXT NOT NULL UNIQUE,
		timestamp_ms INTEGER NOT NULL,
		model TEXT NOT NULL DEFAULT '',
		endpoint TEXT NOT NULL DEFAULT '',
		method TEXT NOT NULL DEFAULT '',
		path TEXT NOT NULL DEFAULT '',
		auth_index TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '',
		source_hash TEXT NOT NULL DEFAULT '',
		api_key_hash TEXT NOT NULL DEFAULT '',
		account_snapshot TEXT NOT NULL DEFAULT '',
		auth_label_snapshot TEXT NOT NULL DEFAULT '',
		auth_provider_snapshot TEXT NOT NULL DEFAULT '',
		auth_project_id_snapshot TEXT NOT NULL DEFAULT '',
		resolved_model TEXT NOT NULL DEFAULT '',
		reasoning_effort TEXT NOT NULL DEFAULT '',
		service_tier TEXT NOT NULL DEFAULT '',
		executor_type TEXT NOT NULL DEFAULT '',
		input_tokens INTEGER NOT NULL DEFAULT 0,
		output_tokens INTEGER NOT NULL DEFAULT 0,
		cached_tokens INTEGER NOT NULL DEFAULT 0,
		cache_read_tokens INTEGER NOT NULL DEFAULT 0,
		cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
		reasoning_tokens INTEGER NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0,
		latency_ms INTEGER NOT NULL DEFAULT 0,
		ttft_ms INTEGER NOT NULL DEFAULT 0,
		failed INTEGER NOT NULL DEFAULT 0,
		fail_status_code INTEGER NOT NULL DEFAULT 0,
		fail_summary TEXT NOT NULL DEFAULT '',
		raw_json TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_events_timestamp_id ON usage_events(timestamp_ms DESC, id DESC)`); err != nil {
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

func (s *Store) ImportUsageEvents(events []UsageEvent) (UsageImportResult, error) {
	result := UsageImportResult{Total: len(events)}
	tx, err := s.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO usage_events (
		event_hash, timestamp_ms, model, endpoint, method, path, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_provider_snapshot, auth_project_id_snapshot,
		resolved_model, reasoning_effort, service_tier, executor_type,
		input_tokens, output_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, reasoning_tokens,
		total_tokens, latency_ms, ttft_ms, failed, fail_status_code, fail_summary, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return result, err
	}
	defer stmt.Close()

	for _, event := range events {
		if event.EventHash == "" {
			result.Failed++
			continue
		}
		if event.TotalTokens == 0 {
			event.TotalTokens = event.InputTokens + event.OutputTokens + event.ReasoningTokens
		}
		raw := string(event.RawJSON)
		res, err := stmt.Exec(
			event.EventHash, event.TimestampMS, event.Model, event.Endpoint, event.Method, event.Path, event.AuthIndex, event.Source, event.SourceHash, event.APIKeyHash,
			event.AccountSnapshot, event.AuthLabelSnapshot, event.AuthProviderSnapshot, event.AuthProjectIDSnapshot,
			event.ResolvedModel, event.ReasoningEffort, event.ServiceTier, event.ExecutorType,
			event.InputTokens, event.OutputTokens, event.CachedTokens, event.CacheReadTokens, event.CacheCreationTokens, event.ReasoningTokens,
			event.TotalTokens, event.LatencyMS, event.TTFTMS, boolToInt(event.Failed), event.FailStatusCode, event.FailSummary, raw,
		)
		if err != nil {
			result.Failed++
			continue
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return result, err
		}
		if affected == 0 {
			result.Skipped++
		} else {
			result.Added++
		}
	}
	return result, tx.Commit()
}

func (s *Store) UsageSummary(query UsageQuery) (UsagePayload, error) {
	where, args := usageWhere(query)
	var payload UsagePayload
	row := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN failed = 0 THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN failed != 0 THEN 1 ELSE 0 END), 0), COALESCE(SUM(total_tokens), 0) FROM usage_events`+where, args...)
	if err := row.Scan(&payload.TotalRequests, &payload.SuccessCount, &payload.FailureCount, &payload.TotalTokens); err != nil {
		return payload, err
	}

	rows, err := s.db.Query(`SELECT endpoint, COUNT(*), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(CASE WHEN failed = 0 THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN failed != 0 THEN 1 ELSE 0 END), 0) FROM usage_events`+where+` GROUP BY endpoint ORDER BY COUNT(*) DESC, endpoint`, args...)
	if err != nil {
		return payload, err
	}
	defer rows.Close()
	for rows.Next() {
		var api model.UsageAPIStat
		if err := rows.Scan(&api.Endpoint, &api.Requests, &api.Tokens, &api.Success, &api.Failure); err != nil {
			return payload, err
		}
		payload.APIs = append(payload.APIs, api)
	}
	return payload, rows.Err()
}

func (s *Store) ExportUsageEvents(query UsageQuery) ([]UsageEvent, error) {
	where, args := usageWhere(query)
	rows, err := s.db.Query(`SELECT `+usageEventColumns()+` FROM usage_events`+where+` ORDER BY timestamp_ms ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []UsageEvent{}
	for rows.Next() {
		event, err := scanUsageEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) ListUsageEvents(query UsageQuery) ([]UsageEvent, bool, error) {
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	where, args := usageWhere(query)
	pageWhere := where
	if query.BeforeMS > 0 {
		if pageWhere == "" {
			pageWhere = " WHERE "
		} else {
			pageWhere += " AND "
		}
		pageWhere += "(timestamp_ms < ? OR (timestamp_ms = ? AND id < ?))"
		args = append(args, query.BeforeMS, query.BeforeMS, query.BeforeID)
	}
	args = append(args, limit+1)
	rows, err := s.db.Query(`SELECT `+usageEventColumns()+` FROM usage_events`+pageWhere+` ORDER BY timestamp_ms DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	events := []UsageEvent{}
	for rows.Next() {
		event, err := scanUsageEvent(rows)
		if err != nil {
			return nil, false, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}
	return events, hasMore, nil
}

func usageWhere(query UsageQuery) (string, []any) {
	parts := []string{}
	args := []any{}
	if query.FromMS > 0 {
		parts = append(parts, "timestamp_ms >= ?")
		args = append(args, query.FromMS)
	}
	if query.ToMS > 0 {
		parts = append(parts, "timestamp_ms <= ?")
		args = append(args, query.ToMS)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + parts[0] + joinAnd(parts[1:]), args
}

func joinAnd(parts []string) string {
	out := ""
	for _, part := range parts {
		out += " AND " + part
	}
	return out
}

func usageEventColumns() string {
	return `id, event_hash, timestamp_ms, model, endpoint, method, path, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_provider_snapshot, auth_project_id_snapshot,
		resolved_model, reasoning_effort, service_tier, executor_type,
		input_tokens, output_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, reasoning_tokens,
		total_tokens, latency_ms, ttft_ms, failed, fail_status_code, fail_summary, raw_json`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUsageEvent(row rowScanner) (UsageEvent, error) {
	var event UsageEvent
	var failed int
	var raw string
	err := row.Scan(
		&event.ID, &event.EventHash, &event.TimestampMS, &event.Model, &event.Endpoint, &event.Method, &event.Path, &event.AuthIndex, &event.Source, &event.SourceHash, &event.APIKeyHash,
		&event.AccountSnapshot, &event.AuthLabelSnapshot, &event.AuthProviderSnapshot, &event.AuthProjectIDSnapshot,
		&event.ResolvedModel, &event.ReasoningEffort, &event.ServiceTier, &event.ExecutorType,
		&event.InputTokens, &event.OutputTokens, &event.CachedTokens, &event.CacheReadTokens, &event.CacheCreationTokens, &event.ReasoningTokens,
		&event.TotalTokens, &event.LatencyMS, &event.TTFTMS, &failed, &event.FailStatusCode, &event.FailSummary, &raw,
	)
	event.Failed = failed != 0
	event.RawJSON = []byte(raw)
	return event, err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s *Store) InsertUsageEvent(ctx context.Context, raw json.RawMessage) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	event, err := usageEventFromRaw(raw)
	if err != nil {
		return err
	}
	_, err = s.ImportUsageEvents([]UsageEvent{event})
	return err
}

func usageEventFromRaw(raw json.RawMessage) (UsageEvent, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return UsageEvent{}, err
	}
	event := UsageEvent{
		EventHash:             readString(data, "event_hash", "eventHash"),
		TimestampMS:           readInt64(data, "timestamp_ms", "timestampMs"),
		Model:                 readString(data, "model", "alias"),
		Endpoint:              readString(data, "endpoint"),
		Method:                readString(data, "method"),
		Path:                  readString(data, "path"),
		AuthIndex:             readString(data, "auth_index", "authIndex"),
		Source:                readString(data, "source"),
		SourceHash:            readString(data, "source_hash", "sourceHash"),
		APIKeyHash:            readString(data, "api_key_hash", "apiKeyHash"),
		AccountSnapshot:       readString(data, "account_snapshot", "accountSnapshot"),
		AuthLabelSnapshot:     readString(data, "auth_label_snapshot", "authLabelSnapshot"),
		AuthProviderSnapshot:  readString(data, "auth_provider_snapshot", "authProviderSnapshot", "provider"),
		AuthProjectIDSnapshot: readString(data, "auth_project_id_snapshot", "authProjectIdSnapshot", "authProjectIDSnapshot"),
		ResolvedModel:         readString(data, "resolved_model", "resolvedModel"),
		ReasoningEffort:       readString(data, "reasoning_effort", "reasoningEffort"),
		ServiceTier:           readString(data, "service_tier", "serviceTier"),
		ExecutorType:          readString(data, "executor_type", "executorType"),
		InputTokens:           readInt64(data, "input_tokens", "inputTokens"),
		OutputTokens:          readInt64(data, "output_tokens", "outputTokens"),
		CachedTokens:          readInt64(data, "cached_tokens", "cachedTokens"),
		CacheReadTokens:       readInt64(data, "cache_read_tokens", "cacheReadTokens"),
		CacheCreationTokens:   readInt64(data, "cache_creation_tokens", "cacheCreationTokens"),
		ReasoningTokens:       readInt64(data, "reasoning_tokens", "reasoningTokens"),
		TotalTokens:           readInt64(data, "total_tokens", "totalTokens"),
		LatencyMS:             readInt64(data, "latency_ms", "latencyMs", "duration_ms", "durationMs"),
		TTFTMS:                readInt64(data, "ttft_ms", "ttftMs"),
		Failed:                readBool(data, "failed"),
		FailStatusCode:        int(readInt64(data, "fail_status_code", "failStatusCode")),
		FailSummary:           readString(data, "fail_summary", "failSummary"),
		RawJSON:               append(json.RawMessage(nil), raw...),
	}
	if event.TimestampMS == 0 {
		event.TimestampMS = readTimestampMS(data, "timestamp")
	}
	if tokens, ok := data["tokens"].(map[string]any); ok {
		if event.InputTokens == 0 {
			event.InputTokens = readInt64(tokens, "input_tokens", "inputTokens")
		}
		if event.OutputTokens == 0 {
			event.OutputTokens = readInt64(tokens, "output_tokens", "outputTokens")
		}
		if event.CachedTokens == 0 {
			event.CachedTokens = readInt64(tokens, "cached_tokens", "cachedTokens")
		}
		if event.CacheReadTokens == 0 {
			event.CacheReadTokens = readInt64(tokens, "cache_read_tokens", "cacheReadTokens")
		}
		if event.CacheCreationTokens == 0 {
			event.CacheCreationTokens = readInt64(tokens, "cache_creation_tokens", "cacheCreationTokens")
		}
		if event.ReasoningTokens == 0 {
			event.ReasoningTokens = readInt64(tokens, "reasoning_tokens", "reasoningTokens")
		}
		if event.TotalTokens == 0 {
			event.TotalTokens = readInt64(tokens, "total_tokens", "totalTokens")
		}
	}
	if fail, ok := data["fail"].(map[string]any); ok {
		if event.FailStatusCode == 0 {
			event.FailStatusCode = int(readInt64(fail, "status_code", "statusCode"))
		}
		if event.FailSummary == "" {
			event.FailSummary = readString(fail, "body", "summary")
		}
	}
	if event.APIKeyHash == "" {
		if apiKey := readString(data, "api_key", "apiKey"); apiKey != "" {
			event.APIKeyHash = sha256Hex([]byte(apiKey))
		}
	}
	if event.SourceHash == "" && event.Source != "" {
		event.SourceHash = sha256Hex([]byte(event.Source))
	}
	if event.EventHash == "" {
		event.EventHash = sha256Hex(raw)
	}
	if event.TimestampMS == 0 {
		event.TimestampMS = time.Now().UnixMilli()
	}
	if event.Endpoint == "" {
		event.Endpoint = event.Path
	}
	if event.Path == "" {
		event.Path = event.Endpoint
	}
	if event.Method == "" && strings.Contains(event.Endpoint, " ") {
		parts := strings.Fields(event.Endpoint)
		if len(parts) >= 2 {
			event.Method = parts[0]
			event.Path = parts[1]
		}
	}
	return event, nil
}

func readString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok && value != nil {
			return strings.TrimSpace(toString(value))
		}
	}
	return ""
}

func readInt64(data map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int64(typed)
		case string:
			parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			return parsed
		case json.Number:
			parsed, _ := typed.Int64()
			return parsed
		}
	}
	return 0
}

func readTimestampMS(data map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value := readString(data, key)
		if value == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return parsed.UnixMilli()
		}
	}
	return 0
}

func readBool(data map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			return typed == "true" || typed == "1"
		case float64:
			return typed != 0
		}
	}
	return false
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
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
