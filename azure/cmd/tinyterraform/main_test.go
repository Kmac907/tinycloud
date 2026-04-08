package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
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
