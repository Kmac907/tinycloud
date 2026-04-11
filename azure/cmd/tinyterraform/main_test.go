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

	if got := stdout.String(); !strings.Contains(got, "SHIM_INIT init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init", got)
	}
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

	if got := stdout.String(); !strings.Contains(got, "SHIM_INIT init") {
		t.Fatalf("stdout = %q, want SHIM_INIT init", got)
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "tinycloud.exe")); err != nil {
		t.Fatalf("repo-root default tinyterraform runtime was not created: %v", err)
	}
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
