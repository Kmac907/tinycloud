package main

import (
	"io"

	"tinycloud/cli/tinycloudcmd"
	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

func main() {
	tinycloudcmd.Main()
}

func run(args []string, stdout, stderr io.Writer) error {
	return tinycloudcmd.Run(args, stdout, stderr)
}

func runSnapshot(args []string, store *state.Store, cfg config.Config, stdout io.Writer) error {
	return tinycloudcmd.RunSnapshot(args, store, cfg, stdout)
}

func runSeed(args []string, store *state.Store) error {
	return tinycloudcmd.RunSeed(args, store)
}

func runEnv(args []string, cfg config.Config, stdout io.Writer) error {
	return tinycloudcmd.RunEnv(args, cfg, stdout)
}

func printUsage(w io.Writer) {
	tinycloudcmd.PrintUsage(w)
}
