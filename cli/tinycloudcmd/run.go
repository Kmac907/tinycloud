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

type startOptions struct {
	detached         bool
	jsonOutput       bool
	servicesOverride string
	backend          string
	env              []string
	publish          []string
	volumes          []string
	network          string
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
		return runStart(ctx, args[1:], stdout, stderr, true)
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

func parseStartOptions(args []string) (startOptions, error) {
	options := startOptions{}
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--detached" || arg == "-d":
			options.detached = true
		case arg == "--json":
			options.jsonOutput = true
		case arg == "--attached":
			options.detached = false
		case strings.HasPrefix(arg, "--services="):
			options.servicesOverride = strings.TrimSpace(strings.TrimPrefix(arg, "--services="))
		case arg == "--services":
			if i+1 >= len(args) {
				return startOptions{}, errors.New("start --services requires a value")
			}
			i++
			options.servicesOverride = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--backend="):
			options.backend = strings.TrimSpace(strings.TrimPrefix(arg, "--backend="))
		case arg == "--backend":
			if i+1 >= len(args) {
				return startOptions{}, errors.New("start --backend requires a value")
			}
			i++
			options.backend = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--env="):
			options.env = append(options.env, strings.TrimSpace(strings.TrimPrefix(arg, "--env=")))
		case arg == "--env" || arg == "-e":
			if i+1 >= len(args) {
				return startOptions{}, errors.New("start --env requires KEY=VALUE")
			}
			i++
			options.env = append(options.env, strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--publish="):
			options.publish = append(options.publish, strings.TrimSpace(strings.TrimPrefix(arg, "--publish=")))
		case arg == "--publish" || arg == "-p":
			if i+1 >= len(args) {
				return startOptions{}, errors.New("start --publish requires a value")
			}
			i++
			options.publish = append(options.publish, strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--volume="):
			options.volumes = append(options.volumes, strings.TrimSpace(strings.TrimPrefix(arg, "--volume=")))
		case arg == "--volume" || arg == "-v":
			if i+1 >= len(args) {
				return startOptions{}, errors.New("start --volume requires a value")
			}
			i++
			options.volumes = append(options.volumes, strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--network="):
			options.network = strings.TrimSpace(strings.TrimPrefix(arg, "--network="))
		case arg == "--network":
			if i+1 >= len(args) {
				return startOptions{}, errors.New("start --network requires a value")
			}
			i++
			options.network = strings.TrimSpace(args[i])
		default:
			return startOptions{}, fmt.Errorf("unknown start option %q", arg)
		}
	}
	return options, nil
}

func resolveRuntimeBackend(flagValue string, env map[string]string) string {
	if value := strings.TrimSpace(flagValue); value != "" {
		return value
	}
	if value := strings.TrimSpace(env["TINYCLOUD_BACKEND"]); value != "" {
		return value
	}
	if dockerAvailable() {
		return "docker"
	}
	return "process"
}

func runStart(ctx cliContext, args []string, stdout, stderr io.Writer, showBanner bool) (int, error) {
	options, err := parseStartOptions(args)
	if err != nil {
		return 2, err
	}
	ui := newTerminalUI(stdout)

	if record, ok, err := activeRuntime(ctx.runtimeRoot); err == nil && ok {
		return 1, fmt.Errorf("TinyCloud is already running via %s", record.Backend)
	}

	runCtx := ctx
	runCtx.env = copyMap(ctx.env)
	for _, value := range options.env {
		key, raw, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return 2, fmt.Errorf("start --env requires KEY=VALUE, got %q", value)
		}
		if strings.HasPrefix(key, "TINYCLOUD_") {
			runCtx.env[key] = raw
		}
	}
	if options.servicesOverride != "" {
		runCtx.env["TINYCLOUD_SERVICES"] = options.servicesOverride
	}
	runCtx.config = tinycloudconfig.FromMap(runCtx.env)
	if !filepath.IsAbs(runCtx.config.DataRoot) {
		runCtx.config.DataRoot = filepath.Clean(filepath.Join(ctx.cwd, runCtx.config.DataRoot))
		runCtx.env["TINYCLOUD_DATA_ROOT"] = runCtx.config.DataRoot
	}
	if err := runCtx.config.Validate(); err != nil {
		return 1, err
	}
	if err := runCtx.config.RequireServices(); err != nil {
		return 1, err
	}
	backend := resolveRuntimeBackend(options.backend, runCtx.env)
	if backend == "" {
		return 1, errors.New("could not determine a TinyCloud runtime backend")
	}
	if backend == "docker" {
		return startDockerRuntime(runCtx, dockerRuntime{
			Image:         strings.TrimSpace(runCtx.env["TINYCLOUD_DOCKER_IMAGE"]),
			ContainerName: dockerContainerName(runCtx.runtimeRoot),
			Network:       options.network,
			Env:           append([]string(nil), options.env...),
			Publish:       append([]string(nil), options.publish...),
			Volumes:       append([]string(nil), options.volumes...),
		}, options.detached, options.jsonOutput, stdout, showBanner)
	}

	binaryPath, err := buildTinyClouddBinary(runCtx.repoRoot, runCtx.runtimeRoot, runCtx.env)
	if err != nil {
		return 1, err
	}
	logPath := filepath.Join(runCtx.runtimeRoot, "tinycloudd.log")

	if options.detached {
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
		if options.jsonOutput {
			return 0, formatJSON(stdout, summary)
		}
		return 0, writeString(stdout, renderDetachedStartOutput(ui, showBanner, "process", startSummary{
			RuntimeID:  fmt.Sprintf("process:%d", cmd.Process.Pid),
			Backend:    "process",
			PID:        cmd.Process.Pid,
			Services:   joinServices(runCtx.config.EnabledServices()),
			LogPath:    logPath,
			Container:  "",
			Image:      "",
			Management: managementValue(runCtx.config),
			Endpoints:  runCtx.config.EndpointMap(),
		}, []string{ui.success("build binary"), ui.success("start daemon"), ui.success("wait for health")}))
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
	logWriter := stdout
	if !options.jsonOutput {
		logWriter = newStructuredLogWriter(stdout, ui)
	}
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if options.jsonOutput {
		if err := formatJSON(stdout, map[string]any{
			"status":   "starting",
			"backend":  "process",
			"services": runCtx.config.EnabledServices(),
		}); err != nil {
			return 1, err
		}
	} else {
		if err := writeString(stdout, renderAttachedStartPrelude(ui, showBanner, "process", startSummary{
			Backend:    "process",
			Services:   joinServices(runCtx.config.EnabledServices()),
			Management: managementValue(runCtx.config),
		}, []string{ui.success("build binary"), ui.progress("start daemon"), ui.progress("stream logs")})); err != nil {
			return 1, err
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
		if flushErr := flushLogWriter(logWriter); flushErr != nil {
			return 1, flushErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("run tinycloudd: %w", err)
	}
	if err := flushLogWriter(logWriter); err != nil {
		return 1, err
	}
	return 0, nil
}

func runStop(ctx cliContext, stdout io.Writer) (int, error) {
	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	if !ok {
		ui := newTerminalUI(stdout)
		return 0, writeString(stdout, renderStop(ui, "none", ""))
	}
	switch record.Backend {
	case "", "process":
		if err := stopProcess(record.PID); err != nil {
			return 1, fmt.Errorf("stop runtime PID %d: %w", record.PID, err)
		}
	case "docker":
		if err := stopDockerRuntime(record); err != nil {
			return 1, err
		}
	default:
		return 1, fmt.Errorf("unsupported runtime backend %q", record.Backend)
	}
	if err := removeRuntimeRecord(ctx.runtimeRoot); err != nil {
		return 1, err
	}
	ui := newTerminalUI(stdout)
	identity := ""
	switch record.Backend {
	case "docker":
		if record.Docker != nil {
			identity = record.Docker.ContainerName
		}
	case "process", "":
		if record.PID > 0 {
			identity = fmt.Sprintf("process:%d", record.PID)
		}
	}
	return 0, writeString(stdout, renderStop(ui, record.Backend, identity))
}

func runRestart(ctx cliContext, args []string, stdout, stderr io.Writer) (int, error) {
	ui := newTerminalUI(stdout)
	detached := false
	explicitMode := false
	backend := ""
	var restartDocker *dockerRuntime
	for _, arg := range args {
		switch arg {
		case "--detached", "-d":
			detached = true
			explicitMode = true
		case "--attached":
			detached = false
			explicitMode = true
		case "--backend=process":
			backend = "process"
		case "--backend=docker":
			backend = "docker"
		default:
			return 2, fmt.Errorf("unknown restart option %q", arg)
		}
	}

	record, ok, err := activeRuntime(ctx.runtimeRoot)
	if err != nil {
		return 1, err
	}
	if ok {
		if err := writeString(stdout, renderRestartHeading(ui, record.Backend)); err != nil {
			return 1, err
		}
		if !explicitMode {
			detached = record.Detached
		}
		if backend == "" {
			backend = record.Backend
		}
		restartDocker = record.Docker
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
	if backend != "" {
		startArgs = append(startArgs, "--backend="+backend)
	}
	if restartDocker != nil {
		for _, value := range restartDocker.Env {
			startArgs = append(startArgs, "--env="+value)
		}
		for _, value := range restartDocker.Publish {
			startArgs = append(startArgs, "--publish="+value)
		}
		for _, value := range restartDocker.Volumes {
			startArgs = append(startArgs, "--volume="+value)
		}
		if restartDocker.Network != "" {
			startArgs = append(startArgs, "--network="+restartDocker.Network)
		}
	}
	return runStart(ctx, startArgs, stdout, stderr, false)
}

func runWait(ctx cliContext, args []string, stdout io.Writer) (int, error) {
	ui := newTerminalUI(stdout)
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
	if !ok {
		return 1, errors.New("TinyCloud is not running")
	}
	if err := waitForHealthy(record.Config, timeout); err != nil {
		return 1, err
	}
	return 0, writeString(stdout, renderWait(ui, record.Backend, timeout.String()))
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
		if record.Backend == "docker" && record.Docker != nil {
			if ui := newTerminalUI(stdout); ui.interactive && follow {
				if err := writeString(stdout, joinLines(
					"Logs",
					strings.TrimRight(ui.keyValues([][2]string{
						{"backend", ui.active("docker")},
						{"container", ui.active(record.Docker.ContainerName)},
					}), "\n"),
				)); err != nil {
					return 1, err
				}
			}
			return 0, dockerLogs(record.Docker.ContainerName, follow, stdout)
		}
		return 1, errors.New("no active TinyCloud runtime log is available")
	}
	if ui := newTerminalUI(stdout); ui.interactive && follow {
		if err := writeString(stdout, joinLines(
			"Logs",
			strings.TrimRight(ui.keyValues([][2]string{
				{"backend", ui.active("process")},
				{"log", ui.active(record.LogPath)},
			}), "\n"),
		)); err != nil {
			return 1, err
		}
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
		"backend": resolveRuntimeBackend("", ctx.env),
	}
	if ok {
		status["backend"] = record.Backend
		if record.Backend == "process" && record.PID > 0 {
			status["runtimeId"] = fmt.Sprintf("process:%d", record.PID)
		}
		if record.Backend == "docker" && record.Docker != nil {
			status["runtimeId"] = firstNonEmpty(record.Docker.ContainerID, record.Docker.ContainerName)
		}
		status["pid"] = record.PID
		status["detached"] = record.Detached
		status["startedAt"] = record.StartedAt
		status["logPath"] = record.LogPath
		status["services"] = record.Config.EnabledServices()
		if record.Docker != nil {
			status["container"] = record.Docker.ContainerName
			status["containerId"] = record.Docker.ContainerID
			status["image"] = record.Docker.Image
		}
		status["status"] = "running"
		if runtimeStatus, err := readRuntimeStatus(record.Config); err == nil {
			status["runtime"] = runtimeStatus
		} else {
			status["runtimeError"] = err.Error()
		}
	}

	if jsonOutput {
		return 0, formatJSON(stdout, status)
	}
	ui := newTerminalUI(stdout)
	view := map[string]string{
		"management": managementValue(record.Config),
		"dataRoot":   record.Config.DataRoot,
	}
	if !ok {
		view["management"] = managementValue(ctx.config)
		view["dataRoot"] = ctx.config.DataRoot
	}
	if services, ok := status["services"].([]tinycloudconfig.Service); ok {
		names := make([]string, 0, len(services))
		for _, service := range services {
			names = append(names, string(service))
		}
		status["services"] = names
	}
	return 0, writeString(stdout, renderRuntimeStatus(ui, status, view))
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
				rows, ok := serviceRowsFromRaw(rawServices)
				if ok {
					return 0, writeString(stdout, renderServicesStatus(newTerminalUI(stdout), rows))
				}
			}
		}
	}

	if jsonOutput {
		return 0, formatJSON(stdout, map[string]any{"services": services})
	}
	ui := newTerminalUI(stdout)
	rows := make([]serviceStatusRow, 0, len(services))
	for _, service := range services {
		health := "disabled"
		if service.Enabled {
			health = "ready"
		}
		rows = append(rows, serviceStatusRow{
			Name:     string(service.Name),
			Family:   service.Family,
			Enabled:  service.Enabled,
			Health:   health,
			Endpoint: service.Endpoint,
		})
	}
	return 0, writeString(stdout, renderServicesStatus(ui, rows))
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
		ui := newTerminalUI(stdout)
		backend := resolveRuntimeBackend("", ctx.env)
		if ok {
			backend = record.Backend
		}
		return 0, writeString(stdout, renderConfigShow(ui, configViewStringMap(cfg, backend)))
	case "validate":
		if err := ctx.config.Validate(); err != nil {
			return 1, err
		}
		ui := newTerminalUI(stdout)
		return 0, writeString(stdout, joinLines(
			"Configuration",
			strings.TrimRight(ui.keyValues([][2]string{{"result", ui.success("valid")}}), "\n"),
		))
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

	ui := newTerminalUI(stdout)
	return 0, writeString(stdout, renderServiceSelectionUpdated(ui, ctx.env["TINYCLOUD_SERVICES"], ok))
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
	ui := newTerminalUI(stdout)
	return 0, writeString(stdout, renderEndpoints(ui, endpoints))
}

func activeRuntime(runtimeRoot string) (runtimeRecord, bool, error) {
	record, err := loadRuntimeRecord(runtimeRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runtimeRecord{}, false, nil
		}
		return runtimeRecord{}, false, err
	}
	running, err := runtimeRunning(record)
	if err != nil {
		return runtimeRecord{}, false, err
	}
	if !running {
		_ = removeRuntimeRecord(runtimeRoot)
		return runtimeRecord{}, false, nil
	}
	return record, true, nil
}

func streamLog(path string, follow bool, stdout io.Writer) error {
	var offset int64
	logWriter := newStructuredLogWriter(stdout, newTerminalUI(stdout))
	for {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			file.Close()
			return err
		}
		written, err := io.Copy(logWriter, file)
		file.Close()
		if err != nil {
			return err
		}
		offset += written
		if !follow {
			return flushLogWriter(logWriter)
		}
		time.Sleep(500 * time.Millisecond)
	}
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

func configViewStringMap(cfg tinycloudconfig.Config, backend string) map[string]string {
	return map[string]string{
		"backend":             backend,
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
		"services":            joinServices(cfg.EnabledServices()),
	}
}

func serviceRowsFromRaw(raw any) ([]serviceStatusRow, bool) {
	services, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	rows := make([]serviceStatusRow, 0, len(services))
	for _, item := range services {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		enabled, _ := record["enabled"].(bool)
		health := "disabled"
		if enabled {
			health = "ready"
		}
		rows = append(rows, serviceStatusRow{
			Name:     fmt.Sprint(record["name"]),
			Family:   fmt.Sprint(record["family"]),
			Enabled:  enabled,
			Health:   health,
			Endpoint: fmt.Sprint(record["endpoint"]),
		})
	}
	return rows, true
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
