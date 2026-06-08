package model

// ModelPrice stores per-million-token prices and optional sync metadata for a model.
type ModelPrice struct {
	Model         string  `json:"model"`
	InputPerMTok  float64 `json:"inputPerMTok"`
	OutputPerMTok float64 `json:"outputPerMTok"`
	Cache         float64 `json:"cache,omitempty"`
	CacheRead     float64 `json:"cacheRead,omitempty"`
	CacheCreation float64 `json:"cacheCreation,omitempty"`
	Source        string  `json:"source,omitempty"`
	SourceModelID string  `json:"sourceModelId,omitempty"`
	RawJSON       string  `json:"rawJson,omitempty"`
	UpdatedAtMS   int64   `json:"updatedAtMs,omitempty"`
	SyncedAtMS    *int64  `json:"syncedAtMs,omitempty"`
}
