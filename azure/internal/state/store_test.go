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
	if summary.ProviderCount != 4 {
		t.Fatalf("Summary().ProviderCount = %d, want %d", summary.ProviderCount, 4)
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
	if summary.TenantCount != 1 || summary.SubscriptionCount != 1 || summary.ProviderCount != 4 {
		t.Fatalf("bootstrap counts = (%d, %d, %d), want (1, 1, 4)", summary.TenantCount, summary.SubscriptionCount, summary.ProviderCount)
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
	if summary.TenantCount != 1 || summary.SubscriptionCount != 1 || summary.ProviderCount != 4 {
		t.Fatalf("bootstrap counts = (%d, %d, %d), want (1, 1, 4)", summary.TenantCount, summary.SubscriptionCount, summary.ProviderCount)
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
	if len(providers) != 4 {
		t.Fatalf("len(providers) = %d, want %d", len(providers), 4)
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

func TestServiceBusQueueRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	namespace, created, err := store.CreateServiceBusNamespace("local-messaging")
	if err != nil {
		t.Fatalf("CreateServiceBusNamespace() error = %v", err)
	}
	if !created {
		t.Fatal("CreateServiceBusNamespace() created = false, want true")
	}
	if namespace.Name != "local-messaging" {
		t.Fatalf("Name = %q, want %q", namespace.Name, "local-messaging")
	}

	namespaces, err := store.ListServiceBusNamespaces()
	if err != nil {
		t.Fatalf("ListServiceBusNamespaces() error = %v", err)
	}
	if len(namespaces) != 1 {
		t.Fatalf("len(namespaces) = %d, want %d", len(namespaces), 1)
	}

	queue, created, err := store.CreateServiceBusQueue("local-messaging", "jobs")
	if err != nil {
		t.Fatalf("CreateServiceBusQueue() error = %v", err)
	}
	if !created {
		t.Fatal("CreateServiceBusQueue() created = false, want true")
	}
	if queue.Name != "jobs" {
		t.Fatalf("Name = %q, want %q", queue.Name, "jobs")
	}

	queues, err := store.ListServiceBusQueues("local-messaging")
	if err != nil {
		t.Fatalf("ListServiceBusQueues() error = %v", err)
	}
	if len(queues) != 1 {
		t.Fatalf("len(queues) = %d, want %d", len(queues), 1)
	}

	message, err := store.SendServiceBusMessage("local-messaging", "jobs", `{"job":"sync"}`)
	if err != nil {
		t.Fatalf("SendServiceBusMessage() error = %v", err)
	}
	if message.ID == "" {
		t.Fatal("message ID is empty")
	}

	messages, err := store.ReceiveServiceBusMessages("local-messaging", "jobs", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveServiceBusMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 1)
	}
	if messages[0].Body != `{"job":"sync"}` {
		t.Fatalf("Body = %q, want %q", messages[0].Body, `{"job":"sync"}`)
	}
	if messages[0].DeliveryCount != 1 {
		t.Fatalf("DeliveryCount = %d, want %d", messages[0].DeliveryCount, 1)
	}

	if err := store.DeleteServiceBusMessage("local-messaging", "jobs", messages[0].ID, messages[0].LockToken); err != nil {
		t.Fatalf("DeleteServiceBusMessage() error = %v", err)
	}

	messages, err = store.ReceiveServiceBusMessages("local-messaging", "jobs", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveServiceBusMessages() after delete error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("len(messages) after delete = %d, want %d", len(messages), 0)
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

func TestSnapshotAndRestorePreserveServiceBusState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusNamespace("local-messaging"); err != nil {
		t.Fatalf("CreateServiceBusNamespace() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusQueue("local-messaging", "jobs"); err != nil {
		t.Fatalf("CreateServiceBusQueue() error = %v", err)
	}
	if _, err := store.SendServiceBusMessage("local-messaging", "jobs", `{"job":"sync"}`); err != nil {
		t.Fatalf("SendServiceBusMessage() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "servicebus.snapshot.json")
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

	namespaces, err := restoreStore.ListServiceBusNamespaces()
	if err != nil {
		t.Fatalf("ListServiceBusNamespaces() error = %v", err)
	}
	if len(namespaces) != 1 {
		t.Fatalf("len(namespaces) = %d, want %d", len(namespaces), 1)
	}

	messages, err := restoreStore.ReceiveServiceBusMessages("local-messaging", "jobs", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveServiceBusMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 1)
	}
	if messages[0].Body != `{"job":"sync"}` {
		t.Fatalf("Body = %q, want %q", messages[0].Body, `{"job":"sync"}`)
	}
}

func TestServiceBusTopicRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusNamespace("local-messaging"); err != nil {
		t.Fatalf("CreateServiceBusNamespace() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusTopic("local-messaging", "events"); err != nil {
		t.Fatalf("CreateServiceBusTopic() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusSubscription("local-messaging", "events", "worker-a"); err != nil {
		t.Fatalf("CreateServiceBusSubscription() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusSubscription("local-messaging", "events", "worker-b"); err != nil {
		t.Fatalf("CreateServiceBusSubscription() error = %v", err)
	}

	topics, err := store.ListServiceBusTopics("local-messaging")
	if err != nil {
		t.Fatalf("ListServiceBusTopics() error = %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("len(topics) = %d, want %d", len(topics), 1)
	}

	subscriptions, err := store.ListServiceBusSubscriptions("local-messaging", "events")
	if err != nil {
		t.Fatalf("ListServiceBusSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 2 {
		t.Fatalf("len(subscriptions) = %d, want %d", len(subscriptions), 2)
	}

	if _, err := store.PublishServiceBusTopicMessage("local-messaging", "events", `{"event":"created"}`); err != nil {
		t.Fatalf("PublishServiceBusTopicMessage() error = %v", err)
	}

	messages, err := store.ReceiveServiceBusSubscriptionMessages("local-messaging", "events", "worker-a", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveServiceBusSubscriptionMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 1)
	}
	if messages[0].Body != `{"event":"created"}` {
		t.Fatalf("Body = %q, want %q", messages[0].Body, `{"event":"created"}`)
	}

	otherMessages, err := store.ReceiveServiceBusSubscriptionMessages("local-messaging", "events", "worker-b", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveServiceBusSubscriptionMessages() other error = %v", err)
	}
	if len(otherMessages) != 1 {
		t.Fatalf("len(otherMessages) = %d, want %d", len(otherMessages), 1)
	}

	if err := store.DeleteServiceBusSubscriptionMessage("local-messaging", "events", "worker-a", messages[0].ID, messages[0].LockToken); err != nil {
		t.Fatalf("DeleteServiceBusSubscriptionMessage() error = %v", err)
	}
}

func TestSnapshotAndRestorePreserveServiceBusTopicState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusNamespace("local-messaging"); err != nil {
		t.Fatalf("CreateServiceBusNamespace() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusTopic("local-messaging", "events"); err != nil {
		t.Fatalf("CreateServiceBusTopic() error = %v", err)
	}
	if _, _, err := store.CreateServiceBusSubscription("local-messaging", "events", "worker-a"); err != nil {
		t.Fatalf("CreateServiceBusSubscription() error = %v", err)
	}
	if _, err := store.PublishServiceBusTopicMessage("local-messaging", "events", `{"event":"created"}`); err != nil {
		t.Fatalf("PublishServiceBusTopicMessage() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "servicebus-topics.snapshot.json")
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

	subscriptions, err := restoreStore.ListServiceBusSubscriptions("local-messaging", "events")
	if err != nil {
		t.Fatalf("ListServiceBusSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("len(subscriptions) = %d, want %d", len(subscriptions), 1)
	}

	messages, err := restoreStore.ReceiveServiceBusSubscriptionMessages("local-messaging", "events", "worker-a", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("ReceiveServiceBusSubscriptionMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 1)
	}
	if messages[0].Body != `{"event":"created"}` {
		t.Fatalf("Body = %q, want %q", messages[0].Body, `{"event":"created"}`)
	}
}

func TestEventHubCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	namespace, created, err := store.CreateEventHubNamespace("local-streaming")
	if err != nil {
		t.Fatalf("CreateEventHubNamespace() error = %v", err)
	}
	if !created || namespace.Name != "local-streaming" {
		t.Fatalf("CreateEventHubNamespace() = %#v, %t", namespace, created)
	}

	namespaces, err := store.ListEventHubNamespaces()
	if err != nil {
		t.Fatalf("ListEventHubNamespaces() error = %v", err)
	}
	if len(namespaces) != 1 {
		t.Fatalf("len(namespaces) = %d, want %d", len(namespaces), 1)
	}

	hub, created, err := store.CreateEventHub("local-streaming", "orders")
	if err != nil {
		t.Fatalf("CreateEventHub() error = %v", err)
	}
	if !created || hub.Name != "orders" {
		t.Fatalf("CreateEventHub() = %#v, %t", hub, created)
	}

	hubs, err := store.ListEventHubs("local-streaming")
	if err != nil {
		t.Fatalf("ListEventHubs() error = %v", err)
	}
	if len(hubs) != 1 {
		t.Fatalf("len(hubs) = %d, want %d", len(hubs), 1)
	}

	first, err := store.PublishEventHubEvent("local-streaming", "orders", `{"event":"created"}`, "tenant-a")
	if err != nil {
		t.Fatalf("PublishEventHubEvent() first error = %v", err)
	}
	second, err := store.PublishEventHubEvent("local-streaming", "orders", `{"event":"updated"}`, "tenant-a")
	if err != nil {
		t.Fatalf("PublishEventHubEvent() second error = %v", err)
	}
	if first.SequenceNumber != 1 || second.SequenceNumber != 2 {
		t.Fatalf("sequence numbers = (%d, %d), want (1, 2)", first.SequenceNumber, second.SequenceNumber)
	}

	events, err := store.ListEventHubEvents("local-streaming", "orders", 2, 10)
	if err != nil {
		t.Fatalf("ListEventHubEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want %d", len(events), 1)
	}
	if events[0].Body != `{"event":"updated"}` {
		t.Fatalf("Body = %q, want %q", events[0].Body, `{"event":"updated"}`)
	}
}

func TestSnapshotAndRestorePreserveEventHubState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
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
	if _, err := store.PublishEventHubEvent("local-streaming", "orders", `{"event":"created"}`, "tenant-a"); err != nil {
		t.Fatalf("PublishEventHubEvent() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "eventhubs.snapshot.json")
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

	events, err := restoreStore.ListEventHubEvents("local-streaming", "orders", 1, 10)
	if err != nil {
		t.Fatalf("ListEventHubEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want %d", len(events), 1)
	}
	if events[0].PartitionKey != "tenant-a" {
		t.Fatalf("PartitionKey = %q, want %q", events[0].PartitionKey, "tenant-a")
	}
}

func TestAppConfigCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	configStore, created, err := store.CreateAppConfigStore("tiny-settings")
	if err != nil {
		t.Fatalf("CreateAppConfigStore() error = %v", err)
	}
	if !created || configStore.Name != "tiny-settings" {
		t.Fatalf("CreateAppConfigStore() = %#v, %t", configStore, created)
	}

	value, err := store.PutAppConfigValue("tiny-settings", "FeatureX:Enabled", "prod", "true", "text/plain")
	if err != nil {
		t.Fatalf("PutAppConfigValue() error = %v", err)
	}
	if value.Value != "true" {
		t.Fatalf("Value = %q, want %q", value.Value, "true")
	}

	got, err := store.GetAppConfigValue("tiny-settings", "FeatureX:Enabled", "prod")
	if err != nil {
		t.Fatalf("GetAppConfigValue() error = %v", err)
	}
	if got.ContentType != "text/plain" {
		t.Fatalf("ContentType = %q, want %q", got.ContentType, "text/plain")
	}

	values, err := store.ListAppConfigValues("tiny-settings")
	if err != nil {
		t.Fatalf("ListAppConfigValues() error = %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("len(values) = %d, want %d", len(values), 1)
	}

	if err := store.DeleteAppConfigValue("tiny-settings", "FeatureX:Enabled", "prod"); err != nil {
		t.Fatalf("DeleteAppConfigValue() error = %v", err)
	}
}

func TestSnapshotAndRestorePreserveAppConfigState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateAppConfigStore("tiny-settings"); err != nil {
		t.Fatalf("CreateAppConfigStore() error = %v", err)
	}
	if _, err := store.PutAppConfigValue("tiny-settings", "FeatureX:Enabled", "prod", "true", "text/plain"); err != nil {
		t.Fatalf("PutAppConfigValue() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "appconfig.snapshot.json")
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

	value, err := restoreStore.GetAppConfigValue("tiny-settings", "FeatureX:Enabled", "prod")
	if err != nil {
		t.Fatalf("GetAppConfigValue() error = %v", err)
	}
	if value.Value != "true" {
		t.Fatalf("Value = %q, want %q", value.Value, "true")
	}
}

func TestCosmosCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	account, created, err := store.CreateCosmosAccount("local-cosmos")
	if err != nil {
		t.Fatalf("CreateCosmosAccount() error = %v", err)
	}
	if !created || account.Name != "local-cosmos" {
		t.Fatalf("CreateCosmosAccount() = %#v, %t", account, created)
	}

	database, created, err := store.CreateCosmosDatabase("local-cosmos", "appdb")
	if err != nil {
		t.Fatalf("CreateCosmosDatabase() error = %v", err)
	}
	if !created || database.Name != "appdb" {
		t.Fatalf("CreateCosmosDatabase() = %#v, %t", database, created)
	}

	container, created, err := store.CreateCosmosContainer("local-cosmos", "appdb", "customers", "/tenantId")
	if err != nil {
		t.Fatalf("CreateCosmosContainer() error = %v", err)
	}
	if !created || container.PartitionKeyPath != "/tenantId" {
		t.Fatalf("CreateCosmosContainer() = %#v, %t", container, created)
	}

	document, err := store.UpsertCosmosDocument("local-cosmos", "appdb", "customers", "cust-001", "tenant-a", map[string]any{
		"id":       "cust-001",
		"tenantId": "tenant-a",
		"name":     "Tiny Cloud",
	})
	if err != nil {
		t.Fatalf("UpsertCosmosDocument() error = %v", err)
	}
	if document.PartitionKey != "tenant-a" {
		t.Fatalf("PartitionKey = %q, want %q", document.PartitionKey, "tenant-a")
	}

	got, err := store.GetCosmosDocument("local-cosmos", "appdb", "customers", "cust-001")
	if err != nil {
		t.Fatalf("GetCosmosDocument() error = %v", err)
	}
	if got.Body["name"] != "Tiny Cloud" {
		t.Fatalf("name = %v, want %q", got.Body["name"], "Tiny Cloud")
	}

	documents, err := store.ListCosmosDocuments("local-cosmos", "appdb", "customers")
	if err != nil {
		t.Fatalf("ListCosmosDocuments() error = %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("len(documents) = %d, want %d", len(documents), 1)
	}

	if err := store.DeleteCosmosDocument("local-cosmos", "appdb", "customers", "cust-001"); err != nil {
		t.Fatalf("DeleteCosmosDocument() error = %v", err)
	}
}

func TestSnapshotAndRestorePreserveCosmosState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, _, err := store.CreateCosmosAccount("local-cosmos"); err != nil {
		t.Fatalf("CreateCosmosAccount() error = %v", err)
	}
	if _, _, err := store.CreateCosmosDatabase("local-cosmos", "appdb"); err != nil {
		t.Fatalf("CreateCosmosDatabase() error = %v", err)
	}
	if _, _, err := store.CreateCosmosContainer("local-cosmos", "appdb", "customers", "/tenantId"); err != nil {
		t.Fatalf("CreateCosmosContainer() error = %v", err)
	}
	if _, err := store.UpsertCosmosDocument("local-cosmos", "appdb", "customers", "cust-001", "tenant-a", map[string]any{
		"id":       "cust-001",
		"tenantId": "tenant-a",
		"name":     "Tiny Cloud",
	}); err != nil {
		t.Fatalf("UpsertCosmosDocument() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "cosmos.snapshot.json")
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

	document, err := restoreStore.GetCosmosDocument("local-cosmos", "appdb", "customers", "cust-001")
	if err != nil {
		t.Fatalf("GetCosmosDocument() error = %v", err)
	}
	if document.Body["name"] != "Tiny Cloud" {
		t.Fatalf("name = %v, want %q", document.Body["name"], "Tiny Cloud")
	}
}

func TestVirtualNetworkCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	network, err := store.UpsertVirtualNetwork("sub-123", "rg-test", "vnet-test", "westus2", []string{"10.0.0.0/16"}, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("UpsertVirtualNetwork() error = %v", err)
	}
	if network.Name != "vnet-test" {
		t.Fatalf("Name = %q, want %q", network.Name, "vnet-test")
	}

	gotNetwork, err := store.GetVirtualNetwork("sub-123", "rg-test", "vnet-test")
	if err != nil {
		t.Fatalf("GetVirtualNetwork() error = %v", err)
	}
	if len(gotNetwork.AddressPrefixes) != 1 || gotNetwork.AddressPrefixes[0] != "10.0.0.0/16" {
		t.Fatalf("AddressPrefixes = %#v, want %#v", gotNetwork.AddressPrefixes, []string{"10.0.0.0/16"})
	}

	networks, err := store.ListVirtualNetworks("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListVirtualNetworks() error = %v", err)
	}
	if len(networks) != 1 {
		t.Fatalf("len(networks) = %d, want %d", len(networks), 1)
	}

	subnet, err := store.UpsertSubnet("sub-123", "rg-test", "vnet-test", "frontend", "10.0.1.0/24")
	if err != nil {
		t.Fatalf("UpsertSubnet() error = %v", err)
	}
	if subnet.Name != "frontend" {
		t.Fatalf("Name = %q, want %q", subnet.Name, "frontend")
	}

	gotSubnet, err := store.GetSubnet("sub-123", "rg-test", "vnet-test", "frontend")
	if err != nil {
		t.Fatalf("GetSubnet() error = %v", err)
	}
	if gotSubnet.AddressPrefix != "10.0.1.0/24" {
		t.Fatalf("AddressPrefix = %q, want %q", gotSubnet.AddressPrefix, "10.0.1.0/24")
	}

	subnets, err := store.ListSubnets("sub-123", "rg-test", "vnet-test")
	if err != nil {
		t.Fatalf("ListSubnets() error = %v", err)
	}
	if len(subnets) != 1 {
		t.Fatalf("len(subnets) = %d, want %d", len(subnets), 1)
	}

	if err := store.DeleteSubnet("sub-123", "rg-test", "vnet-test", "frontend"); err != nil {
		t.Fatalf("DeleteSubnet() error = %v", err)
	}
	if err := store.DeleteVirtualNetwork("sub-123", "rg-test", "vnet-test"); err != nil {
		t.Fatalf("DeleteVirtualNetwork() error = %v", err)
	}
}

func TestSnapshotAndRestorePreserveVirtualNetworkState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertVirtualNetwork("sub-123", "rg-test", "vnet-test", "westus2", []string{"10.0.0.0/16"}, nil); err != nil {
		t.Fatalf("UpsertVirtualNetwork() error = %v", err)
	}
	if _, err := store.UpsertSubnet("sub-123", "rg-test", "vnet-test", "frontend", "10.0.1.0/24"); err != nil {
		t.Fatalf("UpsertSubnet() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "vnet.snapshot.json")
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

	subnet, err := restoreStore.GetSubnet("sub-123", "rg-test", "vnet-test", "frontend")
	if err != nil {
		t.Fatalf("GetSubnet() error = %v", err)
	}
	if subnet.AddressPrefix != "10.0.1.0/24" {
		t.Fatalf("AddressPrefix = %q, want %q", subnet.AddressPrefix, "10.0.1.0/24")
	}
}

func TestPrivateDNSCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	zone, err := store.UpsertPrivateDNSZone("sub-123", "rg-test", "internal.test", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("UpsertPrivateDNSZone() error = %v", err)
	}
	if zone.Name != "internal.test" {
		t.Fatalf("Name = %q, want %q", zone.Name, "internal.test")
	}

	gotZone, err := store.GetPrivateDNSZone("sub-123", "rg-test", "internal.test")
	if err != nil {
		t.Fatalf("GetPrivateDNSZone() error = %v", err)
	}
	if gotZone.Tags["env"] != "test" {
		t.Fatalf("Tags[env] = %q, want %q", gotZone.Tags["env"], "test")
	}

	zones, err := store.ListPrivateDNSZones("sub-123", "rg-test")
	if err != nil {
		t.Fatalf("ListPrivateDNSZones() error = %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("len(zones) = %d, want %d", len(zones), 1)
	}

	recordSet, err := store.UpsertPrivateDNSARecordSet("sub-123", "rg-test", "internal.test", "api", 60, []string{"10.0.0.4", "10.0.0.5"})
	if err != nil {
		t.Fatalf("UpsertPrivateDNSARecordSet() error = %v", err)
	}
	if len(recordSet.IPv4Addresses) != 2 {
		t.Fatalf("len(IPv4Addresses) = %d, want %d", len(recordSet.IPv4Addresses), 2)
	}

	gotRecordSet, err := store.GetPrivateDNSARecordSet("sub-123", "rg-test", "internal.test", "api")
	if err != nil {
		t.Fatalf("GetPrivateDNSARecordSet() error = %v", err)
	}
	if gotRecordSet.TTL != 60 {
		t.Fatalf("TTL = %d, want %d", gotRecordSet.TTL, 60)
	}

	recordSets, err := store.ListPrivateDNSARecordSets("sub-123", "rg-test", "internal.test")
	if err != nil {
		t.Fatalf("ListPrivateDNSARecordSets() error = %v", err)
	}
	if len(recordSets) != 1 {
		t.Fatalf("len(recordSets) = %d, want %d", len(recordSets), 1)
	}

	if err := store.DeletePrivateDNSARecordSet("sub-123", "rg-test", "internal.test", "api"); err != nil {
		t.Fatalf("DeletePrivateDNSARecordSet() error = %v", err)
	}
	if err := store.DeletePrivateDNSZone("sub-123", "rg-test", "internal.test"); err != nil {
		t.Fatalf("DeletePrivateDNSZone() error = %v", err)
	}
}

func TestSnapshotAndRestorePreservePrivateDNSState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertPrivateDNSZone("sub-123", "rg-test", "internal.test", nil); err != nil {
		t.Fatalf("UpsertPrivateDNSZone() error = %v", err)
	}
	if _, err := store.UpsertPrivateDNSARecordSet("sub-123", "rg-test", "internal.test", "api", 60, []string{"10.0.0.4"}); err != nil {
		t.Fatalf("UpsertPrivateDNSARecordSet() error = %v", err)
	}

	snapshotPath := filepath.Join(root, "dns.snapshot.json")
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

	recordSet, err := restoreStore.GetPrivateDNSARecordSet("sub-123", "rg-test", "internal.test", "api")
	if err != nil {
		t.Fatalf("GetPrivateDNSARecordSet() error = %v", err)
	}
	if len(recordSet.IPv4Addresses) != 1 || recordSet.IPv4Addresses[0] != "10.0.0.4" {
		t.Fatalf("IPv4Addresses = %#v, want %#v", recordSet.IPv4Addresses, []string{"10.0.0.4"})
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
