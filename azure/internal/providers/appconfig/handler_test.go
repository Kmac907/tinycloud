package appconfig

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

func TestHandlerRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	cfg := config.FromEnv()
	NewHandler(store, cfg).Register(mux)

	createStoreReq := httptest.NewRequest(http.MethodPost, "/stores", strings.NewReader(`{"name":"tiny-settings"}`))
	createStoreRec := httptest.NewRecorder()
	mux.ServeHTTP(createStoreRec, createStoreReq)
	if createStoreRec.Code != http.StatusCreated {
		t.Fatalf("create store status = %d, want %d", createStoreRec.Code, http.StatusCreated)
	}
	if !strings.Contains(createStoreRec.Body.String(), cfg.AppConfigURL()+"/stores/tiny-settings") {
		t.Fatalf("create store body = %q, want store id", createStoreRec.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/stores/tiny-settings/kv/FeatureX:Enabled?label=prod", strings.NewReader(`{"value":"true","contentType":"text/plain"}`))
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("put status = %d, want %d", putRec.Code, http.StatusCreated)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/stores/tiny-settings/kv", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if !strings.Contains(listRec.Body.String(), `"key":"FeatureX:Enabled"`) {
		t.Fatalf("list body = %q, want config key", listRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/stores/tiny-settings/kv/FeatureX:Enabled?label=prod", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if !strings.Contains(getRec.Body.String(), `"value":"true"`) {
		t.Fatalf("get body = %q, want config value", getRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/stores/tiny-settings/kv/FeatureX:Enabled?label=prod", nil)
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusNoContent)
	}
}

func TestPutRequiresExistingStore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/stores/missing/kv/FeatureX:Enabled", strings.NewReader(`{"value":"true"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
