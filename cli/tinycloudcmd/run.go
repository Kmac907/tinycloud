package tinycloudcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tinycloud/runtime/tinycloudazurecmd"
	"tinycloud/runtime/tinycloudconfig"
)

type cliContext struct {
	cwd         string
	repoRoot    string
	runtimeRoot string
	env         map[string]string
	config      tinycloudconfig.Config
}

func Main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

func Run(args []string, stdout, stderr io.Writer) int {
	code, err := RunE(args, stdout, stderr, os.Getwd)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return code
}

func RunE(args []string, stdout, stderr io.Writer, getwd func() (string, error)) (int, error) {
	if len(args) == 0 {
		PrintUsage(stdout)
		return 0, nil
	}

	ctx, err := loadCLIContext(getwd)
	if err != nil {
		return 1, err
	}

	switch args[0] {
	case "start":
		return runStart(ctx, args[1:], stdout, stderr)
	case "stop":
		return runStop(ctx, stdout)
	case "restart":
		return runRestart(ctx, args[1:], stdout, stderr)
	case "wait":
		return runWait(ctx, args[1:], stdout)
	case "logs":
		return runLogs(ctx, args[1:], stdout)
	case "status":
		return runStatus(ctx, args[1:], stdout)
	case "config":
		return runConfig(ctx, args[1:], stdout)
	case "services":
		return runServices(ctx, args[1:], stdout)
	case "endpoints":
		return runEndpoints(ctx, args[1:], stdout)
	case "init", "reset", "snapshot", "seed", "env":
		if err := tinycloudazurecmd.Run(args, stdout, stderr); err != nil {
			return 1, err
		}
		return 0, nil
	default:
		PrintUsage(stderr)
		return 2, fmt.Errorf("unknown command %q", args[0])
	}
}

func loadCLIContext(getwd func() (string, error)) (cliContext, error) {
	cwd, err := getwd()
	if err != nil {
		return cliContext{}, fmt.Errorf("resolve current directory: %w", err)
	}
	repoRoot, err := resolveRepoRoot(cwd)
	if err != nil {
		return cliContext{}, err
	}
	runtimeRoot := resolveRuntimeRoot(repoRoot)
	env, err := effectiveEnv(runtimeRoot)
	if err != nil {
		return cliContext{}, err
	}
	env = ensureBuildEnv(env, repoRoot)

	cfg := tinycloudconfig.FromMap(env)
	if !filepath.IsAbs(cfg.DataRoot) {
		cfg.DataRoot = filepath.Clean(filepath.Join(cwd, cfg.DataRoot))
		env["TINYCLOUD_DATA_ROOT"] = cfg.DataRoot
	}

	return cliContext{
		cwd:         cwd,
		repoRoot:    repoRoot,
		runtimeRoot: runtimeRoot,
		env:         env,
		config:      cfg,
	}, nil
}

