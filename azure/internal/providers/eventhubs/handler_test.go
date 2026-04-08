package eventhubs

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

	createNamespaceReq := httptest.NewRequest(http.MethodPost, "/namespaces", strings.NewReader(`{"name":"local-streaming"}`))
	createNamespaceRec := httptest.NewRecorder()
	mux.ServeHTTP(createNamespaceRec, createNamespaceReq)
	if createNamespaceRec.Code != http.StatusCreated {
		t.Fatalf("create namespace status = %d, want %d", createNamespaceRec.Code, http.StatusCreated)
	}
	if !strings.Contains(createNamespaceRec.Body.String(), cfg.EventHubsURL()+"/namespaces/local-streaming") {
		t.Fatalf("create namespace body = %q, want namespace id", createNamespaceRec.Body.String())
	}

	createHubReq := httptest.NewRequest(http.MethodPost, "/namespaces/local-streaming/hubs", strings.NewReader(`{"name":"orders"}`))
	createHubRec := httptest.NewRecorder()
	mux.ServeHTTP(createHubRec, createHubReq)
	if createHubRec.Code != http.StatusCreated {
		t.Fatalf("create hub status = %d, want %d", createHubRec.Code, http.StatusCreated)
	}

	listHubsReq := httptest.NewRequest(http.MethodGet, "/namespaces/local-streaming/hubs", nil)
	listHubsRec := httptest.NewRecorder()
	mux.ServeHTTP(listHubsRec, listHubsReq)
	if listHubsRec.Code != http.StatusOK {
		t.Fatalf("list hubs status = %d, want %d", listHubsRec.Code, http.StatusOK)
	}
	if !strings.Contains(listHubsRec.Body.String(), `"name":"orders"`) {
		t.Fatalf("list hubs body = %q, want hub name", listHubsRec.Body.String())
	}

	for _, body := range []string{
		`{"body":"{\"event\":\"created\"}","partitionKey":"tenant-a"}`,
		`{"body":"{\"event\":\"updated\"}","partitionKey":"tenant-a"}`,
	} {
		publishReq := httptest.NewRequest(http.MethodPost, "/namespaces/local-streaming/hubs/orders/events", strings.NewReader(body))
		publishRec := httptest.NewRecorder()
		mux.ServeHTTP(publishRec, publishReq)
		if publishRec.Code != http.StatusCreated {
			t.Fatalf("publish status = %d, want %d, body = %q", publishRec.Code, http.StatusCreated, publishRec.Body.String())
		}
	}

	listEventsReq := httptest.NewRequest(http.MethodGet, "/namespaces/local-streaming/hubs/orders/events?fromSequenceNumber=2&maxEvents=10", nil)
	listEventsRec := httptest.NewRecorder()
	mux.ServeHTTP(listEventsRec, listEventsReq)
	if listEventsRec.Code != http.StatusOK {
		t.Fatalf("list events status = %d, want %d", listEventsRec.Code, http.StatusOK)
	}
	if !strings.Contains(listEventsRec.Body.String(), `"sequenceNumber":2`) {
		t.Fatalf("list events body = %q, want second event", listEventsRec.Body.String())
	}
	if !strings.Contains(listEventsRec.Body.String(), `"partitionKey":"tenant-a"`) {
		t.Fatalf("list events body = %q, want partition key", listEventsRec.Body.String())
	}
}

func TestCreateHubRequiresExistingNamespace(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/namespaces/missing/hubs", strings.NewReader(`{"name":"orders"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListEventsRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateEventHubNamespace("local-streaming"); err != nil {
		t.Fatalf("CreateEventHubNamespace() error = %v", err)
	}
	if _, _, err := store.CreateEventHub("local-streaming", "orders"); err != nil {
		t.Fatalf("CreateEventHub() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/namespaces/local-streaming/hubs/orders/events?fromSequenceNumber=-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
