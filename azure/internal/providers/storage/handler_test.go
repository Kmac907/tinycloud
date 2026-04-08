package storage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

func TestBlobHandlerRoundTrip(t *testing.T) {
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

	createReq := httptest.NewRequest(http.MethodPut, "/devstoreaccount1/images?restype=container", nil)
	createReq.Header.Set("x-ms-version", "2024-01-01")
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusCreated)
	}
	if createRec.Header().Get("x-ms-version") != "2024-01-01" {
		t.Fatalf("x-ms-version = %q, want %q", createRec.Header().Get("x-ms-version"), "2024-01-01")
	}

	listContainersReq := httptest.NewRequest(http.MethodGet, "/devstoreaccount1?comp=list", nil)
	listContainersReq.Header.Set("x-ms-version", "2024-01-01")
	listContainersRec := httptest.NewRecorder()
	mux.ServeHTTP(listContainersRec, listContainersReq)
	if listContainersRec.Code != http.StatusOK {
		t.Fatalf("list containers status = %d, want %d", listContainersRec.Code, http.StatusOK)
	}
	if !strings.Contains(listContainersRec.Body.String(), "<Name>images</Name>") {
		t.Fatalf("list containers body = %q, want container name", listContainersRec.Body.String())
	}
	if !strings.Contains(listContainersRec.Body.String(), cfg.BlobURL()+"/devstoreaccount1") {
		t.Fatalf("list containers body = %q, want service endpoint", listContainersRec.Body.String())
	}

	putBlobReq := httptest.NewRequest(http.MethodPut, "/devstoreaccount1/images/logo.txt", strings.NewReader("tinycloud"))
	putBlobReq.Header.Set("Content-Type", "text/plain")
	putBlobReq.Header.Set("x-ms-version", "2024-01-01")
	putBlobRec := httptest.NewRecorder()
	mux.ServeHTTP(putBlobRec, putBlobReq)
	if putBlobRec.Code != http.StatusCreated {
		t.Fatalf("put blob status = %d, want %d", putBlobRec.Code, http.StatusCreated)
	}
	if putBlobRec.Header().Get("x-ms-blob-type") != "BlockBlob" {
		t.Fatalf("x-ms-blob-type = %q, want %q", putBlobRec.Header().Get("x-ms-blob-type"), "BlockBlob")
	}

	listBlobsReq := httptest.NewRequest(http.MethodGet, "/devstoreaccount1/images?restype=container&comp=list", nil)
	listBlobsReq.Header.Set("x-ms-version", "2024-01-01")
	listBlobsRec := httptest.NewRecorder()
	mux.ServeHTTP(listBlobsRec, listBlobsReq)
	if listBlobsRec.Code != http.StatusOK {
		t.Fatalf("list blobs status = %d, want %d", listBlobsRec.Code, http.StatusOK)
	}
	if !strings.Contains(listBlobsRec.Body.String(), "<Name>logo.txt</Name>") {
		t.Fatalf("list blobs body = %q, want blob name", listBlobsRec.Body.String())
	}

	getBlobReq := httptest.NewRequest(http.MethodGet, "/devstoreaccount1/images/logo.txt", nil)
	getBlobReq.Header.Set("x-ms-version", "2024-01-01")
	getBlobRec := httptest.NewRecorder()
	mux.ServeHTTP(getBlobRec, getBlobReq)
	if getBlobRec.Code != http.StatusOK {
		t.Fatalf("get blob status = %d, want %d", getBlobRec.Code, http.StatusOK)
	}
	if getBlobRec.Header().Get("Content-Length") != "9" {
		t.Fatalf("Content-Length = %q, want %q", getBlobRec.Header().Get("Content-Length"), "9")
	}
	body, err := io.ReadAll(getBlobRec.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "tinycloud" {
		t.Fatalf("body = %q, want %q", string(body), "tinycloud")
	}

	headBlobReq := httptest.NewRequest(http.MethodHead, "/devstoreaccount1/images/logo.txt", nil)
	headBlobReq.Header.Set("x-ms-version", "2024-01-01")
	headBlobRec := httptest.NewRecorder()
	mux.ServeHTTP(headBlobRec, headBlobReq)
	if headBlobRec.Code != http.StatusOK {
		t.Fatalf("head blob status = %d, want %d", headBlobRec.Code, http.StatusOK)
	}
	if headBlobRec.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want %d", headBlobRec.Body.Len(), 0)
	}
	if headBlobRec.Header().Get("x-ms-request-server-encrypted") != "true" {
		t.Fatalf("x-ms-request-server-encrypted = %q, want %q", headBlobRec.Header().Get("x-ms-request-server-encrypted"), "true")
	}
}

func TestPutBlobRequiresExistingContainer(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPut, "/devstoreaccount1/missing/logo.txt", strings.NewReader("tinycloud"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
