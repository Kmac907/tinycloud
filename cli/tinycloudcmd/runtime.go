package tinycloudcmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"tinycloud/runtime/tinycloudconfig"
)

const (
	activeRuntimeFileName = "active-runtime.json"
	runtimeEnvFileName    = "tinycloud.env"
	daemonBinaryName      = "tinycloudd.exe"
	defaultWaitTimeout    = 30 * time.Second
)

type runtimeRecord struct {
	Backend     string                 `json:"backend"`
	PID         int                    `json:"pid,omitempty"`
	StartedAt   string                 `json:"startedAt"`
	RepoRoot    string                 `json:"repoRoot"`
	RuntimeRoot string                 `json:"runtimeRoot"`
	LogPath     string                 `json:"logPath,omitempty"`
	DaemonPath  string                 `json:"daemonPath,omitempty"`
	Detached    bool                   `json:"detached"`
	Env         map[string]string      `json:"env"`
	Config      tinycloudconfig.Config `json:"config"`
	Docker      *dockerRuntime         `json:"docker,omitempty"`
}

func resolveRepoRoot(cwd string) (string, error) {
	if value := os.Getenv("TINYCLOUD_SOURCE_ROOT"); value != "" {
		if info, err := os.Stat(filepath.Join(value, "go.work")); err == nil && !info.IsDir() {
			return value, nil
		}
	}

	for _, start := range candidateSearchRoots(cwd) {
		root, err := findRepoRoot(start)
		if err == nil {
			return root, nil
		}
	}
	return "", errors.New("could not locate the TinyCloud repo root from the current workspace")
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

func findRepoRoot(start string) (string, error) {
	current := filepath.Clean(start)
	for {
		goWork := filepath.Join(current, "go.work")
		cmdPath := filepath.Join(current, "cmd", "tinycloud", "main.go")
		azureGoMod := filepath.Join(current, "azure", "go.mod")
		if fileExists(goWork) && fileExists(cmdPath) && fileExists(azureGoMod) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", os.ErrNotExist
}

func resolveRuntimeRoot(repoRoot string) string {
	if value := os.Getenv("TINYCLOUD_RUNTIME_ROOT"); value != "" {
		return value
	}
	return filepath.Join(repoRoot, ".tinycloud-runtime")
}

func resolveGoWorkdir(repoRoot string) string {
	if value := os.Getenv("TINYCLOUD_GO_WORKDIR"); value != "" {
		return value
	}
	return repoRoot
}

func resolveTinyClouddPackage(repoRoot string) string {
	if fileExists(filepath.Join(repoRoot, "cmd", "tinycloudd", "main.go")) {
		return ".\\cmd\\tinycloudd"
	}
	return ".\\azure\\cmd\\tinycloudd"
}

func buildTinyClouddBinary(repoRoot, runtimeRoot string, env map[string]string) (string, error) {
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		return "", fmt.Errorf("create runtime root: %w", err)
	}
	binaryPath := filepath.Join(runtimeRoot, daemonBinaryName)
	cmd := exec.Command("go", "build", "-o", binaryPath, resolveTinyClouddPackage(repoRoot))
	cmd.Dir = resolveGoWorkdir(repoRoot)
	cmd.Env = inheritEnvWithOverrides(ensureBuildEnv(env, repoRoot))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build tinycloudd: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return binaryPath, nil
}

func ensureBuildEnv(env map[string]string, repoRoot string) map[string]string {
	values := copyMap(env)
	if values["GOCACHE"] == "" {
		values["GOCACHE"] = filepath.Join(repoRoot, ".gocache")
	}
	return values
}

func activeRuntimePath(runtimeRoot string) string {
	return filepath.Join(runtimeRoot, activeRuntimeFileName)
}

func runtimeEnvPath(runtimeRoot string) string {
	return filepath.Join(runtimeRoot, runtimeEnvFileName)
}

func loadRuntimeRecord(runtimeRoot string) (runtimeRecord, error) {
	body, err := os.ReadFile(activeRuntimePath(runtimeRoot))
	if err != nil {
		return runtimeRecord{}, err
	}
	var record runtimeRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return runtimeRecord{}, fmt.Errorf("decode runtime metadata: %w", err)
	}
	if len(record.Env) > 0 {
		record.Config = tinycloudconfig.FromMap(record.Env)
	}
	return record, nil
}

func runtimeRunning(record runtimeRecord) (bool, error) {
	switch record.Backend {
	case "", "process":
		return isProcessRunning(record.PID), nil
	case "docker":
		if record.Docker == nil {
			return false, nil
		}
		return dockerContainerRunning(record.Docker.ContainerName)
	default:
		return false, fmt.Errorf("unsupported runtime backend %q", record.Backend)
	}
}

func saveRuntimeRecord(runtimeRoot string, record runtimeRecord) error {
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		return fmt.Errorf("create runtime root: %w", err)
	}
	body, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode runtime metadata: %w", err)
	}
	return os.WriteFile(activeRuntimePath(runtimeRoot), body, 0o644)
}

func removeRuntimeRecord(runtimeRoot string) error {
	err := os.Remove(activeRuntimePath(runtimeRoot))
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func loadStoredEnv(runtimeRoot string) (map[string]string, error) {
	path := runtimeEnvPath(runtimeRoot)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("open runtime env: %w", err)
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read runtime env: %w", err)
	}
	return values, nil
}

func saveStoredEnv(runtimeRoot string, values map[string]string) error {
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		return fmt.Errorf("create runtime root: %w", err)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.HasPrefix(key, "TINYCLOUD_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var body strings.Builder
	for _, key := range keys {
		body.WriteString(key)
		body.WriteString("=")
		body.WriteString(values[key])
		body.WriteString("\n")
	}
	return os.WriteFile(runtimeEnvPath(runtimeRoot), []byte(body.String()), 0o644)
}

func effectiveEnv(runtimeRoot string) (map[string]string, error) {
	values, err := loadStoredEnv(runtimeRoot)
	if err != nil {
		return nil, err
	}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(key, "TINYCLOUD_") || key == "GOCACHE" {
			values[key] = value
		}
	}
	return values, nil
}

func inheritEnvWithOverrides(overrides map[string]string) []string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func waitForHealthy(cfg tinycloudconfig.Config, timeout time.Duration) error {
	if !cfg.ServiceEnabled(tinycloudconfig.ServiceManagement) {
		return nil
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(cfg.ManagementHTTPURL() + "/_admin/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health returned %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("runtime did not become healthy before timeout")
	}
	return lastErr
}

func readRuntimeStatus(cfg tinycloudconfig.Config) (map[string]any, error) {
	if !cfg.ServiceEnabled(tinycloudconfig.ServiceManagement) {
		return map[string]any{
			"status":   "running",
			"backend":  "process",
			"services": cfg.ServiceCatalog(),
		}, nil
	}

	resp, err := http.Get(cfg.ManagementHTTPURL() + "/_admin/runtime")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("runtime endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false
		}
		text := strings.ToLower(string(output))
		if strings.Contains(text, "no tasks are running") {
			return false
		}
		return strings.Contains(text, fmt.Sprintf(",\"%d\",", pid)) || strings.Contains(text, fmt.Sprintf("\"%d\"", pid))
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func stopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

func formatJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func copyMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
