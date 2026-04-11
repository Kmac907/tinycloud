package tinycloudcmd

import (
	"strings"
	"testing"

	"tinycloud/runtime/tinycloudconfig"
)

func TestRenderDetachedStartOutputIncludesBannerOnlyWhenRequested(t *testing.T) {
	t.Parallel()

	ui := terminalUI{interactive: true}
	summary := startSummary{
		RuntimeID:  "runtime-123",
		Backend:    "docker",
		Container:  "tinycloud-test",
		Image:      "tinycloud-azure",
		Services:   "management",
		Management: "http://127.0.0.1:4566",
		Endpoints: map[string]string{
			"management": "http://127.0.0.1:4566",
		},
	}

	withBanner := renderDetachedStartOutput(ui, true, "docker", summary, []string{"✓ build image"})
	if !strings.Contains(withBanner, tinyCloudBanner) {
		t.Fatalf("renderDetachedStartOutput() missing banner:\n%s", withBanner)
	}

	withoutBanner := renderDetachedStartOutput(ui, false, "docker", summary, []string{"✓ build image"})
	if strings.Contains(withoutBanner, tinyCloudBanner) {
		t.Fatalf("renderDetachedStartOutput() unexpectedly included banner:\n%s", withoutBanner)
	}
}

func TestRenderServicesStatusUsesTableLayout(t *testing.T) {
	t.Parallel()

	output := renderServicesStatus(terminalUI{}, []serviceStatusRow{
		{
			Name:     "management",
			Family:   "control-plane",
			Enabled:  true,
			Health:   "ready",
			Endpoint: "http://127.0.0.1:4566",
		},
		{
			Name:     "blob",
			Family:   "storage",
			Enabled:  false,
			Health:   "disabled",
			Endpoint: "http://127.0.0.1:4577",
		},
	})

	for _, fragment := range []string{
		"Service Status   1 enabled   1 disabled   0 failed",
		"SERVICE",
		"FAMILY",
		"management",
		"✓ yes",
		"○ disabled",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("renderServicesStatus() missing %q in:\n%s", fragment, output)
		}
	}
}

func TestTerminalUIColorsOnlyTheStatusIcon(t *testing.T) {
	t.Parallel()

	ui := terminalUI{color: true}
	got := ui.success("running")
	want := ansiGreen + "✓" + ansiReset + " running"
	if got != want {
		t.Fatalf("success() = %q, want %q", got, want)
	}

	got = ui.warning("required")
	want = ansiYellow + "‼" + ansiReset + " required"
	if got != want {
		t.Fatalf("warning() = %q, want %q", got, want)
	}
}

func TestConfigViewStringMapIncludesBackendAndServices(t *testing.T) {
	t.Parallel()

	cfg := tinycloudconfig.FromMap(map[string]string{
		"TINYCLOUD_SERVICES":        "management,storage",
		"TINYCLOUD_DATA_ROOT":       `C:\temp\tinycloud-data`,
		"TINYCLOUD_MGMT_HTTP_PORT":  "4566",
		"TINYCLOUD_MGMT_HTTPS_PORT": "4567",
	})

	view := configViewStringMap(cfg, "docker")
	if view["backend"] != "docker" {
		t.Fatalf("configViewStringMap()[backend] = %q, want docker", view["backend"])
	}
	if view["services"] != "management,blob,queue,table" {
		t.Fatalf("configViewStringMap()[services] = %q", view["services"])
	}
	if view["dataRoot"] != `C:\temp\tinycloud-data` {
		t.Fatalf("configViewStringMap()[dataRoot] = %q", view["dataRoot"])
	}
}

func TestServiceRowsFromRawBuildsHumanRows(t *testing.T) {
	t.Parallel()

	rows, ok := serviceRowsFromRaw([]any{
		map[string]any{
			"name":     "management",
			"family":   "control-plane",
			"enabled":  true,
			"endpoint": "http://127.0.0.1:4566",
		},
		map[string]any{
			"name":     "blob",
			"family":   "storage",
			"enabled":  false,
			"endpoint": "http://127.0.0.1:4577",
		},
	})
	if !ok {
		t.Fatal("serviceRowsFromRaw() returned ok=false")
	}
	if len(rows) != 2 {
		t.Fatalf("serviceRowsFromRaw() rows = %d, want 2", len(rows))
	}
	if rows[0].Health != "ready" || rows[1].Health != "disabled" {
		t.Fatalf("serviceRowsFromRaw() healths = %#v", rows)
	}
}
