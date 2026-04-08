package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	root   string
	dbPath string
	mu     sync.Mutex
}

type Document struct {
	Version         string                   `json:"version"`
	UpdatedAt       string                   `json:"updatedAt"`
	Resources       map[string]ResourceGroup `json:"resources"`
	BlobContainers  []BlobContainer          `json:"blobContainers,omitempty"`
	Blobs           []BlobObject             `json:"blobs,omitempty"`
	Queues          []StorageQueue           `json:"queues,omitempty"`
	QueueMessages   []QueueMessage           `json:"queueMessages,omitempty"`
	Tables          []StorageTable           `json:"tables,omitempty"`
	TableEntities   []TableEntity            `json:"tableEntities,omitempty"`
	StorageAccounts []StorageAccount         `json:"storageAccounts,omitempty"`
	KeyVaults       []KeyVault               `json:"keyVaults,omitempty"`
	KeyVaultSecrets []KeyVaultSecret         `json:"keyVaultSecrets,omitempty"`
	Deployments     []Deployment             `json:"deployments,omitempty"`
}

type ResourceGroup struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Type              string            `json:"type,omitempty"`
	SubscriptionID    string            `json:"subscriptionId,omitempty"`
	Location          string            `json:"location"`
	Tags              map[string]string `json:"tags"`
	ManagedBy         string            `json:"managedBy,omitempty"`
	CreatedAt         string            `json:"createdAt,omitempty"`
	UpdatedAt         string            `json:"updatedAt,omitempty"`
	ProvisioningState string            `json:"provisioningState,omitempty"`
}

type Tenant struct {
	ID string
}

type Subscription struct {
	ID       string
	TenantID string
}

type Provider struct {
	Namespace         string
	RegistrationState string
}

type Operation struct {
	ID             string
	SubscriptionID string
	ResourceID     string
	Status         string
	Operation      string
	ErrorCode      string
	ErrorMessage   string
	CreatedAt      string
	UpdatedAt      string
}

type BlobContainer struct {
	AccountName string
	Name        string
	CreatedAt   string
	UpdatedAt   string
}

type BlobObject struct {
	AccountName   string
	ContainerName string
	Name          string
	ContentType   string
	Body          []byte
	ETag          string
	CreatedAt     string
	UpdatedAt     string
}

