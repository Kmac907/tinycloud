package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func runtimeExeMatches(t *testing.T, runtimeRoot, prefix string) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(runtimeRoot, prefix+"-*.exe"))
	if err != nil {
		t.Fatalf("Glob(%q) error = %v", filepath.Join(runtimeRoot, prefix+"-*.exe"), err)
	}
	if len(matches) == 0 {
		t.Fatalf("no %s helper executables found under %s", prefix, runtimeRoot)
	}
	return matches
}

func writeRuntimeWrapperProbe(t *testing.T, path string) {
	t.Helper()

	script := `param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_TINYCLOUD_EXE)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_TINYCLOUD_EXE"
    exit 1
}
if (-not (Test-Path $env:TINYTERRAFORM_LAUNCHER_TINYCLOUD_EXE)) {
    Write-Error "missing launcher-built tinycloud helper"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_TERRAFORM_EXE)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_TERRAFORM_EXE"
    exit 1
}
if (-not (Test-Path $env:TINYTERRAFORM_LAUNCHER_TERRAFORM_EXE)) {
    Write-Error "missing launcher-resolved terraform exe"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_DIR)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_AZ_SHIM_DIR"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_LOG)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_AZ_SHIM_LOG"
    exit 1
}
if (-not (Test-Path (Join-Path $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_DIR "az.cmd"))) {
    Write-Error "missing launcher-managed az.cmd"
    exit 1
}
if (-not (Test-Path (Join-Path $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_DIR "azshim.ps1"))) {
    Write-Error "missing launcher-managed azshim.ps1"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_OVERRIDE_PATH)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_OVERRIDE_PATH"
    exit 1
}
if (-not (Test-Path $env:TINYTERRAFORM_LAUNCHER_OVERRIDE_PATH)) {
    Write-Error "missing launcher-created terraform override"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_ARM_SUBSCRIPTION_ID)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_ARM_SUBSCRIPTION_ID"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_ARM_TENANT_ID)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_ARM_TENANT_ID"
    exit 1
}
if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_LAUNCHER_TINY_MGMT_HTTPS_CERT)) {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_TINY_MGMT_HTTPS_CERT"
    exit 1
}
if (-not (Test-Path $env:TINYTERRAFORM_LAUNCHER_TINY_MGMT_HTTPS_CERT)) {
    Write-Error "missing launcher runtime cert"
    exit 1
}
if ($env:TINYTERRAFORM_LAUNCHER_CERT_TRUSTED -ne "1") {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_CERT_TRUSTED"
    exit 1
}
if ($env:TINYTERRAFORM_LAUNCHER_HOSTS_MAPPED -ne "1") {
    Write-Error "missing TINYTERRAFORM_LAUNCHER_HOSTS_MAPPED"
    exit 1
}
$cert = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new($env:TINYTERRAFORM_LAUNCHER_TINY_MGMT_HTTPS_CERT)
$trusted = Get-ChildItem Cert:\CurrentUser\Root | Where-Object { $_.Thumbprint -eq $cert.Thumbprint }
if (-not $trusted) {
    Write-Error "launcher did not trust runtime cert"
    exit 1
}
$env:ARM_SUBSCRIPTION_ID = $env:TINYTERRAFORM_LAUNCHER_ARM_SUBSCRIPTION_ID
$env:ARM_TENANT_ID = $env:TINYTERRAFORM_LAUNCHER_ARM_TENANT_ID
$env:TINYTERRAFORM_AZ_LOG = $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_LOG
$azVersion = & (Join-Path $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_DIR "az.cmd") version
if ($LASTEXITCODE -ne 0) {
    Write-Error "launcher-managed az shim failed version"
    exit 1
}
$azAccount = & (Join-Path $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_DIR "az.cmd") account show
if ($LASTEXITCODE -ne 0) {
    Write-Error "launcher-managed az shim failed account show"
    exit 1
}
$azLog = if (Test-Path $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_LOG) { Get-Content -Raw $env:TINYTERRAFORM_LAUNCHER_AZ_SHIM_LOG } else { "" }
if (-not $azLog.Contains("version") -or -not $azLog.Contains("account show")) {
    Write-Error "launcher-managed az shim log missing expected commands"
    exit 1
}
$hostsPath = if (-not [string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_HOSTS_PATH)) { $env:TINYTERRAFORM_HOSTS_PATH } else { Join-Path $env:SystemRoot "System32\\drivers\\etc\\hosts" }
$hostsContent = Get-Content -Raw $hostsPath
if (-not $hostsContent.Contains("# tinycloud terraform begin")) {
    Write-Error "missing launcher-managed hosts marker"
    exit 1
}
$healthReady = $false
$healthError = $null
for ($attempt = 0; $attempt -lt 10; $attempt++) {
    try {
        Invoke-RestMethod "http://127.0.0.1:4566/_admin/healthz" -TimeoutSec 2 | Out-Null
        $healthReady = $true
        break
    } catch {
        $healthError = $_
        Start-Sleep -Milliseconds 500
    }
}
if (-not $healthReady) {
    Write-Error ("launcher runtime health probe did not become ready: " + $healthError)
    exit 1
}
Write-Output ("WRAPPER_OK terraform=" + $env:TINYTERRAFORM_LAUNCHER_TERRAFORM_EXE + " override=" + $env:TINYTERRAFORM_LAUNCHER_OVERRIDE_PATH + " cert=" + $env:TINYTERRAFORM_LAUNCHER_TINY_MGMT_HTTPS_CERT + " args=" + ($Args -join " "))
`
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestRunERequiresArguments(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code, err := runE(nil, strings.NewReader(""), io.Discard, &stderr, os.Getwd, func(string) (string, error) {
		return "", errors.New("should not be called")
	})
	if err == nil {
		t.Fatal("runE() error = nil, want usage error")
	}
	if code != 2 {
		t.Fatalf("runE() code = %d, want %d", code, 2)
	}
}

func TestResolvePowerShellExePrefersPwsh(t *testing.T) {
	t.Parallel()

	path, err := resolvePowerShellExe(func(name string) (string, error) {
		switch name {
		case "pwsh":
			return `C:\Program Files\PowerShell\7\pwsh.exe`, nil
		default:
			return "", errors.New("not found")
		}
	})
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}
	if path != `C:\Program Files\PowerShell\7\pwsh.exe` {
		t.Fatalf("resolvePowerShellExe() = %q", path)
	}
}

