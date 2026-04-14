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

func TestResolveTerraformWorkingDirDefaultsToCurrentDirectory(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	got, err := ResolveTerraformWorkingDir(cwd, []string{"apply", "-auto-approve"})
	if err != nil {
		t.Fatalf("ResolveTerraformWorkingDir() error = %v", err)
	}
	if got != cwd {
		t.Fatalf("ResolveTerraformWorkingDir() = %q, want %q", got, cwd)
	}
}

func TestResolveTerraformWorkingDirHonorsChdirEquals(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	got, err := ResolveTerraformWorkingDir(cwd, []string{"-chdir=examples\\rg", "apply"})
	if err != nil {
		t.Fatalf("ResolveTerraformWorkingDir() error = %v", err)
	}
	want := filepath.Join(cwd, "examples", "rg")
	if got != want {
		t.Fatalf("ResolveTerraformWorkingDir() = %q, want %q", got, want)
	}
}

func TestResolveTerraformWorkingDirHonorsSplitChdir(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	got, err := ResolveTerraformWorkingDir(cwd, []string{"-chdir", "examples\\rg", "apply"})
	if err != nil {
		t.Fatalf("ResolveTerraformWorkingDir() error = %v", err)
	}
	want := filepath.Join(cwd, "examples", "rg")
	if got != want {
		t.Fatalf("ResolveTerraformWorkingDir() = %q, want %q", got, want)
	}
}

func TestEnsureTerraformOverrideCreatesAndCleansUpFile(t *testing.T) {
	t.Parallel()

	terraformDir := filepath.Join(t.TempDir(), "nested", "terraform")
	overridePath, cleanup, err := EnsureTerraformOverride(terraformDir)
	if err != nil {
		t.Fatalf("EnsureTerraformOverride() error = %v", err)
	}
	if filepath.Dir(overridePath) != terraformDir {
		t.Fatalf("EnsureTerraformOverride() path dir = %q, want %q", filepath.Dir(overridePath), terraformDir)
	}
	if _, err := os.Stat(overridePath); err != nil {
		t.Fatalf("override file missing at %q: %v", overridePath, err)
	}
	content, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", overridePath, err)
	}
	if string(content) == "" {
		t.Fatal("override file is empty")
	}
	cleanup()
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Fatalf("override file still exists after cleanup, err = %v", err)
	}
}
