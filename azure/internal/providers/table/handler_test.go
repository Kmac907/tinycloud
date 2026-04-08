package table

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
	if _, err := store.UpsertStorageAccount("sub-123", "rg-test", "storagetest", "westus2", "StorageV2", "Standard_LRS", nil); err != nil {
		t.Fatalf("UpsertStorageAccount() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	createReq := httptest.NewRequest(http.MethodPost, "/storagetest/Tables", strings.NewReader(`{"TableName":"customers"}`))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusCreated)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/storagetest/Tables", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if !strings.Contains(listRec.Body.String(), `"TableName":"customers"`) {
		t.Fatalf("list body = %q, want table name", listRec.Body.String())
	}

	upsertReq := httptest.NewRequest(http.MethodPost, "/storagetest/customers", strings.NewReader(`{"PartitionKey":"retail","RowKey":"cust-001","Name":"Tiny Cloud","Active":true}`))
	upsertRec := httptest.NewRecorder()
	mux.ServeHTTP(upsertRec, upsertReq)
	if upsertRec.Code != http.StatusCreated {
		t.Fatalf("upsert status = %d, want %d", upsertRec.Code, http.StatusCreated)
	}

	listEntitiesReq := httptest.NewRequest(http.MethodGet, "/storagetest/customers", nil)
	listEntitiesRec := httptest.NewRecorder()
	mux.ServeHTTP(listEntitiesRec, listEntitiesReq)
	if listEntitiesRec.Code != http.StatusOK {
		t.Fatalf("list entities status = %d, want %d", listEntitiesRec.Code, http.StatusOK)
	}
	if !strings.Contains(listEntitiesRec.Body.String(), `"Name":"Tiny Cloud"`) {
		t.Fatalf("list entities body = %q, want entity property", listEntitiesRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/storagetest/customers/retail/cust-001", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if !strings.Contains(getRec.Body.String(), `"Active":true`) {
		t.Fatalf("get body = %q, want entity property", getRec.Body.String())
	}

	deleteEntityReq := httptest.NewRequest(http.MethodDelete, "/storagetest/customers/retail/cust-001", nil)
	deleteEntityRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteEntityRec, deleteEntityReq)
	if deleteEntityRec.Code != http.StatusNoContent {
		t.Fatalf("delete entity status = %d, want %d", deleteEntityRec.Code, http.StatusNoContent)
	}

	deleteTableReq := httptest.NewRequest(http.MethodDelete, "/storagetest/Tables/customers", nil)
	deleteTableRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteTableRec, deleteTableReq)
	if deleteTableRec.Code != http.StatusNoContent {
		t.Fatalf("delete table status = %d, want %d", deleteTableRec.Code, http.StatusNoContent)
	}
}

func TestCreateTableRequiresExistingStorageAccount(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/missing/Tables", strings.NewReader(`{"TableName":"customers"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
