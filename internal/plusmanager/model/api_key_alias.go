package model

// APIKeyAlias stores a stable display alias for a hashed API key.
type APIKeyAlias struct {
	APIKeyHash  string `json:"apiKeyHash"`
	Alias       string `json:"alias"`
	UpdatedAtMS int64  `json:"updatedAtMs,omitempty"`
}
