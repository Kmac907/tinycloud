package queue

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
	cfg := config.FromEnv()
	NewHandler(store, cfg).Register(mux)

	createReq := httptest.NewRequest(http.MethodPut, "/storagetest/jobs?restype=queue", nil)
	createReq.Header.Set("x-ms-version", "2024-01-01")
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusCreated)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/storagetest?comp=list", nil)
	listReq.Header.Set("x-ms-version", "2024-01-01")
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if !strings.Contains(listRec.Body.String(), "<Name>jobs</Name>") {
		t.Fatalf("list body = %q, want queue name", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), cfg.QueueURL()+"/storagetest") {
		t.Fatalf("list body = %q, want service endpoint", listRec.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPost, "/storagetest/jobs/messages", strings.NewReader(`<QueueMessage><MessageText>work-item-1</MessageText></QueueMessage>`))
	putReq.Header.Set("x-ms-version", "2024-01-01")
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("put status = %d, want %d", putRec.Code, http.StatusCreated)
	}
	if !strings.Contains(putRec.Body.String(), "<MessageId>") {
		t.Fatalf("put body = %q, want message id", putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/storagetest/jobs/messages?numofmessages=1&visibilitytimeout=30", nil)
	getReq.Header.Set("x-ms-version", "2024-01-01")
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if !strings.Contains(getRec.Body.String(), "<MessageText>work-item-1</MessageText>") {
		t.Fatalf("get body = %q, want message text", getRec.Body.String())
	}

	messageID := extractXML(getRec.Body.String(), "MessageId")
	popReceipt := extractXML(getRec.Body.String(), "PopReceipt")
	if messageID == "" || popReceipt == "" {
		t.Fatalf("get body = %q, want message id and pop receipt", getRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/storagetest/jobs/messages/"+messageID+"?popreceipt="+popReceipt, nil)
	deleteReq.Header.Set("x-ms-version", "2024-01-01")
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusNoContent)
	}
}

func TestCreateQueueRequiresExistingStorageAccount(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPut, "/missing/jobs?restype=queue", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func extractXML(body, name string) string {
	start := "<" + name + ">"
	end := "</" + name + ">"
	startIndex := strings.Index(body, start)
	endIndex := strings.Index(body, end)
	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		return ""
	}
	return body[startIndex+len(start) : endIndex]
}