func runStart(ctx cliContext, args []string, stdout, stderr io.Writer) (int, error) {
	detached := false
	jsonOutput := false
	servicesOverride := ""
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--detached" || arg == "-d":
			detached = true
		case arg == "--json":
			jsonOutput = true
		case arg == "--attached":
			detached = false
		case strings.HasPrefix(arg, "--services="):
			servicesOverride = strings.TrimSpace(strings.TrimPrefix(arg, "--services="))
		case arg == "--services":
			if i+1 >= len(args) {
				return 2, errors.New("start --services requires a value")
			}
			i++
			servicesOverride = strings.TrimSpace(args[i])
		default:
			return 2, fmt.Errorf("unknown start option %q", arg)
		}
	}

	if record, ok, err := activeRuntime(ctx.runtimeRoot); err == nil && ok && isProcessRunning(record.PID) {
		return 1, fmt.Errorf("TinyCloud is already running with PID %d", record.PID)
	}

	runCtx := ctx
	runCtx.env = copyMap(ctx.env)
	if servicesOverride != "" {
		runCtx.env["TINYCLOUD_SERVICES"] = servicesOverride
		runCtx.config = tinycloudconfig.FromMap(runCtx.env)
		if !filepath.IsAbs(runCtx.config.DataRoot) {
			runCtx.config.DataRoot = filepath.Clean(filepath.Join(ctx.cwd, runCtx.config.DataRoot))
			runCtx.env["TINYCLOUD_DATA_ROOT"] = runCtx.config.DataRoot
		}
	}
	if err := runCtx.config.Validate(); err != nil {
		return 1, err
	}
	if err := runCtx.config.RequireServices(); err != nil {
		return 1, err
	}

	binaryPath, err := buildTinyClouddBinary(runCtx.repoRoot, runCtx.runtimeRoot, runCtx.env)
	if err != nil {
		return 1, err
	}
	logPath := filepath.Join(runCtx.runtimeRoot, "tinycloudd.log")

	if detached {
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return 1, fmt.Errorf("open runtime log: %w", err)
		}
		defer logFile.Close()

		cmd := exec.Command(binaryPath)
		cmd.Dir = runCtx.repoRoot
		cmd.Env = inheritEnvWithOverrides(runCtx.env)
		cmd.Stdout = logFile
		cmd.Stderr = logFile

		if err := cmd.Start(); err != nil {
			return 1, fmt.Errorf("start tinycloudd: %w", err)
		}

		record := runtimeRecord{
			Backend:     "process",
			PID:         cmd.Process.Pid,
			StartedAt:   time.Now().UTC().Format(time.RFC3339Nano),
			RepoRoot:    runCtx.repoRoot,
			RuntimeRoot: runCtx.runtimeRoot,
			LogPath:     logPath,
			DaemonPath:  binaryPath,
			Detached:    true,
			Env:         copyMap(runCtx.env),
			Config:      runCtx.config,
		}
		if err := saveRuntimeRecord(runCtx.runtimeRoot, record); err != nil {
			_ = stopProcess(cmd.Process.Pid)
			return 1, err
		}
		if err := waitForHealthy(runCtx.config, defaultWaitTimeout); err != nil {
			_ = stopProcess(cmd.Process.Pid)
			_ = removeRuntimeRecord(runCtx.runtimeRoot)
			return 1, fmt.Errorf("wait for runtime: %w", err)
		}

		summary := map[string]any{
			"status":       "running",
			"runtimeId":    fmt.Sprintf("process:%d", cmd.Process.Pid),
			"backend":      "process",
			"pid":          cmd.Process.Pid,
			"services":     runCtx.config.EnabledServices(),
			"logPath":      logPath,
			"endpoints":    runCtx.config.EndpointMap(),
			"nextCommands": []string{"tinycloud status runtime", "tinycloud logs -f", "tinycloud stop"},
		}
		if runCtx.config.ServiceEnabled(tinycloudconfig.ServiceManagement) {
			summary["management"] = runCtx.config.ManagementHTTPURL()
		}
		if jsonOutput {
			return 0, formatJSON(stdout, summary)
		}
		_, err = fmt.Fprintf(stdout, "runtime=running\nruntimeId=process:%d\nbackend=process\npid=%d\nservices=%s\nlog=%s\n", cmd.Process.Pid, cmd.Process.Pid, joinServices(runCtx.config.EnabledServices()), logPath)
		if runCtx.config.ServiceEnabled(tinycloudconfig.ServiceManagement) {
			_, err = fmt.Fprintf(stdout, "management=%s\n", runCtx.config.ManagementHTTPURL())
		}
		for _, endpoint := range sortedEndpointLines(runCtx.config.EndpointMap()) {
			_, err = fmt.Fprintln(stdout, endpoint)
		}
		_, err = fmt.Fprintln(stdout, "next=tinycloud status runtime")
		_, err = fmt.Fprintln(stdout, "next=tinycloud logs -f")
		_, err = fmt.Fprintln(stdout, "next=tinycloud stop")
		return 0, err
	}

	record := runtimeRecord{
		Backend:     "process",
		StartedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		RepoRoot:    runCtx.repoRoot,
		RuntimeRoot: runCtx.runtimeRoot,
		LogPath:     logPath,
		DaemonPath:  binaryPath,
		Detached:    false,
		Env:         copyMap(runCtx.env),
		Config:      runCtx.config,
	}

	cmd := exec.Command(binaryPath)
	cmd.Dir = runCtx.repoRoot
	cmd.Env = inheritEnvWithOverrides(runCtx.env)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if jsonOutput {
		if err := formatJSON(stdout, map[string]any{
			"status":   "starting",
			"backend":  "process",
			"services": runCtx.config.EnabledServices(),
		}); err != nil {
			return 1, err
		}
	} else {
		_, _ = fmt.Fprintf(stdout, "runtime=starting\nbackend=process\nservices=%s\n", joinServices(runCtx.config.EnabledServices()))
		if runCtx.config.ServiceEnabled(tinycloudconfig.ServiceManagement) {
			_, _ = fmt.Fprintf(stdout, "management=%s\n", runCtx.config.ManagementHTTPURL())
		}
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start tinycloudd: %w", err)
	}
	record.PID = cmd.Process.Pid
	if err := saveRuntimeRecord(runCtx.runtimeRoot, record); err != nil {
		_ = stopProcess(cmd.Process.Pid)
		return 1, err
	}
	defer removeRuntimeRecord(runCtx.runtimeRoot)

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("run tinycloudd: %w", err)
	}
	return 0, nil
}

