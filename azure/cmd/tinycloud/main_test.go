package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tinycloud/internal/config"
)

func TestRunEnvTerraformIncludesARMAndBlobSettings(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()
	var stdout bytes.Buffer

	if err := runEnv([]string{"terraform"}, cfg, &stdout); err != nil {
		t.Fatalf("runEnv() error = %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ARM_ENDPOINT=" + cfg.ManagementHTTPURL(),
		"ARM_METADATA_HOST=" + cfg.ManagementHost(),
		"ARM_METADATA_HOSTNAME=" + cfg.ManagementTLSHost(),
		"ARM_MSI_ENDPOINT=" + cfg.ManagedIdentityURL(),
		"ARM_SUBSCRIPTION_ID=" + cfg.SubscriptionID,
		"ARM_TENANT_ID=" + cfg.TenantID,
		"ARM_USE_MSI=true",
		"TINY_BLOB_ENDPOINT=" + cfg.BlobURL(),
		"TINY_APPCONFIG_ENDPOINT=" + cfg.AppConfigURL(),
		"TINY_COSMOS_ENDPOINT=" + cfg.CosmosURL(),
		"TINY_DNS_SERVER=" + cfg.DNSAddress(),
		"TINY_EVENTHUBS_ENDPOINT=" + cfg.EventHubsURL(),
		"TINY_MGMT_HTTPS_CERT=" + cfg.ManagementTLSCertPath(),
		"TINY_OAUTH_TOKEN=" + cfg.OAuthTokenURL(),
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}

func TestRunEnvPulumiIncludesARMAndBlobSettings(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()
	var stdout bytes.Buffer

	if err := runEnv([]string{"pulumi"}, cfg, &stdout); err != nil {
		t.Fatalf("runEnv() error = %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ARM_ENDPOINT=" + cfg.ManagementHTTPURL(),
		"ARM_METADATA_HOST=" + cfg.ManagementHost(),
		"ARM_SUBSCRIPTION_ID=" + cfg.SubscriptionID,
		"ARM_TENANT_ID=" + cfg.TenantID,
		"AZURE_STORAGE_ENDPOINT=" + cfg.BlobURL(),
		"AZURE_APPCONFIG_ENDPOINT=" + cfg.AppConfigURL(),
		"AZURE_COSMOS_ENDPOINT=" + cfg.CosmosURL(),
		"AZURE_PRIVATE_DNS_SERVER=" + cfg.DNSAddress(),
		"AZURE_EVENTHUBS_ENDPOINT=" + cfg.EventHubsURL(),
		"AZURE_OAUTH_TOKEN_URL=" + cfg.OAuthTokenURL(),
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}

func TestRepoRootTinyCloudScriptRunsEnvPulumi(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinycloud script test requires Windows")
	}

	powerShellExe, err := exec.LookPath("pwsh")
	if err != nil {
		powerShellExe, err = exec.LookPath("powershell")
		if err != nil {
			t.Fatalf("resolve PowerShell: %v", err)
		}
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	scriptPath := filepath.Join(repoRoot, "scripts", "tinycloud.ps1")
	runtimeRoot := filepath.Join(t.TempDir(), "tinycloud-runtime")

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "env", "pulumi")
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(azureRoot, ".gocache"),
		"TINYCLOUD_RUNTIME_ROOT="+runtimeRoot,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "tinycloud.exe")); err != nil {
		t.Fatalf("tinycloud.exe was not built in runtime root: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ARM_ENDPOINT=",
		"ARM_METADATA_HOST=",
		"ARM_SUBSCRIPTION_ID=",
		"ARM_TENANT_ID=",
		"AZURE_STORAGE_ENDPOINT=",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}

func TestRepoRootTinyCloudScriptFallsBackToAzureCommandPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinycloud script test requires Windows")
	}

	powerShellExe, err := exec.LookPath("pwsh")
	if err != nil {
		powerShellExe, err = exec.LookPath("powershell")
		if err != nil {
			t.Fatalf("resolve PowerShell: %v", err)
		}
	}

	verifyRoot := t.TempDir()
	scriptDir := filepath.Join(verifyRoot, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(scriptDir) error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	originalScriptPath := filepath.Join(repoRoot, "scripts", "tinycloud.ps1")
	scriptContents, err := os.ReadFile(originalScriptPath)
	if err != nil {
		t.Fatalf("ReadFile(script) error = %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "tinycloud.ps1")
	if err := os.WriteFile(scriptPath, scriptContents, 0o644); err != nil {
		t.Fatalf("WriteFile(script) error = %v", err)
	}

	goWorkContents := "go 1.26\r\n\r\nuse ./azure\r\n"
	if err := os.WriteFile(filepath.Join(verifyRoot, "go.work"), []byte(goWorkContents), 0o644); err != nil {
		t.Fatalf("WriteFile(go.work) error = %v", err)
	}

	azureModuleRoot := filepath.Join(verifyRoot, "azure")
	if err := os.MkdirAll(filepath.Join(azureModuleRoot, "cmd", "tinycloud"), 0o755); err != nil {
		t.Fatalf("MkdirAll(moduleRoot) error = %v", err)
	}
	goModContents := "module tinycloud\r\n\r\ngo 1.26\r\n"
	if err := os.WriteFile(filepath.Join(azureModuleRoot, "go.mod"), []byte(goModContents), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	mainContents := "package main\r\n\r\nimport (\r\n\t\"fmt\"\r\n\t\"os\"\r\n)\r\n\r\nfunc main() {\r\n\tfmt.Println(\"FAKE_TINYCLOUD \" + os.Args[1])\r\n}\r\n"
	if err := os.WriteFile(filepath.Join(azureModuleRoot, "cmd", "tinycloud", "main.go"), []byte(mainContents), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	runtimeRoot := filepath.Join(verifyRoot, "tinycloud-runtime")
	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "sentinel")
	cmd.Dir = verifyRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(verifyRoot, "azure", ".gocache"),
		"TINYCLOUD_RUNTIME_ROOT="+runtimeRoot,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "FAKE_TINYCLOUD sentinel" {
		t.Fatalf("stdout = %q, want %q", got, "FAKE_TINYCLOUD sentinel")
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "tinycloud.exe")); err != nil {
		t.Fatalf("tinycloud.exe was not built in runtime root: %v", err)
	}
}

func TestRepoRootTinyCloudScriptDefaultsGoCacheToRepoRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinycloud script test requires Windows")
	}

	powerShellExe, err := exec.LookPath("pwsh")
	if err != nil {
		powerShellExe, err = exec.LookPath("powershell")
		if err != nil {
			t.Fatalf("resolve PowerShell: %v", err)
		}
	}

	verifyRoot := t.TempDir()
	scriptDir := filepath.Join(verifyRoot, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(scriptDir) error = %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	originalScriptPath := filepath.Join(repoRoot, "scripts", "tinycloud.ps1")
	scriptContents, err := os.ReadFile(originalScriptPath)
	if err != nil {
		t.Fatalf("ReadFile(script) error = %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "tinycloud.ps1")
	if err := os.WriteFile(scriptPath, scriptContents, 0o644); err != nil {
		t.Fatalf("WriteFile(script) error = %v", err)
	}

	goWorkContents := "go 1.26\r\n\r\nuse ./azure\r\n"
	if err := os.WriteFile(filepath.Join(verifyRoot, "go.work"), []byte(goWorkContents), 0o644); err != nil {
		t.Fatalf("WriteFile(go.work) error = %v", err)
	}

	azureModuleRoot := filepath.Join(verifyRoot, "azure")
	if err := os.MkdirAll(filepath.Join(azureModuleRoot, "cmd", "tinycloud"), 0o755); err != nil {
		t.Fatalf("MkdirAll(moduleRoot) error = %v", err)
	}
	goModContents := "module tinycloud\r\n\r\ngo 1.26\r\n"
	if err := os.WriteFile(filepath.Join(azureModuleRoot, "go.mod"), []byte(goModContents), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	mainContents := "package main\r\n\r\nimport \"fmt\"\r\n\r\nfunc main() {\r\n\tfmt.Println(\"FAKE_GOCACHE\")\r\n}\r\n"
	if err := os.WriteFile(filepath.Join(azureModuleRoot, "cmd", "tinycloud", "main.go"), []byte(mainContents), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	runtimeRoot := filepath.Join(verifyRoot, "tinycloud-runtime")
	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Dir = verifyRoot
	env := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(value), "GOCACHE=") {
			env = append(env, value)
		}
	}
	cmd.Env = append(env, "TINYCLOUD_RUNTIME_ROOT="+runtimeRoot)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "FAKE_GOCACHE" {
		t.Fatalf("stdout = %q, want %q", got, "FAKE_GOCACHE")
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "tinycloud.exe")); err != nil {
		t.Fatalf("tinycloud.exe was not built in runtime root: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(verifyRoot, ".gocache"))
	if err != nil {
		t.Fatalf("ReadDir(.gocache) error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("repo-root .gocache was created but empty")
	}
}

func TestRepoRootGoRunTinyCloudEnvPulumi(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("repo-root go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	cmd := exec.Command("go", "run", ".\\azure\\cmd\\tinycloud", "env", "pulumi")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ARM_ENDPOINT=",
		"ARM_METADATA_HOST=",
		"ARM_SUBSCRIPTION_ID=",
		"ARM_TENANT_ID=",
		"AZURE_STORAGE_ENDPOINT=",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}

func TestRepoRootGoRunTopLevelTinyCloudEnvPulumi(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("repo-root go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	cmd := exec.Command("go", "run", ".\\cmd\\tinycloud", "env", "pulumi")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run() error = %v, stderr = %q", err, stderr.String())
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ARM_ENDPOINT=",
		"ARM_METADATA_HOST=",
		"ARM_SUBSCRIPTION_ID=",
		"ARM_TENANT_ID=",
		"AZURE_STORAGE_ENDPOINT=",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}
