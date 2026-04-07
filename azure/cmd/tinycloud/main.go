package main

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

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg := config.FromEnv()
	logger := telemetry.NewJSONLogger(stderr)

	if len(args) == 0 {
		printUsage(stdout)
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
		_, err = fmt.Fprintf(stdout, "state=%s\nresources=%d\nupdatedAt=%s\n", summary.StatePath, summary.ResourceCount, summary.UpdatedAt)
		return err
	case "endpoints":
		for name, value := range cfg.EndpointMap() {
			if _, err := fmt.Fprintf(stdout, "%s=%s\n", name, value); err != nil {
				return err
			}
		}
		return nil
	case "env":
		return runEnv(args[1:], cfg, stdout)
	case "snapshot":
		return runSnapshot(args[1:], store, cfg, stdout)
	case "seed":
		return runSeed(args[1:], store)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSnapshot(args []string, store *state.Store, cfg config.Config, stdout io.Writer) error {
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

func runSeed(args []string, store *state.Store) error {
	if len(args) < 2 || args[0] != "apply" {
		return errors.New("usage: tinycloud seed apply <path>")
	}
	return store.ApplySeed(args[1])
}

func runEnv(args []string, cfg config.Config, stdout io.Writer) error {
	if len(args) < 1 {
		return errors.New("env requires a target")
	}

	switch args[0] {
	case "terraform":
		lines := []string{
			fmt.Sprintf("ARM_ENDPOINT=%s", cfg.ManagementHTTPURL()),
			fmt.Sprintf("ARM_METADATA_HOST=%s", cfg.ManagementHost()),
		}
		_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
		return err
	case "pulumi":
		lines := []string{
			fmt.Sprintf("ARM_ENDPOINT=%s", cfg.ManagementHTTPURL()),
			fmt.Sprintf("ARM_METADATA_HOST=%s", cfg.ManagementHost()),
		}
		_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
		return err
	default:
		return fmt.Errorf("unknown env target %q", args[0])
	}
}

func printUsage(w io.Writer) {
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