type StorageQueue struct {
	AccountName string `json:"accountName"`
	Name        string `json:"name"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type QueueMessage struct {
	AccountName   string `json:"accountName"`
	QueueName     string `json:"queueName"`
	ID            string `json:"id"`
	MessageText   string `json:"messageText"`
	PopReceipt    string `json:"popReceipt"`
	DequeueCount  int    `json:"dequeueCount"`
	VisibleAt     string `json:"visibleAt"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type StorageTable struct {
	AccountName string `json:"accountName"`
	Name        string `json:"name"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type TableEntity struct {
	AccountName   string         `json:"accountName"`
	TableName     string         `json:"tableName"`
	PartitionKey  string         `json:"partitionKey"`
	RowKey        string         `json:"rowKey"`
	Properties    map[string]any `json:"properties"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
}

type StorageAccount struct {
	ID                string            `json:"id"`
	SubscriptionID    string            `json:"subscriptionId"`
	ResourceGroupName string            `json:"resourceGroupName"`
	Name              string            `json:"name"`
	Location          string            `json:"location"`
	Kind              string            `json:"kind"`
	SKUName           string            `json:"skuName"`
	Tags              map[string]string `json:"tags"`
	ProvisioningState string            `json:"provisioningState"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
}

type KeyVault struct {
	ID                string            `json:"id"`
	SubscriptionID    string            `json:"subscriptionId"`
	ResourceGroupName string            `json:"resourceGroupName"`
	Name              string            `json:"name"`
	Location          string            `json:"location"`
	TenantID          string            `json:"tenantId"`
	SKUName           string            `json:"skuName"`
	Tags              map[string]string `json:"tags"`
	ProvisioningState string            `json:"provisioningState"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
}

type KeyVaultSecret struct {
	VaultName   string `json:"vaultName"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	ContentType string `json:"contentType,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type Deployment struct {
	ID                string            `json:"id"`
	SubscriptionID    string            `json:"subscriptionId"`
	ResourceGroupName string            `json:"resourceGroupName"`
	Name              string            `json:"name"`
	Location          string            `json:"location"`
	Mode              string            `json:"mode"`
	TemplateJSON      string            `json:"templateJson"`
	ParametersJSON    string            `json:"parametersJson"`
	OutputsJSON       string            `json:"outputsJson"`
	Tags              map[string]string `json:"tags"`
	ProvisioningState string            `json:"provisioningState"`
	ErrorCode         string            `json:"errorCode"`
	ErrorMessage      string            `json:"errorMessage"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
}

type Summary struct {
	TenantCount       int
	SubscriptionCount int
	ProviderCount     int
	StatePath         string
	ResourceCount     int
	UpdatedAt         string
}

const (
	defaultTenantID       = "00000000-0000-0000-0000-000000000001"
	defaultSubscriptionID = "11111111-1111-1111-1111-111111111111"
)

var defaultProviders = []string{
	"Microsoft.Resources",
	"Microsoft.Storage",
	"Microsoft.KeyVault",
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("state root is required")
	}
	return &Store{
		root:   root,
		dbPath: filepath.Join(root, "state.db"),
	}, nil
}

func (s *Store) Init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	return s.ensureDocumentLocked(db)
}

func (s *Store) Summary() (Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Summary{}, err
	}
	defer db.Close()

	doc, err := s.readLocked(db)
	if err != nil {
		return Summary{}, err
	}
	tenantCount, err := s.countLocked(db, "tenants")
	if err != nil {
		return Summary{}, err
	}
	subscriptionCount, err := s.countLocked(db, "subscriptions")
	if err != nil {
		return Summary{}, err
	}
	providerCount, err := s.countLocked(db, "providers")
	if err != nil {
		return Summary{}, err
	}

	return Summary{
		TenantCount:       tenantCount,
		SubscriptionCount: subscriptionCount,
		ProviderCount:     providerCount,
		StatePath:         s.dbPath,
		ResourceCount:     len(doc.Resources),
		UpdatedAt:         doc.UpdatedAt,
	}, nil
}

func (s *Store) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	return s.writeLocked(db, newDocument())
}

func (s *Store) Snapshot(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	doc, err := s.readLocked(db)
	if err != nil {
		return err
	}

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	return os.WriteFile(path, body, 0o644)
}

func (s *Store) Restore(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc Document
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("parse snapshot: %w", err)
	}
	if doc.Resources == nil {
		doc.Resources = map[string]ResourceGroup{}
	}
	if doc.BlobContainers == nil {
		doc.BlobContainers = []BlobContainer{}
	}
	if doc.Blobs == nil {
		doc.Blobs = []BlobObject{}
	}
	if doc.Queues == nil {
		doc.Queues = []StorageQueue{}
	}
	if doc.QueueMessages == nil {
		doc.QueueMessages = []QueueMessage{}
	}
	if doc.Tables == nil {
		doc.Tables = []StorageTable{}
	}
	if doc.TableEntities == nil {
		doc.TableEntities = []TableEntity{}
	}
	if doc.StorageAccounts == nil {
		doc.StorageAccounts = []StorageAccount{}
	}
	if doc.KeyVaults == nil {
		doc.KeyVaults = []KeyVault{}
	}
	if doc.KeyVaultSecrets == nil {
		doc.KeyVaultSecrets = []KeyVaultSecret{}
	}
	if doc.Deployments == nil {
		doc.Deployments = []Deployment{}
	}

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	return s.writeLocked(db, doc)
}

func (s *Store) ApplySeed(path string) error {
	return s.Restore(path)
}

func (s *Store) ListSubscriptions() ([]Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT id, tenant_id FROM subscriptions ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var subscription Subscription
		if err := rows.Scan(&subscription.ID, &subscription.TenantID); err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriptions: %w", err)
	}
	return subscriptions, nil
}

func (s *Store) ListTenants() ([]Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT id FROM tenants ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		if err := rows.Scan(&tenant.ID); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenants: %w", err)
	}
	return tenants, nil
}

func (s *Store) ListProviders() ([]Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT namespace, registration_state FROM providers ORDER BY namespace`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var provider Provider
		if err := rows.Scan(&provider.Namespace, &provider.RegistrationState); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		providers = append(providers, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return providers, nil
}

func (s *Store) GetProvider(namespace string) (Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Provider{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return Provider{}, err
	}

	var provider Provider
	err = db.QueryRow(`SELECT namespace, registration_state FROM providers WHERE namespace = ?`, namespace).Scan(
		&provider.Namespace,
		&provider.RegistrationState,
	)
	if err != nil {
		return Provider{}, err
	}
	return provider, nil
}

func (s *Store) RegisterProvider(namespace string) (Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Provider{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return Provider{}, err
	}

	if _, err := db.Exec(`
INSERT INTO providers (namespace, registration_state) VALUES (?, 'Registered')
ON CONFLICT(namespace) DO UPDATE SET registration_state = 'Registered'
`, namespace); err != nil {
		return Provider{}, fmt.Errorf("register provider: %w", err)
	}

	return Provider{
		Namespace:         namespace,
		RegistrationState: "Registered",
	}, nil
}

func (s *Store) UpsertResourceGroup(subscriptionID, name, location, managedBy string, tags map[string]string) (ResourceGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return ResourceGroup{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return ResourceGroup{}, err
	}

	if tags == nil {
		tags = map[string]string{}
	}
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, name)
	nowValue := now()

	var createdAt string
	err = db.QueryRow(`SELECT created_at FROM resource_groups WHERE id = ?`, id).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return ResourceGroup{}, fmt.Errorf("read existing resource group: %w", err)
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return ResourceGroup{}, fmt.Errorf("marshal resource group tags: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO resource_groups (
    id, subscription_id, name, location, tags_json, managed_by, created_at, updated_at, provisioning_state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    subscription_id = excluded.subscription_id,
    name = excluded.name,
    location = excluded.location,
    tags_json = excluded.tags_json,
    managed_by = excluded.managed_by,
    updated_at = excluded.updated_at,
    provisioning_state = excluded.provisioning_state
`, id, subscriptionID, name, location, string(tagsJSON), managedBy, createdAt, nowValue, "Succeeded"); err != nil {
		return ResourceGroup{}, fmt.Errorf("upsert resource group: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO metadata (key, value) VALUES ('updated_at', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, nowValue); err != nil {
		return ResourceGroup{}, fmt.Errorf("update state timestamp: %w", err)
	}

	return s.getResourceGroupLocked(db, subscriptionID, name)
}

func (s *Store) GetResourceGroup(subscriptionID, name string) (ResourceGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return ResourceGroup{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return ResourceGroup{}, err
	}

	return s.getResourceGroupLocked(db, subscriptionID, name)
}

func (s *Store) ListResourceGroups(subscriptionID string) ([]ResourceGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, subscription_id, name, location, tags_json, managed_by, created_at, updated_at, provisioning_state
FROM resource_groups
WHERE subscription_id = ?
ORDER BY name`, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("list resource groups: %w", err)
	}
	defer rows.Close()

	var resourceGroups []ResourceGroup
	for rows.Next() {
		resourceGroup, scanErr := scanResourceGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		resourceGroups = append(resourceGroups, resourceGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resource groups: %w", err)
	}
	return resourceGroups, nil
}

func (s *Store) DeleteResourceGroup(subscriptionID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, name)
	result, err := db.Exec(`DELETE FROM resource_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete resource group: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete resource group rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if _, err := db.Exec(`
INSERT INTO metadata (key, value) VALUES ('updated_at', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, now()); err != nil {
		return fmt.Errorf("update state timestamp: %w", err)
	}
	return nil
}

func (s *Store) CreateOperation(subscriptionID, resourceID, operation, status string) (Operation, error) {
	return s.CreateOperationResult(subscriptionID, resourceID, operation, status, "", "")
}

func (s *Store) CreateOperationResult(subscriptionID, resourceID, operation, status, errorCode, errorMessage string) (Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Operation{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return Operation{}, err
	}

	id := fmt.Sprintf("op-%d", time.Now().UTC().UnixNano())
	nowValue := now()
	if _, err := db.Exec(`
INSERT INTO operations (
    id, subscription_id, resource_id, operation_name, status, error_code, error_message, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, subscriptionID, resourceID, operation, status, errorCode, errorMessage, nowValue, nowValue,
	); err != nil {
		return Operation{}, fmt.Errorf("create operation: %w", err)
	}

	return Operation{
		ID:             id,
		SubscriptionID: subscriptionID,
		ResourceID:     resourceID,
		Status:         status,
		Operation:      operation,
		ErrorCode:      errorCode,
		ErrorMessage:   errorMessage,
		CreatedAt:      nowValue,
		UpdatedAt:      nowValue,
	}, nil
}

func (s *Store) GetOperation(subscriptionID, operationID string) (Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Operation{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return Operation{}, err
	}

	var operation Operation
	err = db.QueryRow(`
SELECT id, subscription_id, resource_id, operation_name, status, error_code, error_message, created_at, updated_at
FROM operations
WHERE subscription_id = ? AND id = ?`, subscriptionID, operationID).Scan(
		&operation.ID,
		&operation.SubscriptionID,
		&operation.ResourceID,
		&operation.Operation,
		&operation.Status,
		&operation.ErrorCode,
		&operation.ErrorMessage,
		&operation.CreatedAt,
		&operation.UpdatedAt,
	)
	if err != nil {
		return Operation{}, err
	}
	return operation, nil
}

func (s *Store) CreateBlobContainer(accountName, name string) (BlobContainer, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return BlobContainer{}, false, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return BlobContainer{}, false, err
	}

	var existing BlobContainer
	err = db.QueryRow(`
SELECT account_name, name, created_at, updated_at
FROM blob_containers
WHERE account_name = ? AND name = ?`, accountName, name).Scan(
		&existing.AccountName,
		&existing.Name,
		&existing.CreatedAt,
		&existing.UpdatedAt,
	)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return BlobContainer{}, false, fmt.Errorf("read blob container: %w", err)
	}

	nowValue := now()
	if _, err := db.Exec(`
INSERT INTO blob_containers (account_name, name, created_at, updated_at)
VALUES (?, ?, ?, ?)`, accountName, name, nowValue, nowValue); err != nil {
		return BlobContainer{}, false, fmt.Errorf("create blob container: %w", err)
	}

	return BlobContainer{
		AccountName: accountName,
		Name:        name,
		CreatedAt:   nowValue,
		UpdatedAt:   nowValue,
	}, true, nil
}

func (s *Store) ListBlobContainers(accountName string) ([]BlobContainer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT account_name, name, created_at, updated_at
FROM blob_containers
WHERE account_name = ?
ORDER BY name`, accountName)
	if err != nil {
		return nil, fmt.Errorf("list blob containers: %w", err)
	}
	defer rows.Close()

	var containers []BlobContainer
	for rows.Next() {
		var container BlobContainer
		if err := rows.Scan(&container.AccountName, &container.Name, &container.CreatedAt, &container.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan blob container: %w", err)
		}
		containers = append(containers, container)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blob containers: %w", err)
	}
	return containers, nil
}

func (s *Store) PutBlob(accountName, containerName, blobName, contentType string, body []byte) (BlobObject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return BlobObject{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return BlobObject{}, err
	}

	var containerExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM blob_containers WHERE account_name = ? AND name = ?
)`, accountName, containerName).Scan(&containerExists); err != nil {
		return BlobObject{}, fmt.Errorf("query blob container: %w", err)
	}
	if !containerExists {
		return BlobObject{}, sql.ErrNoRows
	}

	nowValue := now()
	etag := fmt.Sprintf("\"%d\"", time.Now().UTC().UnixNano())

	var createdAt string
	err = db.QueryRow(`
SELECT created_at
FROM blobs
WHERE account_name = ? AND container_name = ? AND name = ?`, accountName, containerName, blobName).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return BlobObject{}, fmt.Errorf("read blob timestamps: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO blobs (
    account_name, container_name, name, content_type, body, etag, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(account_name, container_name, name) DO UPDATE SET
    content_type = excluded.content_type,
    body = excluded.body,
    etag = excluded.etag,
    updated_at = excluded.updated_at
`, accountName, containerName, blobName, contentType, body, etag, createdAt, nowValue); err != nil {
		return BlobObject{}, fmt.Errorf("put blob: %w", err)
	}

	return BlobObject{
		AccountName:   accountName,
		ContainerName: containerName,
		Name:          blobName,
		ContentType:   contentType,
		Body:          append([]byte(nil), body...),
		ETag:          etag,
		CreatedAt:     createdAt,
		UpdatedAt:     nowValue,
	}, nil
}

func (s *Store) GetBlob(accountName, containerName, blobName string) (BlobObject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return BlobObject{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return BlobObject{}, err
	}

	var blob BlobObject
	err = db.QueryRow(`
SELECT account_name, container_name, name, content_type, body, etag, created_at, updated_at
FROM blobs
WHERE account_name = ? AND container_name = ? AND name = ?`, accountName, containerName, blobName).Scan(
		&blob.AccountName,
		&blob.ContainerName,
		&blob.Name,
		&blob.ContentType,
		&blob.Body,
		&blob.ETag,
		&blob.CreatedAt,
		&blob.UpdatedAt,
	)
	if err != nil {
		return BlobObject{}, err
	}
	return blob, nil
}

func (s *Store) ListBlobs(accountName, containerName string) ([]BlobObject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT account_name, container_name, name, content_type, body, etag, created_at, updated_at
FROM blobs
WHERE account_name = ? AND container_name = ?
ORDER BY name`, accountName, containerName)
	if err != nil {
		return nil, fmt.Errorf("list blobs: %w", err)
	}
	defer rows.Close()

	var blobs []BlobObject
	for rows.Next() {
		var blob BlobObject
		if err := rows.Scan(
			&blob.AccountName,
			&blob.ContainerName,
			&blob.Name,
			&blob.ContentType,
			&blob.Body,
			&blob.ETag,
			&blob.CreatedAt,
			&blob.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan blob: %w", err)
		}
		blobs = append(blobs, blob)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blobs: %w", err)
	}
	return blobs, nil
}

func (s *Store) DeleteBlob(accountName, containerName, blobName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM blobs
WHERE account_name = ? AND container_name = ? AND name = ?`, accountName, containerName, blobName)
	if err != nil {
		return fmt.Errorf("delete blob: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete blob rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) CreateQueue(accountName, name string) (StorageQueue, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return StorageQueue{}, false, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return StorageQueue{}, false, err
	}

	var accountExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM storage_accounts WHERE name = ?
)`, accountName).Scan(&accountExists); err != nil {
		return StorageQueue{}, false, fmt.Errorf("query storage account: %w", err)
	}
	if !accountExists {
		return StorageQueue{}, false, sql.ErrNoRows
	}

	var existing StorageQueue
	err = db.QueryRow(`
