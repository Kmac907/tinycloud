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