func runStop(ctx cliContext, stdout io.Writer) (int, error) {
	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	if !ok {
		_, err := fmt.Fprintln(stdout, "runtime=stopped")
		return 0, err
	}
	if isProcessRunning(record.PID) {
		if err := stopProcess(record.PID); err != nil {
			return 1, fmt.Errorf("stop runtime PID %d: %w", record.PID, err)
		}
	}
	if err := removeRuntimeRecord(ctx.runtimeRoot); err != nil {
		return 1, err
	}
	_, err = fmt.Fprintf(stdout, "runtime=stopped\nbackend=%s\n", record.Backend)
	return 0, err
}

func runRestart(ctx cliContext, args []string, stdout, stderr io.Writer) (int, error) {
	detached := false
	explicitMode := false
	for _, arg := range args {
		switch arg {
		case "--detached", "-d":
			detached = true
			explicitMode = true
		case "--attached":
			detached = false
			explicitMode = true
		default:
			return 2, fmt.Errorf("unknown restart option %q", arg)
		}
	}

	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	if ok {
		if !explicitMode {
			detached = record.Detached
		}
		mergedEnv := copyMap(record.Env)
		for key, value := range ctx.env {
			if strings.HasPrefix(key, "TINYCLOUD_") || key == "GOCACHE" {
				mergedEnv[key] = value
			}
		}
		ctx.env = ensureBuildEnv(mergedEnv, ctx.repoRoot)
		ctx.config = tinycloudconfig.FromMap(ctx.env)
		if !filepath.IsAbs(ctx.config.DataRoot) {
			ctx.config.DataRoot = filepath.Clean(filepath.Join(ctx.cwd, ctx.config.DataRoot))
			ctx.env["TINYCLOUD_DATA_ROOT"] = ctx.config.DataRoot
		}
		if _, err := runStop(ctx, io.Discard); err != nil {
			return 1, err
		}
	}

	startArgs := []string{}
	if detached {
		startArgs = append(startArgs, "--detached")
	}
	return runStart(ctx, startArgs, stdout, stderr)
}