SELECT account_name, name, created_at, updated_at
FROM storage_queues
WHERE account_name = ? AND name = ?`, accountName, name).Scan(
		&existing.AccountName,
		&existing.Name,
		&existing.CreatedAt,
		&existing.UpdatedAt,
	)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return StorageQueue{}, false, fmt.Errorf("read storage queue: %w", err)
	}

	nowValue := now()
	if _, err := db.Exec(`
INSERT INTO storage_queues (account_name, name, created_at, updated_at)
VALUES (?, ?, ?, ?)`, accountName, name, nowValue, nowValue); err != nil {
		return StorageQueue{}, false, fmt.Errorf("create storage queue: %w", err)
	}

	return StorageQueue{
		AccountName: accountName,
		Name:        name,
		CreatedAt:   nowValue,
		UpdatedAt:   nowValue,
	}, true, nil
}

func (s *Store) ListQueues(accountName string) ([]StorageQueue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT account_name, name, created_at, updated_at
FROM storage_queues
WHERE account_name = ?
ORDER BY name`, accountName)
	if err != nil {
		return nil, fmt.Errorf("list storage queues: %w", err)
	}
	defer rows.Close()

	var queues []StorageQueue
	for rows.Next() {
		var queue StorageQueue
		if err := rows.Scan(&queue.AccountName, &queue.Name, &queue.CreatedAt, &queue.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan storage queue: %w", err)
		}
		queues = append(queues, queue)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage queues: %w", err)
	}
	return queues, nil
}

func (s *Store) EnqueueMessage(accountName, queueName, messageText string) (QueueMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return QueueMessage{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return QueueMessage{}, err
	}

	var queueExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM storage_queues WHERE account_name = ? AND name = ?
)`, accountName, queueName).Scan(&queueExists); err != nil {
		return QueueMessage{}, fmt.Errorf("query storage queue: %w", err)
	}
	if !queueExists {
		return QueueMessage{}, sql.ErrNoRows
	}

	nowValue := now()
	messageID := fmt.Sprintf("msg-%d", time.Now().UTC().UnixNano())
	popReceipt := fmt.Sprintf("pop-%d", time.Now().UTC().UnixNano())
	if _, err := db.Exec(`
INSERT INTO queue_messages (
    account_name, queue_name, id, message_text, pop_receipt, dequeue_count, visible_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		accountName, queueName, messageID, messageText, popReceipt, 0, nowValue, nowValue, nowValue,
	); err != nil {
		return QueueMessage{}, fmt.Errorf("enqueue queue message: %w", err)
	}

	return QueueMessage{
		AccountName:  accountName,
		QueueName:    queueName,
		ID:           messageID,
		MessageText:  messageText,
		PopReceipt:   popReceipt,
		DequeueCount: 0,
		VisibleAt:    nowValue,
		CreatedAt:    nowValue,
		UpdatedAt:    nowValue,
	}, nil
}

func (s *Store) ReceiveMessages(accountName, queueName string, maxMessages int, visibilityTimeout time.Duration) ([]QueueMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}
	if maxMessages <= 0 {
		maxMessages = 1
	}

	var queueExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM storage_queues WHERE account_name = ? AND name = ?
)`, accountName, queueName).Scan(&queueExists); err != nil {
		return nil, fmt.Errorf("query storage queue: %w", err)
	}
	if !queueExists {
		return nil, sql.ErrNoRows
	}

	nowValue := time.Now().UTC()
	rows, err := db.Query(`
SELECT account_name, queue_name, id, message_text, pop_receipt, dequeue_count, visible_at, created_at, updated_at
FROM queue_messages
WHERE account_name = ? AND queue_name = ? AND visible_at <= ?
ORDER BY created_at
LIMIT ?`, accountName, queueName, nowValue.Format(time.RFC3339Nano), maxMessages)
	if err != nil {
		return nil, fmt.Errorf("receive queue messages: %w", err)
	}
	defer rows.Close()

	var messages []QueueMessage
	for rows.Next() {
		var message QueueMessage
		if err := rows.Scan(
			&message.AccountName,
			&message.QueueName,
			&message.ID,
			&message.MessageText,
			&message.PopReceipt,
			&message.DequeueCount,
			&message.VisibleAt,
			&message.CreatedAt,
			&message.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan queue message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate queue messages: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close queue messages rows: %w", err)
	}

	for i := range messages {
		messages[i].DequeueCount++
		messages[i].PopReceipt = fmt.Sprintf("pop-%d", time.Now().UTC().UnixNano())
		messages[i].VisibleAt = nowValue.Add(visibilityTimeout).Format(time.RFC3339Nano)
		messages[i].UpdatedAt = nowValue.Format(time.RFC3339Nano)
		if _, err := db.Exec(`
UPDATE queue_messages
SET pop_receipt = ?, dequeue_count = ?, visible_at = ?, updated_at = ?
WHERE account_name = ? AND queue_name = ? AND id = ?`,
			messages[i].PopReceipt,
			messages[i].DequeueCount,
			messages[i].VisibleAt,
			messages[i].UpdatedAt,
			messages[i].AccountName,
			messages[i].QueueName,
			messages[i].ID,
		); err != nil {
			return nil, fmt.Errorf("update queue message visibility: %w", err)
		}
	}
	return messages, nil
}

func (s *Store) DeleteMessage(accountName, queueName, messageID, popReceipt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM queue_messages
WHERE account_name = ? AND queue_name = ? AND id = ? AND pop_receipt = ?`,
		accountName, queueName, messageID, popReceipt,
	)
	if err != nil {
		return fmt.Errorf("delete queue message: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete queue message rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) CreateTable(accountName, name string) (StorageTable, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return StorageTable{}, false, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return StorageTable{}, false, err
	}

	var accountExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM storage_accounts WHERE name = ?
)`, accountName).Scan(&accountExists); err != nil {
		return StorageTable{}, false, fmt.Errorf("query storage account: %w", err)
	}
	if !accountExists {
		return StorageTable{}, false, sql.ErrNoRows
	}

	var existing StorageTable
	err = db.QueryRow(`
SELECT account_name, name, created_at, updated_at
FROM storage_tables
WHERE account_name = ? AND name = ?`, accountName, name).Scan(
		&existing.AccountName,
		&existing.Name,
		&existing.CreatedAt,
		&existing.UpdatedAt,
	)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return StorageTable{}, false, fmt.Errorf("read storage table: %w", err)
	}

	nowValue := now()
	if _, err := db.Exec(`
INSERT INTO storage_tables (account_name, name, created_at, updated_at)
VALUES (?, ?, ?, ?)`, accountName, name, nowValue, nowValue); err != nil {
		return StorageTable{}, false, fmt.Errorf("create storage table: %w", err)
	}

	return StorageTable{
		AccountName: accountName,
		Name:        name,
		CreatedAt:   nowValue,
		UpdatedAt:   nowValue,
	}, true, nil
}

func (s *Store) ListTables(accountName string) ([]StorageTable, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT account_name, name, created_at, updated_at
FROM storage_tables
WHERE account_name = ?
ORDER BY name`, accountName)
	if err != nil {
		return nil, fmt.Errorf("list storage tables: %w", err)
	}
	defer rows.Close()

	var tables []StorageTable
	for rows.Next() {
		var table StorageTable
		if err := rows.Scan(&table.AccountName, &table.Name, &table.CreatedAt, &table.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan storage table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage tables: %w", err)
	}
	return tables, nil
}

