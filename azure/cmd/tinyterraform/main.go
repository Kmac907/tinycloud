package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	code, err := runE(args, stdin, stdout, stderr, os.Getwd, exec.LookPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return code
}

func runE(args []string, stdin io.Reader, stdout, stderr io.Writer, getwd func() (string, error), lookPath func(string) (string, error)) (int, error) {
	if runtime.GOOS != "windows" {
		return 1, errors.New("tinyterraform currently supports Windows only")
	}
	if len(args) == 0 {
		return 2, errors.New("usage: tinyterraform <terraform arguments>")
	}

	powerShellExe, err := resolvePowerShellExe(lookPath)
	if err != nil {
		return 1, err
	}

	cwd, err := getwd()
	if err != nil {
		return 1, fmt.Errorf("resolve current directory: %w", err)
	}

	scriptPath, err := resolveTinyTerraformScript(cwd)
	if err != nil {
		return 1, err
	}

	commandArgs := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath}, args...)
	cmd := exec.Command(powerShellExe, commandArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("run tinyterraform wrapper: %w", err)
	}

	return 0, nil
}

func resolvePowerShellExe(lookPath func(string) (string, error)) (string, error) {
	for _, candidate := range []string{"pwsh", "powershell"} {
		path, err := lookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("PowerShell was not found. Install PowerShell or ensure pwsh/powershell is on PATH")
}

func resolveTinyTerraformScript(cwd string) (string, error) {
	relativePath := filepath.Join("scripts", "tinyterraform.ps1")
	for _, start := range candidateSearchRoots(cwd) {
		path, err := findUpward(start, relativePath)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("could not locate %s from the current workspace", relativePath)
}

func candidateSearchRoots(cwd string) []string {
	values := []string{cwd}

	if exePath, err := os.Executable(); err == nil {
		values = append(values, filepath.Dir(exePath))
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		values = append(values, filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")))
	}

	return uniquePaths(values)
}

func findUpward(start, relativePath string) (string, error) {
	current := filepath.Clean(start)
	for {
		candidate := filepath.Join(current, relativePath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", os.ErrNotExist
}

func uniquePaths(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}

	for _, value := range values {
		if value == "" {
			continue
		}
		cleaned := filepath.Clean(value)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}

	return result
}
