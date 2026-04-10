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