func TestResolvePowerShellExeFallsBackToWindowsPowerShell(t *testing.T) {
	t.Parallel()

	path, err := resolvePowerShellExe(func(name string) (string, error) {
		switch name {
		case "powershell":
			return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, nil
		default:
			return "", errors.New("not found")
		}
	})
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}
	if path != `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe` {
		t.Fatalf("resolvePowerShellExe() = %q", path)
	}
}

func TestFindUpwardFindsWrapperScript(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "tinyterraform.ps1")
	if err := os.WriteFile(scriptPath, []byte("Write-Host test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	nestedDir := filepath.Join(root, "examples", "terraform", "resource-group")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	path, err := findUpward(nestedDir, filepath.Join("scripts", "tinyterraform.ps1"))
	if err != nil {
		t.Fatalf("findUpward() error = %v", err)
	}
	if path != scriptPath {
		t.Fatalf("findUpward() = %q, want %q", path, scriptPath)
	}
}

func TestResolveTinyTerraformScriptHonorsExplicitScriptOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-tinyterraform.ps1")
	if err := os.WriteFile(override, []byte("Write-Host override"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("TINYTERRAFORM_SCRIPT", override)

	path, err := resolveTinyTerraformScript(t.TempDir())
	if err != nil {
		t.Fatalf("resolveTinyTerraformScript() error = %v", err)
	}
	if path != override {
		t.Fatalf("resolveTinyTerraformScript() = %q, want %q", path, override)
	}
}

func TestResolveTinyTerraformScriptHonorsSourceRootOverride(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "scripts", "tinyterraform.ps1")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("Write-Host source-root"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("TINYCLOUD_SOURCE_ROOT", root)

	path, err := resolveTinyTerraformScript(t.TempDir())
	if err != nil {
		t.Fatalf("resolveTinyTerraformScript() error = %v", err)
	}
	if path != scriptPath {
		t.Fatalf("resolveTinyTerraformScript() = %q, want %q", path, scriptPath)
	}
}

func TestResolveTinyTerraformScriptHonorsRelativePathOverride(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "azure", "scripts", "tinyterraform.ps1")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("Write-Host relative-path"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("TINYCLOUD_SOURCE_ROOT", root)
	t.Setenv("TINYTERRAFORM_SCRIPT_RELATIVE_PATH", filepath.Join("azure", "scripts", "tinyterraform.ps1"))

	path, err := resolveTinyTerraformScript(t.TempDir())
	if err != nil {
		t.Fatalf("resolveTinyTerraformScript() error = %v", err)
	}
	if path != scriptPath {
		t.Fatalf("resolveTinyTerraformScript() = %q, want %q", path, scriptPath)
	}
}

func TestResolveTinyTerraformScriptFindsRepoRootWrapperFromRepoRootWorkingDirectory(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	want := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")

	path, err := resolveTinyTerraformScript(repoRoot)
	if err != nil {
		t.Fatalf("resolveTinyTerraformScript() error = %v", err)
	}
	if path != want {
		t.Fatalf("resolveTinyTerraformScript() = %q, want %q", path, want)
	}
}

func TestBuildPowerShellCommandArgsPassesThroughFlags(t *testing.T) {
	t.Parallel()

	args := buildPowerShellCommandArgs(`C:\repo\scripts\tinyterraform.ps1`, []string{"apply", "-auto-approve", "-input=false"})
	expectedPrefix := []string{
		"-NoProfile",
		"-ExecutionPolicy",
		"Bypass",
		"-Command",
		"& { param([string]$ScriptPath, [Parameter(ValueFromRemainingArguments=$true)][string[]]$ForwardArgs) & $ScriptPath @ForwardArgs; if ($null -ne $LASTEXITCODE) { exit $LASTEXITCODE } }",
		`C:\repo\scripts\tinyterraform.ps1`,
		"apply",
		"-auto-approve",
		"-input=false",
	}

	if len(args) != len(expectedPrefix) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expectedPrefix))
	}
	for i, value := range expectedPrefix {
		if args[i] != value {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], value)
		}
	}
}

