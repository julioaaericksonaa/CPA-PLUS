package model

import "encoding/json"

type UsageEvent struct {
	ID                    int64           `json:"id,omitempty"`
	EventHash             string          `json:"event_hash"`
	TimestampMS           int64           `json:"timestamp_ms"`
	Model                 string          `json:"model"`
	Endpoint              string          `json:"endpoint"`
	Method                string          `json:"method"`
	Path                  string          `json:"path"`
	AuthIndex             string          `json:"auth_index"`
	Source                string          `json:"source"`
	SourceHash            string          `json:"source_hash"`
	APIKeyHash            string          `json:"api_key_hash"`
	AccountSnapshot       string          `json:"account_snapshot"`
	AuthLabelSnapshot     string          `json:"auth_label_snapshot"`
	AuthProviderSnapshot  string          `json:"auth_provider_snapshot"`
	AuthProjectIDSnapshot string          `json:"auth_project_id_snapshot,omitempty"`
	ResolvedModel         string          `json:"resolved_model,omitempty"`
	ReasoningEffort       string          `json:"reasoning_effort,omitempty"`
	ServiceTier           string          `json:"service_tier,omitempty"`
	ExecutorType          string          `json:"executor_type,omitempty"`
	InputTokens           int64           `json:"input_tokens"`
	OutputTokens          int64           `json:"output_tokens"`
	CachedTokens          int64           `json:"cached_tokens"`
	CacheReadTokens       int64           `json:"cache_read_tokens"`
	CacheCreationTokens   int64           `json:"cache_creation_tokens"`
	ReasoningTokens       int64           `json:"reasoning_tokens"`
	TotalTokens           int64           `json:"total_tokens"`
	LatencyMS             int64           `json:"latency_ms,omitempty"`
	TTFTMS                int64           `json:"ttft_ms,omitempty"`
	Failed                bool            `json:"failed"`
	FailStatusCode        int             `json:"fail_status_code,omitempty"`
	FailSummary           string          `json:"fail_summary,omitempty"`
	RawJSON               json.RawMessage `json:"-"`
}

type UsageImportResult struct {
	Added   int `json:"added"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
	Failed  int `json:"failed"`
}

type UsageQuery struct {
	FromMS   int64
	ToMS     int64
	Limit    int
	BeforeMS int64
	BeforeID int64
}

type UsageAPIStat struct {
	Endpoint string `json:"endpoint"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
	Success  int64  `json:"success"`
	Failure  int64  `json:"failure"`
}

type UsagePayload struct {
	TotalRequests int64          `json:"total_requests"`
	SuccessCount  int64          `json:"success_count"`
	FailureCount  int64          `json:"failure_count"`
	TotalTokens   int64          `json:"total_tokens"`
	APIs          []UsageAPIStat `json:"apis"`
}
