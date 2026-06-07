package model

// ModelPrice stores per-million-token input and output prices for a model.
type ModelPrice struct {
	Model         string  `json:"model"`
	InputPerMTok  float64 `json:"inputPerMTok"`
	OutputPerMTok float64 `json:"outputPerMTok"`
}
