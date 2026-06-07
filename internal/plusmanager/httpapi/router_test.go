package httpapi

import (
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