func TestResolveTerraformExeHonorsEnvironmentOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho terraform shim\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("TERRAFORM_EXE", override)

	path, err := resolveTerraformExe(func(string) (string, error) {
		return "", errors.New("not found")
	})
	if err != nil {
		t.Fatalf("resolveTerraformExe() error = %v", err)
	}
	if path != override {
		t.Fatalf("resolveTerraformExe() = %q, want %q", path, override)
	}
}

func TestTinyTerraformScriptPreservesMachineReadableStdout(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform script test requires Windows")
	}

	powerShellExe, err := resolvePowerShellExe(exec.LookPath)
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	scriptPath := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")

	override := filepath.Join(t.TempDir(), "terraform.cmd")
	shimOutput := `{"terraform_version":"shim","platform":"windows_amd64","provider_selections":{},"terraform_outdated":false}`
	shimScript := "@echo off\r\necho " + shimOutput + "\r\nexit /b 0\r\n"
	if err := os.WriteFile(override, []byte(shimScript), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "version", "-json")
	cmd.Env = append(os.Environ(), "TERRAFORM_EXE="+override)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != shimOutput {
		t.Fatalf("stdout = %q, want %q", got, shimOutput)
	}
}

func TestTinyTerraformScriptHonorsMainPackageOverrideOnInit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform script test requires Windows")
	}

	powerShellExe, err := resolvePowerShellExe(exec.LookPath)
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	scriptPath := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")

	workingDir := t.TempDir()
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho SHIM_INIT %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	goCache := filepath.Join(workingDir, "gocache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "init")
	cmd.Dir = workingDir
	cmd.Env = append(
		os.Environ(),
		"TERRAFORM_EXE="+override,
		"TINYCLOUD_SOURCE_ROOT="+repoRoot,
		"TINYCLOUD_MAIN_PACKAGE=tinycloud/cmd/tinycloud",
		"TINYTERRAFORM_RUNTIME_ROOT="+filepath.Join(workingDir, "tinyterraform-runtime"),
		"GOCACHE="+goCache,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	got := stdout.String()
	if !strings.Contains(got, "SHIM_INIT init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init", got)
	}
	if strings.Contains(got, "Resetting TinyCloud runtime state for terraform init") {
		t.Fatalf("stdout = %q, want launcher-owned init path without wrapper reset message", got)
	}
	runtimeExeMatches(t, filepath.Join(workingDir, "tinyterraform-runtime"), "tinyterraform")
}

