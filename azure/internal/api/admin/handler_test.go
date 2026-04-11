package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

func TestResolveDataPathUsesDataRootByDefault(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	handler := NewHandler(store, config.Config{DataRoot: root})
	path, err := handler.resolveDataPath("", "tinycloud.snapshot.json")
	if err != nil {
		t.Fatalf("resolveDataPath() error = %v", err)
	}

	want := filepath.Join(root, "tinycloud.snapshot.json")
	if path != want {
		t.Fatalf("resolveDataPath() = %q, want %q", path, want)
	}
}

func TestResolveDataPathRejectsTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	handler := NewHandler(store, config.Config{DataRoot: root})
	if _, err := handler.resolveDataPath(filepath.Join("..", "escape.json"), ""); err == nil {
		t.Fatal("resolveDataPath() error = nil, want rejection")
	}
}

func TestRuntimeReportsServiceSelection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	cfg := config.Config{
		DataRoot:       root,
		AdvertiseHost:  "127.0.0.1",
		ManagementHTTP: "4566",
		ManagementTLS:  "4567",
		Blob:           "4577",
		Queue:          "4578",
		Table:          "4579",
		KeyVault:       "4580",
		ServiceBus:     "4581",
		AppConfig:      "4582",
		Cosmos:         "4583",
		DNS:            "4584",
		EventHubs:      "4585",
		Services:       config.ParseServiceSelection("management,storage"),
	}

	handler := NewHandler(store, cfg)
	req := httptest.NewRequest(http.MethodGet, "/_admin/runtime", nil)
	rec := httptest.NewRecorder()

	handler.runtime(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("runtime status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		EnabledServices []string `json:"enabledServices"`
		Services        []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(body.EnabledServices) != 4 {
		t.Fatalf("enabledServices len = %d, want %d", len(body.EnabledServices), 4)
	}

	states := map[string]bool{}
	for _, service := range body.Services {
		states[service.Name] = service.Enabled
	}
	if !states["management"] || !states["blob"] || !states["queue"] || !states["table"] {
		t.Fatalf("storage selection missing expected enabled services: %#v", states)
	}
	if states["keyVault"] || states["serviceBus"] || states["dns"] {
		t.Fatalf("runtime reported unexpected enabled services: %#v", states)
	}
}
