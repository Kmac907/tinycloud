package storage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	NewHandler(store).Register(mux)

	createReq := httptest.NewRequest(http.MethodPut, "/devstoreaccount1/images?restype=container", nil)
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusCreated)
	}

	listContainersReq := httptest.NewRequest(http.MethodGet, "/devstoreaccount1?comp=list", nil)
	listContainersRec := httptest.NewRecorder()
	mux.ServeHTTP(listContainersRec, listContainersReq)
	if listContainersRec.Code != http.StatusOK {
		t.Fatalf("list containers status = %d, want %d", listContainersRec.Code, http.StatusOK)
	}
	if !strings.Contains(listContainersRec.Body.String(), "<Name>images</Name>") {
		t.Fatalf("list containers body = %q, want container name", listContainersRec.Body.String())
	}

	putBlobReq := httptest.NewRequest(http.MethodPut, "/devstoreaccount1/images/logo.txt", strings.NewReader("tinycloud"))
	putBlobReq.Header.Set("Content-Type", "text/plain")
	putBlobRec := httptest.NewRecorder()
	mux.ServeHTTP(putBlobRec, putBlobReq)
	if putBlobRec.Code != http.StatusCreated {
		t.Fatalf("put blob status = %d, want %d", putBlobRec.Code, http.StatusCreated)
	}

	listBlobsReq := httptest.NewRequest(http.MethodGet, "/devstoreaccount1/images?restype=container&comp=list", nil)
	listBlobsRec := httptest.NewRecorder()
	mux.ServeHTTP(listBlobsRec, listBlobsReq)
	if listBlobsRec.Code != http.StatusOK {
		t.Fatalf("list blobs status = %d, want %d", listBlobsRec.Code, http.StatusOK)
	}
	if !strings.Contains(listBlobsRec.Body.String(), "<Name>logo.txt</Name>") {
		t.Fatalf("list blobs body = %q, want blob name", listBlobsRec.Body.String())
	}

	getBlobReq := httptest.NewRequest(http.MethodGet, "/devstoreaccount1/images/logo.txt", nil)
	getBlobRec := httptest.NewRecorder()
	mux.ServeHTTP(getBlobRec, getBlobReq)
	if getBlobRec.Code != http.StatusOK {
		t.Fatalf("get blob status = %d, want %d", getBlobRec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(getBlobRec.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "tinycloud" {
		t.Fatalf("body = %q, want %q", string(body), "tinycloud")
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
	NewHandler(store).Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/devstoreaccount1/missing/logo.txt", strings.NewReader("tinycloud"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
