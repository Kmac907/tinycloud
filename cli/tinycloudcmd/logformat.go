package tinycloudcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

type flushWriter interface {
	io.Writer
	Flush() error
}

type structuredLogWriter struct {
	dst     io.Writer
	ui      terminalUI
	mu      sync.Mutex
	pending bytes.Buffer
}

func newStructuredLogWriter(dst io.Writer, ui terminalUI) io.Writer {
	if !ui.interactive {
		return dst
	}
	return &structuredLogWriter{
		dst: dst,
		ui:  ui,
	}
}

func (w *structuredLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.pending.Write(p); err != nil {
		return 0, err
	}

	for {
		line, err := w.pending.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				_, _ = w.pending.WriteString(line)
				return len(p), nil
			}
			return 0, err
		}
		if err := w.writeLine(strings.TrimRight(line, "\r\n")); err != nil {
			return 0, err
		}
	}
}

func (w *structuredLogWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pending.Len() == 0 {
		return nil
	}
	line := strings.TrimRight(w.pending.String(), "\r\n")
	w.pending.Reset()
	return w.writeLine(line)
}

func (w *structuredLogWriter) writeLine(line string) error {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	if rendered, ok := renderStructuredLogLine(w.ui, line); ok {
		return writeString(w.dst, rendered)
	}
	_, err := fmt.Fprintln(w.dst, line)
	return err
}

func renderStructuredLogLine(ui terminalUI, line string) (string, bool) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return "", false
	}

	switch {
	case isHTTPRequestLog(raw):
		return renderRequestLogSection(ui, raw), true
	case isRuntimeEventLog(raw):
		return renderRuntimeEventSection(ui, raw), true
	default:
		return renderGenericLogSection(ui, raw), true
	}
}

func isHTTPRequestLog(raw map[string]any) bool {
	message := strings.TrimSpace(fmt.Sprint(raw["message"]))
	if message == "http request" {
		return true
	}
	_, hasMethod := raw["method"]
	_, hasPath := raw["path"]
	_, hasStatus := raw["status"]
	return hasMethod && hasPath && hasStatus
}

func isRuntimeEventLog(raw map[string]any) bool {
	if strings.TrimSpace(fmt.Sprint(raw["message"])) == "tinycloud server starting" {
		return true
	}
	for _, key := range []string{
		"addr",
		"managementTLSAddr",
		"blobAddr",
		"queueAddr",
		"tableAddr",
		"keyVaultAddr",
		"serviceBusAddr",
		"appConfigAddr",
		"cosmosAddr",
		"dnsAddr",
		"eventHubsAddr",
		"enabledServices",
		"disabledServices",
		"dataRoot",
	} {
		if _, ok := raw[key]; ok {
			return true
		}
	}
	return false
}

func renderRequestLogSection(ui terminalUI, raw map[string]any) string {
	items := [][2]string{}
	if value := strings.TrimSpace(fmt.Sprint(raw["method"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"method", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["path"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"path", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["status"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"status", renderHTTPStatus(ui, value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["duration"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"duration", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["remoteIP"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"remote ip", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["requestID"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"request id", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["level"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"level", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["ts"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"time", ui.active(value)})
	}
	return renderLogSection("Request", ui, items)
}

func renderRuntimeEventSection(ui terminalUI, raw map[string]any) string {
	items := [][2]string{}
	if value := strings.TrimSpace(fmt.Sprint(raw["message"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"message", ui.active(value)})
	}
	for _, field := range [][2]string{
		{"addr", "management"},
		{"managementTLSAddr", "management https"},
		{"blobAddr", "blob"},
		{"queueAddr", "queue"},
		{"tableAddr", "table"},
		{"keyVaultAddr", "key vault"},
		{"serviceBusAddr", "service bus"},
		{"appConfigAddr", "app config"},
		{"cosmosAddr", "cosmos"},
		{"dnsAddr", "dns"},
		{"eventHubsAddr", "event hubs"},
		{"dataRoot", "data root"},
	} {
		if value := strings.TrimSpace(fmt.Sprint(raw[field[0]])); value != "" && value != "<nil>" {
			items = append(items, [2]string{field[1], ui.active(value)})
		}
	}
	if value := joinAnyList(raw["enabledServices"]); value != "" {
		items = append(items, [2]string{"enabled services", ui.active(value)})
	}
	if value := joinAnyList(raw["disabledServices"]); value != "" {
		items = append(items, [2]string{"disabled services", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["level"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"level", ui.active(value)})
	}
	if value := strings.TrimSpace(fmt.Sprint(raw["ts"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"time", ui.active(value)})
	}
	return renderLogSection("Runtime Event", ui, items)
}

func renderGenericLogSection(ui terminalUI, raw map[string]any) string {
	title := "Log Event"
	if value := strings.TrimSpace(fmt.Sprint(raw["message"])); value != "" && value != "<nil>" {
		title = toTitleCase(value)
	}

	keys := make([]string, 0, len(raw))
	for key := range raw {
		if key == "message" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := [][2]string{}
	if value := strings.TrimSpace(fmt.Sprint(raw["message"])); value != "" && value != "<nil>" {
		items = append(items, [2]string{"message", ui.active(value)})
	}
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(raw[key]))
		if value == "" || value == "<nil>" {
			continue
		}
		if list := joinAnyList(raw[key]); list != "" && strings.HasPrefix(value, "[") {
			value = list
		}
		items = append(items, [2]string{normalizeLogFieldLabel(key), ui.active(value)})
	}
	return renderLogSection(title, ui, items)
}

func renderLogSection(title string, ui terminalUI, items [][2]string) string {
	return joinLines(
		title,
		strings.TrimRight(ui.keyValues(items), "\n"),
	) + "\n"
}

func renderHTTPStatus(ui terminalUI, value string) string {
	switch {
	case strings.HasPrefix(value, "2"):
		return ui.success(value)
	case strings.HasPrefix(value, "4"):
		return ui.warning(value)
	case strings.HasPrefix(value, "5"):
		return ui.failure(value)
	default:
		return ui.active(value)
	}
}

func joinAnyList(value any) string {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text == "" || text == "<nil>" {
			continue
		}
		values = append(values, text)
	}
	return strings.Join(values, ", ")
}

func normalizeLogFieldLabel(value string) string {
	switch value {
	case "requestID":
		return "request id"
	case "remoteIP":
		return "remote ip"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func toTitleCase(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return "Log Event"
	}
	for i, part := range parts {
		lower := strings.ToLower(part)
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
}

func flushLogWriter(w io.Writer) error {
	fw, ok := w.(flushWriter)
	if !ok {
		return nil
	}
	return fw.Flush()
}
