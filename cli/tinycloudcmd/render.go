package tinycloudcmd

import (
	"fmt"
	"sort"
	"strings"

	"tinycloud/runtime/tinycloudconfig"
)

type startSummary struct {
	RuntimeID  string
	Backend    string
	PID        int
	Container  string
	Image      string
	Services   string
	LogPath    string
	Management string
	Endpoints  map[string]string
}

type serviceStatusRow struct {
	Name     string
	Family   string
	Enabled  bool
	Health   string
	Endpoint string
}

func managementValue(cfg tinycloudconfig.Config) string {
	if cfg.ServiceEnabled(tinycloudconfig.ServiceManagement) {
		return cfg.ManagementHTTPURL()
	}
	return ""
}

func renderDetachedStartOutput(ui terminalUI, showBanner bool, backend string, summary startSummary, steps []string) string {
	lines := []string{}
	if showBanner {
		lines = append(lines, strings.TrimRight(ui.renderBanner(), "\n"))
	}
	lines = append(lines, fmt.Sprintf("TinyCloud CLI   backend: %s", backend), "", "Starting runtime")
	for _, step := range steps {
		lines = append(lines, "  "+step)
	}
	lines = append(lines, "", "Runtime")

	items := [][2]string{
		{"status", ui.success("running")},
		{"runtime id", ui.active(summary.RuntimeID)},
		{"backend", ui.active(summary.Backend)},
	}
	if summary.Container != "" {
		items = append(items, [2]string{"container", ui.active(summary.Container)})
	}
	if summary.Image != "" {
		items = append(items, [2]string{"image", ui.active(summary.Image)})
	}
	if summary.PID > 0 {
		items = append(items, [2]string{"pid", ui.active(fmt.Sprintf("%d", summary.PID))})
	}
	items = append(items, [2]string{"services", ui.active(summary.Services)})
	if summary.Management != "" {
		items = append(items, [2]string{"management", ui.active(summary.Management)})
	}
	if summary.LogPath != "" {
		items = append(items, [2]string{"log", ui.active(summary.LogPath)})
	}
	lines = append(lines, strings.TrimRight(ui.keyValues(items), "\n"))

	if len(summary.Endpoints) > 0 {
		lines = append(lines, "", "Endpoints")
		lines = append(lines, strings.TrimRight(ui.renderTable(table{
			headers: []string{"NAME", "URL"},
			rows:    endpointRows(summary.Endpoints),
		}), "\n"))
	}

	lines = append(lines, "", "Next", "  tinycloud status runtime", "  tinycloud status services", "  tinycloud logs -f", "  tinycloud stop")
	return joinLines(lines...)
}

func renderAttachedStartPrelude(ui terminalUI, showBanner bool, backend string, summary startSummary, steps []string) string {
	lines := []string{}
	if showBanner {
		lines = append(lines, strings.TrimRight(ui.renderBanner(), "\n"))
	}
	lines = append(lines, fmt.Sprintf("TinyCloud CLI   backend: %s", backend), "", "Starting runtime")
	for _, step := range steps {
		lines = append(lines, "  "+step)
	}
	lines = append(lines, "", "Runtime")
	items := [][2]string{
		{"backend", ui.active(summary.Backend)},
		{"services", ui.active(summary.Services)},
	}
	if summary.Management != "" {
		items = append(items, [2]string{"management", ui.active(summary.Management)})
	}
	if summary.Container != "" {
		items = append(items, [2]string{"container", ui.active(summary.Container)})
	}
	if summary.Image != "" {
		items = append(items, [2]string{"image", ui.active(summary.Image)})
	}
	if summary.PID > 0 {
		items = append(items, [2]string{"pid", ui.active(fmt.Sprintf("%d", summary.PID))})
	}
	lines = append(lines, strings.TrimRight(ui.keyValues(items), "\n"))
	return joinLines(lines...)
}

