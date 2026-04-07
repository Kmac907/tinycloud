package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesSQLiteDatabase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	wantPath := filepath.Join(root, "state.db")
	if summary.StatePath != wantPath {
		t.Fatalf("Summary().StatePath = %q, want %q", summary.StatePath, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("Stat(%q) error = %v", wantPath, err)
	}
	if summary.TenantCount != 1 {
		t.Fatalf("Summary().TenantCount = %d, want %d", summary.TenantCount, 1)
	}
	if summary.SubscriptionCount != 1 {
		t.Fatalf("Summary().SubscriptionCount = %d, want %d", summary.SubscriptionCount, 1)
	}
	if summary.ProviderCount != 1 {
		t.Fatalf("Summary().ProviderCount = %d, want %d", summary.ProviderCount, 1)
	}
}

func TestSnapshotCreatesParentDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	path := filepath.Join(root, "nested", "tinycloud.snapshot.json")
	if err := store.Snapshot(path); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}

func TestRestoreLoadsSnapshotIntoSQLite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "seed.json")
	body, err := json.Marshal(Document{
		Version: "foundation-v1",
		Resources: map[string]ResourceGroup{
			"/subscriptions/sub-123/resourceGroups/rg-test": {
				ID:       "/subscriptions/sub-123/resourceGroups/rg-test",
				Name:     "rg-test",
				Location: "westus2",
				Tags: map[string]string{
					"env": "test",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(snapshotPath, body, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := store.Restore(snapshotPath); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.ResourceCount != 1 {
		t.Fatalf("Summary().ResourceCount = %d, want %d", summary.ResourceCount, 1)
	}
	if summary.TenantCount != 1 || summary.SubscriptionCount != 1 || summary.ProviderCount != 1 {
		t.Fatalf("bootstrap counts = (%d, %d, %d), want (1, 1, 1)", summary.TenantCount, summary.SubscriptionCount, summary.ProviderCount)
	}

	exportPath := filepath.Join(root, "export.json")
	if err := store.Snapshot(exportPath); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	exported, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var doc Document
	if err := json.Unmarshal(exported, &doc); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := doc.Resources["/subscriptions/sub-123/resourceGroups/rg-test"]; !ok {
		t.Fatal("exported snapshot is missing restored resource group")
	}
}

func TestInitIsIdempotentForBootstrapRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if err := store.Init(); err != nil {
		t.Fatalf("Init() first error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() second error = %v", err)
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.TenantCount != 1 || summary.SubscriptionCount != 1 || summary.ProviderCount != 1 {
		t.Fatalf("bootstrap counts = (%d, %d, %d), want (1, 1, 1)", summary.TenantCount, summary.SubscriptionCount, summary.ProviderCount)
	}
}

func TestListBootstrapEntities(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	subscriptions, err := store.ListSubscriptions()
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("len(subscriptions) = %d, want %d", len(subscriptions), 1)
	}

	providers, err := store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("len(providers) = %d, want %d", len(providers), 1)
	}
}