func (s *Store) DeleteTable(accountName, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM storage_tables
WHERE account_name = ? AND name = ?`, accountName, name)
	if err != nil {
		return fmt.Errorf("delete storage table: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete storage table rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if _, err := db.Exec(`
DELETE FROM table_entities
WHERE account_name = ? AND table_name = ?`, accountName, name); err != nil {
		return fmt.Errorf("delete table entities: %w", err)
	}
	return nil
}

func (s *Store) UpsertTableEntity(accountName, tableName, partitionKey, rowKey string, properties map[string]any) (TableEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return TableEntity{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return TableEntity{}, err
	}

	var tableExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM storage_tables WHERE account_name = ? AND name = ?
)`, accountName, tableName).Scan(&tableExists); err != nil {
		return TableEntity{}, fmt.Errorf("query storage table: %w", err)
	}
	if !tableExists {
		return TableEntity{}, sql.ErrNoRows
	}
	if properties == nil {
		properties = map[string]any{}
	}

	nowValue := now()
	var createdAt string
	err = db.QueryRow(`
SELECT created_at
FROM table_entities
WHERE account_name = ? AND table_name = ? AND partition_key = ? AND row_key = ?`,
		accountName, tableName, partitionKey, rowKey,
	).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return TableEntity{}, fmt.Errorf("read table entity timestamps: %w", err)
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return TableEntity{}, fmt.Errorf("marshal table entity properties: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO table_entities (
    account_name, table_name, partition_key, row_key, properties_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(account_name, table_name, partition_key, row_key) DO UPDATE SET
    properties_json = excluded.properties_json,
    updated_at = excluded.updated_at`,
		accountName, tableName, partitionKey, rowKey, string(propertiesJSON), createdAt, nowValue,
	); err != nil {
		return TableEntity{}, fmt.Errorf("upsert table entity: %w", err)
	}

	return TableEntity{
		AccountName:  accountName,
		TableName:    tableName,
		PartitionKey: partitionKey,
		RowKey:       rowKey,
		Properties:   properties,
		CreatedAt:    createdAt,
		UpdatedAt:    nowValue,
	}, nil
}

func (s *Store) GetTableEntity(accountName, tableName, partitionKey, rowKey string) (TableEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return TableEntity{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return TableEntity{}, err
	}

	return s.getTableEntityLocked(db, accountName, tableName, partitionKey, rowKey)
}

func (s *Store) ListTableEntities(accountName, tableName string) ([]TableEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT account_name, table_name, partition_key, row_key, properties_json, created_at, updated_at
FROM table_entities
WHERE account_name = ? AND table_name = ?
ORDER BY partition_key, row_key`, accountName, tableName)
	if err != nil {
		return nil, fmt.Errorf("list table entities: %w", err)
	}
	defer rows.Close()

	var entities []TableEntity
	for rows.Next() {
		entity, err := scanTableEntity(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table entities: %w", err)
	}
	return entities, nil
}

func (s *Store) DeleteTableEntity(accountName, tableName, partitionKey, rowKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM table_entities
WHERE account_name = ? AND table_name = ? AND partition_key = ? AND row_key = ?`,
		accountName, tableName, partitionKey, rowKey,
	)
	if err != nil {
		return fmt.Errorf("delete table entity: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete table entity rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) UpsertStorageAccount(subscriptionID, resourceGroupName, name, location, kind, skuName string, tags map[string]string) (StorageAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return StorageAccount{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return StorageAccount{}, err
	}

	if tags == nil {
		tags = map[string]string{}
	}
	if kind == "" {
		kind = "StorageV2"
	}
	if skuName == "" {
		skuName = "Standard_LRS"
	}

	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s", subscriptionID, resourceGroupName, name)
	nowValue := now()

	var createdAt string
	err = db.QueryRow(`SELECT created_at FROM storage_accounts WHERE id = ?`, id).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return StorageAccount{}, fmt.Errorf("read existing storage account: %w", err)
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return StorageAccount{}, fmt.Errorf("marshal storage account tags: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO storage_accounts (
    id, subscription_id, resource_group_name, name, location, kind, sku_name, tags_json, provisioning_state, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    location = excluded.location,
    kind = excluded.kind,
    sku_name = excluded.sku_name,
    tags_json = excluded.tags_json,
    provisioning_state = excluded.provisioning_state,
    updated_at = excluded.updated_at
`, id, subscriptionID, resourceGroupName, name, location, kind, skuName, string(tagsJSON), "Succeeded", createdAt, nowValue); err != nil {
		return StorageAccount{}, fmt.Errorf("upsert storage account: %w", err)
	}

	return s.getStorageAccountLocked(db, subscriptionID, resourceGroupName, name)
}

func (s *Store) GetStorageAccount(subscriptionID, resourceGroupName, name string) (StorageAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return StorageAccount{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return StorageAccount{}, err
	}

	return s.getStorageAccountLocked(db, subscriptionID, resourceGroupName, name)
}

func (s *Store) GetStorageAccountByName(name string) (StorageAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return StorageAccount{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return StorageAccount{}, err
	}

	row := db.QueryRow(`
SELECT id, subscription_id, resource_group_name, name, location, kind, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM storage_accounts
WHERE name = ?
ORDER BY created_at
LIMIT 1`, name)
	return scanStorageAccount(row)
}

func (s *Store) ListStorageAccounts(subscriptionID, resourceGroupName string) ([]StorageAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, subscription_id, resource_group_name, name, location, kind, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM storage_accounts
WHERE subscription_id = ? AND resource_group_name = ?
ORDER BY name`, subscriptionID, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("list storage accounts: %w", err)
	}
	defer rows.Close()

	var accounts []StorageAccount
	for rows.Next() {
		account, err := scanStorageAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage accounts: %w", err)
	}
	return accounts, nil
}

func (s *Store) DeleteStorageAccount(subscriptionID, resourceGroupName, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM storage_accounts
WHERE subscription_id = ? AND resource_group_name = ? AND name = ?`, subscriptionID, resourceGroupName, name)
	if err != nil {
		return fmt.Errorf("delete storage account: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete storage account rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) UpsertKeyVault(subscriptionID, resourceGroupName, name, location, tenantID, skuName string, tags map[string]string) (KeyVault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return KeyVault{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return KeyVault{}, err
	}

	if tags == nil {
		tags = map[string]string{}
	}
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	if skuName == "" {
		skuName = "standard"
	}

	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s", subscriptionID, resourceGroupName, name)
	nowValue := now()

	var createdAt string
	err = db.QueryRow(`SELECT created_at FROM key_vaults WHERE id = ?`, id).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return KeyVault{}, fmt.Errorf("read existing key vault: %w", err)
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return KeyVault{}, fmt.Errorf("marshal key vault tags: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO key_vaults (
    id, subscription_id, resource_group_name, name, location, tenant_id, sku_name, tags_json, provisioning_state, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    location = excluded.location,
    tenant_id = excluded.tenant_id,
    sku_name = excluded.sku_name,
    tags_json = excluded.tags_json,
    provisioning_state = excluded.provisioning_state,
    updated_at = excluded.updated_at
`, id, subscriptionID, resourceGroupName, name, location, tenantID, skuName, string(tagsJSON), "Succeeded", createdAt, nowValue); err != nil {
		return KeyVault{}, fmt.Errorf("upsert key vault: %w", err)
	}

	return s.getKeyVaultLocked(db, subscriptionID, resourceGroupName, name)
}

func (s *Store) GetKeyVault(subscriptionID, resourceGroupName, name string) (KeyVault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return KeyVault{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return KeyVault{}, err
	}

	return s.getKeyVaultLocked(db, subscriptionID, resourceGroupName, name)
}

func (s *Store) ListKeyVaults(subscriptionID, resourceGroupName string) ([]KeyVault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, subscription_id, resource_group_name, name, location, tenant_id, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM key_vaults
WHERE subscription_id = ? AND resource_group_name = ?
ORDER BY name`, subscriptionID, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("list key vaults: %w", err)
	}
	defer rows.Close()

	var vaults []KeyVault
	for rows.Next() {
		vault, err := scanKeyVault(rows)
		if err != nil {
			return nil, err
		}
		vaults = append(vaults, vault)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate key vaults: %w", err)
	}
	return vaults, nil
}

func (s *Store) GetKeyVaultByName(name string) (KeyVault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return KeyVault{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return KeyVault{}, err
	}

	row := db.QueryRow(`
SELECT id, subscription_id, resource_group_name, name, location, tenant_id, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM key_vaults
WHERE name = ?
ORDER BY created_at
LIMIT 1`, name)
	return scanKeyVault(row)
}

func (s *Store) DeleteKeyVault(subscriptionID, resourceGroupName, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM key_vaults
WHERE subscription_id = ? AND resource_group_name = ? AND name = ?`, subscriptionID, resourceGroupName, name)
	if err != nil {
		return fmt.Errorf("delete key vault: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete key vault rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if _, err := db.Exec(`DELETE FROM key_vault_secrets WHERE vault_name = ?`, name); err != nil {
		return fmt.Errorf("delete key vault secrets: %w", err)
	}
	return nil
}

func (s *Store) PutKeyVaultSecret(vaultName, name, value, contentType string) (KeyVaultSecret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return KeyVaultSecret{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return KeyVaultSecret{}, err
	}

	var vaultExists bool
	if err := db.QueryRow(`
SELECT EXISTS(
    SELECT 1 FROM key_vaults WHERE name = ?
)`, vaultName).Scan(&vaultExists); err != nil {
		return KeyVaultSecret{}, fmt.Errorf("query key vault: %w", err)
	}
	if !vaultExists {
		return KeyVaultSecret{}, sql.ErrNoRows
	}

	nowValue := now()
	var createdAt string
	err = db.QueryRow(`
SELECT created_at
FROM key_vault_secrets
WHERE vault_name = ? AND name = ?`, vaultName, name).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return KeyVaultSecret{}, fmt.Errorf("read key vault secret timestamps: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO key_vault_secrets (
    vault_name, name, value, content_type, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(vault_name, name) DO UPDATE SET
    value = excluded.value,
    content_type = excluded.content_type,
    updated_at = excluded.updated_at
`, vaultName, name, value, contentType, createdAt, nowValue); err != nil {
		return KeyVaultSecret{}, fmt.Errorf("put key vault secret: %w", err)
	}

	return KeyVaultSecret{
		VaultName:   vaultName,
		Name:        name,
		Value:       value,
		ContentType: contentType,
		CreatedAt:   createdAt,
		UpdatedAt:   nowValue,
	}, nil
}

