package main

import (
	"io"

	"tinycloud-root/cli/tinycloudcmd"
	"tinycloud/internal/config"
	"tinycloud/internal/state"
	"tinycloud/runtime/tinycloudazurecmd"
)

func main() {
	tinycloudcmd.Main()
}

func run(args []string, stdout, stderr io.Writer) error {
	return tinycloudazurecmd.Run(args, stdout, stderr)
}

func runSnapshot(args []string, store *state.Store, cfg config.Config, stdout io.Writer) error {
	return tinycloudazurecmd.RunSnapshot(args, store, cfg, stdout)
}

func runSeed(args []string, store *state.Store) error {
	return tinycloudazurecmd.RunSeed(args, store)
}

func runEnv(args []string, cfg config.Config, stdout io.Writer) error {
	return tinycloudazurecmd.RunEnv(args, cfg, stdout)
}

func printUsage(w io.Writer) {
	tinycloudazurecmd.PrintUsage(w)
}
