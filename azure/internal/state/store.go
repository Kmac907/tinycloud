package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store struct {
	root      string
	statePath string
	mu        sync.Mutex
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
		root:      root,
		statePath: filepath.Join(root, "state.json"),
	}, nil
}

func (s *Store) Init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("create state root: %w", err)
	}

	if _, err := os.Stat(s.statePath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	doc := newDocument()
	return s.writeLocked(doc)
}

func (s *Store) Summary() (Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readLocked()
	if err != nil {
		return Summary{}, err
	}

	return Summary{
		StatePath:     s.statePath,
		ResourceCount: len(doc.Resources),
		UpdatedAt:     doc.UpdatedAt,
	}, nil
}

func (s *Store) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	return s.writeLocked(newDocument())
}

func (s *Store) Snapshot(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readLocked()
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
	doc.UpdatedAt = now()
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("create state root: %w", err)
	}
	return s.writeLocked(doc)
}

func (s *Store) ApplySeed(path string) error {
	return s.Restore(path)
}

func (s *Store) readLocked() (Document, error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return Document{}, err
	}

	body, err := os.ReadFile(s.statePath)
	if errors.Is(err, os.ErrNotExist) {
		doc := newDocument()
		if err := s.writeLocked(doc); err != nil {
			return Document{}, err
		}
		return doc, nil
	}
	if err != nil {
		return Document{}, err
	}

	var doc Document
	if err := json.Unmarshal(body, &doc); err != nil {
		return Document{}, fmt.Errorf("parse state: %w", err)
	}
	if doc.Resources == nil {
		doc.Resources = map[string]ResourceGroup{}
	}
	return doc, nil
}

func (s *Store) writeLocked(doc Document) error {
	doc.UpdatedAt = now()

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.statePath, body, 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
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
