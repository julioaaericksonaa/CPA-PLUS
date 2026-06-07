package httpapi

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

type UsageStore interface {
	UsageSummary(store.UsageQuery) (store.UsagePayload, error)
	ImportUsageEvents([]store.UsageEvent) (store.UsageImportResult, error)
	ExportUsageEvents(store.UsageQuery) ([]store.UsageEvent, error)
	ListUsageEvents(store.UsageQuery) ([]store.UsageEvent, bool, error)
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
	RegisterUsageRoutes(group, opts)
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

func RegisterUsageRoutes(group *gin.RouterGroup, opts Options) {
	if group == nil || !opts.Enabled {
		return
	}
	group.GET("/usage", func(c *gin.Context) {
		usageStore := usageStoreFromOptions(opts)
		if usageStore == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage store unavailable"})
			return
		}
		payload, err := usageStore.UsageSummary(store.UsageQuery{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "usage summary failed"})
			return
		}
		c.JSON(http.StatusOK, payload)
	})
	group.GET("/usage/export", func(c *gin.Context) {
		handleExportUsage(c, usageStoreFromOptions(opts))
	})
	group.POST("/usage/import", func(c *gin.Context) {
		handleImportUsage(c, usageStoreFromOptions(opts))
	})
	group.GET("/dashboard/summary", func(c *gin.Context) {
		handleDashboardSummary(c, usageStoreFromOptions(opts))
	})
	group.POST("/monitoring/analytics", func(c *gin.Context) {
		handleMonitoringAnalytics(c, usageStoreFromOptions(opts))
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

func usageStoreFromOptions(opts Options) UsageStore {
	if usageStore, ok := opts.Store.(UsageStore); ok {
		return usageStore
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

func handleImportUsage(c *gin.Context, rawStore any) {
	usageStore, ok := rawStore.(UsageStore)
	if !ok || usageStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage store unavailable"})
		return
	}
	events := []store.UsageEvent{}
	scanner := bufio.NewScanner(c.Request.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		event, err := parseUsageEventLine([]byte(line))
		if err != nil {
			events = append(events, store.UsageEvent{})
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read usage import failed"})
		return
	}
	result, err := usageStore.ImportUsageEvents(events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import usage failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func handleExportUsage(c *gin.Context, rawStore any) {
	usageStore, ok := rawStore.(UsageStore)
	if !ok || usageStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage store unavailable"})
		return
	}
	events, err := usageStore.ExportUsageEvents(store.UsageQuery{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export usage failed"})
		return
	}
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Content-Disposition", `attachment; filename="usage-events.jsonl"`)
	for _, event := range events {
		if len(event.RawJSON) > 0 && json.Valid(event.RawJSON) {
			c.Writer.Write(event.RawJSON)
		} else {
			b, _ := json.Marshal(event)
			c.Writer.Write(b)
		}
		c.Writer.Write([]byte("\n"))
	}
}

type dashboardSummaryResponse struct {
	GeneratedAtMS              int64                               `json:"generated_at_ms"`
	Window                     dashboardSummaryWindow              `json:"window"`
	Today                      dashboardTodaySummary               `json:"today"`
	Rolling30M                 dashboardRollingSummary             `json:"rolling_30m"`
	TopModelsToday             []dashboardTopModel                 `json:"top_models_today"`
	ModelCostRank              []dashboardTopModel                 `json:"model_cost_rank"`
	TrafficTimeline            []any                               `json:"traffic_timeline"`
	HourlyActivity             []any                               `json:"hourly_activity"`
	TodayRequestHealthTimeline dashboardTodayRequestHealthTimeline `json:"today_request_health_timeline"`
	TokenMix                   []any                               `json:"token_mix"`
	ChannelHealth              []any                               `json:"channel_health"`
	FailureSources             []any                               `json:"failure_sources"`
	RecentFailures             []recentFailure                     `json:"recent_failures"`
}

type dashboardSummaryWindow struct {
	TodayStartMS      int64 `json:"today_start_ms"`
	NowMS             int64 `json:"now_ms"`
	Rolling30MStartMS int64 `json:"rolling_30m_start_ms"`
}

type dashboardTodaySummary struct {
	TotalCalls          int64    `json:"total_calls"`
	SuccessCalls        int64    `json:"success_calls"`
	FailureCalls        int64    `json:"failure_calls"`
	SuccessRate         float64  `json:"success_rate"`
	InputTokens         int64    `json:"input_tokens"`
	OutputTokens        int64    `json:"output_tokens"`
	CachedTokens        int64    `json:"cached_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	ReasoningTokens     int64    `json:"reasoning_tokens"`
	TotalTokens         int64    `json:"total_tokens"`
	TotalCost           float64  `json:"total_cost"`
	AverageLatencyMS    *float64 `json:"average_latency_ms"`
	ZeroTokenCalls      int64    `json:"zero_token_calls"`
}

type dashboardRollingSummary struct {
	RPM         float64 `json:"rpm"`
	TPM         float64 `json:"tpm"`
	TotalCalls  int64   `json:"total_calls"`
	TotalTokens int64   `json:"total_tokens"`
}

type dashboardTopModel struct {
	Model       string  `json:"model"`
	Calls       int64   `json:"calls"`
	Tokens      int64   `json:"tokens"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"success_rate"`
}

type dashboardTodayRequestHealthTimeline struct {
	FromMS       int64   `json:"from_ms"`
	ToMS         int64   `json:"to_ms"`
	BucketMS     int64   `json:"bucket_ms"`
	SuccessCalls int64   `json:"success_calls"`
	FailureCalls int64   `json:"failure_calls"`
	TotalCalls   int64   `json:"total_calls"`
	SuccessRate  float64 `json:"success_rate"`
	Points       []any   `json:"points"`
}

type recentFailure struct {
	TimestampMS    int64  `json:"timestamp_ms"`
	Model          string `json:"model"`
	APIKeyHash     string `json:"api_key_hash"`
	Source         string `json:"source,omitempty"`
	SourceHash     string `json:"source_hash"`
	AuthIndex      string `json:"auth_index"`
	Endpoint       string `json:"endpoint"`
	DurationMS     *int64 `json:"duration_ms"`
	FailStatusCode *int   `json:"fail_status_code"`
	FailSummary    string `json:"fail_summary,omitempty"`
}

func handleDashboardSummary(c *gin.Context, rawStore any) {
	usageStore, ok := rawStore.(UsageStore)
	if !ok || usageStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage store unavailable"})
		return
	}
	now := queryInt64(c, "now_ms", time.Now().UnixMilli())
	todayStart := queryInt64(c, "today_start_ms", now)
	query := store.UsageQuery{FromMS: todayStart, ToMS: now}
	summary, err := usageStore.UsageSummary(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dashboard summary failed"})
		return
	}
	failures, err := recentFailures(usageStore, query, queryInt(c, "recent_failures", 10))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dashboard failures failed"})
		return
	}
	c.JSON(http.StatusOK, dashboardSummaryResponse{
		GeneratedAtMS:              now,
		Window:                     dashboardSummaryWindow{TodayStartMS: todayStart, NowMS: now, Rolling30MStartMS: now - 30*60*1000},
		Today:                      toDashboardTodaySummary(summary),
		Rolling30M:                 dashboardRollingSummary{TotalCalls: summary.TotalRequests, TotalTokens: summary.TotalTokens, RPM: float64(summary.TotalRequests) / 30.0, TPM: float64(summary.TotalTokens) / 30.0},
		TopModelsToday:             []dashboardTopModel{},
		ModelCostRank:              []dashboardTopModel{},
		TrafficTimeline:            []any{},
		HourlyActivity:             []any{},
		TodayRequestHealthTimeline: dashboardTodayRequestHealthTimeline{FromMS: todayStart, ToMS: now, BucketMS: 60 * 60 * 1000, SuccessCalls: summary.SuccessCount, FailureCalls: summary.FailureCount, TotalCalls: summary.TotalRequests, SuccessRate: successRate(summary.SuccessCount, summary.TotalRequests), Points: []any{}},
		TokenMix:                   []any{},
		ChannelHealth:              []any{},
		FailureSources:             []any{},
		RecentFailures:             failures,
	})
}

type analyticsRequest struct {
	FromMS  int64            `json:"from_ms"`
	ToMS    int64            `json:"to_ms"`
	NowMS   int64            `json:"now_ms"`
	Include analyticsInclude `json:"include"`
}

type analyticsInclude struct {
	Summary        bool                   `json:"summary"`
	RecentFailures int                    `json:"recent_failures"`
	EventsPage     analyticsEventsPageReq `json:"events_page"`
	Granularity    string                 `json:"granularity"`
}

type analyticsEventsPageReq struct {
	Limit    int   `json:"limit"`
	BeforeMS int64 `json:"before_ms"`
	BeforeID int64 `json:"before_id"`
}

type analyticsResponse struct {
	GeneratedAtMS      int64                    `json:"generated_at_ms"`
	Granularity        string                   `json:"granularity"`
	Summary            *analyticsSummary        `json:"summary,omitempty"`
	Timeline           []any                    `json:"timeline,omitempty"`
	HourlyDistribution []any                    `json:"hourly_distribution,omitempty"`
	ModelShare         []any                    `json:"model_share,omitempty"`
	ModelStats         []any                    `json:"model_stats,omitempty"`
	ChannelShare       []any                    `json:"channel_share,omitempty"`
	FailureSources     []any                    `json:"failure_sources,omitempty"`
	AccountStats       []any                    `json:"account_stats,omitempty"`
	APIKeyStats        []any                    `json:"api_key_stats,omitempty"`
	FilterOptions      map[string]any           `json:"filter_options,omitempty"`
	TaskBuckets        []any                    `json:"task_buckets,omitempty"`
	RecentFailures     []recentFailure          `json:"recent_failures,omitempty"`
	Events             *analyticsEventsResponse `json:"events,omitempty"`
}

type analyticsSummary struct {
	dashboardTodaySummary
	RPM30M                float64  `json:"rpm_30m"`
	TPM30M                float64  `json:"tpm_30m"`
	AvgDailyRequests      float64  `json:"avg_daily_requests"`
	AvgDailyTokens        float64  `json:"avg_daily_tokens"`
	ApproxTasks           int64    `json:"approx_tasks"`
	ApproxTaskFailures    int64    `json:"approx_task_failures"`
	ApproxTaskSuccessRate float64  `json:"approx_task_success_rate"`
	ZeroTokenModels       []string `json:"zero_token_models"`
}

type analyticsEventsResponse struct {
	Items        []store.UsageEvent `json:"items"`
	NextBeforeMS int64              `json:"next_before_ms"`
	NextBeforeID int64              `json:"next_before_id,omitempty"`
	HasMore      bool               `json:"has_more"`
	TotalCount   int64              `json:"total_count,omitempty"`
}

func handleMonitoringAnalytics(c *gin.Context, rawStore any) {
	usageStore, ok := rawStore.(UsageStore)
	if !ok || usageStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage store unavailable"})
		return
	}
	var req analyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	now := req.NowMS
	if now == 0 {
		now = time.Now().UnixMilli()
	}
	query := store.UsageQuery{FromMS: req.FromMS, ToMS: req.ToMS}
	granularity := req.Include.Granularity
	if granularity == "" {
		granularity = "hour"
	}
	resp := analyticsResponse{GeneratedAtMS: now, Granularity: granularity}
	if req.Include.Summary {
		summary, err := usageStore.UsageSummary(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics summary failed"})
			return
		}
		daily := toDashboardTodaySummary(summary)
		resp.Summary = &analyticsSummary{
			dashboardTodaySummary: daily,
			RPM30M:                float64(summary.TotalRequests) / 30.0,
			TPM30M:                float64(summary.TotalTokens) / 30.0,
			AvgDailyRequests:      float64(summary.TotalRequests),
			AvgDailyTokens:        float64(summary.TotalTokens),
			ApproxTasks:           summary.TotalRequests,
			ApproxTaskFailures:    summary.FailureCount,
			ApproxTaskSuccessRate: successRate(summary.SuccessCount, summary.TotalRequests),
			ZeroTokenModels:       []string{},
		}
	}
	if req.Include.RecentFailures > 0 {
		failures, err := recentFailures(usageStore, query, req.Include.RecentFailures)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failures failed"})
			return
		}
		resp.RecentFailures = failures
	}
	if req.Include.EventsPage.Limit > 0 {
		pageQuery := query
		pageQuery.Limit = req.Include.EventsPage.Limit
		pageQuery.BeforeMS = req.Include.EventsPage.BeforeMS
		pageQuery.BeforeID = req.Include.EventsPage.BeforeID
		events, hasMore, err := usageStore.ListUsageEvents(pageQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics events failed"})
			return
		}
		var nextMS, nextID int64
		if len(events) > 0 {
			last := events[len(events)-1]
			nextMS = last.TimestampMS
			nextID = last.ID
		}
		resp.Events = &analyticsEventsResponse{Items: events, NextBeforeMS: nextMS, NextBeforeID: nextID, HasMore: hasMore}
	}
	c.JSON(http.StatusOK, resp)
}

func toDashboardTodaySummary(summary store.UsagePayload) dashboardTodaySummary {
	return dashboardTodaySummary{
		TotalCalls:   summary.TotalRequests,
		SuccessCalls: summary.SuccessCount,
		FailureCalls: summary.FailureCount,
		SuccessRate:  successRate(summary.SuccessCount, summary.TotalRequests),
		TotalTokens:  summary.TotalTokens,
	}
}

func recentFailures(usageStore UsageStore, query store.UsageQuery, limit int) ([]recentFailure, error) {
	if limit <= 0 {
		limit = 10
	}
	query.Limit = 500
	events, _, err := usageStore.ListUsageEvents(query)
	if err != nil {
		return nil, err
	}
	out := []recentFailure{}
	for _, event := range events {
		if !event.Failed {
			continue
		}
		latency := event.LatencyMS
		status := event.FailStatusCode
		out = append(out, recentFailure{
			TimestampMS:    event.TimestampMS,
			Model:          event.Model,
			APIKeyHash:     event.APIKeyHash,
			Source:         event.Source,
			SourceHash:     event.SourceHash,
			AuthIndex:      event.AuthIndex,
			Endpoint:       event.Endpoint,
			DurationMS:     &latency,
			FailStatusCode: &status,
			FailSummary:    event.FailSummary,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func successRate(success, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total)
}

func queryInt64(c *gin.Context, name string, fallback int64) int64 {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryInt(c *gin.Context, name string, fallback int) int {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseUsageEventLine(raw []byte) (store.UsageEvent, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return store.UsageEvent{}, err
	}
	event := store.UsageEvent{
		EventHash:             readString(data, "event_hash", "eventHash"),
		TimestampMS:           readInt64(data, "timestamp_ms", "timestampMs"),
		Model:                 readString(data, "model"),
		Endpoint:              readString(data, "endpoint"),
		Method:                readString(data, "method"),
		Path:                  readString(data, "path"),
		AuthIndex:             readString(data, "auth_index", "authIndex"),
		Source:                readString(data, "source"),
		SourceHash:            readString(data, "source_hash", "sourceHash"),
		APIKeyHash:            readString(data, "api_key_hash", "apiKeyHash"),
		AccountSnapshot:       readString(data, "account_snapshot", "accountSnapshot"),
		AuthLabelSnapshot:     readString(data, "auth_label_snapshot", "authLabelSnapshot"),
		AuthProviderSnapshot:  readString(data, "auth_provider_snapshot", "authProviderSnapshot"),
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
		TotalTokens:           readInt64(data, "total_tokens", "totalTokens", "tokens"),
		LatencyMS:             readInt64(data, "latency_ms", "latencyMs", "duration_ms", "durationMs"),
		TTFTMS:                readInt64(data, "ttft_ms", "ttftMs"),
		Failed:                readBool(data, "failed"),
		FailStatusCode:        int(readInt64(data, "fail_status_code", "failStatusCode")),
		FailSummary:           readString(data, "fail_summary", "failSummary"),
		RawJSON:               append([]byte(nil), raw...),
	}
	if event.EventHash == "" {
		hash := sha256.Sum256(raw)
		event.EventHash = hex.EncodeToString(hash[:])
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