func runWait(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	timeout := defaultWaitTimeout
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case strings.HasPrefix(arg, "--timeout="):
			value := strings.TrimPrefix(arg, "--timeout=")
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return 2, fmt.Errorf("parse --timeout: %w", err)
			}
			timeout = parsed
		case arg == "--timeout":
			if i+1 >= len(args) {
				return 2, errors.New("wait --timeout requires a value")
			}
			i++
			parsed, err := time.ParseDuration(args[i])
			if err != nil {
				return 2, fmt.Errorf("parse --timeout: %w", err)
			}
			timeout = parsed
		default:
			return 2, fmt.Errorf("unknown wait option %q", arg)
		}
	}

	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	if !ok || !isProcessRunning(record.PID) {
		return 1, errors.New("TinyCloud is not running")
	}
	if err := waitForHealthy(record.Config, timeout); err != nil {
		return 1, err
	}
	_, err = fmt.Fprintln(stdout, "runtime=ready")
	return 0, err
}

func runLogs(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	follow := false
	for _, arg := range args {
		switch arg {
		case "--follow", "-f":
			follow = true
		default:
			return 2, fmt.Errorf("unknown logs option %q", arg)
		}
	}
	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	if !ok || record.LogPath == "" {
		return 1, errors.New("no active TinyCloud runtime log is available")
	}
	return 0, streamLog(record.LogPath, follow, stdout)
}

func runStatus(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	target := "runtime"
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "runtime", "services":
			target = arg
		case "--json":
			jsonOutput = true
		default:
			return 2, fmt.Errorf("unknown status option %q", arg)
		}
	}
	switch target {
	case "runtime":
		return statusRuntime(ctx, jsonOutput, stdout)
	case "services":
		return statusServices(ctx, jsonOutput, stdout)
	default:
		return 2, fmt.Errorf("unknown status target %q", target)
	}
}

func statusRuntime(ctx cliContext, jsonOutput bool, stdout io.Writer) (int, error) {
	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}

	status := map[string]any{
		"status":  "stopped",
		"backend": "process",
	}
	if ok {
		status["pid"] = record.PID
		status["detached"] = record.Detached
		status["startedAt"] = record.StartedAt
		status["logPath"] = record.LogPath
		status["services"] = record.Config.EnabledServices()
		if isProcessRunning(record.PID) {
			status["status"] = "running"
			if runtimeStatus, err := readRuntimeStatus(record.Config); err == nil {
				status["runtime"] = runtimeStatus
			} else {
				status["runtimeError"] = err.Error()
			}
		}
	}

	if jsonOutput {
		return 0, formatJSON(stdout, status)
	}
	_, err = fmt.Fprintf(stdout, "runtime=%s\nbackend=%v\n", status["status"], status["backend"])
	if pid, ok := status["pid"]; ok {
		_, err = fmt.Fprintf(stdout, "pid=%v\n", pid)
	}
	if services, ok := status["services"].([]tinycloudconfig.Service); ok {
		_, err = fmt.Fprintf(stdout, "services=%s\n", joinServices(services))
	}
	if logPath, ok := status["logPath"]; ok {
		_, err = fmt.Fprintf(stdout, "log=%v\n", logPath)
	}
	return 0, err
}

