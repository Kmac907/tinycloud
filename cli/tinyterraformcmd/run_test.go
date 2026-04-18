package tinyterraformcmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestResolveLauncherTinyCloudRuntimeRootUsesNestedRuntimeDirectory(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	got := ResolveLauncherTinyCloudRuntimeRoot(runtimeRoot)
	want := filepath.Join(runtimeRoot, "tinycloud-runtime")
	if got != want {
		t.Fatalf("ResolveLauncherTinyCloudRuntimeRoot() = %q, want %q", got, want)
	}
}

func TestTinyCloudRuntimeEnvUsesIsolatedRuntimeAndDataRoots(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	runtimeRoot := t.TempDir()
	env := TinyCloudRuntimeEnv(repoRoot, runtimeRoot)

	values := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}

	if got := values["TINYCLOUD_RUNTIME_ROOT"]; got != filepath.Join(runtimeRoot, "tinycloud-runtime") {
		t.Fatalf("TINYCLOUD_RUNTIME_ROOT = %q, want %q", got, filepath.Join(runtimeRoot, "tinycloud-runtime"))
	}
	if got := values["TINYCLOUD_DATA_ROOT"]; got != filepath.Join(runtimeRoot, "data") {
		t.Fatalf("TINYCLOUD_DATA_ROOT = %q, want %q", got, filepath.Join(runtimeRoot, "data"))
	}
	if got := values["TINYCLOUD_MGMT_HTTP_PORT"]; got != "4566" {
		t.Fatalf("TINYCLOUD_MGMT_HTTP_PORT = %q, want %q", got, "4566")
	}
	if got := values["TINYCLOUD_MGMT_HTTPS_PORT"]; got != "443" {
		t.Fatalf("TINYCLOUD_MGMT_HTTPS_PORT = %q, want %q", got, "443")
	}
}

func TestParseTerraformEnvRequiresExpectedKeys(t *testing.T) {
	t.Parallel()

	_, err := ParseTerraformEnv("ARM_SUBSCRIPTION_ID=sub\n", []string{"ARM_SUBSCRIPTION_ID", "ARM_TENANT_ID"})
	if err == nil {
		t.Fatal("ParseTerraformEnv() error = nil, want missing-key error")
	}
}

func TestTinyTerraformHostsBlockContainsExpectedMapping(t *testing.T) {
	t.Parallel()

	block := tinyterraformHostsBlock()
	if !strings.Contains(block, "# tinycloud terraform begin") {
		t.Fatalf("tinyterraformHostsBlock() = %q, want start marker", block)
	}
	if !strings.Contains(block, "127.0.0.1 management.azure.com") {
		t.Fatalf("tinyterraformHostsBlock() = %q, want mapping", block)
	}
	if !strings.Contains(block, "# tinycloud terraform end") {
		t.Fatalf("tinyterraformHostsBlock() = %q, want end marker", block)
	}
}

func TestResolveTinyTerraformHostsPathHonorsOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "hosts")
	t.Setenv("TINYTERRAFORM_HOSTS_PATH", override)

	got, err := ResolveTinyTerraformHostsPath()
	if err != nil {
		t.Fatalf("ResolveTinyTerraformHostsPath() error = %v", err)
	}
	if got != override {
		t.Fatalf("ResolveTinyTerraformHostsPath() = %q, want %q", got, override)
	}
}

func TestPowerShellSingleQuotedEscapesEmbeddedQuotes(t *testing.T) {
	t.Parallel()

	got := PowerShellSingleQuoted(`C:\temp\it's\cert.pem`)
	want := `'C:\temp\it''s\cert.pem'`
	if got != want {
		t.Fatalf("PowerShellSingleQuoted() = %q, want %q", got, want)
	}
}

func TestLoadTinyTerraformAzShimScriptIncludesExpectedCommands(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	script, err := LoadTinyTerraformAzShimScript(repoRoot)
	if err != nil {
		t.Fatalf("LoadTinyTerraformAzShimScript() error = %v", err)
	}
	for _, expected := range []string{
		`account" -and $Args[1] -eq "show`,
		`account" -and $Args[1] -eq "get-access-token`,
		`Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:4566/oauth/token"`,
		`unsupported az command`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("LoadTinyTerraformAzShimScript() missing %q", expected)
		}
	}
}

func TestEnsureTinyTerraformAzShimCreatesExpectedFiles(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	shimDir, shimLog, err := EnsureTinyTerraformAzShim(runtimeRoot, repoRoot, func(name string) (string, error) {
		if name != "pwsh" {
			t.Fatalf("ResolvePowerShellExe lookPath asked for %q, want pwsh first", name)
		}
		return `C:\Program Files\PowerShell\7\pwsh.exe`, nil
	})
	if err != nil {
		t.Fatalf("EnsureTinyTerraformAzShim() error = %v", err)
	}
	if got, want := shimDir, filepath.Join(runtimeRoot, "shim"); got != want {
		t.Fatalf("shimDir = %q, want %q", got, want)
	}
	if got, want := shimLog, filepath.Join(runtimeRoot, "azshim.log"); got != want {
		t.Fatalf("shimLog = %q, want %q", got, want)
	}
	for _, path := range []string{filepath.Join(shimDir, "az.cmd"), filepath.Join(shimDir, "azshim.ps1")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected shim file %q: %v", path, err)
		}
	}
	launcher, err := os.ReadFile(filepath.Join(shimDir, "az.cmd"))
	if err != nil {
		t.Fatalf("ReadFile(az.cmd) error = %v", err)
	}
	if !strings.Contains(string(launcher), `pwsh.exe`) || !strings.Contains(string(launcher), `azshim.ps1`) {
		t.Fatalf("az.cmd = %q, want embedded pwsh launcher and azshim target", string(launcher))
	}
	expected, err := LoadTinyTerraformAzShimScript(repoRoot)
	if err != nil {
		t.Fatalf("LoadTinyTerraformAzShimScript() error = %v", err)
	}
	actual, err := os.ReadFile(filepath.Join(shimDir, "azshim.ps1"))
	if err != nil {
		t.Fatalf("ReadFile(azshim.ps1) error = %v", err)
	}
	if got := string(actual); got != expected {
		t.Fatalf("azshim.ps1 = %q, want shared asset contents", got)
	}
}