func TestTinyTerraformScriptHonorsGoWorkdirOverrideOnInit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform script test requires Windows")
	}

	powerShellExe, err := resolvePowerShellExe(exec.LookPath)
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	scriptPath := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")

	workingDir := t.TempDir()
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho SHIM_INIT %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	goCache := filepath.Join(workingDir, "gocache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	goWorkDir := filepath.Join(workingDir, "gowork")
	if err := os.MkdirAll(goWorkDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	goWorkContents := "go 1.26\r\n\r\nuse " + filepath.ToSlash(repoRoot) + "\r\n"
	if err := os.WriteFile(filepath.Join(goWorkDir, "go.work"), []byte(goWorkContents), 0o644); err != nil {
		t.Fatalf("WriteFile(go.work) error = %v", err)
	}

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "init")
	cmd.Dir = workingDir
	cmd.Env = append(
		os.Environ(),
		"TERRAFORM_EXE="+override,
		"TINYCLOUD_SOURCE_ROOT="+repoRoot,
		"TINYCLOUD_GO_WORKDIR="+goWorkDir,
		"TINYCLOUD_MAIN_PACKAGE=tinycloud/cmd/tinycloud",
		"TINYTERRAFORM_RUNTIME_ROOT="+filepath.Join(workingDir, "tinyterraform-runtime"),
		"GOCACHE="+goCache,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "SHIM_INIT init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init", got)
	}
}

func TestTinyTerraformScriptAutoDetectsSourceRootFromNestedScriptPathOnInit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform script test requires Windows")
	}

	powerShellExe, err := resolvePowerShellExe(exec.LookPath)
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	verifyRoot := filepath.Join(repoRoot, ".verify-auto-source-root")
	t.Cleanup(func() {
		_ = os.RemoveAll(verifyRoot)
	})
	scriptDir := filepath.Join(verifyRoot, "azure", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "tinyterraform.ps1")
	originalScript := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")
	content, err := os.ReadFile(originalScript)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	workingDir := t.TempDir()
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho SHIM_INIT %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	goCache := filepath.Join(workingDir, "gocache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	goWorkDir := filepath.Join(workingDir, "gowork")
	if err := os.MkdirAll(goWorkDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	goWorkContents := "go 1.26\r\n\r\nuse " + filepath.ToSlash(repoRoot) + "\r\n"
	if err := os.WriteFile(filepath.Join(goWorkDir, "go.work"), []byte(goWorkContents), 0o644); err != nil {
		t.Fatalf("WriteFile(go.work) error = %v", err)
	}

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "init")
	cmd.Dir = workingDir
	cmd.Env = append(
		os.Environ(),
		"TERRAFORM_EXE="+override,
		"TINYCLOUD_GO_WORKDIR="+goWorkDir,
		"TINYCLOUD_MAIN_PACKAGE=tinycloud/cmd/tinycloud",
		"TINYTERRAFORM_RUNTIME_ROOT="+filepath.Join(workingDir, "tinyterraform-runtime"),
		"GOCACHE="+goCache,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "SHIM_INIT init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init", got)
	}
}

