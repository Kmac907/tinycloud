package tinyterraformcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
)

var runtimeExeSeq atomic.Uint64

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
	if subcommand == "init" {
		return RunLocalInit(args, stdin, stdout, stderr, getwd, lookPath)
	}

	powerShellExe, err := ResolvePowerShellExe(lookPath)
	if err != nil {
		return 1, err
	}

	cwd, err := getwd()
	if err != nil {
		return 1, fmt.Errorf("resolve current directory: %w", err)
	}

	terraformDir, err := ResolveTerraformWorkingDir(cwd, args)
	if err != nil {
		return 1, err
	}

	wrapperEnv, cleanup, err := RuntimeWrapperEnv(cwd, terraformDir, lookPath)
	if err != nil {
		return 1, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	scriptPath, err := ResolveTinyTerraformScript(cwd)
	if err != nil {
		return 1, err
	}

	commandArgs := BuildPowerShellCommandArgs(scriptPath, args)
	return RunCommandWithEnv(powerShellExe, commandArgs, wrapperEnv, stdin, stdout, stderr)
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
	return RunCommandWithEnv(command, args, nil, stdin, stdout, stderr)
}

func RunCommandWithEnv(command string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(command, args...)
	if env != nil {
		cmd.Env = env
	}
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

func RuntimeWrapperEnv(cwd, terraformDir string, lookPath func(string) (string, error)) ([]string, func(), error) {
	repoRoot, err := ResolveTinyCloudRepoRoot(cwd)
	if err != nil {
		return nil, nil, err
	}

	terraformExe, err := ResolveTerraformExe(lookPath)
	if err != nil {
		return nil, nil, err
	}

	runtimeRoot := ResolveTinyTerraformRuntimeRoot(repoRoot)
	tinycloudExe, err := BuildTinyCloudExe(repoRoot, runtimeRoot)
	if err != nil {
		return nil, nil, err
	}

	overridePath, cleanup, err := EnsureTerraformOverride(terraformDir)
	if err != nil {
		return nil, nil, err
	}

	return append(
		os.Environ(),
		"TINYTERRAFORM_LAUNCHER_TINYCLOUD_EXE="+tinycloudExe,
		"TINYTERRAFORM_LAUNCHER_TERRAFORM_EXE="+terraformExe,
		"TINYTERRAFORM_LAUNCHER_OVERRIDE_PATH="+overridePath,
	), cleanup, nil
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

func RunLocalInit(args []string, stdin io.Reader, stdout, stderr io.Writer, getwd func() (string, error), lookPath func(string) (string, error)) (int, error) {
	terraformExe, err := ResolveTerraformExe(lookPath)
	if err != nil {
		return 1, err
	}

	cwd, err := getwd()
	if err != nil {
		return 1, fmt.Errorf("resolve current directory: %w", err)
	}

	repoRoot, err := ResolveTinyCloudRepoRoot(cwd)
	if err != nil {
		return 1, err
	}

	runtimeRoot := ResolveTinyTerraformRuntimeRoot(repoRoot)
	tinycloudExe, err := BuildTinyCloudExe(repoRoot, runtimeRoot)
	if err != nil {
		return 1, err
	}

	if code, err := RunCommand(tinycloudExe, []string{"reset"}, nil, stdout, stderr); err != nil || code != 0 {
		if err != nil {
			return 1, fmt.Errorf("reset tinycloud state: %w", err)
		}
		return code, nil
	}

	if code, err := RunCommand(tinycloudExe, []string{"init"}, nil, stdout, stderr); err != nil || code != 0 {
		if err != nil {
			return 1, fmt.Errorf("initialize tinycloud state: %w", err)
		}
		return code, nil
	}

	var envStdout bytes.Buffer
	if code, err := RunCommand(tinycloudExe, []string{"env", "terraform"}, nil, &envStdout, stderr); err != nil || code != 0 {
		if err != nil {
			return 1, fmt.Errorf("load tinycloud terraform environment: %w", err)
		}
		return code, nil
	}

	terraformEnv, err := TerraformInitEnv(envStdout.String())
	if err != nil {
		return 1, err
	}

	return RunTerraformInit(terraformExe, cwd, args, stdin, stdout, stderr, terraformEnv)
}

func ResolveTinyCloudRepoRoot(cwd string) (string, error) {
	if sourceRoot := os.Getenv("TINYCLOUD_SOURCE_ROOT"); sourceRoot != "" {
		cleaned := filepath.Clean(sourceRoot)
		if LooksLikeTinyCloudRepoRoot(cleaned) {
			return cleaned, nil
		}
		parent := filepath.Dir(cleaned)
		if parent != cleaned && LooksLikeTinyCloudRepoRoot(parent) {
			return parent, nil
		}
	}

	for _, start := range CandidateSearchRoots(cwd) {
		current := filepath.Clean(start)
		for {
			if LooksLikeTinyCloudRepoRoot(current) {
				return current, nil
			}

			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}

	return "", errors.New("could not locate the TinyCloud repo root from the current workspace")
}

func LooksLikeTinyCloudRepoRoot(path string) bool {
	if path == "" {
		return false
	}
	if info, err := os.Stat(filepath.Join(path, "go.work")); err != nil || info.IsDir() {
		return false
	}
	if info, err := os.Stat(filepath.Join(path, "cmd", "tinycloud", "main.go")); err != nil || info.IsDir() {
		return false
	}
	if info, err := os.Stat(filepath.Join(path, "azure", "go.mod")); err != nil || info.IsDir() {
		return false
	}
	return true
}

func ResolveTinyCloudGoWorkdir(repoRoot string) string {
	if value := os.Getenv("TINYCLOUD_GO_WORKDIR"); value != "" {
		return filepath.Clean(value)
	}
	return repoRoot
}

func ResolveTinyCloudMainPackage(repoRoot string) string {
	if packageValue := os.Getenv("TINYCLOUD_MAIN_PACKAGE"); packageValue != "" {
		switch filepath.Clean(strings.ReplaceAll(packageValue, "/", `\`)) {
		case `tinycloud\cmd\tinycloud`:
			if topLevelMain := filepath.Join(repoRoot, "cmd", "tinycloud", "main.go"); fileExists(topLevelMain) {
				return topLevelMain
			}
			return filepath.Join(repoRoot, "azure", "cmd", "tinycloud", "main.go")
		default:
			return packageValue
		}
	}

	if topLevelMain := filepath.Join(repoRoot, "cmd", "tinycloud", "main.go"); fileExists(topLevelMain) {
		return topLevelMain
	}
	return filepath.Join(repoRoot, "azure", "cmd", "tinycloud", "main.go")
}

func ResolveTinyTerraformRuntimeRoot(repoRoot string) string {
	if value := os.Getenv("TINYTERRAFORM_RUNTIME_ROOT"); value != "" {
		return value
	}
	return filepath.Join(repoRoot, ".tinyterraform-runtime")
}

func BuildTinyCloudExe(repoRoot, runtimeRoot string) (string, error) {
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		return "", fmt.Errorf("create tinyterraform runtime root: %w", err)
	}

	tinycloudExe := RuntimeExePath(runtimeRoot, "tinycloud")
	goWorkdir := ResolveTinyCloudGoWorkdir(repoRoot)
	mainPackage := ResolveTinyCloudMainPackage(repoRoot)

	cmd := exec.Command("go", "build", "-o", tinycloudExe, mainPackage)
	cmd.Dir = goWorkdir
	cmd.Env = GoBuildEnv(repoRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build tinycloud: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return tinycloudExe, nil
}

func RuntimeExePath(runtimeRoot, base string) string {
	seq := runtimeExeSeq.Add(1)
	return filepath.Join(runtimeRoot, fmt.Sprintf("%s-%d-%d.exe", base, os.Getpid(), seq))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func GoBuildEnv(repoRoot string) []string {
	env := os.Environ()
	if os.Getenv("GOCACHE") != "" {
		return env
	}
	return append(env, "GOCACHE="+filepath.Join(repoRoot, ".gocache"))
}

func ResolveTerraformWorkingDir(cwd string, args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-chdir":
			if i+1 >= len(args) {
				return "", errors.New("terraform -chdir requires a value")
			}
			return resolveTerraformDirValue(cwd, args[i+1]), nil
		case strings.HasPrefix(arg, "-chdir="):
			value := strings.TrimPrefix(arg, "-chdir=")
			if value == "" {
				return "", errors.New("terraform -chdir requires a value")
			}
			return resolveTerraformDirValue(cwd, value), nil
		}
	}
	return cwd, nil
}

func resolveTerraformDirValue(cwd, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(cwd, value))
}

func EnsureTerraformOverride(terraformDir string) (string, func(), error) {
	overridePath := filepath.Join(terraformDir, "tinycloud_providers_override.tf")
	if err := os.MkdirAll(terraformDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create terraform working directory: %w", err)
	}

	overrideBody := strings.Join([]string{
		`provider "azurerm" {`,
		`  features {}`,
		`  use_cli = true`,
		`  resource_provider_registrations = "none"`,
		``,
		`  enhanced_validation {`,
		`    locations = false`,
		`    resource_providers = false`,
		`  }`,
		`}`,
		``,
	}, "\n")
	if err := os.WriteFile(overridePath, []byte(overrideBody), 0o644); err != nil {
		return "", nil, fmt.Errorf("write terraform override: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(overridePath)
	}
	return overridePath, cleanup, nil
}

func TerraformInitEnv(raw string) (map[string]string, error) {
	values := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		values[parts[0]] = parts[1]
	}

	required := []string{"ARM_SUBSCRIPTION_ID", "ARM_TENANT_ID"}
	for _, key := range required {
		if strings.TrimSpace(values[key]) == "" {
			return nil, fmt.Errorf("TinyCloud Terraform environment is missing %s", key)
		}
	}
	return values, nil
}

func RunTerraformInit(terraformExe, cwd string, args []string, stdin io.Reader, stdout, stderr io.Writer, terraformEnv map[string]string) (int, error) {
	clearKeys := map[string]struct{}{
		"ARM_ENDPOINT":          {},
		"ARM_ENVIRONMENT":       {},
		"ARM_METADATA_HOST":     {},
		"ARM_METADATA_HOSTNAME": {},
		"ARM_MSI_ENDPOINT":      {},
		"ARM_USE_MSI":           {},
	}

	env := make([]string, 0, len(os.Environ())+len(terraformEnv))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, skip := clearKeys[key]; skip {
			continue
		}
		env = append(env, entry)
	}
	env = append(env,
		"ARM_SUBSCRIPTION_ID="+terraformEnv["ARM_SUBSCRIPTION_ID"],
		"ARM_TENANT_ID="+terraformEnv["ARM_TENANT_ID"],
	)

	cmd := exec.Command(terraformExe, args...)
	cmd.Dir = cwd
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("run terraform init: %w", err)
	}
	return 0, nil
}
