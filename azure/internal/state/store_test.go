package state

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	tenants, err := store.ListTenants()
	if err != nil {
		t.Fatalf("ListTenants() error = %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("len(tenants) = %d, want %d", len(tenants), 1)
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

func TestSnapshotAndRestorePreserveBlobState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateBlobContainer("devstoreaccount1", "images"); err != nil {
		t.Fatalf("CreateBlobContainer() error = %v", err)
	}
	if _, err := store.PutBlob("devstoreaccount1", "images", "logo.txt", "text/plain", []byte("tinycloud")); err != nil {
		t.Fatalf("PutBlob() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "blob.snapshot.json")
	if err := store.Snapshot(snapshotPath); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	restoreRoot := t.TempDir()
	restoreStore, err := NewStore(restoreRoot)
	if err != nil {
		t.Fatalf("NewStore() restore error = %v", err)
	}
	if err := restoreStore.Restore(snapshotPath); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	blob, err := restoreStore.GetBlob("devstoreaccount1", "images", "logo.txt")
	if err != nil {
		t.Fatalf("GetBlob() error = %v", err)
	}
	if string(blob.Body) != "tinycloud" {
		t.Fatalf("Body = %q, want %q", string(blob.Body), "tinycloud")
	}
}

func TestStorageAccountCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	account, err := store.UpsertStorageAccount("sub-123", "rg-test", "storagetest", "westus2", "StorageV2", "Standard_LRS", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("UpsertStorageAccount() error = %v", err)
	}
	if account.Name != "storagetest" {
		t.Fatalf("Name = %q, want %q", account.Name, "storagetest")
	}

	got, err := store.GetStorageAccount("sub-123", "rg-test", "storagetest")
	if err != nil {
		t.Fatalf("GetStorageAccount() error = %v", err)
	}
	if got.Location != "westus2" {
		t.Fatalf("Location = %q, want %q", got.Location, "westus2")
	}

	accounts, err := store.ListStorageAccounts("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListStorageAccounts() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("len(accounts) = %d, want %d", len(accounts), 1)
	}

	if err := store.DeleteStorageAccount("sub-123", "rg-test", "storagetest"); err != nil {
		t.Fatalf("DeleteStorageAccount() error = %v", err)
	}
	accounts, err = store.ListStorageAccounts("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListStorageAccounts() after delete error = %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("len(accounts) after delete = %d, want %d", len(accounts), 0)
	}
}

func TestQueueStorageRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertStorageAccount("sub-123", "rg-test", "storagetest", "westus2", "StorageV2", "Standard_LRS", nil); err != nil {
		t.Fatalf("UpsertStorageAccount() error = %v", err)
	}

	account, err := store.GetStorageAccountByName("storagetest")
	if err != nil {
		t.Fatalf("GetStorageAccountByName() error = %v", err)
	}
	if account.ResourceGroupName != "rg-test" {
		t.Fatalf("ResourceGroupName = %q, want %q", account.ResourceGroupName, "rg-test")
	}

	queue, created, err := store.CreateQueue("storagetest", "jobs")
	if err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}
	if !created {
		t.Fatal("CreateQueue() created = false, want true")
	}
	if queue.Name != "jobs" {
		t.Fatalf("Name = %q, want %q", queue.Name, "jobs")
	}

	queues, err := store.ListQueues("storagetest")
	if err != nil {
		t.Fatalf("ListQueues() error = %v", err)
	}
	if len(queues) != 1 {
		t.Fatalf("len(queues) = %d, want %d", len(queues), 1)
	}

	message, err := store.EnqueueMessage("storagetest", "jobs", "work-item-1")
	if err != nil {
		t.Fatalf("EnqueueMessage() error = %v", err)
	}
	if message.ID == "" {
		t.Fatal("message ID is empty")
	}

	messages, err := store.ReceiveMessages("storagetest", "jobs", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 1)
	}
	if messages[0].MessageText != "work-item-1" {
		t.Fatalf("MessageText = %q, want %q", messages[0].MessageText, "work-item-1")
	}
	if messages[0].DequeueCount != 1 {
		t.Fatalf("DequeueCount = %d, want %d", messages[0].DequeueCount, 1)
	}

	if err := store.DeleteMessage("storagetest", "jobs", messages[0].ID, messages[0].PopReceipt); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}

	messages, err = store.ReceiveMessages("storagetest", "jobs", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveMessages() after delete error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("len(messages) after delete = %d, want %d", len(messages), 0)
	}
}

func TestTableStorageRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertStorageAccount("sub-123", "rg-test", "storagetest", "westus2", "StorageV2", "Standard_LRS", nil); err != nil {
		t.Fatalf("UpsertStorageAccount() error = %v", err)
	}

	table, created, err := store.CreateTable("storagetest", "customers")
	if err != nil {
		t.Fatalf("CreateTable() error = %v", err)
	}
	if !created {
		t.Fatal("CreateTable() created = false, want true")
	}
	if table.Name != "customers" {
		t.Fatalf("Name = %q, want %q", table.Name, "customers")
	}

	tables, err := store.ListTables("storagetest")
	if err != nil {
		t.Fatalf("ListTables() error = %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("len(tables) = %d, want %d", len(tables), 1)
	}

	entity, err := store.UpsertTableEntity("storagetest", "customers", "retail", "cust-001", map[string]any{
		"Name":   "Tiny Cloud",
		"Active": true,
	})
	if err != nil {
		t.Fatalf("UpsertTableEntity() error = %v", err)
	}
	if entity.RowKey != "cust-001" {
		t.Fatalf("RowKey = %q, want %q", entity.RowKey, "cust-001")
	}

	got, err := store.GetTableEntity("storagetest", "customers", "retail", "cust-001")
	if err != nil {
		t.Fatalf("GetTableEntity() error = %v", err)
	}
	if got.Properties["Name"] != "Tiny Cloud" {
		t.Fatalf("Name = %v, want %q", got.Properties["Name"], "Tiny Cloud")
	}

	entities, err := store.ListTableEntities("storagetest", "customers")
	if err != nil {
		t.Fatalf("ListTableEntities() error = %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("len(entities) = %d, want %d", len(entities), 1)
	}

	if err := store.DeleteTableEntity("storagetest", "customers", "retail", "cust-001"); err != nil {
		t.Fatalf("DeleteTableEntity() error = %v", err)
	}
	entities, err = store.ListTableEntities("storagetest", "customers")
	if err != nil {
		t.Fatalf("ListTableEntities() after delete error = %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("len(entities) after delete = %d, want %d", len(entities), 0)
	}

	if err := store.DeleteTable("storagetest", "customers"); err != nil {
		t.Fatalf("DeleteTable() error = %v", err)
	}
	tables, err = store.ListTables("storagetest")
	if err != nil {
		t.Fatalf("ListTables() after delete error = %v", err)
	}
	if len(tables) != 0 {
		t.Fatalf("len(tables) after delete = %d, want %d", len(tables), 0)
	}
}

func TestSnapshotAndRestorePreserveQueueState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertStorageAccount("sub-123", "rg-test", "storagetest", "westus2", "StorageV2", "Standard_LRS", nil); err != nil {
		t.Fatalf("UpsertStorageAccount() error = %v", err)
	}
	if _, _, err := store.CreateQueue("storagetest", "jobs"); err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}
	if _, err := store.EnqueueMessage("storagetest", "jobs", "work-item-1"); err != nil {
		t.Fatalf("EnqueueMessage() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "queues.snapshot.json")
	if err := store.Snapshot(snapshotPath); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	restoreRoot := t.TempDir()
	restoreStore, err := NewStore(restoreRoot)
	if err != nil {
		t.Fatalf("NewStore() restore error = %v", err)
	}
	if err := restoreStore.Restore(snapshotPath); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	queues, err := restoreStore.ListQueues("storagetest")
	if err != nil {
		t.Fatalf("ListQueues() error = %v", err)
	}
	if len(queues) != 1 {
		t.Fatalf("len(queues) = %d, want %d", len(queues), 1)
	}

	messages, err := restoreStore.ReceiveMessages("storagetest", "jobs", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 1)
	}
	if messages[0].MessageText != "work-item-1" {
		t.Fatalf("MessageText = %q, want %q", messages[0].MessageText, "work-item-1")
	}
}

func TestSnapshotAndRestorePreserveTableState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertStorageAccount("sub-123", "rg-test", "storagetest", "westus2", "StorageV2", "Standard_LRS", nil); err != nil {
		t.Fatalf("UpsertStorageAccount() error = %v", err)
	}
	if _, _, err := store.CreateTable("storagetest", "customers"); err != nil {
		t.Fatalf("CreateTable() error = %v", err)
	}
	if _, err := store.UpsertTableEntity("storagetest", "customers", "retail", "cust-001", map[string]any{"Name": "Tiny Cloud"}); err != nil {
		t.Fatalf("UpsertTableEntity() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "tables.snapshot.json")
	if err := store.Snapshot(snapshotPath); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	restoreRoot := t.TempDir()
	restoreStore, err := NewStore(restoreRoot)
	if err != nil {
		t.Fatalf("NewStore() restore error = %v", err)
	}
	if err := restoreStore.Restore(snapshotPath); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	tables, err := restoreStore.ListTables("storagetest")
	if err != nil {
		t.Fatalf("ListTables() error = %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("len(tables) = %d, want %d", len(tables), 1)
	}

	entity, err := restoreStore.GetTableEntity("storagetest", "customers", "retail", "cust-001")
	if err != nil {
		t.Fatalf("GetTableEntity() error = %v", err)
	}
	if entity.Properties["Name"] != "Tiny Cloud" {
		t.Fatalf("Name = %v, want %q", entity.Properties["Name"], "Tiny Cloud")
	}
}

func TestKeyVaultCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	vault, err := store.UpsertKeyVault("sub-123", "rg-test", "vaulttest", "westus2", "tenant-123", "standard", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("UpsertKeyVault() error = %v", err)
	}
	if vault.Name != "vaulttest" {
		t.Fatalf("Name = %q, want %q", vault.Name, "vaulttest")
	}

	got, err := store.GetKeyVault("sub-123", "rg-test", "vaulttest")
	if err != nil {
		t.Fatalf("GetKeyVault() error = %v", err)
	}
	if got.TenantID != "tenant-123" {
		t.Fatalf("TenantID = %q, want %q", got.TenantID, "tenant-123")
	}

	gotByName, err := store.GetKeyVaultByName("vaulttest")
	if err != nil {
		t.Fatalf("GetKeyVaultByName() error = %v", err)
	}
	if gotByName.ResourceGroupName != "rg-test" {
		t.Fatalf("ResourceGroupName = %q, want %q", gotByName.ResourceGroupName, "rg-test")
	}

	vaults, err := store.ListKeyVaults("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListKeyVaults() error = %v", err)
	}
	if len(vaults) != 1 {
		t.Fatalf("len(vaults) = %d, want %d", len(vaults), 1)
	}

	if err := store.DeleteKeyVault("sub-123", "rg-test", "vaulttest"); err != nil {
		t.Fatalf("DeleteKeyVault() error = %v", err)
	}
	vaults, err = store.ListKeyVaults("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListKeyVaults() after delete error = %v", err)
	}
	if len(vaults) != 0 {
		t.Fatalf("len(vaults) after delete = %d, want %d", len(vaults), 0)
	}
}

