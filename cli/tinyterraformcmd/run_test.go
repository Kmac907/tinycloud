package tinyterraformcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTinyCloudMainPackagePrefersTopLevelMainFile(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	topLevelMain := filepath.Join(repoRoot, "cmd", "tinycloud", "main.go")
	if err := os.MkdirAll(filepath.Dir(topLevelMain), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(topLevelMain, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if got := ResolveTinyCloudMainPackage(repoRoot); got != topLevelMain {
		t.Fatalf("ResolveTinyCloudMainPackage() = %q, want %q", got, topLevelMain)
	}
}

func TestResolveTinyCloudMainPackageFallsBackToAzureMainFile(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	azureMain := filepath.Join(repoRoot, "azure", "cmd", "tinycloud", "main.go")
	if err := os.MkdirAll(filepath.Dir(azureMain), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(azureMain, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if got := ResolveTinyCloudMainPackage(repoRoot); got != azureMain {
		t.Fatalf("ResolveTinyCloudMainPackage() = %q, want %q", got, azureMain)
	}
}
