package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/store"
)

type ModelPriceStore interface {
	ListModelPrices() ([]store.ModelPrice, error)
	ReplaceModelPrices([]store.ModelPrice) error
}

type APIKeyAliasStore interface {
	ListAPIKeyAliases() ([]store.APIKeyAlias, error)
	UpsertAPIKeyAliasesWithActiveHashes([]store.APIKeyAlias, []string, bool) error
	DeleteAPIKeyAlias(string) error
}

type Options struct {
	Enabled bool
	Store   any
}

func RegisterRoutes(group *gin.RouterGroup, opts Options) {
	if group == nil || !opts.Enabled {
		return
	}

	group.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"mode":   "integrated",
		})
	})
	group.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"integrated":    true,
			"setupRequired": false,
		})
	})
	RegisterModelPriceRoutes(group, opts)
	RegisterAPIKeyAliasRoutes(group, opts)
}

func RegisterCompatibilityRoutes(engine *gin.Engine, opts Options) {
	if engine == nil || !opts.Enabled {
		return
	}
	engine.GET("/usage-service/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":            "cpa-manager-plus",
			"mode":               "integrated",
			"configured":         true,
			"projectInitialized": true,
			"setupRequired":      false,
			"adminReady":         true,
			"dataKeyReady":       true,
		})
	})
	engine.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "cpa-manager-plus",
			"mode":    "integrated",
		})
	})
}

func RegisterModelPriceRoutes(group *gin.RouterGroup, opts Options) {
	if group == nil || !opts.Enabled {
		return
	}
	group.GET("/model-prices", func(c *gin.Context) {
		handleGetModelPrices(c, modelPriceStoreFromOptions(opts))
	})
	group.PUT("/model-prices", func(c *gin.Context) {
		handlePutModelPrices(c, modelPriceStoreFromOptions(opts))
	})
}

func RegisterAPIKeyAliasRoutes(group *gin.RouterGroup, opts Options) {
	if group == nil || !opts.Enabled {
		return
	}
	group.GET("/api-key-aliases", func(c *gin.Context) {
		handleGetAPIKeyAliases(c, apiKeyAliasStoreFromOptions(opts))
	})
	group.PUT("/api-key-aliases", func(c *gin.Context) {
		handlePutAPIKeyAliases(c, apiKeyAliasStoreFromOptions(opts))
	})
	group.DELETE("/api-key-aliases/:apiKeyHash", func(c *gin.Context) {
		handleDeleteAPIKeyAlias(c, apiKeyAliasStoreFromOptions(opts))
	})
}

type httpModelPrice struct {
	Prompt        float64 `json:"prompt,omitempty"`
	Completion    float64 `json:"completion,omitempty"`
	Cache         float64 `json:"cache,omitempty"`
	Input         float64 `json:"input,omitempty"`
	Output        float64 `json:"output,omitempty"`
	InputPerMTok  float64 `json:"inputPerMTok,omitempty"`
	OutputPerMTok float64 `json:"outputPerMTok,omitempty"`
}

type modelPricesResponse struct {
	Prices map[string]httpModelPrice `json:"prices"`
}

type apiKeyAliasesResponse struct {
	Items []store.APIKeyAlias `json:"items"`
}

type putAPIKeyAliasesRequest struct {
	Items                   []store.APIKeyAlias `json:"items"`
	ActiveAPIKeyHashes      []string            `json:"activeApiKeyHashes,omitempty"`
	AllowOrphanAliasCleanup bool                `json:"allowOrphanAliasCleanup,omitempty"`
}

func modelPriceStoreFromOptions(opts Options) ModelPriceStore {
	if priceStore, ok := opts.Store.(ModelPriceStore); ok {
		return priceStore
	}
	return nil
}

func apiKeyAliasStoreFromOptions(opts Options) APIKeyAliasStore {
	if aliasStore, ok := opts.Store.(APIKeyAliasStore); ok {
		return aliasStore
	}
	return nil
}

func handleGetModelPrices(c *gin.Context, priceStore ModelPriceStore) {
	if priceStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model price store unavailable"})
		return
	}
	prices, err := priceStore.ListModelPrices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list model prices failed"})
		return
	}
	c.JSON(http.StatusOK, modelPricesResponse{Prices: toHTTPModelPrices(prices)})
}

func handlePutModelPrices(c *gin.Context, priceStore ModelPriceStore) {
	prices, ok := bindModelPrices(c)
	if !ok {
		return
	}
	if priceStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model price store unavailable"})
		return
	}
	if err := priceStore.ReplaceModelPrices(prices); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "replace model prices failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

func bindModelPrices(c *gin.Context) ([]store.ModelPrice, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return nil, false
	}
	var wrapped modelPricesResponse
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Prices != nil {
		return fromHTTPModelPrices(wrapped.Prices), true
	}
	var prices []store.ModelPrice
	if err := json.Unmarshal(body, &prices); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return nil, false
	}
	return prices, true
}

func toHTTPModelPrices(prices []store.ModelPrice) map[string]httpModelPrice {
	out := make(map[string]httpModelPrice, len(prices))
	for _, price := range prices {
		model := strings.TrimSpace(price.Model)
		if model == "" {
			continue
		}
		out[model] = httpModelPrice{
			Prompt:        price.InputPerMTok,
			Completion:    price.OutputPerMTok,
			Input:         price.InputPerMTok,
			Output:        price.OutputPerMTok,
			InputPerMTok:  price.InputPerMTok,
			OutputPerMTok: price.OutputPerMTok,
		}
	}
	return out
}

func handleGetAPIKeyAliases(c *gin.Context, aliasStore APIKeyAliasStore) {
	if aliasStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "api key alias store unavailable"})
		return
	}
	aliases, err := aliasStore.ListAPIKeyAliases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list api key aliases failed"})
		return
	}
	c.JSON(http.StatusOK, apiKeyAliasesResponse{Items: aliases})
}

func handlePutAPIKeyAliases(c *gin.Context, aliasStore APIKeyAliasStore) {
	var req putAPIKeyAliasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if req.Items == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api key aliases are required"})
		return
	}
	if aliasStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "api key alias store unavailable"})
		return
	}
	if err := aliasStore.UpsertAPIKeyAliasesWithActiveHashes(req.Items, req.ActiveAPIKeyHashes, req.AllowOrphanAliasCleanup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	aliases, err := aliasStore.ListAPIKeyAliases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list api key aliases failed"})
		return
	}
	c.JSON(http.StatusOK, apiKeyAliasesResponse{Items: aliases})
}

func handleDeleteAPIKeyAlias(c *gin.Context, aliasStore APIKeyAliasStore) {
	if aliasStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "api key alias store unavailable"})
		return
	}
	if err := aliasStore.DeleteAPIKeyAlias(c.Param("apiKeyHash")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func fromHTTPModelPrices(prices map[string]httpModelPrice) []store.ModelPrice {
	out := make([]store.ModelPrice, 0, len(prices))
	for model, price := range prices {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		input := price.Prompt
		if input == 0 {
			input = price.Input
		}
		if input == 0 {
			input = price.InputPerMTok
		}
		output := price.Completion
		if output == 0 {
			output = price.Output
		}
		if output == 0 {
			output = price.OutputPerMTok
		}
		out = append(out, store.ModelPrice{
			Model:         model,
			InputPerMTok:  input,
			OutputPerMTok: output,
		})
	}
	return out
}
