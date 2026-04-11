package main

import (
	"io"

	"tinycloud/cli/tinyterraformcmd"
)

func main() {
	tinyterraformcmd.Main()
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return tinyterraformcmd.Run(args, stdin, stdout, stderr)
}

func runE(args []string, stdin io.Reader, stdout, stderr io.Writer, getwd func() (string, error), lookPath func(string) (string, error)) (int, error) {
	return tinyterraformcmd.RunE(args, stdin, stdout, stderr, getwd, lookPath)
}

func terraformSubcommand(args []string) string {
	return tinyterraformcmd.TerraformSubcommand(args)
}

func normalizeTerraformArgs(args []string) []string {
	return tinyterraformcmd.NormalizeTerraformArgs(args)
}

func requiresTinyCloudRuntime(subcommand string) bool {
	return tinyterraformcmd.RequiresTinyCloudRuntime(subcommand)
}

func consumesTerraformGlobalArgValue(arg string) bool {
	return tinyterraformcmd.ConsumesTerraformGlobalArgValue(arg)
}

func requestsTerraformHelp(args []string) bool {
	return tinyterraformcmd.RequestsTerraformHelp(args)
}

func runCommand(command string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return tinyterraformcmd.RunCommand(command, args, stdin, stdout, stderr)
}

func resolveTerraformExe(lookPath func(string) (string, error)) (string, error) {
	return tinyterraformcmd.ResolveTerraformExe(lookPath)
}

func resolvePowerShellExe(lookPath func(string) (string, error)) (string, error) {
	return tinyterraformcmd.ResolvePowerShellExe(lookPath)
}

func resolveTinyTerraformScript(cwd string) (string, error) {
	return tinyterraformcmd.ResolveTinyTerraformScript(cwd)
}

func resolveTinyTerraformScriptRelativePath() string {
	return tinyterraformcmd.ResolveTinyTerraformScriptRelativePath()
}

func candidateSearchRoots(cwd string) []string {
	return tinyterraformcmd.CandidateSearchRoots(cwd)
}

func findUpward(start, relativePath string) (string, error) {
	return tinyterraformcmd.FindUpward(start, relativePath)
}

func buildPowerShellCommandArgs(scriptPath string, args []string) []string {
	return tinyterraformcmd.BuildPowerShellCommandArgs(scriptPath, args)
}

func uniquePaths(values []string) []string {
	return tinyterraformcmd.UniquePaths(values)
}