func TestRepoRootTinyTerraformScriptUsesRepoRootDefaultsOnInit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform script test requires Windows")
	}

	powerShellExe, err := resolvePowerShellExe(exec.LookPath)
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	scriptPath := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")
	runtimeRoot := filepath.Join(repoRoot, ".tinyterraform-runtime")
	_ = os.RemoveAll(runtimeRoot)
	t.Cleanup(func() {
		_ = os.RemoveAll(runtimeRoot)
	})

	workingDir := t.TempDir()
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho SHIM_INIT %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	goCache := filepath.Join(workingDir, "gocache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "init")
	cmd.Dir = workingDir
	cmd.Env = append(
		os.Environ(),
		"TERRAFORM_EXE="+override,
		"GOCACHE="+goCache,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	got := stdout.String()
	if !strings.Contains(got, "SHIM_INIT init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init", got)
	}
	if strings.Contains(got, "Resetting TinyCloud runtime state for terraform init") {
		t.Fatalf("stdout = %q, want launcher-owned init path without wrapper reset message", got)
	}
	runtimeExeMatches(t, runtimeRoot, "tinyterraform")
	runtimeExeMatches(t, runtimeRoot, "tinycloud")
}

func TestRepoRootTinyTerraformScriptDoesNotRequireAzureWrapperForPassthrough(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform script test requires Windows")
	}

	powerShellExe, err := resolvePowerShellExe(exec.LookPath)
	if err != nil {
		t.Fatalf("resolvePowerShellExe() error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	originalScriptPath := filepath.Join(repoRoot, "scripts", "tinyterraform.ps1")

	verifyRoot := t.TempDir()
	scriptDir := filepath.Join(verifyRoot, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	scriptContents, err := os.ReadFile(originalScriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "tinyterraform.ps1")
	if err := os.WriteFile(scriptPath, scriptContents, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sourceRoot := filepath.Join(verifyRoot, "azure", "cmd", "tinycloud")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.go"), []byte("package main\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	override := filepath.Join(verifyRoot, "terraform.cmd")
	shimOutput := `{"terraform_version":"shim","platform":"windows_amd64","provider_selections":{},"terraform_outdated":false}`
	shimScript := "@echo off\r\necho " + shimOutput + "\r\nexit /b 0\r\n"
	if err := os.WriteFile(override, []byte(shimScript), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "version", "-json")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "TERRAFORM_EXE="+override)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != shimOutput {
		t.Fatalf("stdout = %q, want %q", got, shimOutput)
	}
}

func TestRepoRootGoRunTinyTerraformVersionJSON(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	override := filepath.Join(t.TempDir(), "terraform.cmd")
	shimOutput := `{"terraform_version":"shim","platform":"windows_amd64","provider_selections":{},"terraform_outdated":false}`
	shimScript := "@echo off\r\necho " + shimOutput + "\r\nexit /b 0\r\n"
	if err := os.WriteFile(override, []byte(shimScript), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	cmd := exec.Command("go", "run", ".\\azure\\cmd\\tinyterraform", "--", "version", "-json")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"TERRAFORM_EXE="+override,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != shimOutput {
		t.Fatalf("stdout = %q, want %q", got, shimOutput)
	}
}

func TestRepoRootGoRunTopLevelTinyTerraformVersionJSON(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	override := filepath.Join(t.TempDir(), "terraform.cmd")
	shimOutput := `{"terraform_version":"shim","platform":"windows_amd64","provider_selections":{},"terraform_outdated":false}`
	shimScript := "@echo off\r\necho " + shimOutput + "\r\nexit /b 0\r\n"
	if err := os.WriteFile(override, []byte(shimScript), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	cmd := exec.Command("go", "run", ".\\cmd\\tinyterraform", "--", "version", "-json")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"TERRAFORM_EXE="+override,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != shimOutput {
		t.Fatalf("stdout = %q, want %q", got, shimOutput)
	}
}

func TestRepoRootGoRunTopLevelTinyTerraformInitDoesNotRequireScript(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	workingDir := t.TempDir()
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho SHIM_INIT %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	runtimeRoot := filepath.Join(workingDir, "tinyterraform-runtime")
	cmd := exec.Command("go", "run", ".\\cmd\\tinyterraform", "--", "-chdir="+workingDir, "init")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(workingDir, "gocache"),
		"TERRAFORM_EXE="+override,
		"TINYTERRAFORM_RUNTIME_ROOT="+runtimeRoot,
		"TINYTERRAFORM_SCRIPT="+filepath.Join(workingDir, "missing-tinyterraform.ps1"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "SHIM_INIT -chdir="+workingDir+" init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init output", got)
	}
	runtimeExeMatches(t, runtimeRoot, "tinycloud")
}

func TestRepoRootGoRunAzureTinyTerraformInitDoesNotRequireScript(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	workingDir := t.TempDir()
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho SHIM_INIT %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	runtimeRoot := filepath.Join(workingDir, "tinyterraform-runtime")
	cmd := exec.Command("go", "run", ".\\azure\\cmd\\tinyterraform", "--", "-chdir="+workingDir, "init")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(workingDir, "gocache"),
		"TERRAFORM_EXE="+override,
		"TINYTERRAFORM_RUNTIME_ROOT="+runtimeRoot,
		"TINYTERRAFORM_SCRIPT="+filepath.Join(workingDir, "missing-tinyterraform.ps1"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "SHIM_INIT -chdir="+workingDir+" init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init output", got)
	}
	runtimeExeMatches(t, runtimeRoot, "tinycloud")
}

func TestRepoRootGoRunTopLevelTinyTerraformApplyBuildsTinyCloudBeforeWrapper(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	workingDir := t.TempDir()
	scriptPath := filepath.Join(workingDir, "probe-wrapper.ps1")
	writeRuntimeWrapperProbe(t, scriptPath)
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho PROBE_TERRAFORM %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	runtimeRoot := filepath.Join(workingDir, "tinyterraform-runtime")
	hostsPath := filepath.Join(workingDir, "hosts")
	if err := os.WriteFile(hostsPath, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile(hosts) error = %v", err)
	}
	cmd := exec.Command("go", "run", ".\\cmd\\tinyterraform", "--", "-chdir="+workingDir, "apply", "-auto-approve")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(workingDir, "gocache"),
		"TERRAFORM_EXE="+override,
		"TINYTERRAFORM_RUNTIME_ROOT="+runtimeRoot,
		"TINYTERRAFORM_SCRIPT="+scriptPath,
		"TINYTERRAFORM_HOSTS_PATH="+hostsPath,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	got := stdout.String()
	overridePath := filepath.Join(workingDir, "tinycloud_providers_override.tf")
	if !strings.Contains(got, "WRAPPER_OK terraform="+override+" override="+overridePath+" cert=") || !strings.Contains(got, "apply -auto-approve") {
		t.Fatalf("stdout = %q, want WRAPPER_OK terraform override-path cert apply output", got)
	}
	runtimeExeMatches(t, runtimeRoot, "tinycloud")
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Fatalf("override file still exists after launcher cleanup, err = %v", err)
	}
}

func TestRepoRootGoRunAzureTinyTerraformApplyBuildsTinyCloudBeforeWrapper(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinyterraform go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	workingDir := t.TempDir()
	scriptPath := filepath.Join(workingDir, "probe-wrapper.ps1")
	writeRuntimeWrapperProbe(t, scriptPath)
	override := filepath.Join(workingDir, "terraform.cmd")
	if err := os.WriteFile(override, []byte("@echo off\r\necho PROBE_TERRAFORM %*\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(terraform.cmd) error = %v", err)
	}

	runtimeRoot := filepath.Join(workingDir, "tinyterraform-runtime")
	hostsPath := filepath.Join(workingDir, "hosts")
	if err := os.WriteFile(hostsPath, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile(hosts) error = %v", err)
	}
	cmd := exec.Command("go", "run", ".\\azure\\cmd\\tinyterraform", "--", "-chdir="+workingDir, "apply", "-auto-approve")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(workingDir, "gocache"),
		"TERRAFORM_EXE="+override,
		"TINYTERRAFORM_RUNTIME_ROOT="+runtimeRoot,
		"TINYTERRAFORM_SCRIPT="+scriptPath,
		"TINYTERRAFORM_HOSTS_PATH="+hostsPath,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	got := stdout.String()
	overridePath := filepath.Join(workingDir, "tinycloud_providers_override.tf")
	if !strings.Contains(got, "WRAPPER_OK terraform="+override+" override="+overridePath+" cert=") || !strings.Contains(got, "apply -auto-approve") {
		t.Fatalf("stdout = %q, want WRAPPER_OK terraform override-path cert apply output", got)
	}
	runtimeExeMatches(t, runtimeRoot, "tinycloud")
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Fatalf("override file still exists after launcher cleanup, err = %v", err)
	}
}

