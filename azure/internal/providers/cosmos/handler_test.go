package cosmos

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

	createAccountReq := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(`{"name":"local-cosmos"}`))
	createAccountRec := httptest.NewRecorder()
	mux.ServeHTTP(createAccountRec, createAccountReq)
	if createAccountRec.Code != http.StatusCreated {
		t.Fatalf("create account status = %d, want %d", createAccountRec.Code, http.StatusCreated)
	}
	if !strings.Contains(createAccountRec.Body.String(), cfg.CosmosURL()+"/accounts/local-cosmos") {
		t.Fatalf("create account body = %q, want account id", createAccountRec.Body.String())
	}

	createDatabaseReq := httptest.NewRequest(http.MethodPost, "/accounts/local-cosmos/dbs", strings.NewReader(`{"id":"appdb"}`))
	createDatabaseRec := httptest.NewRecorder()
	mux.ServeHTTP(createDatabaseRec, createDatabaseReq)
	if createDatabaseRec.Code != http.StatusCreated {
		t.Fatalf("create database status = %d, want %d", createDatabaseRec.Code, http.StatusCreated)
	}

	createContainerReq := httptest.NewRequest(http.MethodPost, "/accounts/local-cosmos/dbs/appdb/colls", strings.NewReader(`{"id":"customers","partitionKeyPath":"/tenantId"}`))
	createContainerRec := httptest.NewRecorder()
	mux.ServeHTTP(createContainerRec, createContainerReq)
	if createContainerRec.Code != http.StatusCreated {
		t.Fatalf("create container status = %d, want %d", createContainerRec.Code, http.StatusCreated)
	}

	upsertDocumentReq := httptest.NewRequest(http.MethodPost, "/accounts/local-cosmos/dbs/appdb/colls/customers/docs", strings.NewReader(`{"id":"cust-001","partitionKey":"tenant-a","tenantId":"tenant-a","name":"Tiny Cloud"}`))
	upsertDocumentRec := httptest.NewRecorder()
	mux.ServeHTTP(upsertDocumentRec, upsertDocumentReq)
	if upsertDocumentRec.Code != http.StatusCreated {
		t.Fatalf("upsert document status = %d, want %d", upsertDocumentRec.Code, http.StatusCreated)
	}

	listDocumentsReq := httptest.NewRequest(http.MethodGet, "/accounts/local-cosmos/dbs/appdb/colls/customers/docs", nil)
	listDocumentsRec := httptest.NewRecorder()
	mux.ServeHTTP(listDocumentsRec, listDocumentsReq)
	if listDocumentsRec.Code != http.StatusOK {
		t.Fatalf("list documents status = %d, want %d", listDocumentsRec.Code, http.StatusOK)
	}
	if !strings.Contains(listDocumentsRec.Body.String(), `"name":"Tiny Cloud"`) {
		t.Fatalf("list documents body = %q, want document", listDocumentsRec.Body.String())
	}

	getDocumentReq := httptest.NewRequest(http.MethodGet, "/accounts/local-cosmos/dbs/appdb/colls/customers/docs/cust-001", nil)
	getDocumentRec := httptest.NewRecorder()
	mux.ServeHTTP(getDocumentRec, getDocumentReq)
	if getDocumentRec.Code != http.StatusOK {
		t.Fatalf("get document status = %d, want %d", getDocumentRec.Code, http.StatusOK)
	}

	deleteDocumentReq := httptest.NewRequest(http.MethodDelete, "/accounts/local-cosmos/dbs/appdb/colls/customers/docs/cust-001", nil)
	deleteDocumentRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteDocumentRec, deleteDocumentReq)
	if deleteDocumentRec.Code != http.StatusNoContent {
		t.Fatalf("delete document status = %d, want %d", deleteDocumentRec.Code, http.StatusNoContent)
	}
}

func TestCreateDatabaseRequiresExistingAccount(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/accounts/missing/dbs", strings.NewReader(`{"id":"appdb"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
