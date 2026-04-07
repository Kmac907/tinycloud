package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Location string            `json:"location"`
	Tags     map[string]string `json:"tags"`
}

type Summary struct {
	StatePath     string
	ResourceCount int
	UpdatedAt     string
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

	return Summary{
		StatePath:     s.dbPath,
		ResourceCount: len(doc.Resources),
		UpdatedAt:     doc.UpdatedAt,
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
CREATE TABLE IF NOT EXISTS resource_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    tags_json TEXT NOT NULL
);`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init state schema: %w", err)
	}

	return db, nil
}

func (s *Store) ensureDocumentLocked(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM metadata WHERE key = 'version'`).Scan(&count); err != nil {
		return fmt.Errorf("query state metadata: %w", err)
	}
	if count > 0 {
		return nil
	}
	return s.writeLocked(db, newDocument())
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

	rows, err := db.Query(`SELECT id, name, location, tags_json FROM resource_groups`)
	if err != nil {
		return Document{}, fmt.Errorf("read resource groups: %w", err)
	}
	defer rows.Close()

	doc.Resources = map[string]ResourceGroup{}
	for rows.Next() {
		var rg ResourceGroup
		var tagsJSON string
		if err := rows.Scan(&rg.ID, &rg.Name, &rg.Location, &tagsJSON); err != nil {
			return Document{}, fmt.Errorf("scan resource group: %w", err)
		}
		if err := json.Unmarshal([]byte(tagsJSON), &rg.Tags); err != nil {
			return Document{}, fmt.Errorf("parse resource group tags: %w", err)
		}
		if rg.Tags == nil {
			rg.Tags = map[string]string{}
		}
		doc.Resources[rg.ID] = rg
	}
	if err := rows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate resource groups: %w", err)
	}

	return doc, nil
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
		if rg.Tags == nil {
			rg.Tags = map[string]string{}
		}
		tagsJSON, marshalErr := json.Marshal(rg.Tags)
		if marshalErr != nil {
			err = fmt.Errorf("marshal resource group tags: %w", marshalErr)
			return err
		}
		if _, err = tx.Exec(
			`INSERT INTO resource_groups (id, name, location, tags_json) VALUES (?, ?, ?, ?)`,
			rg.ID,
			rg.Name,
			rg.Location,
			string(tagsJSON),
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