func TestTerraformSubcommandSkipsFlags(t *testing.T) {
	t.Parallel()

	value := terraformSubcommand([]string{"-chdir=examples", "apply", "-auto-approve"})
	if value != "apply" {
		t.Fatalf("terraformSubcommand() = %q, want %q", value, "apply")
	}
}

func TestTerraformSubcommandSkipsGlobalFlagValues(t *testing.T) {
	t.Parallel()

	value := terraformSubcommand([]string{"-chdir", ".", "version", "-json"})
	if value != "version" {
		t.Fatalf("terraformSubcommand() = %q, want %q", value, "version")
	}

	value = terraformSubcommand([]string{"-chdir=", ".", "version", "-json"})
	if value != "version" {
		t.Fatalf("terraformSubcommand() with PowerShell-split -chdir= = %q, want %q", value, "version")
	}
}

func TestNormalizeTerraformArgsDropsGoRunSeparator(t *testing.T) {
	t.Parallel()

	args := normalizeTerraformArgs([]string{"--", "version", "-json"})
	expected := []string{"version", "-json"}

	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expected))
	}
	for i, value := range expected {
		if args[i] != value {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], value)
		}
	}
}

func TestNormalizeTerraformArgsRejoinsPowerShellSplitChdirEquals(t *testing.T) {
	t.Parallel()

	args := normalizeTerraformArgs([]string{"-chdir=", ".", "version", "-json"})
	expected := []string{"-chdir=.", "version", "-json"}

	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expected))
	}
	for i, value := range expected {
		if args[i] != value {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], value)
		}
	}
}