func statusServices(ctx cliContext, jsonOutput bool, stdout io.Writer) (int, error) {
	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}

	services := ctx.config.ServiceCatalog()
	if ok {
		services = record.Config.ServiceCatalog()
		if runtimeStatus, err := readRuntimeStatus(record.Config); err == nil {
			if rawServices, found := runtimeStatus["services"]; found {
				if jsonOutput {
					return 0, formatJSON(stdout, map[string]any{"services": rawServices})
				}
				return 0, printGenericServices(stdout, rawServices)
			}
		}
	}

	if jsonOutput {
		return 0, formatJSON(stdout, map[string]any{"services": services})
	}
	for _, service := range services {
		_, err = fmt.Fprintf(stdout, "name=%s enabled=%t family=%s endpoint=%s\n", service.Name, service.Enabled, service.Family, service.Endpoint)
		if err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func runConfig(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("config requires a subcommand")
	}
	switch args[0] {
	case "show":
		jsonOutput := hasJSONFlag(args[1:])
		record, ok, err := activeRuntime(ctx.runtimeRoot)
		if err != nil {
			return 1, err
		}
		cfg := ctx.config
		if ok {
			cfg = record.Config
		}
		if jsonOutput {
			return 0, formatJSON(stdout, configView(cfg))
		}
		return 0, printConfig(stdout, cfg)
	case "validate":
		if err := ctx.config.Validate(); err != nil {
			return 1, err
		}
		_, err := fmt.Fprintln(stdout, "config=valid")
		return 0, err
	default:
		return 2, fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func runServices(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("services requires a subcommand")
	}
	switch args[0] {
	case "list":
		return statusServices(ctx, hasJSONFlag(args[1:]), stdout)
	case "enable":
		return updateServices(ctx, args[1:], true, stdout)
	case "disable":
		return updateServices(ctx, args[1:], false, stdout)
	default:
		return 2, fmt.Errorf("unknown services subcommand %q", args[0])
	}
}

func updateServices(ctx cliContext, values []string, enable bool, stdout io.Writer) (int, error) {
	if len(values) == 0 {
		return 2, errors.New("services enable/disable requires one or more service names or families")
	}
	selection := tinycloudconfig.ParseServiceSelection(strings.Join(values, ","))
	if err := selection.Validate(); err != nil {
		return 1, err
	}

	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	activeConfig := ctx.config
	if ok {
		activeConfig = record.Config
		ctx.env = copyMap(record.Env)
	}

	current := map[tinycloudconfig.Service]struct{}{}
	for _, service := range activeConfig.EnabledServices() {
		current[service] = struct{}{}
	}
	for _, service := range selection.Names() {
		if enable {
			current[service] = struct{}{}
		} else {
			delete(current, service)
		}
	}

	ctx.env = ensureBuildEnv(ctx.env, ctx.repoRoot)
	ctx.env["TINYCLOUD_SERVICES"] = joinServices(serviceSetNames(current))
	if len(current) == 0 {
		ctx.env["TINYCLOUD_SERVICES"] = "none"
	}
	if err := saveStoredEnv(ctx.runtimeRoot, ctx.env); err != nil {
		return 1, err
	}

	_, err = fmt.Fprintf(stdout, "services=%s\n", ctx.env["TINYCLOUD_SERVICES"])
	if ok && isProcessRunning(record.PID) {
		_, err = fmt.Fprintln(stdout, "restartRequired=true")
		_, err = fmt.Fprintln(stdout, "next=tinycloud restart")
	}
	return 0, err
}

func runEndpoints(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	jsonOutput := hasJSONFlag(args)
	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	cfg := ctx.config
	if ok {
		cfg = record.Config
	}
	endpoints := cfg.EndpointMap()
	if jsonOutput {
		return 0, formatJSON(stdout, endpoints)
	}
	keys := make([]string, 0, len(endpoints))
	for key := range endpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(stdout, "%s=%s\n", key, endpoints[key]); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func activeRuntime(runtimeRoot string) (runtimeRecord, bool, error) {
	record, err := loadRuntimeRecord(runtimeRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runtimeRecord{}, false, nil
		}
		return runtimeRecord{}, false, err
	}
	if !isProcessRunning(record.PID) {
		_ = removeRuntimeRecord(runtimeRoot)
		return runtimeRecord{}, false, nil
	}
	return record, true, nil
}

func streamLog(path string, follow bool, stdout io.Writer) error {
	var offset int64
	for {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			file.Close()
			return err
		}
		written, err := io.Copy(stdout, file)
		file.Close()
		if err != nil {
			return err
		}
		offset += written
		if !follow {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func printGenericServices(stdout io.Writer, raw any) error {
	services, ok := raw.([]any)
	if !ok {
		return formatJSON(stdout, raw)
	}
	for _, item := range services {
		record, ok := item.(map[string]any)
		if !ok {
			if err := formatJSON(stdout, raw); err != nil {
				return err
			}
			return nil
		}
		_, err := fmt.Fprintf(stdout, "name=%v enabled=%v family=%v endpoint=%v\n", record["name"], record["enabled"], record["family"], record["endpoint"])
		if err != nil {
			return err
		}
	}
	return nil
}

func printConfig(stdout io.Writer, cfg tinycloudconfig.Config) error {
	lines := []string{
		"listenHost=" + cfg.ListenHost,
		"advertiseHost=" + cfg.AdvertiseHost,
		"dataRoot=" + cfg.DataRoot,
		"managementHttpPort=" + cfg.ManagementHTTP,
		"managementHttpsPort=" + cfg.ManagementTLS,
		"blobPort=" + cfg.Blob,
		"queuePort=" + cfg.Queue,
		"tablePort=" + cfg.Table,
		"keyVaultPort=" + cfg.KeyVault,
		"serviceBusPort=" + cfg.ServiceBus,
		"appConfigPort=" + cfg.AppConfig,
		"cosmosPort=" + cfg.Cosmos,
		"dnsPort=" + cfg.DNS,
		"eventHubsPort=" + cfg.EventHubs,
		"services=" + joinServices(cfg.EnabledServices()),
	}
	_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	return err
}

func configView(cfg tinycloudconfig.Config) map[string]any {
	return map[string]any{
		"listenHost":          cfg.ListenHost,
		"advertiseHost":       cfg.AdvertiseHost,
		"dataRoot":            cfg.DataRoot,
		"managementHttpPort":  cfg.ManagementHTTP,
		"managementHttpsPort": cfg.ManagementTLS,
		"blobPort":            cfg.Blob,
		"queuePort":           cfg.Queue,
		"tablePort":           cfg.Table,
		"keyVaultPort":        cfg.KeyVault,
		"serviceBusPort":      cfg.ServiceBus,
		"appConfigPort":       cfg.AppConfig,
		"cosmosPort":          cfg.Cosmos,
		"dnsPort":             cfg.DNS,
		"eventHubsPort":       cfg.EventHubs,
		"tenantId":            cfg.TenantID,
		"subscriptionId":      cfg.SubscriptionID,
		"services":            cfg.EnabledServices(),
	}
}

func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func serviceSetNames(values map[tinycloudconfig.Service]struct{}) []tinycloudconfig.Service {
	names := make([]tinycloudconfig.Service, 0, len(values))
	for _, service := range []tinycloudconfig.Service{
		tinycloudconfig.ServiceManagement,
		tinycloudconfig.ServiceBlob,
		tinycloudconfig.ServiceQueue,
		tinycloudconfig.ServiceTable,
		tinycloudconfig.ServiceKeyVault,
		tinycloudconfig.ServiceServiceBus,
		tinycloudconfig.ServiceAppConfig,
		tinycloudconfig.ServiceCosmos,
		tinycloudconfig.ServiceDNS,
		tinycloudconfig.ServiceEventHubs,
	} {
		if _, ok := values[service]; ok {
			names = append(names, service)
		}
	}
	return names
}

func joinServices(services []tinycloudconfig.Service) string {
	if len(services) == 0 {
		return "none"
	}
	values := make([]string, 0, len(services))
	for _, service := range services {
		values = append(values, string(service))
	}
	return strings.Join(values, ",")
}

func sortedEndpointLines(endpoints map[string]string) []string {
	keys := make([]string, 0, len(endpoints))
	for key := range endpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("endpoint.%s=%s", key, endpoints[key]))
	}
	return lines
}

func PrintUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `tinycloud commands:
  start [--detached] [--services <list>] [--json]
  stop
  restart [--detached|--attached]
  wait [--timeout <duration>]
  logs [-f]
  status [runtime|services] [--json]
  config show [--json]
  config validate
  services list [--json]
  services enable <names...>
  services disable <names...>
  init
  reset
  endpoints [--json]
  snapshot create [path]
  snapshot restore <path>
  seed apply <path>
  env terraform
  env pulumi`)
}