func (s *Store) GetKeyVaultSecret(vaultName, name string) (KeyVaultSecret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return KeyVaultSecret{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return KeyVaultSecret{}, err
	}

	var secret KeyVaultSecret
	err = db.QueryRow(`
SELECT vault_name, name, value, content_type, created_at, updated_at
FROM key_vault_secrets
WHERE vault_name = ? AND name = ?`, vaultName, name).Scan(
		&secret.VaultName,
		&secret.Name,
		&secret.Value,
		&secret.ContentType,
		&secret.CreatedAt,
		&secret.UpdatedAt,
	)
	if err != nil {
		return KeyVaultSecret{}, err
	}
	return secret, nil
}

func (s *Store) ListKeyVaultSecrets(vaultName string) ([]KeyVaultSecret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT vault_name, name, value, content_type, created_at, updated_at
FROM key_vault_secrets
WHERE vault_name = ?
ORDER BY name`, vaultName)
	if err != nil {
		return nil, fmt.Errorf("list key vault secrets: %w", err)
	}
	defer rows.Close()

	var secrets []KeyVaultSecret
	for rows.Next() {
		var secret KeyVaultSecret
		if err := rows.Scan(
			&secret.VaultName,
			&secret.Name,
			&secret.Value,
			&secret.ContentType,
			&secret.CreatedAt,
			&secret.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan key vault secret: %w", err)
		}
		secrets = append(secrets, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate key vault secrets: %w", err)
	}
	return secrets, nil
}

func (s *Store) DeleteKeyVaultSecret(vaultName, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM key_vault_secrets
WHERE vault_name = ? AND name = ?`, vaultName, name)
	if err != nil {
		return fmt.Errorf("delete key vault secret: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete key vault secret rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) UpsertDeployment(subscriptionID, resourceGroupName, name, location, mode, templateJSON, parametersJSON, outputsJSON, provisioningState, errorCode, errorMessage string, tags map[string]string) (Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Deployment{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return Deployment{}, err
	}

	if tags == nil {
		tags = map[string]string{}
	}
	if mode == "" {
		mode = "Incremental"
	}
	if provisioningState == "" {
		provisioningState = "Accepted"
	}

	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", subscriptionID, resourceGroupName, name)
	nowValue := now()

	var createdAt string
	err = db.QueryRow(`SELECT created_at FROM deployments WHERE id = ?`, id).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		createdAt = nowValue
	} else if err != nil {
		return Deployment{}, fmt.Errorf("read existing deployment: %w", err)
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return Deployment{}, fmt.Errorf("marshal deployment tags: %w", err)
	}

	if _, err := db.Exec(`
INSERT INTO deployments (
    id, subscription_id, resource_group_name, name, location, mode, template_json, parameters_json, outputs_json, tags_json, provisioning_state, error_code, error_message, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    location = excluded.location,
    mode = excluded.mode,
    template_json = excluded.template_json,
    parameters_json = excluded.parameters_json,
    outputs_json = excluded.outputs_json,
    tags_json = excluded.tags_json,
    provisioning_state = excluded.provisioning_state,
    error_code = excluded.error_code,
    error_message = excluded.error_message,
    updated_at = excluded.updated_at
`, id, subscriptionID, resourceGroupName, name, location, mode, templateJSON, parametersJSON, outputsJSON, string(tagsJSON), provisioningState, errorCode, errorMessage, createdAt, nowValue); err != nil {
		return Deployment{}, fmt.Errorf("upsert deployment: %w", err)
	}

	return s.getDeploymentLocked(db, subscriptionID, resourceGroupName, name)
}

func (s *Store) GetDeployment(subscriptionID, resourceGroupName, name string) (Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return Deployment{}, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return Deployment{}, err
	}

	return s.getDeploymentLocked(db, subscriptionID, resourceGroupName, name)
}

func (s *Store) ListDeployments(subscriptionID, resourceGroupName string) ([]Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.openLocked()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := s.ensureDocumentLocked(db); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, subscription_id, resource_group_name, name, location, mode, template_json, parameters_json, outputs_json, tags_json, provisioning_state, error_code, error_message, created_at, updated_at
FROM deployments
WHERE subscription_id = ? AND resource_group_name = ?
ORDER BY name`, subscriptionID, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		deployment, err := scanDeployment(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deployments: %w", err)
	}
	return deployments, nil
}

func (s *Store) openLocked() (*sql.DB, error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, fmt.Errorf("create state root: %w", err)
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return nil, fmt.Errorf("open state db: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY
);
CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS providers (
    namespace TEXT PRIMARY KEY,
    registration_state TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS blob_containers (
    account_name TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (account_name, name)
);
CREATE TABLE IF NOT EXISTS blobs (
    account_name TEXT NOT NULL,
    container_name TEXT NOT NULL,
    name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    body BLOB NOT NULL,
    etag TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (account_name, container_name, name)
);
CREATE TABLE IF NOT EXISTS storage_queues (
    account_name TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (account_name, name)
);
CREATE TABLE IF NOT EXISTS queue_messages (
    account_name TEXT NOT NULL,
    queue_name TEXT NOT NULL,
    id TEXT NOT NULL,
    message_text TEXT NOT NULL,
    pop_receipt TEXT NOT NULL,
    dequeue_count INTEGER NOT NULL,
    visible_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (account_name, queue_name, id)
);
CREATE TABLE IF NOT EXISTS storage_tables (
    account_name TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (account_name, name)
);
CREATE TABLE IF NOT EXISTS table_entities (
    account_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    row_key TEXT NOT NULL,
    properties_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (account_name, table_name, partition_key, row_key)
);
CREATE TABLE IF NOT EXISTS storage_accounts (
    id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    resource_group_name TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    kind TEXT NOT NULL,
    sku_name TEXT NOT NULL,
    tags_json TEXT NOT NULL,
    provisioning_state TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS key_vaults (
    id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    resource_group_name TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    sku_name TEXT NOT NULL,
    tags_json TEXT NOT NULL,
    provisioning_state TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS key_vault_secrets (
    vault_name TEXT NOT NULL,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    content_type TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (vault_name, name)
);
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    resource_group_name TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    mode TEXT NOT NULL,
    template_json TEXT NOT NULL,
    parameters_json TEXT NOT NULL,
    outputs_json TEXT NOT NULL,
    tags_json TEXT NOT NULL,
    provisioning_state TEXT NOT NULL,
    error_code TEXT NOT NULL,
    error_message TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS operations (
    id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    operation_name TEXT NOT NULL,
    status TEXT NOT NULL,
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS resource_groups (
    id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    tags_json TEXT NOT NULL,
    managed_by TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT '',
    provisioning_state TEXT NOT NULL DEFAULT 'Succeeded'
);`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init state schema: %w", err)
	}
	for _, statement := range []string{
		`ALTER TABLE resource_groups ADD COLUMN subscription_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE resource_groups ADD COLUMN managed_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE resource_groups ADD COLUMN created_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE resource_groups ADD COLUMN updated_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE resource_groups ADD COLUMN provisioning_state TEXT NOT NULL DEFAULT 'Succeeded'`,
	} {
		if _, err := db.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			_ = db.Close()
			return nil, fmt.Errorf("migrate state schema: %w", err)
		}
	}

	return db, nil
}