func TestSnapshotAndRestorePreserveKeyVaultState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertKeyVault("sub-123", "rg-test", "vaulttest", "westus2", "tenant-123", "standard", map[string]string{"env": "test"}); err != nil {
		t.Fatalf("UpsertKeyVault() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "keyvault.snapshot.json")
	if err := store.Snapshot(snapshotPath); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	restoreRoot := t.TempDir()
	restoreStore, err := NewStore(restoreRoot)
	if err != nil {
		t.Fatalf("NewStore() restore error = %v", err)
	}
	if err := restoreStore.Restore(snapshotPath); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	vault, err := restoreStore.GetKeyVault("sub-123", "rg-test", "vaulttest")
	if err != nil {
		t.Fatalf("GetKeyVault() error = %v", err)
	}
	if vault.TenantID != "tenant-123" {
		t.Fatalf("TenantID = %q, want %q", vault.TenantID, "tenant-123")
	}
}

func TestKeyVaultSecretCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertKeyVault("sub-123", "rg-test", "vaulttest", "westus2", "tenant-123", "standard", nil); err != nil {
		t.Fatalf("UpsertKeyVault() error = %v", err)
	}

	secret, err := store.PutKeyVaultSecret("vaulttest", "app-secret", "super-secret-value", "text/plain")
	if err != nil {
		t.Fatalf("PutKeyVaultSecret() error = %v", err)
	}
	if secret.Name != "app-secret" {
		t.Fatalf("Name = %q, want %q", secret.Name, "app-secret")
	}

	got, err := store.GetKeyVaultSecret("vaulttest", "app-secret")
	if err != nil {
		t.Fatalf("GetKeyVaultSecret() error = %v", err)
	}
	if got.Value != "super-secret-value" {
		t.Fatalf("Value = %q, want %q", got.Value, "super-secret-value")
	}

	secrets, err := store.ListKeyVaultSecrets("vaulttest")
	if err != nil {
		t.Fatalf("ListKeyVaultSecrets() error = %v", err)
	}
	if len(secrets) != 1 {
		t.Fatalf("len(secrets) = %d, want %d", len(secrets), 1)
	}

	if err := store.DeleteKeyVaultSecret("vaulttest", "app-secret"); err != nil {
		t.Fatalf("DeleteKeyVaultSecret() error = %v", err)
	}
	if _, err := store.GetKeyVaultSecret("vaulttest", "app-secret"); err == nil {
		t.Fatal("GetKeyVaultSecret() after delete error = nil, want error")
	} else if err != sql.ErrNoRows {
		t.Fatalf("GetKeyVaultSecret() after delete error = %v, want %v", err, sql.ErrNoRows)
	}
}

