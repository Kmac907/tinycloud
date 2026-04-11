package tinyterraformcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Main() {
	os.Exit(Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	code, err := RunE(args, stdin, stdout, stderr, os.Getwd, exec.LookPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return code
}

func RunE(args []string, stdin io.Reader, stdout, stderr io.Writer, getwd func() (string, error), lookPath func(string) (string, error)) (int, error) {
	if runtime.GOOS != "windows" {
		return 1, errors.New("tinyterraform currently supports Windows only")
	}
	args = NormalizeTerraformArgs(args)
	if len(args) == 0 {
		return 2, errors.New("usage: tinyterraform <terraform arguments>")
	}

	subcommand := TerraformSubcommand(args)
	if RequestsTerraformHelp(args) || !RequiresTinyCloudRuntime(subcommand) {
		terraformExe, err := ResolveTerraformExe(lookPath)
		if err != nil {
			return 1, err
		}
		return RunCommand(terraformExe, args, stdin, stdout, stderr)
	}

	powerShellExe, err := ResolvePowerShellExe(lookPath)
	if err != nil {
		return 1, err
	}

	cwd, err := getwd()
	if err != nil {
		return 1, fmt.Errorf("resolve current directory: %w", err)
	}

	scriptPath, err := ResolveTinyTerraformScript(cwd)
	if err != nil {
		return 1, err
	}

	commandArgs := BuildPowerShellCommandArgs(scriptPath, args)
	return RunCommand(powerShellExe, commandArgs, stdin, stdout, stderr)
}

func TerraformSubcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if ConsumesTerraformGlobalArgValue(arg) {
			i++
			continue
		}
		if arg != "" && arg[0] != '-' {
			return arg
		}
	}
	return ""
}

func NormalizeTerraformArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(normalized) == 0 && arg == "--" {
			continue
		}
		if arg == "-chdir=" && i+1 < len(args) {
			normalized = append(normalized, "-chdir="+args[i+1])
			i++
			continue
		}
		normalized = append(normalized, arg)
	}
	return normalized
}

func RequiresTinyCloudRuntime(subcommand string) bool {
	switch subcommand {
	case "", "help", "version", "fmt", "validate", "providers", "state", "output", "show", "graph", "workspace", "force-unlock", "taint", "untaint", "login", "logout", "console":
		return false
	default:
		return true
	}
}

func ConsumesTerraformGlobalArgValue(arg string) bool {
	return arg == "-chdir" || arg == "-chdir="
}

func RequestsTerraformHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-help", "--help", "-h":
			return true
		}
	}
	return false
}

func RunCommand(command string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("run command %q: %w", command, err)
	}
	return 0, nil
}

func ResolveTerraformExe(lookPath func(string) (string, error)) (string, error) {
	if terraformExe := os.Getenv("TERRAFORM_EXE"); terraformExe != "" {
		if _, err := os.Stat(terraformExe); err == nil {
			return terraformExe, nil
		}
	}

	for _, candidate := range []string{"terraform", "terraform.exe"} {
		path, err := lookPath(candidate)
		if err == nil {
			return path, nil
		}
	}

	candidates := []string{
		`C:\Program Files\Terraform\terraform.exe`,
		`C:\HashiCorp\Terraform\terraform.exe`,
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidates = append(candidates,
			filepath.Join(localAppData, `Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe`),
			filepath.Join(localAppData, `Programs\Terraform\terraform.exe`),
		)
	}
	if home := os.Getenv("HOME"); home != "" {
		candidates = append(candidates, filepath.Join(home, `AppData\Local\Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe`))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	userDirs, err := os.ReadDir(`C:\Users`)
	if err == nil {
		for _, userDir := range userDirs {
			if !userDir.IsDir() {
				continue
			}
			candidate := filepath.Join(`C:\Users`, userDir.Name(), `AppData\Local\Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe`)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", errors.New("terraform.exe was not found. Install Terraform or set TERRAFORM_EXE")
}

func ResolvePowerShellExe(lookPath func(string) (string, error)) (string, error) {
	for _, candidate := range []string{"pwsh", "powershell"} {
		path, err := lookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("PowerShell was not found. Install PowerShell or ensure pwsh/powershell is on PATH")
}

func ResolveTinyTerraformScript(cwd string) (string, error) {
	if scriptPath := os.Getenv("TINYTERRAFORM_SCRIPT"); scriptPath != "" {
		if info, err := os.Stat(scriptPath); err == nil && !info.IsDir() {
			return scriptPath, nil
		}
	}

	relativePath := ResolveTinyTerraformScriptRelativePath()
	if sourceRoot := os.Getenv("TINYCLOUD_SOURCE_ROOT"); sourceRoot != "" {
		candidate := filepath.Join(sourceRoot, relativePath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	for _, start := range CandidateSearchRoots(cwd) {
		path, err := FindUpward(start, relativePath)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("could not locate %s from the current workspace", relativePath)
}

func ResolveTinyTerraformScriptRelativePath() string {
	if relativePath := os.Getenv("TINYTERRAFORM_SCRIPT_RELATIVE_PATH"); relativePath != "" {
		return filepath.Clean(relativePath)
	}

	return filepath.Join("scripts", "tinyterraform.ps1")
}

func CandidateSearchRoots(cwd string) []string {
	values := []string{cwd}

	if exePath, err := os.Executable(); err == nil {
		values = append(values, filepath.Dir(exePath))
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		values = append(values, filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")))
	}

	return UniquePaths(values)
}

func FindUpward(start, relativePath string) (string, error) {
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

func BuildPowerShellCommandArgs(scriptPath string, args []string) []string {
	command := "& { param([string]$ScriptPath, [Parameter(ValueFromRemainingArguments=$true)][string[]]$ForwardArgs) & $ScriptPath @ForwardArgs; if ($null -ne $LASTEXITCODE) { exit $LASTEXITCODE } }"
	commandArgs := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command, scriptPath}
	return append(commandArgs, args...)
}

func UniquePaths(values []string) []string {
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