func (s *Store) ensureDocumentLocked(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM metadata WHERE key = 'version'`).Scan(&count); err != nil {
		return fmt.Errorf("query state metadata: %w", err)
	}
	if count == 0 {
		if err := s.writeLocked(db, newDocument()); err != nil {
			return err
		}
	}
	return s.ensureBootstrapLocked(db)
}

func (s *Store) readLocked(db *sql.DB) (Document, error) {
	if err := s.ensureDocumentLocked(db); err != nil {
		return Document{}, err
	}

	doc := newDocument()
	if err := db.QueryRow(`SELECT value FROM metadata WHERE key = 'version'`).Scan(&doc.Version); err != nil {
		return Document{}, fmt.Errorf("read state version: %w", err)
	}
	if err := db.QueryRow(`SELECT value FROM metadata WHERE key = 'updated_at'`).Scan(&doc.UpdatedAt); err != nil {
		return Document{}, fmt.Errorf("read state updated_at: %w", err)
	}

	rows, err := db.Query(`SELECT id, subscription_id, name, location, tags_json, managed_by, created_at, updated_at, provisioning_state FROM resource_groups`)
	if err != nil {
		return Document{}, fmt.Errorf("read resource groups: %w", err)
	}
	defer rows.Close()

	doc.Resources = map[string]ResourceGroup{}
	for rows.Next() {
		rg, scanErr := scanResourceGroup(rows)
		if scanErr != nil {
			return Document{}, scanErr
		}
		doc.Resources[rg.ID] = rg
	}
	if err := rows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate resource groups: %w", err)
	}

	containerRows, err := db.Query(`
SELECT account_name, name, created_at, updated_at
FROM blob_containers
ORDER BY account_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read blob containers: %w", err)
	}
	defer containerRows.Close()

	for containerRows.Next() {
		var container BlobContainer
		if err := containerRows.Scan(&container.AccountName, &container.Name, &container.CreatedAt, &container.UpdatedAt); err != nil {
			return Document{}, fmt.Errorf("scan blob container: %w", err)
		}
		doc.BlobContainers = append(doc.BlobContainers, container)
	}
	if err := containerRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate blob containers: %w", err)
	}

	blobRows, err := db.Query(`
SELECT account_name, container_name, name, content_type, body, etag, created_at, updated_at
FROM blobs
ORDER BY account_name, container_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read blobs: %w", err)
	}
	defer blobRows.Close()

	for blobRows.Next() {
		var blob BlobObject
		if err := blobRows.Scan(
			&blob.AccountName,
			&blob.ContainerName,
			&blob.Name,
			&blob.ContentType,
			&blob.Body,
			&blob.ETag,
			&blob.CreatedAt,
			&blob.UpdatedAt,
		); err != nil {
			return Document{}, fmt.Errorf("scan blob: %w", err)
		}
		doc.Blobs = append(doc.Blobs, blob)
	}
	if err := blobRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate blobs: %w", err)
	}

	queueRows, err := db.Query(`
SELECT account_name, name, created_at, updated_at
FROM storage_queues
ORDER BY account_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read storage queues: %w", err)
	}
	defer queueRows.Close()

	for queueRows.Next() {
		var queue StorageQueue
		if err := queueRows.Scan(&queue.AccountName, &queue.Name, &queue.CreatedAt, &queue.UpdatedAt); err != nil {
			return Document{}, fmt.Errorf("scan storage queue: %w", err)
		}
		doc.Queues = append(doc.Queues, queue)
	}
	if err := queueRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate storage queues: %w", err)
	}

	queueMessageRows, err := db.Query(`
SELECT account_name, queue_name, id, message_text, pop_receipt, dequeue_count, visible_at, created_at, updated_at
FROM queue_messages
ORDER BY account_name, queue_name, created_at`)
	if err != nil {
		return Document{}, fmt.Errorf("read queue messages: %w", err)
	}
	defer queueMessageRows.Close()

	for queueMessageRows.Next() {
		var message QueueMessage
		if err := queueMessageRows.Scan(
			&message.AccountName,
			&message.QueueName,
			&message.ID,
			&message.MessageText,
			&message.PopReceipt,
			&message.DequeueCount,
			&message.VisibleAt,
			&message.CreatedAt,
			&message.UpdatedAt,
		); err != nil {
			return Document{}, fmt.Errorf("scan queue message: %w", err)
		}
		doc.QueueMessages = append(doc.QueueMessages, message)
	}
	if err := queueMessageRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate queue messages: %w", err)
	}

	tableRows, err := db.Query(`
SELECT account_name, name, created_at, updated_at
FROM storage_tables
ORDER BY account_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read storage tables: %w", err)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var table StorageTable
		if err := tableRows.Scan(&table.AccountName, &table.Name, &table.CreatedAt, &table.UpdatedAt); err != nil {
			return Document{}, fmt.Errorf("scan storage table: %w", err)
		}
		doc.Tables = append(doc.Tables, table)
	}
	if err := tableRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate storage tables: %w", err)
	}

	tableEntityRows, err := db.Query(`
SELECT account_name, table_name, partition_key, row_key, properties_json, created_at, updated_at
FROM table_entities
ORDER BY account_name, table_name, partition_key, row_key`)
	if err != nil {
		return Document{}, fmt.Errorf("read table entities: %w", err)
	}
	defer tableEntityRows.Close()

	for tableEntityRows.Next() {
		entity, err := scanTableEntity(tableEntityRows)
		if err != nil {
			return Document{}, err
		}
		doc.TableEntities = append(doc.TableEntities, entity)
	}
	if err := tableEntityRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate table entities: %w", err)
	}

	storageAccountRows, err := db.Query(`
SELECT id, subscription_id, resource_group_name, name, location, kind, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM storage_accounts
ORDER BY subscription_id, resource_group_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read storage accounts: %w", err)
	}
	defer storageAccountRows.Close()

	for storageAccountRows.Next() {
		account, err := scanStorageAccount(storageAccountRows)
		if err != nil {
			return Document{}, err
		}
		doc.StorageAccounts = append(doc.StorageAccounts, account)
	}
	if err := storageAccountRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate storage accounts: %w", err)
	}

	keyVaultRows, err := db.Query(`
SELECT id, subscription_id, resource_group_name, name, location, tenant_id, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM key_vaults
ORDER BY subscription_id, resource_group_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read key vaults: %w", err)
	}
	defer keyVaultRows.Close()

	for keyVaultRows.Next() {
		vault, err := scanKeyVault(keyVaultRows)
		if err != nil {
			return Document{}, err
		}
		doc.KeyVaults = append(doc.KeyVaults, vault)
	}
	if err := keyVaultRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate key vaults: %w", err)
	}

	keyVaultSecretRows, err := db.Query(`
SELECT vault_name, name, value, content_type, created_at, updated_at
FROM key_vault_secrets
ORDER BY vault_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read key vault secrets: %w", err)
	}
	defer keyVaultSecretRows.Close()

	for keyVaultSecretRows.Next() {
		var secret KeyVaultSecret
		if err := keyVaultSecretRows.Scan(
			&secret.VaultName,
			&secret.Name,
			&secret.Value,
			&secret.ContentType,
			&secret.CreatedAt,
			&secret.UpdatedAt,
		); err != nil {
			return Document{}, fmt.Errorf("scan key vault secret: %w", err)
		}
		doc.KeyVaultSecrets = append(doc.KeyVaultSecrets, secret)
	}
	if err := keyVaultSecretRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate key vault secrets: %w", err)
	}

	deploymentRows, err := db.Query(`
SELECT id, subscription_id, resource_group_name, name, location, mode, template_json, parameters_json, outputs_json, tags_json, provisioning_state, error_code, error_message, created_at, updated_at
FROM deployments
ORDER BY subscription_id, resource_group_name, name`)
	if err != nil {
		return Document{}, fmt.Errorf("read deployments: %w", err)
	}
	defer deploymentRows.Close()

	for deploymentRows.Next() {
		deployment, err := scanDeployment(deploymentRows)
		if err != nil {
			return Document{}, err
		}
		doc.Deployments = append(doc.Deployments, deployment)
	}
	if err := deploymentRows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate deployments: %w", err)
	}

	return doc, nil
}

func (s *Store) ensureBootstrapLocked(db *sql.DB) error {
	if _, err := db.Exec(`
INSERT INTO tenants (id) VALUES (?)
ON CONFLICT(id) DO NOTHING;
INSERT INTO subscriptions (id, tenant_id) VALUES (?, ?)
ON CONFLICT(id) DO NOTHING;
`, defaultTenantID, defaultSubscriptionID, defaultTenantID); err != nil {
		return fmt.Errorf("ensure bootstrap records: %w", err)
	}
	for _, namespace := range defaultProviders {
		if _, err := db.Exec(`
INSERT INTO providers (namespace, registration_state) VALUES (?, 'Registered')
ON CONFLICT(namespace) DO NOTHING;
`, namespace); err != nil {
			return fmt.Errorf("ensure bootstrap provider %s: %w", namespace, err)
		}
	}
	return nil
}

func (s *Store) countLocked(db *sql.DB, table string) (int, error) {
	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return count, nil
}

