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
	Version   string                   `json:"version"`
	UpdatedAt string                   `json:"updatedAt"`
	Resources map[string]ResourceGroup `json:"resources"`
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
	defaultProvider       = "Microsoft.Resources"
)

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
) VALUES (?, ?, ?, ?, ?, '', '', ?, ?)`,
		id, subscriptionID, resourceID, operation, status, nowValue, nowValue,
	); err != nil {
		return Operation{}, fmt.Errorf("create operation: %w", err)
	}

	return Operation{
		ID:             id,
		SubscriptionID: subscriptionID,
		ResourceID:     resourceID,
		Status:         status,
		Operation:      operation,
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

	return doc, nil
}

func (s *Store) ensureBootstrapLocked(db *sql.DB) error {
	if _, err := db.Exec(`
INSERT INTO tenants (id) VALUES (?)
ON CONFLICT(id) DO NOTHING;
INSERT INTO subscriptions (id, tenant_id) VALUES (?, ?)
ON CONFLICT(id) DO NOTHING;
INSERT INTO providers (namespace, registration_state) VALUES (?, 'Registered')
ON CONFLICT(namespace) DO NOTHING;
`, defaultTenantID, defaultSubscriptionID, defaultTenantID, defaultProvider); err != nil {
		return fmt.Errorf("ensure bootstrap records: %w", err)
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
		Version:   "foundation-v1",
		UpdatedAt: now(),
		Resources: map[string]ResourceGroup{},
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

func parseResourceGroupID(id string) (string, string) {
	parts := strings.Split(strings.Trim(id, "/"), "/")
	if len(parts) >= 4 && strings.EqualFold(parts[0], "subscriptions") && strings.EqualFold(parts[2], "resourceGroups") {
		return parts[1], parts[3]
	}
	return "", ""
}