func TestRequiresTinyCloudRuntime(t *testing.T) {
	t.Parallel()

	if requiresTinyCloudRuntime("help") {
		t.Fatal("requiresTinyCloudRuntime(help) = true, want false")
	}
	if requiresTinyCloudRuntime("login") {
		t.Fatal("requiresTinyCloudRuntime(login) = true, want false")
	}
	if requiresTinyCloudRuntime("logout") {
		t.Fatal("requiresTinyCloudRuntime(logout) = true, want false")
	}
	if requiresTinyCloudRuntime("console") {
		t.Fatal("requiresTinyCloudRuntime(console) = true, want false")
	}
	if requiresTinyCloudRuntime("version") {
		t.Fatal("requiresTinyCloudRuntime(version) = true, want false")
	}
	if !requiresTinyCloudRuntime("apply") {
		t.Fatal("requiresTinyCloudRuntime(apply) = false, want true")
	}
}

func TestRequestsTerraformHelp(t *testing.T) {
	t.Parallel()

	if !requestsTerraformHelp([]string{"apply", "-help"}) {
		t.Fatal("requestsTerraformHelp(apply -help) = false, want true")
	}
	if !requestsTerraformHelp([]string{"-h"}) {
		t.Fatal("requestsTerraformHelp(-h) = false, want true")
	}
	if requestsTerraformHelp([]string{"apply", "-auto-approve"}) {
		t.Fatal("requestsTerraformHelp(apply -auto-approve) = true, want false")
	}
}

func TestConsumesTerraformGlobalArgValue(t *testing.T) {
	t.Parallel()

	if !consumesTerraformGlobalArgValue("-chdir") {
		t.Fatal("consumesTerraformGlobalArgValue(-chdir) = false, want true")
	}
	if !consumesTerraformGlobalArgValue("-chdir=") {
		t.Fatal("consumesTerraformGlobalArgValue(-chdir=) = false, want true")
	}
	if consumesTerraformGlobalArgValue("-chdir=examples") {
		t.Fatal("consumesTerraformGlobalArgValue(-chdir=examples) = true, want false")
	}
}
