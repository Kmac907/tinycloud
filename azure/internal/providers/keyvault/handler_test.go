package keyvault

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
	if _, err := store.UpsertKeyVault("sub-123", "rg-test", "vaulttest", "westus2", "tenant-123", "standard", nil); err != nil {
		t.Fatalf("UpsertKeyVault() error = %v", err)
	}

	mux := http.NewServeMux()
	cfg := config.FromEnv()
	NewHandler(store, cfg).Register(mux)

	putReq := httptest.NewRequest(http.MethodPut, "/vaulttest/secrets/app-secret?api-version=7.4", strings.NewReader(`{"value":"super-secret-value","contentType":"text/plain"}`))
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d", putRec.Code, http.StatusOK)
	}
	if !strings.Contains(putRec.Body.String(), `"value":"super-secret-value"`) {
		t.Fatalf("put body = %q, want secret value", putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/vaulttest/secrets/app-secret?api-version=7.4", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if !strings.Contains(getRec.Body.String(), cfg.KeyVaultURL()+"/vaulttest/secrets/app-secret") {
		t.Fatalf("get body = %q, want secret id", getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/vaulttest/secrets?api-version=7.4", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if strings.Contains(listRec.Body.String(), `"value":"super-secret-value"`) {
		t.Fatalf("list body = %q, want no secret values", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"id":"`+cfg.KeyVaultURL()+`/vaulttest/secrets/app-secret"`) {
		t.Fatalf("list body = %q, want secret id", listRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/vaulttest/secrets/app-secret?api-version=7.4", nil)
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusOK)
	}

	getMissingReq := httptest.NewRequest(http.MethodGet, "/vaulttest/secrets/app-secret?api-version=7.4", nil)
	getMissingRec := httptest.NewRecorder()
	mux.ServeHTTP(getMissingRec, getMissingReq)
	if getMissingRec.Code != http.StatusNotFound {
		t.Fatalf("get missing status = %d, want %d", getMissingRec.Code, http.StatusNotFound)
	}
}

func TestPutSecretRequiresExistingVault(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPut, "/missing/secrets/app-secret?api-version=7.4", strings.NewReader(`{"value":"super-secret-value"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
