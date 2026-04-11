package tinycloudcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"tinycloud/internal/app"
	"tinycloud/internal/config"
	"tinycloud/internal/state"
	"tinycloud/internal/telemetry"
)

func Main() {
	if err := Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

func Run(args []string, stdout, stderr io.Writer) error {
	cfg := config.FromEnv()
	logger := telemetry.NewJSONLogger(stderr)

	if len(args) == 0 {
		PrintUsage(stdout)
		return nil
	}

	store, err := state.NewStore(cfg.DataRoot)
	if err != nil {
		return fmt.Errorf("init state store: %w", err)
	}

	switch args[0] {
	case "start":
		server := app.NewServer(cfg, store, logger)
		return server.Run(context.Background())
	case "init":
		return store.Init()
	case "reset":
		return store.Reset()
	case "status":
		summary, err := store.Summary()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "state=%s\ntenants=%d\nsubscriptions=%d\nproviders=%d\nresources=%d\nupdatedAt=%s\n",
			summary.StatePath,
			summary.TenantCount,
			summary.SubscriptionCount,
			summary.ProviderCount,
			summary.ResourceCount,
			summary.UpdatedAt,
		)
		return err
	case "endpoints":
		for name, value := range cfg.EndpointMap() {
			if _, err := fmt.Fprintf(stdout, "%s=%s\n", name, value); err != nil {
				return err
			}
		}
		return nil
	case "env":
		return RunEnv(args[1:], cfg, stdout)
	case "snapshot":
		return RunSnapshot(args[1:], store, cfg, stdout)
	case "seed":
		return RunSeed(args[1:], store)
	default:
		PrintUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func RunSnapshot(args []string, store *state.Store, cfg config.Config, stdout io.Writer) error {
	if len(args) < 1 {
		return errors.New("snapshot requires a subcommand")
	}

	switch args[0] {
	case "create":
		path := filepath.Join(cfg.DataRoot, "tinycloud.snapshot.json")
		if len(args) > 1 {
			path = args[1]
		}
		if err := store.Snapshot(path); err != nil {
			return err
		}
		_, err := fmt.Fprintf(stdout, "snapshot=%s\n", path)
		return err
	case "restore":
		if len(args) < 2 {
			return errors.New("snapshot restore requires a path")
		}
		return store.Restore(args[1])
	default:
		return fmt.Errorf("unknown snapshot subcommand %q", args[0])
	}
}

func RunSeed(args []string, store *state.Store) error {
	if len(args) < 2 || args[0] != "apply" {
		return errors.New("usage: tinycloud seed apply <path>")
	}
	return store.ApplySeed(args[1])
}

func RunEnv(args []string, cfg config.Config, stdout io.Writer) error {
	if len(args) < 1 {
		return errors.New("env requires a target")
	}

	switch args[0] {
	case "terraform":
		if err := app.EnsureManagementTLSCertFiles(cfg); err != nil {
			return err
		}
		lines := []string{
			fmt.Sprintf("ARM_ENDPOINT=%s", cfg.ManagementHTTPURL()),
			fmt.Sprintf("ARM_METADATA_HOST=%s", cfg.ManagementHost()),
			fmt.Sprintf("ARM_METADATA_HOSTNAME=%s", cfg.ManagementTLSHost()),
			fmt.Sprintf("ARM_MSI_ENDPOINT=%s", cfg.ManagedIdentityURL()),
			fmt.Sprintf("ARM_SUBSCRIPTION_ID=%s", cfg.SubscriptionID),
			fmt.Sprintf("ARM_TENANT_ID=%s", cfg.TenantID),
			fmt.Sprintf("ARM_USE_MSI=true"),
			fmt.Sprintf("TINY_BLOB_ENDPOINT=%s", cfg.BlobURL()),
			fmt.Sprintf("TINY_APPCONFIG_ENDPOINT=%s", cfg.AppConfigURL()),
			fmt.Sprintf("TINY_COSMOS_ENDPOINT=%s", cfg.CosmosURL()),
			fmt.Sprintf("TINY_DNS_SERVER=%s", cfg.DNSAddress()),
			fmt.Sprintf("TINY_EVENTHUBS_ENDPOINT=%s", cfg.EventHubsURL()),
			fmt.Sprintf("TINY_MGMT_HTTPS_CERT=%s", cfg.ManagementTLSCertPath()),
			fmt.Sprintf("TINY_OAUTH_TOKEN=%s", cfg.OAuthTokenURL()),
		}
		_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
		return err
	case "pulumi":
		lines := []string{
			fmt.Sprintf("ARM_ENDPOINT=%s", cfg.ManagementHTTPURL()),
			fmt.Sprintf("ARM_METADATA_HOST=%s", cfg.ManagementHost()),
			fmt.Sprintf("ARM_SUBSCRIPTION_ID=%s", cfg.SubscriptionID),
			fmt.Sprintf("ARM_TENANT_ID=%s", cfg.TenantID),
			fmt.Sprintf("AZURE_STORAGE_ENDPOINT=%s", cfg.BlobURL()),
			fmt.Sprintf("AZURE_APPCONFIG_ENDPOINT=%s", cfg.AppConfigURL()),
			fmt.Sprintf("AZURE_COSMOS_ENDPOINT=%s", cfg.CosmosURL()),
			fmt.Sprintf("AZURE_PRIVATE_DNS_SERVER=%s", cfg.DNSAddress()),
			fmt.Sprintf("AZURE_EVENTHUBS_ENDPOINT=%s", cfg.EventHubsURL()),
			fmt.Sprintf("AZURE_OAUTH_TOKEN_URL=%s", cfg.OAuthTokenURL()),
		}
		_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
		return err
	default:
		return fmt.Errorf("unknown env target %q", args[0])
	}
}

func PrintUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `tinycloud commands:
  start
  init
  reset
  status
  endpoints
  snapshot create [path]
  snapshot restore <path>
  seed apply <path>
  env terraform
  env pulumi`)
}