func TestSnapshotAndRestorePreserveKeyVaultSecrets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertKeyVault("sub-123", "rg-test", "vaulttest", "westus2", "tenant-123", "standard", nil); err != nil {
		t.Fatalf("UpsertKeyVault() error = %v", err)
	}
	if _, err := store.PutKeyVaultSecret("vaulttest", "app-secret", "super-secret-value", "text/plain"); err != nil {
		t.Fatalf("PutKeyVaultSecret() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "keyvault-secrets.snapshot.json")
	if err := store.Snapshot(snapshotPath); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	restoreRoot := t.TempDir()
	restoreStore, err := NewStore(restoreRoot)
	if err != nil {
		t.Fatalf("NewStore() restore error = %v", err)
	}
	if err := restoreStore.Restore(snapshotPath); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	secret, err := restoreStore.GetKeyVaultSecret("vaulttest", "app-secret")
	if err != nil {
		t.Fatalf("GetKeyVaultSecret() error = %v", err)
	}
	if secret.Value != "super-secret-value" {
		t.Fatalf("Value = %q, want %q", secret.Value, "super-secret-value")
	}
}

func TestDeploymentCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	deployment, err := store.UpsertDeployment(
		"sub-123",
		"rg-test",
		"deploy-one",
		"westus2",
		"Incremental",
		`{"resources":[]}`,
		`{"name":{"value":"tiny"}}`,
		`{}`,
		"Failed",
		"DeploymentNotSupported",
		"ARM deployment execution is not implemented yet",
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("UpsertDeployment() error = %v", err)
	}
	if deployment.Name != "deploy-one" {
		t.Fatalf("Name = %q, want %q", deployment.Name, "deploy-one")
	}
	if deployment.ProvisioningState != "Failed" {
		t.Fatalf("ProvisioningState = %q, want %q", deployment.ProvisioningState, "Failed")
	}

	got, err := store.GetDeployment("sub-123", "rg-test", "deploy-one")
	if err != nil {
		t.Fatalf("GetDeployment() error = %v", err)
	}
	if got.ErrorCode != "DeploymentNotSupported" {
		t.Fatalf("ErrorCode = %q, want %q", got.ErrorCode, "DeploymentNotSupported")
	}

	deployments, err := store.ListDeployments("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListDeployments() error = %v", err)
	}
	if len(deployments) != 1 {
		t.Fatalf("len(deployments) = %d, want %d", len(deployments), 1)
	}
}
