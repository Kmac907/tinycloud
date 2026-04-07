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
	if summary.ProviderCount != 3 {
		t.Fatalf("Summary().ProviderCount = %d, want %d", summary.ProviderCount, 3)
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
	if summary.TenantCount != 1 || summary.SubscriptionCount != 1 || summary.ProviderCount != 3 {
		t.Fatalf("bootstrap counts = (%d, %d, %d), want (1, 1, 3)", summary.TenantCount, summary.SubscriptionCount, summary.ProviderCount)
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
	if summary.TenantCount != 1 || summary.SubscriptionCount != 1 || summary.ProviderCount != 3 {
		t.Fatalf("bootstrap counts = (%d, %d, %d), want (1, 1, 3)", summary.TenantCount, summary.SubscriptionCount, summary.ProviderCount)
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
	if len(providers) != 3 {
		t.Fatalf("len(providers) = %d, want %d", len(providers), 3)
	}
}

func TestResourceGroupCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	resourceGroup, err := store.UpsertResourceGroup("sub-123", "rg-test", "westus2", "owner", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}
	if resourceGroup.Type != "Microsoft.Resources/resourceGroups" {
		t.Fatalf("Type = %q, want %q", resourceGroup.Type, "Microsoft.Resources/resourceGroups")
	}

	list, err := store.ListResourceGroups("sub-123")
	if err != nil {
		t.Fatalf("ListResourceGroups() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want %d", len(list), 1)
	}

	got, err := store.GetResourceGroup("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("GetResourceGroup() error = %v", err)
	}
	if got.ManagedBy != "owner" {
		t.Fatalf("ManagedBy = %q, want %q", got.ManagedBy, "owner")
	}

	if err := store.DeleteResourceGroup("sub-123", "rg-test"); err != nil {
		t.Fatalf("DeleteResourceGroup() error = %v", err)
	}
	list, err = store.ListResourceGroups("sub-123")
	if err != nil {
		t.Fatalf("ListResourceGroups() error = %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("len(list) = %d, want %d", len(list), 0)
	}
}

func TestOperationsRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	operation, err := store.CreateOperation("sub-123", "/subscriptions/sub-123/resourceGroups/rg-test", "Microsoft.Resources/resourceGroups/write", "Succeeded")
	if err != nil {
		t.Fatalf("CreateOperation() error = %v", err)
	}

	got, err := store.GetOperation("sub-123", operation.ID)
	if err != nil {
		t.Fatalf("GetOperation() error = %v", err)
	}
	if got.Status != "Succeeded" {
		t.Fatalf("Status = %q, want %q", got.Status, "Succeeded")
	}
}

func TestRegisterProviderPersists(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	provider, err := store.RegisterProvider("Microsoft.Custom")
	if err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}
	if provider.RegistrationState != "Registered" {
		t.Fatalf("RegistrationState = %q, want %q", provider.RegistrationState, "Registered")
	}

	got, err := store.GetProvider("Microsoft.Custom")
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
	if got.Namespace != "Microsoft.Custom" {
		t.Fatalf("Namespace = %q, want %q", got.Namespace, "Microsoft.Custom")
	}
}

func TestBlobContainerAndBlobRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	container, created, err := store.CreateBlobContainer("devstoreaccount1", "images")
	if err != nil {
		t.Fatalf("CreateBlobContainer() error = %v", err)
	}
	if !created {
		t.Fatal("CreateBlobContainer() created = false, want true")
	}
	if container.Name != "images" {
		t.Fatalf("Name = %q, want %q", container.Name, "images")
	}

	blob, err := store.PutBlob("devstoreaccount1", "images", "logo.txt", "text/plain", []byte("tinycloud"))
	if err != nil {
		t.Fatalf("PutBlob() error = %v", err)
	}
	if blob.ETag == "" {
		t.Fatal("ETag is empty")
	}

	got, err := store.GetBlob("devstoreaccount1", "images", "logo.txt")
	if err != nil {
		t.Fatalf("GetBlob() error = %v", err)
	}
	if string(got.Body) != "tinycloud" {
		t.Fatalf("Body = %q, want %q", string(got.Body), "tinycloud")
	}

	containers, err := store.ListBlobContainers("devstoreaccount1")
	if err != nil {
		t.Fatalf("ListBlobContainers() error = %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("len(containers) = %d, want %d", len(containers), 1)
	}

	blobs, err := store.ListBlobs("devstoreaccount1", "images")
	if err != nil {
		t.Fatalf("ListBlobs() error = %v", err)
	}
	if len(blobs) != 1 {
		t.Fatalf("len(blobs) = %d, want %d", len(blobs), 1)
	}

	if err := store.DeleteBlob("devstoreaccount1", "images", "logo.txt"); err != nil {
		t.Fatalf("DeleteBlob() error = %v", err)
	}
	blobs, err = store.ListBlobs("devstoreaccount1", "images")
	if err != nil {
		t.Fatalf("ListBlobs() after delete error = %v", err)
	}
	if len(blobs) != 0 {
		t.Fatalf("len(blobs) after delete = %d, want %d", len(blobs), 0)
	}
}
