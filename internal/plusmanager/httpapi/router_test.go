package httpapi

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/store"
)

func TestRegisterRoutesStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true})

	req := httptest.NewRequest(http.MethodGet, "/v0/management/plus/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterRoutesInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true})

	req := httptest.NewRequest(http.MethodGet, "/v0/management/plus/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterCompatibilityRoutesInfoAndStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterCompatibilityRoutes(r, Options{Enabled: true})

	infoReq := httptest.NewRequest(http.MethodGet, "/usage-service/info", nil)
	infoW := httptest.NewRecorder()
	r.ServeHTTP(infoW, infoReq)
	if infoW.Code != http.StatusOK {
		t.Fatalf("info status code = %d, want 200; body=%s", infoW.Code, infoW.Body.String())
	}
	if !strings.Contains(infoW.Body.String(), "cpa-manager-plus") {
		t.Fatalf("info body missing service id: %s", infoW.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/status", nil)
	statusW := httptest.NewRecorder()
	r.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", statusW.Code, statusW.Body.String())
	}
}

func TestRegisterRoutesDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: false})

	req := httptest.NewRequest(http.MethodGet, "/v0/management/plus/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want 404", w.Code)
	}
}

func TestRegisterRoutesModelPricesGetPut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer s.Close()

	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true, Store: s})

	putBody := strings.NewReader(`[{"model":"gpt-test","inputPerMTok":1.25,"outputPerMTok":5.5}]`)
	putReq := httptest.NewRequest(http.MethodPut, "/v0/management/plus/model-prices", putBody)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusNoContent {
		t.Fatalf("PUT status code = %d, want 204; body=%s", putW.Code, putW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/model-prices", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET status code = %d, want 200; body=%s", getW.Code, getW.Body.String())
	}
	var got struct {
		Prices map[string]map[string]float64 `json:"prices"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &got); err != nil {
		t.Fatalf("GET response is not JSON model prices: %v; body=%s", err, getW.Body.String())
	}
	if len(got.Prices) != 1 || got.Prices["gpt-test"]["input"] != 1.25 || got.Prices["gpt-test"]["output"] != 5.5 {
		t.Fatalf("GET model prices = %#v", got)
	}
}

func TestRegisterRoutesAPIKeyAliasesGetPutDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer s.Close()

	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true, Store: s})

	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	putBody := strings.NewReader(`{"items":[{"apiKeyHash":"` + hash + `","alias":"Main key","updatedAtMs":123}]}`)
	putReq := httptest.NewRequest(http.MethodPut, "/v0/management/plus/api-key-aliases", putBody)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("PUT status code = %d, want 200; body=%s", putW.Code, putW.Body.String())
	}
	var putGot struct {
		Items []store.APIKeyAlias `json:"items"`
	}
	if err := json.Unmarshal(putW.Body.Bytes(), &putGot); err != nil {
		t.Fatalf("PUT response is not JSON aliases: %v; body=%s", err, putW.Body.String())
	}
	if len(putGot.Items) != 1 || putGot.Items[0].APIKeyHash != hash || putGot.Items[0].Alias != "Main key" || putGot.Items[0].UpdatedAtMS != 123 {
		t.Fatalf("PUT aliases response = %#v", putGot)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/api-key-aliases", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET status code = %d, want 200; body=%s", getW.Code, getW.Body.String())
	}
	var getGot struct {
		Items []store.APIKeyAlias `json:"items"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &getGot); err != nil {
		t.Fatalf("GET response is not JSON aliases: %v; body=%s", err, getW.Body.String())
	}
	if len(getGot.Items) != 1 || getGot.Items[0].Alias != "Main key" {
		t.Fatalf("GET aliases response = %#v", getGot)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v0/management/plus/api-key-aliases/"+hash, nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("DELETE status code = %d, want 204; body=%s", deleteW.Code, deleteW.Body.String())
	}

	getAfterDeleteReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/api-key-aliases", nil)
	getAfterDeleteW := httptest.NewRecorder()
	r.ServeHTTP(getAfterDeleteW, getAfterDeleteReq)
	if getAfterDeleteW.Code != http.StatusOK {
		t.Fatalf("GET after delete status code = %d, want 200; body=%s", getAfterDeleteW.Code, getAfterDeleteW.Body.String())
	}
	var afterDeleteGot struct {
		Items []store.APIKeyAlias `json:"items"`
	}
	if err := json.Unmarshal(getAfterDeleteW.Body.Bytes(), &afterDeleteGot); err != nil {
		t.Fatalf("GET after delete response is not JSON aliases: %v; body=%s", err, getAfterDeleteW.Body.String())
	}
	if len(afterDeleteGot.Items) != 0 {
		t.Fatalf("GET after delete aliases response = %#v, want empty", afterDeleteGot)
	}
}

func TestRegisterRoutesModelPricesObjectSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer s.Close()

	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true, Store: s})

	putBody := strings.NewReader(`{"prices":{"gpt-test":{"prompt":1.25,"completion":5.5}}}`)
	putReq := httptest.NewRequest(http.MethodPut, "/v0/management/plus/model-prices", putBody)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusNoContent {
		t.Fatalf("PUT status code = %d, want 204; body=%s", putW.Code, putW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/model-prices", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET status code = %d, want 200; body=%s", getW.Code, getW.Body.String())
	}
	var got struct {
		Prices map[string]map[string]float64 `json:"prices"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &got); err != nil {
		t.Fatalf("GET response is not JSON price map: %v; body=%s", err, getW.Body.String())
	}
	if got.Prices["gpt-test"]["prompt"] != 1.25 || got.Prices["gpt-test"]["completion"] != 5.5 {
		t.Fatalf("GET model price map = %#v", got)
	}
}

func TestRegisterRoutesModelPricesStoreNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true})

	req := httptest.NewRequest(http.MethodGet, "/v0/management/plus/model-prices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want 503; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterRoutesModelPricesPutStoreNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true})

	req := httptest.NewRequest(http.MethodPut, "/v0/management/plus/model-prices", strings.NewReader(`[]`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want 503; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterRoutesModelPricesPutInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer s.Close()

	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true, Store: s})

	req := httptest.NewRequest(http.MethodPut, "/v0/management/plus/model-prices", strings.NewReader(`{"model":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

type failingModelPriceStore struct{}

func (failingModelPriceStore) ListModelPrices() ([]store.ModelPrice, error) {
	return nil, errors.New("list failed")
}

func (failingModelPriceStore) ReplaceModelPrices([]store.ModelPrice) error {
	return errors.New("replace failed")
}

func TestRegisterRoutesModelPricesStoreErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true, Store: failingModelPriceStore{}})

	getReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/model-prices", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusInternalServerError {
		t.Fatalf("GET status code = %d, want 500; body=%s", getW.Code, getW.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/v0/management/plus/model-prices", strings.NewReader(`[]`))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusInternalServerError {
		t.Fatalf("PUT status code = %d, want 500; body=%s", putW.Code, putW.Body.String())
	}
}

