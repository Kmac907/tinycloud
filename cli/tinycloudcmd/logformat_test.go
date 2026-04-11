package tinycloudcmd

import (
	"strings"
	"testing"
)

func TestRenderStructuredLogLineFormatsRuntimeEventSection(t *testing.T) {
	t.Parallel()

	line := `{"addr":"0.0.0.0:4566","appConfigAddr":"0.0.0.0:4582","blobAddr":"0.0.0.0:4577","cosmosAddr":"0.0.0.0:4583","dataRoot":"/var/lib/tinycloud","disabledServices":[],"dnsAddr":"0.0.0.0:4584","enabledServices":["management","blob"],"eventHubsAddr":"0.0.0.0:4585","keyVaultAddr":"0.0.0.0:4580","level":"info","managementTLSAddr":"0.0.0.0:4567","message":"tinycloud server starting","queueAddr":"0.0.0.0:4578","serviceBusAddr":"0.0.0.0:4581","tableAddr":"0.0.0.0:4579","ts":"2026-04-11T10:06:17.994073525Z"}`

	output, ok := renderStructuredLogLine(terminalUI{}, line)
	if !ok {
		t.Fatal("renderStructuredLogLine() returned ok=false")
	}

	for _, fragment := range []string{
		"Runtime Event",
		"message",
		"● tinycloud server starting",
		"management",
		"management https",
		"enabled services",
		"management, blob",
		"data root",
		"/var/lib/tinycloud",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in:\n%s", fragment, output)
		}
	}
}

func TestRenderStructuredLogLineFormatsRequestSection(t *testing.T) {
	t.Parallel()

	line := `{"duration":"45.019µs","level":"info","message":"http request","method":"GET","path":"/_admin/healthz","remoteIP":"172.17.0.1:45714","requestID":"abc123","status":200,"ts":"2026-04-11T10:06:18.11988844Z"}`

	output, ok := renderStructuredLogLine(terminalUI{}, line)
	if !ok {
		t.Fatal("renderStructuredLogLine() returned ok=false")
	}

	for _, fragment := range []string{
		"Request",
		"method",
		"● GET",
		"path",
		"/_admin/healthz",
		"status",
		"✓ 200",
		"request id",
		"abc123",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in:\n%s", fragment, output)
		}
	}
}

func TestRenderStructuredLogLineFallsBackForNonJSON(t *testing.T) {
	t.Parallel()

	if _, ok := renderStructuredLogLine(terminalUI{}, "plain text line"); ok {
		t.Fatal("renderStructuredLogLine() unexpectedly formatted non-JSON line")
	}
}