func (s *Store) writeLocked(db *sql.DB, doc Document) (err error) {
	doc.UpdatedAt = now()
	if doc.Version == "" {
		doc.Version = "foundation-v1"
	}
	if doc.Resources == nil {
		doc.Resources = map[string]ResourceGroup{}
	}
	if doc.BlobContainers == nil {
		doc.BlobContainers = []BlobContainer{}
	}
	if doc.Blobs == nil {
		doc.Blobs = []BlobObject{}
	}
	if doc.Queues == nil {
		doc.Queues = []StorageQueue{}
	}
	if doc.QueueMessages == nil {
		doc.QueueMessages = []QueueMessage{}
	}
	if doc.Tables == nil {
		doc.Tables = []StorageTable{}
	}
	if doc.TableEntities == nil {
		doc.TableEntities = []TableEntity{}
	}
	if doc.StorageAccounts == nil {
		doc.StorageAccounts = []StorageAccount{}
	}
	if doc.KeyVaults == nil {
		doc.KeyVaults = []KeyVault{}
	}
	if doc.KeyVaultSecrets == nil {
		doc.KeyVaultSecrets = []KeyVaultSecret{}
	}
	if doc.Deployments == nil {
		doc.Deployments = []Deployment{}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin state tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM resource_groups`); err != nil {
		err = fmt.Errorf("clear resource groups: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM blobs`); err != nil {
		err = fmt.Errorf("clear blobs: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM queue_messages`); err != nil {
		err = fmt.Errorf("clear queue messages: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM table_entities`); err != nil {
		err = fmt.Errorf("clear table entities: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM storage_tables`); err != nil {
		err = fmt.Errorf("clear storage tables: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM storage_queues`); err != nil {
		err = fmt.Errorf("clear storage queues: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM blob_containers`); err != nil {
		err = fmt.Errorf("clear blob containers: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM storage_accounts`); err != nil {
		err = fmt.Errorf("clear storage accounts: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM key_vaults`); err != nil {
		err = fmt.Errorf("clear key vaults: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM key_vault_secrets`); err != nil {
		err = fmt.Errorf("clear key vault secrets: %w", err)
		return err
	}
	if _, err = tx.Exec(`DELETE FROM deployments`); err != nil {
		err = fmt.Errorf("clear deployments: %w", err)
		return err
	}
	for id, rg := range doc.Resources {
		if rg.ID == "" {
			rg.ID = id
		}
		if rg.Type == "" {
			rg.Type = "Microsoft.Resources/resourceGroups"
		}
		if rg.Tags == nil {
			rg.Tags = map[string]string{}
		}
		if rg.SubscriptionID == "" {
			subscriptionID, resourceGroupName := parseResourceGroupID(rg.ID)
			if subscriptionID != "" {
				rg.SubscriptionID = subscriptionID
			}
			if rg.Name == "" {
				rg.Name = resourceGroupName
			}
		}
		if rg.CreatedAt == "" {
			rg.CreatedAt = doc.UpdatedAt
		}
		rg.UpdatedAt = doc.UpdatedAt
		if rg.ProvisioningState == "" {
			rg.ProvisioningState = "Succeeded"
		}
		tagsJSON, marshalErr := json.Marshal(rg.Tags)
		if marshalErr != nil {
			err = fmt.Errorf("marshal resource group tags: %w", marshalErr)
			return err
		}
		if _, err = tx.Exec(
			`INSERT INTO resource_groups (
id, subscription_id, name, location, tags_json, managed_by, created_at, updated_at, provisioning_state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			rg.ID,
			rg.SubscriptionID,
			rg.Name,
			rg.Location,
			string(tagsJSON),
			rg.ManagedBy,
			rg.CreatedAt,
			rg.UpdatedAt,
			rg.ProvisioningState,
		); err != nil {
			err = fmt.Errorf("insert resource group: %w", err)
			return err
		}
	}
	for _, container := range doc.BlobContainers {
		if container.CreatedAt == "" {
			container.CreatedAt = doc.UpdatedAt
		}
		if container.UpdatedAt == "" {
			container.UpdatedAt = doc.UpdatedAt
		}
		if _, err = tx.Exec(
			`INSERT INTO blob_containers (account_name, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			container.AccountName,
			container.Name,
			container.CreatedAt,
			container.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert blob container: %w", err)
			return err
		}
	}
	for _, blob := range doc.Blobs {
		if blob.ContentType == "" {
			blob.ContentType = "application/octet-stream"
		}
		if blob.ETag == "" {
			blob.ETag = fmt.Sprintf("\"%d\"", time.Now().UTC().UnixNano())
		}
		if blob.CreatedAt == "" {
			blob.CreatedAt = doc.UpdatedAt
		}
		if blob.UpdatedAt == "" {
			blob.UpdatedAt = doc.UpdatedAt
		}
		if _, err = tx.Exec(
			`INSERT INTO blobs (account_name, container_name, name, content_type, body, etag, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			blob.AccountName,
			blob.ContainerName,
			blob.Name,
			blob.ContentType,
			blob.Body,
			blob.ETag,
			blob.CreatedAt,
			blob.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert blob: %w", err)
			return err
		}
	}
	for _, queue := range doc.Queues {
		if queue.CreatedAt == "" {
			queue.CreatedAt = doc.UpdatedAt
		}
		if queue.UpdatedAt == "" {
			queue.UpdatedAt = doc.UpdatedAt
		}
		if _, err = tx.Exec(
			`INSERT INTO storage_queues (account_name, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			queue.AccountName,
			queue.Name,
			queue.CreatedAt,
			queue.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert storage queue: %w", err)
			return err
		}
	}
	for _, message := range doc.QueueMessages {
		if message.PopReceipt == "" {
			message.PopReceipt = fmt.Sprintf("pop-%d", time.Now().UTC().UnixNano())
		}
		if message.VisibleAt == "" {
			message.VisibleAt = doc.UpdatedAt
		}
		if message.CreatedAt == "" {
			message.CreatedAt = doc.UpdatedAt
		}
		if message.UpdatedAt == "" {
			message.UpdatedAt = doc.UpdatedAt
		}
		if _, err = tx.Exec(
			`INSERT INTO queue_messages (account_name, queue_name, id, message_text, pop_receipt, dequeue_count, visible_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			message.AccountName,
			message.QueueName,
			message.ID,
			message.MessageText,
			message.PopReceipt,
			message.DequeueCount,
			message.VisibleAt,
			message.CreatedAt,
			message.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert queue message: %w", err)
			return err
		}
	}
	for _, table := range doc.Tables {
		if table.CreatedAt == "" {
			table.CreatedAt = doc.UpdatedAt
		}
		if table.UpdatedAt == "" {
			table.UpdatedAt = doc.UpdatedAt
		}
		if _, err = tx.Exec(
			`INSERT INTO storage_tables (account_name, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			table.AccountName,
			table.Name,
			table.CreatedAt,
			table.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert storage table: %w", err)
			return err
		}
	}
	for _, entity := range doc.TableEntities {
		if entity.Properties == nil {
			entity.Properties = map[string]any{}
		}
		if entity.CreatedAt == "" {
			entity.CreatedAt = doc.UpdatedAt
		}
		if entity.UpdatedAt == "" {
			entity.UpdatedAt = doc.UpdatedAt
		}
		propertiesJSON, marshalErr := json.Marshal(entity.Properties)
		if marshalErr != nil {
			err = fmt.Errorf("marshal table entity properties: %w", marshalErr)
			return err
		}
		if _, err = tx.Exec(
			`INSERT INTO table_entities (account_name, table_name, partition_key, row_key, properties_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			entity.AccountName,
			entity.TableName,
			entity.PartitionKey,
			entity.RowKey,
			string(propertiesJSON),
			entity.CreatedAt,
			entity.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert table entity: %w", err)
			return err
		}
	}
	for _, account := range doc.StorageAccounts {
		if account.Kind == "" {
			account.Kind = "StorageV2"
		}
		if account.SKUName == "" {
			account.SKUName = "Standard_LRS"
		}
		if account.Tags == nil {
			account.Tags = map[string]string{}
		}
		if account.ProvisioningState == "" {
			account.ProvisioningState = "Succeeded"
		}
		if account.CreatedAt == "" {
			account.CreatedAt = doc.UpdatedAt
		}
		if account.UpdatedAt == "" {
			account.UpdatedAt = doc.UpdatedAt
		}
		tagsJSON, marshalErr := json.Marshal(account.Tags)
		if marshalErr != nil {
			err = fmt.Errorf("marshal storage account tags: %w", marshalErr)
			return err
		}
		if _, err = tx.Exec(
			`INSERT INTO storage_accounts (id, subscription_id, resource_group_name, name, location, kind, sku_name, tags_json, provisioning_state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			account.ID,
			account.SubscriptionID,
			account.ResourceGroupName,
			account.Name,
			account.Location,
			account.Kind,
			account.SKUName,
			string(tagsJSON),
			account.ProvisioningState,
			account.CreatedAt,
			account.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert storage account: %w", err)
			return err
		}
	}
	for _, vault := range doc.KeyVaults {
		if vault.TenantID == "" {
			vault.TenantID = defaultTenantID
		}
		if vault.SKUName == "" {
			vault.SKUName = "standard"
		}
		if vault.Tags == nil {
			vault.Tags = map[string]string{}
		}
		if vault.ProvisioningState == "" {
			vault.ProvisioningState = "Succeeded"
		}
		if vault.CreatedAt == "" {
			vault.CreatedAt = doc.UpdatedAt
		}
		if vault.UpdatedAt == "" {
			vault.UpdatedAt = doc.UpdatedAt
		}
		tagsJSON, marshalErr := json.Marshal(vault.Tags)
		if marshalErr != nil {
			err = fmt.Errorf("marshal key vault tags: %w", marshalErr)
			return err
		}
		if _, err = tx.Exec(
			`INSERT INTO key_vaults (id, subscription_id, resource_group_name, name, location, tenant_id, sku_name, tags_json, provisioning_state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			vault.ID,
			vault.SubscriptionID,
			vault.ResourceGroupName,
			vault.Name,
			vault.Location,
			vault.TenantID,
			vault.SKUName,
			string(tagsJSON),
			vault.ProvisioningState,
			vault.CreatedAt,
			vault.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert key vault: %w", err)
			return err
		}
	}
	for _, secret := range doc.KeyVaultSecrets {
		if secret.CreatedAt == "" {
			secret.CreatedAt = doc.UpdatedAt
		}
		if secret.UpdatedAt == "" {
			secret.UpdatedAt = doc.UpdatedAt
		}
		if _, err = tx.Exec(
			`INSERT INTO key_vault_secrets (vault_name, name, value, content_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			secret.VaultName,
			secret.Name,
			secret.Value,
			secret.ContentType,
			secret.CreatedAt,
			secret.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert key vault secret: %w", err)
			return err
		}
	}
	for _, deployment := range doc.Deployments {
		if deployment.Mode == "" {
			deployment.Mode = "Incremental"
		}
		if deployment.Tags == nil {
			deployment.Tags = map[string]string{}
		}
		if deployment.ProvisioningState == "" {
			deployment.ProvisioningState = "Accepted"
		}
		if deployment.CreatedAt == "" {
			deployment.CreatedAt = doc.UpdatedAt
		}
		if deployment.UpdatedAt == "" {
			deployment.UpdatedAt = doc.UpdatedAt
		}
		tagsJSON, marshalErr := json.Marshal(deployment.Tags)
		if marshalErr != nil {
			err = fmt.Errorf("marshal deployment tags: %w", marshalErr)
			return err
		}
		if _, err = tx.Exec(
			`INSERT INTO deployments (id, subscription_id, resource_group_name, name, location, mode, template_json, parameters_json, outputs_json, tags_json, provisioning_state, error_code, error_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			deployment.ID,
			deployment.SubscriptionID,
			deployment.ResourceGroupName,
			deployment.Name,
			deployment.Location,
			deployment.Mode,
			deployment.TemplateJSON,
			deployment.ParametersJSON,
			deployment.OutputsJSON,
			string(tagsJSON),
			deployment.ProvisioningState,
			deployment.ErrorCode,
			deployment.ErrorMessage,
			deployment.CreatedAt,
			deployment.UpdatedAt,
		); err != nil {
			err = fmt.Errorf("insert deployment: %w", err)
			return err
		}
	}

	if _, err = tx.Exec(`
INSERT INTO metadata (key, value) VALUES ('version', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, doc.Version); err != nil {
		err = fmt.Errorf("write state version: %w", err)
		return err
	}
	if _, err = tx.Exec(`
INSERT INTO metadata (key, value) VALUES ('updated_at', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, doc.UpdatedAt); err != nil {
		err = fmt.Errorf("write state updated_at: %w", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		err = fmt.Errorf("commit state tx: %w", err)
		return err
	}
	return nil
}

func newDocument() Document {
	return Document{
		Version:         "foundation-v1",
		UpdatedAt:       now(),
		Resources:       map[string]ResourceGroup{},
		BlobContainers:  []BlobContainer{},
		Blobs:           []BlobObject{},
		Queues:          []StorageQueue{},
		QueueMessages:   []QueueMessage{},
		Tables:          []StorageTable{},
		TableEntities:   []TableEntity{},
		StorageAccounts: []StorageAccount{},
		KeyVaults:       []KeyVault{},
		KeyVaultSecrets: []KeyVaultSecret{},
		Deployments:     []Deployment{},
	}
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func (s *Store) getResourceGroupLocked(db *sql.DB, subscriptionID, name string) (ResourceGroup, error) {
	row := db.QueryRow(`
SELECT id, subscription_id, name, location, tags_json, managed_by, created_at, updated_at, provisioning_state
FROM resource_groups
WHERE subscription_id = ? AND name = ?`, subscriptionID, name)
	return scanResourceGroup(row)
}

func scanResourceGroup(scanner interface {
	Scan(dest ...any) error
}) (ResourceGroup, error) {
	var rg ResourceGroup
	var tagsJSON string
	if err := scanner.Scan(
		&rg.ID,
		&rg.SubscriptionID,
		&rg.Name,
		&rg.Location,
		&tagsJSON,
		&rg.ManagedBy,
		&rg.CreatedAt,
		&rg.UpdatedAt,
		&rg.ProvisioningState,
	); err != nil {
		return ResourceGroup{}, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &rg.Tags); err != nil {
		return ResourceGroup{}, fmt.Errorf("parse resource group tags: %w", err)
	}
	if rg.Tags == nil {
		rg.Tags = map[string]string{}
	}
	rg.Type = "Microsoft.Resources/resourceGroups"
	return rg, nil
}

func (s *Store) getStorageAccountLocked(db *sql.DB, subscriptionID, resourceGroupName, name string) (StorageAccount, error) {
	row := db.QueryRow(`
SELECT id, subscription_id, resource_group_name, name, location, kind, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM storage_accounts
WHERE subscription_id = ? AND resource_group_name = ? AND name = ?`, subscriptionID, resourceGroupName, name)
	return scanStorageAccount(row)
}

func scanStorageAccount(scanner interface {
	Scan(dest ...any) error
}) (StorageAccount, error) {
	var account StorageAccount
	var tagsJSON string
	if err := scanner.Scan(
		&account.ID,
		&account.SubscriptionID,
		&account.ResourceGroupName,
		&account.Name,
		&account.Location,
		&account.Kind,
		&account.SKUName,
		&tagsJSON,
		&account.ProvisioningState,
		&account.CreatedAt,
		&account.UpdatedAt,
	); err != nil {
		return StorageAccount{}, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &account.Tags); err != nil {
		return StorageAccount{}, fmt.Errorf("parse storage account tags: %w", err)
	}
	if account.Tags == nil {
		account.Tags = map[string]string{}
	}
	return account, nil
}

func (s *Store) getTableEntityLocked(db *sql.DB, accountName, tableName, partitionKey, rowKey string) (TableEntity, error) {
	row := db.QueryRow(`
SELECT account_name, table_name, partition_key, row_key, properties_json, created_at, updated_at
FROM table_entities
WHERE account_name = ? AND table_name = ? AND partition_key = ? AND row_key = ?`,
		accountName, tableName, partitionKey, rowKey,
	)
	return scanTableEntity(row)
}

func scanTableEntity(scanner interface {
	Scan(dest ...any) error
}) (TableEntity, error) {
	var entity TableEntity
	var propertiesJSON string
	if err := scanner.Scan(
		&entity.AccountName,
		&entity.TableName,
		&entity.PartitionKey,
		&entity.RowKey,
		&propertiesJSON,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		return TableEntity{}, err
	}
	if err := json.Unmarshal([]byte(propertiesJSON), &entity.Properties); err != nil {
		return TableEntity{}, fmt.Errorf("parse table entity properties: %w", err)
	}
	if entity.Properties == nil {
		entity.Properties = map[string]any{}
	}
	return entity, nil
}

func (s *Store) getKeyVaultLocked(db *sql.DB, subscriptionID, resourceGroupName, name string) (KeyVault, error) {
	row := db.QueryRow(`
SELECT id, subscription_id, resource_group_name, name, location, tenant_id, sku_name, tags_json, provisioning_state, created_at, updated_at
FROM key_vaults
WHERE subscription_id = ? AND resource_group_name = ? AND name = ?`, subscriptionID, resourceGroupName, name)
	return scanKeyVault(row)
}

func scanKeyVault(scanner interface {
	Scan(dest ...any) error
}) (KeyVault, error) {
	var vault KeyVault
	var tagsJSON string
	if err := scanner.Scan(
		&vault.ID,
		&vault.SubscriptionID,
		&vault.ResourceGroupName,
		&vault.Name,
		&vault.Location,
		&vault.TenantID,
		&vault.SKUName,
		&tagsJSON,
		&vault.ProvisioningState,
		&vault.CreatedAt,
		&vault.UpdatedAt,
	); err != nil {
		return KeyVault{}, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &vault.Tags); err != nil {
		return KeyVault{}, fmt.Errorf("parse key vault tags: %w", err)
	}
	if vault.Tags == nil {
		vault.Tags = map[string]string{}
	}
	return vault, nil
}

func (s *Store) getDeploymentLocked(db *sql.DB, subscriptionID, resourceGroupName, name string) (Deployment, error) {
	row := db.QueryRow(`
SELECT id, subscription_id, resource_group_name, name, location, mode, template_json, parameters_json, outputs_json, tags_json, provisioning_state, error_code, error_message, created_at, updated_at
FROM deployments
WHERE subscription_id = ? AND resource_group_name = ? AND name = ?`, subscriptionID, resourceGroupName, name)
	return scanDeployment(row)
}

func scanDeployment(scanner interface {
	Scan(dest ...any) error
}) (Deployment, error) {
	var deployment Deployment
	var tagsJSON string
	if err := scanner.Scan(
		&deployment.ID,
		&deployment.SubscriptionID,
		&deployment.ResourceGroupName,
		&deployment.Name,
		&deployment.Location,
		&deployment.Mode,
		&deployment.TemplateJSON,
		&deployment.ParametersJSON,
		&deployment.OutputsJSON,
		&tagsJSON,
		&deployment.ProvisioningState,
		&deployment.ErrorCode,
		&deployment.ErrorMessage,
		&deployment.CreatedAt,
		&deployment.UpdatedAt,
	); err != nil {
		return Deployment{}, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &deployment.Tags); err != nil {
		return Deployment{}, fmt.Errorf("parse deployment tags: %w", err)
	}
	if deployment.Tags == nil {
		deployment.Tags = map[string]string{}
	}
	return deployment, nil
}

func parseResourceGroupID(id string) (string, string) {
	parts := strings.Split(strings.Trim(id, "/"), "/")
	if len(parts) >= 4 && strings.EqualFold(parts[0], "subscriptions") && strings.EqualFold(parts[2], "resourceGroups") {
		return parts[1], parts[3]
	}
	return "", ""
}