func TestRegisterRoutesUsageImportExportUsageDashboardAndAnalytics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer s.Close()

	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true, Store: s})

	jsonl := strings.Join([]string{
		`{"event_hash":"evt-1","timestamp_ms":1700000000000,"model":"gpt-test","endpoint":"/v1/chat/completions","method":"POST","path":"/v1/chat/completions","auth_index":"auth-1","source":"cli","source_hash":"src-1","api_key_hash":"key-1","input_tokens":10,"output_tokens":20,"total_tokens":30,"latency_ms":101}`,
		`{"eventHash":"evt-2","timestampMs":1700000001000,"model":"gpt-test","endpoint":"/v1/responses","method":"POST","path":"/v1/responses","authIndex":"auth-2","source":"worker","sourceHash":"src-2","apiKeyHash":"key-2","tokens":5,"failed":true,"failStatusCode":500,"failSummary":"boom"}`,
		``,
	}, "\n")
	importReq := httptest.NewRequest(http.MethodPost, "/v0/management/plus/usage/import", strings.NewReader(jsonl))
	importW := httptest.NewRecorder()
	r.ServeHTTP(importW, importReq)
	if importW.Code != http.StatusOK {
		t.Fatalf("import status code = %d, want 200; body=%s", importW.Code, importW.Body.String())
	}
	var importResp struct {
		Added   int `json:"added"`
		Skipped int `json:"skipped"`
		Total   int `json:"total"`
		Failed  int `json:"failed"`
	}
	if err := json.Unmarshal(importW.Body.Bytes(), &importResp); err != nil {
		t.Fatalf("import response JSON error = %v; body=%s", err, importW.Body.String())
	}
	if importResp.Added != 2 || importResp.Skipped != 0 || importResp.Total != 2 || importResp.Failed != 0 {
		t.Fatalf("import response = %#v", importResp)
	}

	usageReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/usage", nil)
	usageW := httptest.NewRecorder()
	r.ServeHTTP(usageW, usageReq)
	if usageW.Code != http.StatusOK {
		t.Fatalf("usage status code = %d, want 200; body=%s", usageW.Code, usageW.Body.String())
	}
	var usageResp struct {
		TotalRequests int `json:"total_requests"`
		SuccessCount  int `json:"success_count"`
		FailureCount  int `json:"failure_count"`
		TotalTokens   int `json:"total_tokens"`
		APIs          []struct {
			Endpoint string `json:"endpoint"`
			Requests int    `json:"requests"`
		} `json:"apis"`
	}
	if err := json.Unmarshal(usageW.Body.Bytes(), &usageResp); err != nil {
		t.Fatalf("usage response JSON error = %v; body=%s", err, usageW.Body.String())
	}
	if usageResp.TotalRequests != 2 || usageResp.SuccessCount != 1 || usageResp.FailureCount != 1 || usageResp.TotalTokens != 35 || len(usageResp.APIs) != 2 {
		t.Fatalf("usage response = %#v", usageResp)
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/usage/export", nil)
	exportW := httptest.NewRecorder()
	r.ServeHTTP(exportW, exportReq)
	if exportW.Code != http.StatusOK {
		t.Fatalf("export status code = %d, want 200; body=%s", exportW.Code, exportW.Body.String())
	}
	if cd := exportW.Header().Get("Content-Disposition"); !strings.Contains(cd, "filename=") {
		t.Fatalf("export missing filename Content-Disposition: %q", cd)
	}
	scanner := bufio.NewScanner(strings.NewReader(exportW.Body.String()))
	exported := 0
	for scanner.Scan() {
		exported++
	}
	if exported != 2 {
		t.Fatalf("exported rows = %d, want 2; body=%s", exported, exportW.Body.String())
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/v0/management/plus/dashboard/summary?today_start_ms=1699999900000&now_ms=1700000100000", nil)
	dashboardW := httptest.NewRecorder()
	r.ServeHTTP(dashboardW, dashboardReq)
	if dashboardW.Code != http.StatusOK {
		t.Fatalf("dashboard status code = %d, want 200; body=%s", dashboardW.Code, dashboardW.Body.String())
	}
	var dashboardResp struct {
		GeneratedAtMS int64 `json:"generated_at_ms"`
		Today         struct {
			TotalCalls int `json:"total_calls"`
		} `json:"today"`
		Rolling30M struct {
			TotalCalls int `json:"total_calls"`
		} `json:"rolling_30m"`
		RecentFailure []any `json:"recent_failures"`
	}
	if err := json.Unmarshal(dashboardW.Body.Bytes(), &dashboardResp); err != nil {
		t.Fatalf("dashboard response JSON error = %v; body=%s", err, dashboardW.Body.String())
	}
	if dashboardResp.GeneratedAtMS == 0 || dashboardResp.Today.TotalCalls != 2 || dashboardResp.Rolling30M.TotalCalls != 2 || len(dashboardResp.RecentFailure) != 1 {
		t.Fatalf("dashboard response = %#v", dashboardResp)
	}

	analyticsReq := httptest.NewRequest(http.MethodPost, "/v0/management/plus/monitoring/analytics", strings.NewReader(`{"from_ms":1699999900000,"to_ms":1700000100000,"include":{"summary":true,"events_page":{"limit":10},"recent_failures":5}}`))
	analyticsReq.Header.Set("Content-Type", "application/json")
	analyticsW := httptest.NewRecorder()
	r.ServeHTTP(analyticsW, analyticsReq)
	if analyticsW.Code != http.StatusOK {
		t.Fatalf("analytics status code = %d, want 200; body=%s", analyticsW.Code, analyticsW.Body.String())
	}
	var analyticsResp struct {
		GeneratedAtMS int64  `json:"generated_at_ms"`
		Granularity   string `json:"granularity"`
		Summary       *struct {
			TotalCalls int `json:"total_calls"`
		} `json:"summary"`
		Events *struct {
			Items []struct {
				EventHash   string `json:"event_hash"`
				TotalTokens int    `json:"total_tokens"`
				Failed      bool   `json:"failed"`
			} `json:"items"`
			HasMore bool `json:"has_more"`
		} `json:"events"`
		RecentFailures []any `json:"recent_failures"`
	}
	if err := json.Unmarshal(analyticsW.Body.Bytes(), &analyticsResp); err != nil {
		t.Fatalf("analytics response JSON error = %v; body=%s", err, analyticsW.Body.String())
	}
	if analyticsResp.GeneratedAtMS == 0 || analyticsResp.Granularity == "" || analyticsResp.Summary == nil || analyticsResp.Summary.TotalCalls != 2 || analyticsResp.Events == nil || len(analyticsResp.Events.Items) != 2 || len(analyticsResp.RecentFailures) != 1 {
		t.Fatalf("analytics response = %#v", analyticsResp)
	}
}
