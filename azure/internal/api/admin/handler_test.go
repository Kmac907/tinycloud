package admin

import (
	"path/filepath"
	"testing"

	"tinycloud/internal/state"
)

func TestResolveDataPathUsesDataRootByDefault(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	handler := NewHandler(store, root)
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

	handler := NewHandler(store, root)
	if _, err := handler.resolveDataPath(filepath.Join("..", "escape.json"), ""); err == nil {
		t.Fatal("resolveDataPath() error = nil, want rejection")
	}
}
