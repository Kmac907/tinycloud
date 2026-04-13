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

func TestRuntimeExePathUsesUniqueNames(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	first := RuntimeExePath(runtimeRoot, "tinycloud")
	second := RuntimeExePath(runtimeRoot, "tinycloud")

	if first == second {
		t.Fatalf("RuntimeExePath() returned duplicate paths: %q", first)
	}
	if filepath.Dir(first) != runtimeRoot {
		t.Fatalf("RuntimeExePath() dir = %q, want %q", filepath.Dir(first), runtimeRoot)
	}
	if filepath.Dir(second) != runtimeRoot {
		t.Fatalf("RuntimeExePath() dir = %q, want %q", filepath.Dir(second), runtimeRoot)
	}
	if filepath.Base(first) == "tinycloud.exe" || filepath.Base(second) == "tinycloud.exe" {
		t.Fatalf("RuntimeExePath() should use collision-resistant file names, got %q and %q", filepath.Base(first), filepath.Base(second))
	}
}
