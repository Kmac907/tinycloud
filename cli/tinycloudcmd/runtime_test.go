package tinycloudcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tinycloud/runtime/tinycloudconfig"
)

func TestLoadRuntimeRecordRebuildsConfigFromEnv(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	record := runtimeRecord{
		Backend: "process",
		Env: map[string]string{
			"TINYCLOUD_SERVICES":        "management,storage",
			"TINYCLOUD_LISTEN_HOST":     "127.0.0.1",
			"TINYCLOUD_ADVERTISE_HOST":  "127.0.0.1",
			"TINYCLOUD_MGMT_HTTP_PORT":  "4566",
			"TINYCLOUD_MGMT_HTTPS_PORT": "4567",
			"TINYCLOUD_BLOB_PORT":       "4577",
			"TINYCLOUD_QUEUE_PORT":      "4578",
			"TINYCLOUD_TABLE_PORT":      "4579",
		},
	}
	if err := saveRuntimeRecord(runtimeRoot, record); err != nil {
		t.Fatalf("saveRuntimeRecord() error = %v", err)
	}

	loaded, err := loadRuntimeRecord(runtimeRoot)
	if err != nil {
		t.Fatalf("loadRuntimeRecord() error = %v", err)
	}

	if !loaded.Config.ServiceEnabled(tinycloudconfig.ServiceManagement) ||
		!loaded.Config.ServiceEnabled(tinycloudconfig.ServiceBlob) ||
		!loaded.Config.ServiceEnabled(tinycloudconfig.ServiceQueue) ||
		!loaded.Config.ServiceEnabled(tinycloudconfig.ServiceTable) {
		t.Fatalf("loaded config lost enabled services: %#v", loaded.Config.EnabledServices())
	}
}

func TestRunEServicesDisablePrintsRestartGuidanceWhenRuntimeActive(t *testing.T) {
	t.Parallel()

	repoRoot := repoRootFromTestFile(t)
	runtimeRoot := filepath.Join(t.TempDir(), "tinycloud-runtime")
	if err := saveStoredEnv(runtimeRoot, map[string]string{
		"TINYCLOUD_RUNTIME_ROOT": runtimeRoot,
	}); err != nil {
		t.Fatalf("saveStoredEnv() error = %v", err)
	}
	if err := saveRuntimeRecord(runtimeRoot, runtimeRecord{
		Backend:     "process",
		PID:         os.Getpid(),
		StartedAt:   "2026-04-10T00:00:00Z",
		RepoRoot:    repoRoot,
		RuntimeRoot: runtimeRoot,
		Env: map[string]string{
			"TINYCLOUD_RUNTIME_ROOT": runtimeRoot,
			"TINYCLOUD_SERVICES":     "management,storage,messaging",
		},
	}); err != nil {
		t.Fatalf("saveRuntimeRecord() error = %v", err)
	}

	oldRuntimeRoot := os.Getenv("TINYCLOUD_RUNTIME_ROOT")
	t.Cleanup(func() {
		if oldRuntimeRoot == "" {
			_ = os.Unsetenv("TINYCLOUD_RUNTIME_ROOT")
			return
		}
		_ = os.Setenv("TINYCLOUD_RUNTIME_ROOT", oldRuntimeRoot)
	})
	if err := os.Setenv("TINYCLOUD_RUNTIME_ROOT", runtimeRoot); err != nil {
		t.Fatalf("Setenv(TINYCLOUD_RUNTIME_ROOT) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, err := RunE([]string{"services", "disable", "storage"}, &stdout, &stderr, func() (string, error) {
		return repoRoot, nil
	})
	if err != nil {
		t.Fatalf("RunE() error = %v, stderr = %q", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want 0, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, fragment := range []string{
		"services=management,serviceBus,eventHubs",
		"restartRequired=true",
		"next=tinycloud restart",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}

func repoRootFromTestFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