func renderRuntimeStatus(ui terminalUI, status map[string]any, cfg map[string]string) string {
	rows := [][]string{
		{"status", renderRuntimeState(ui, fmt.Sprint(status["status"]))},
		{"backend", ui.active(fmt.Sprint(status["backend"]))},
	}
	if value := strings.TrimSpace(fmt.Sprint(status["runtimeId"])); value != "" && value != "<nil>" {
		rows = append(rows, []string{"runtime id", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(status["container"])); value != "" && value != "<nil>" {
		rows = append(rows, []string{"container", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(status["image"])); value != "" && value != "<nil>" {
		rows = append(rows, []string{"image", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(status["pid"])); value != "" && value != "<nil>" && value != "0" {
		rows = append(rows, []string{"pid", ui.active(value)})
	}
	if services, ok := status["services"].([]string); ok && len(services) > 0 {
		rows = append(rows, []string{"services", ui.active(strings.Join(services, ","))})
	}
	if value := strings.TrimSpace(cfg["management"]); value != "" {
		rows = append(rows, []string{"management", ui.active(value)})
	}
	if value := strings.TrimSpace(cfg["dataRoot"]); value != "" {
		rows = append(rows, []string{"data root", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(status["startedAt"])); value != "" && value != "<nil>" {
		rows = append(rows, []string{"started at", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(status["runtimeError"])); value != "" && value != "<nil>" {
		rows = append(rows, []string{"error", ui.warning(value)})
	}
	return joinLines(
		"Runtime Status",
		"",
		strings.TrimRight(ui.renderTable(table{
			headers: []string{"FIELD", "VALUE"},
			rows:    rows,
		}), "\n"),
	)
}

func renderServicesStatus(ui terminalUI, rows []serviceStatusRow) string {
	enabledCount := 0
	disabledCount := 0
	failedCount := 0
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		switch row.Health {
		case "failed":
			failedCount++
		case "disabled":
			disabledCount++
		default:
			if row.Enabled {
				enabledCount++
			} else {
				disabledCount++
			}
		}
		tableRows = append(tableRows, []string{
			row.Name,
			renderServiceStatus(ui, row),
		})
	}
	return joinLines(
		fmt.Sprintf("Service Status   %d enabled   %d disabled   %d failed", enabledCount, disabledCount, failedCount),
		"",
		strings.TrimRight(ui.renderTable(table{
			headers: []string{"SERVICE", "STATUS"},
			rows:    tableRows,
		}), "\n"),
	)
}

func renderServicesList(ui terminalUI, rows []serviceStatusRow) string {
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{
			row.Name,
			renderEnabled(ui, row.Enabled),
			row.Family,
			row.Endpoint,
		})
	}
	return joinLines(
		"Services",
		"",
		strings.TrimRight(ui.renderTable(table{
			headers: []string{"SERVICE", "ENABLED", "FAMILY", "ENDPOINT"},
			rows:    tableRows,
		}), "\n"),
	)
}

func renderConfigShow(ui terminalUI, cfg map[string]string) string {
	runtimeItems := [][2]string{
		{"backend", ui.active(cfg["backend"])},
		{"data root", ui.active(cfg["dataRoot"])},
		{"listen host", ui.active(cfg["listenHost"])},
		{"advertise host", ui.active(cfg["advertiseHost"])},
	}
	portItems := [][2]string{
		{"management http", ui.active(cfg["managementHttpPort"])},
		{"management https", ui.active(cfg["managementHttpsPort"])},
		{"blob", ui.active(cfg["blobPort"])},
		{"queue", ui.active(cfg["queuePort"])},
		{"table", ui.active(cfg["tablePort"])},
		{"key vault", ui.active(cfg["keyVaultPort"])},
		{"service bus", ui.active(cfg["serviceBusPort"])},
		{"app config", ui.active(cfg["appConfigPort"])},
		{"cosmos", ui.active(cfg["cosmosPort"])},
		{"dns", ui.active(cfg["dnsPort"])},
		{"event hubs", ui.active(cfg["eventHubsPort"])},
	}
	serviceItems := [][2]string{
		{"enabled", ui.active(cfg["services"])},
	}
	return joinLines(
		"Configuration",
		"",
		"Runtime",
		strings.TrimRight(ui.keyValues(runtimeItems), "\n"),
		"",
		"Ports",
		strings.TrimRight(ui.keyValues(portItems), "\n"),
		"",
		"Services",
		strings.TrimRight(ui.keyValues(serviceItems), "\n"),
	)
}

func renderEndpoints(ui terminalUI, endpoints map[string]string) string {
	return joinLines(
		"Endpoints",
		"",
		strings.TrimRight(ui.renderTable(table{
			headers: []string{"NAME", "URL"},
			rows:    endpointRows(endpoints),
		}), "\n"),
	)
}

func renderStop(ui terminalUI, backend, identity string) string {
	items := [][2]string{{"backend", ui.active(backend)}}
	if identity != "" {
		items = append(items, [2]string{"runtime", ui.active(identity)})
	}
	items = append(items, [2]string{"result", ui.success("stopped")})
	return joinLines("Stopping TinyCloud", strings.TrimRight(ui.keyValues(items), "\n"))
}

func renderWait(ui terminalUI, backend string, timeout string) string {
	return joinLines(
		"Waiting For TinyCloud",
		strings.TrimRight(ui.keyValues([][2]string{
			{"backend", ui.active(backend)},
			{"timeout", ui.active(timeout)},
			{"result", ui.success("ready")},
		}), "\n"),
	)
}

func renderRestartHeading(ui terminalUI, backend string) string {
	return joinLines(
		"Restarting TinyCloud",
		strings.TrimRight(ui.keyValues([][2]string{{"backend", ui.active(backend)}}), "\n"),
	)
}

func renderServiceSelectionUpdated(ui terminalUI, services string, running bool) string {
	items := [][2]string{{"services", ui.active(services)}}
	if running {
		items = append(items,
			[2]string{"runtime", ui.active("running")},
			[2]string{"restart", ui.warning("required")},
		)
	}
	lines := []string{
		"Service Selection Updated",
		strings.TrimRight(ui.keyValues(items), "\n"),
	}
	if running {
		lines = append(lines, "", "Next", "  tinycloud restart", "  tinycloud status services")
	}
	return joinLines(lines...)
}

func endpointRows(endpoints map[string]string) [][]string {
	keys := make([]string, 0, len(endpoints))
	for key := range endpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, endpoints[key]})
	}
	return rows
}

func renderRuntimeState(ui terminalUI, value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "running", "ready":
		return ui.success(value)
	case "stopped", "disabled":
		return ui.inactive(value)
	case "failed", "error":
		return ui.failure(value)
	default:
		return ui.active(value)
	}
}

func renderHealth(ui terminalUI, value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ready", "running":
		return ui.success(value)
	case "disabled", "stopped":
		return ui.inactive(value)
	case "failed":
		return ui.failure(value)
	case "starting":
		return ui.progress(value)
	default:
		return ui.active(value)
	}
}

func renderEnabled(ui terminalUI, enabled bool) string {
	if enabled {
		return ui.success("yes")
	}
	return ui.inactive("no")
}

func renderServiceStatus(ui terminalUI, row serviceStatusRow) string {
	if !row.Enabled {
		return ui.inactive("disabled")
	}
	return renderHealth(ui, row.Health)
}
