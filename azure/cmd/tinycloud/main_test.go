package main

import (
	"bytes"
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
		"ARM_SUBSCRIPTION_ID=" + cfg.SubscriptionID,
		"ARM_TENANT_ID=" + cfg.TenantID,
		"TINY_BLOB_ENDPOINT=" + cfg.BlobURL(),
		"TINY_APPCONFIG_ENDPOINT=" + cfg.AppConfigURL(),
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
		"AZURE_OAUTH_TOKEN_URL=" + cfg.OAuthTokenURL(),
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}
